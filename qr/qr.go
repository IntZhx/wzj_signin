package qr

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"wzj_signin/db"
	"wzj_signin/model"
)

var wsUrl string = "wss://www.teachermate.com.cn/faye"

func InitQrSign(courseId int, signId int) {
	// 启动一个独立 WS 连接监听二维码更新
	go Start(courseId, signId)
}

func extractQrUrlFromMessage(msg []byte) string {
	qrCodeUrl := ""
	var qrArr []model.QRCodeUrlData
	if err := json.Unmarshal(msg, &qrArr); err == nil {
		for _, it := range qrArr {
			if strings.TrimSpace(it.Data.QrURL) != "" {
				qrCodeUrl = strings.TrimSpace(it.Data.QrURL)
				break
			}
		}
	}
	if qrCodeUrl != "" {
		return qrCodeUrl
	}

	var single model.QRCodeUrlData
	if err := json.Unmarshal(msg, &single); err == nil {
		return strings.TrimSpace(single.Data.QrURL)
	}
	return ""
}

func receiveHandler(connection *websocket.Conn, done chan struct{}, signId int) {
	defer close(done)
	loggedOnce := false
	for {
		_, msg, err := connection.ReadMessage()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				// 如果外部误设置了 read deadline，超时不应导致整个监听退出
				continue
			}
			log.Println("Error in receive:", err)
			return
		}

		qrCodeUrl := extractQrUrlFromMessage(msg)
		if qrCodeUrl == "" {
			continue
		}
		if !loggedOnce {
			log.Println("QR url received:", signId, qrCodeUrl)
			loggedOnce = true
		}

		// 缓存拉长：避免用户稍晚打开页面 key 已过期
		result := db.RedisSet("wzj:qr:"+fmt.Sprint(signId), qrCodeUrl, 15*time.Minute)
		if result.Err() != nil {
			log.Println("Error setting key:", result.Err())
			return
		}
	}
}

func Start(courseId int, signId int) {
	done := make(chan struct{})
	log.Println("QR WS start:", "courseId=", courseId, "signId=", signId)

	conn, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		log.Println("Error connecting to Websocket Server:", err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	// 1) handshake
	handshake := `[{"channel":"/meta/handshake","version":"1.0","supportedConnectionTypes":["websocket"],"id":"1"}]`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(handshake)); err != nil {
		log.Println("Error during handshake write:", err)
		return
	}

	clientID, err := waitForClientID(conn, 6*time.Second)
	if err != nil {
		log.Println("Handshake failed:", err)
		return
	}
	log.Println("QR WS handshake ok:", "clientId=", clientID, "signId=", signId)
	// waitForClientID 会设置 ReadDeadline；握手成功后要清除，否则后续 ReadMessage 会很快 i/o timeout
	_ = conn.SetReadDeadline(time.Time{})

	// 2) subscribe
	subscribe := fmt.Sprintf(`[{"channel":"/meta/subscribe","clientId":"%s","subscription":"/attendance/%d/%d/qr","id":"2"}]`, clientID, courseId, signId)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(subscribe)); err != nil {
		log.Println("Error during subscribe write:", err)
		return
	}
	log.Println("QR WS subscribed:", "/attendance/", courseId, "/", signId, "/qr")
	// 不强依赖 subscribe ack（有的环境下 ack 会延迟），先启动接收
	go receiveHandler(conn, done, signId)

	// 3) connect loop（提高频率，降低首码/轮换延迟）
	var counter = 3
	connectTicker := time.NewTicker(1 * time.Second)
	defer connectTicker.Stop()

	for {
		select {
		case <-connectTicker.C:
			counter = counter + 1
			connectString := fmt.Sprintf(`[{"channel":"/meta/connect","clientId":"%s","connectionType":"websocket","id":"%d"}]`, clientID, counter)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(connectString)); err != nil {
				log.Println("Error during writing to websocket:", err)
				return
			}
		case <-done:
			return
		}
	}
}

func waitForClientID(conn *websocket.Conn, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// 读超时继续等
			if websocket.IsUnexpectedCloseError(err) {
				return "", err
			}
			continue
		}

		// handshake 响应通常是数组
		var arr []model.WSData
		if err := json.Unmarshal(msg, &arr); err == nil {
			for _, it := range arr {
				if it.Channel == "/meta/handshake" && it.Successful && strings.TrimSpace(it.ClientID) != "" {
					return strings.TrimSpace(it.ClientID), nil
				}
			}
			continue
		}

		var single model.WSData
		if err := json.Unmarshal(msg, &single); err == nil {
			if single.Channel == "/meta/handshake" && single.Successful && strings.TrimSpace(single.ClientID) != "" {
				return strings.TrimSpace(single.ClientID), nil
			}
		}
	}
	return "", fmt.Errorf("timeout waiting for clientId")
}
