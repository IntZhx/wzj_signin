package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"wzj_signin/db"
	"wzj_signin/mail"
	"wzj_signin/model"
	"wzj_signin/qr"

	"github.com/spf13/viper"
)

var getAllSignsUrl = "https://v18.teachermate.cn/wechat-api/v1/class-attendance/student/active_signs"
var signInUrl = "https://v18.teachermate.cn/wechat-api/v1/class-attendance/student-sign-in"

func effectiveServerAddress() string {
	serverAddress := viper.GetString("app.url")
	if envAddress := os.Getenv("SERVER_ADDRESS"); envAddress != "" {
		serverAddress = envAddress
	}
	// 兼容通过 PORT 运行在非 8080 端口的情况（仅在未指定 SERVER_ADDRESS 时生效）
	if os.Getenv("SERVER_ADDRESS") == "" {
		if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
			serverAddress = strings.ReplaceAll(serverAddress, ":8080", ":"+port)
		}
	}
	return serverAddress
}

// 获取每一个OpenId的全部签到
func GetAllSigns(openId string) ([]model.SignData, error) {
	req, err := http.NewRequest("GET", getAllSignsUrl, nil)
	if err != nil {
		log.Println("Error creating GetAllSigns request:", err)
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Set("Openid", openId)
	req.Header.Set("Host", "v18.teachermate.cn")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error sending GetAllSigns request:", err)
		return nil, err
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	log.Println(openId+":GetAllSigns Response:", string(body))
	if string(body) == `{"message":"登录信息失效，请退出后重试"}` {
		// 1s后过期
		result := db.RedisExpire("wzj:user:"+openId, 1*time.Second)
		log.Println(openId + ":Invalid OpenId!")
		if result.Err() != nil {
			log.Println("Error setting key:", result.Err())
			return nil, result.Err()
		}
		return nil, errors.New("无效OpenId")
	}
	var signList []model.SignData
	json.Unmarshal(body, &signList)
	return signList, nil
}

// 辅助函数：从 Redis 获取用户自定义经纬度
// 期望格式: "经度,纬度" 例如 "113.399319,23.038859"
// 返回: lat(纬度), lon(经度), success
func GetUserLocation(openId string) (float64, float64, bool) {
	// 读取键名为 wzj:gps:openid
	val, err := db.RedisGet("wzj:gps:" + openId).Result()
	if err != nil || val == "" {
		return 0, 0, false
	}

	// 此时 val 应该是 "113.399319,23.038859"
	val = strings.ReplaceAll(val, "，", ",") // 兼容中文逗号
	parts := strings.Split(val, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}

	// parts[0] 是 Lon (经度)
	// parts[1] 是 Lat (纬度)
	lon, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lat, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)

	if err1 != nil || err2 != nil {
		log.Println("Error parsing user location:", val, err1, err2)
		return 0, 0, false
	}

	return lat, lon, true
}

// 提交签到
func Signin(sign model.SignData, openId string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// 生成0到1000之间的随机整数
	randomNum := r.Intn(1001)

	courseId := sign.CourseID
	signId := sign.SignID
	courseName := sign.Name

	// 1. 避免重复签到
	openidSign := fmt.Sprintf("wzj:repeat:%s%d", openId, signId)
	if _, err := db.RedisGet(openidSign).Result(); err == nil {
		log.Println(randomNum, "Repeated Sign", openId, signId)
		return
	}

	// 2. 二维码签到处理
	if sign.IsQR != 0 {
		serverAddress := effectiveServerAddress()
		qrPage := serverAddress + "/static/qr.html?sign=" + fmt.Sprint(signId) + "&course=" + fmt.Sprint(courseId) + "&v=" + fmt.Sprint(time.Now().Unix())
		mail_title := courseName + "正在二维码签到，需要手动完成"
		mail_content := "立刻点击下方二维码网址（或复制到浏览器打开），使用微信扫一扫完成签到。\n签到完成后之前提交的OpenID可能会立刻失效，如果需要再次监控需要重新添加新的OpenID到监控池。\n二维码页面：" + qrPage

		// 给前端一个可轮询的 pending 提示（方便弹窗/新标签页打开）
		_ = db.RedisSet("wzj:qr:pending:"+openId, fmt.Sprintf("%d,%d", courseId, signId), 10*time.Minute).Err()

		go qr.InitQrSign(courseId, signId)
		mail.SendEmail(mail_title, mail_content, FindEmailByOpenId(openId))
		CoolDownFor5Min(openId, signId)
	}

	// 3. 延时处理
	if sign.IsGPS == 1 || ((sign.IsGPS + sign.IsQR) == 0) {
		delay_time := viper.GetInt("app.normal_delay")
		log.Println(randomNum, "delay for", delay_time)
		time.Sleep(time.Duration(delay_time) * time.Second)
	}

	// 4. 再次检查重复
	if _, err := db.RedisGet(openidSign).Result(); err == nil {
		log.Println(randomNum, "Repeated Sign x2", openId, signId)
		return
	}

	// ================= GPS 核心逻辑：获取坐标 =================

	// 默认坐标 (来自配置文件 config.yml) - 需要你在 config.yml 的 app 下也配置 lat/lon
	lat := viper.GetFloat64("app.lat")
	lon := viper.GetFloat64("app.lon")

	// 尝试获取用户自定义坐标 (来自 Redis)
	userLat, userLon, ok := GetUserLocation(openId)
	if ok {
		lat = userLat
		lon = userLon
		log.Printf("[%d] Using User GPS: %s (Lat: %f, Lon: %f)", randomNum, openId, lat, lon)
	} else {
		log.Printf("[%d] Using Default GPS (Lat: %f, Lon: %f)", randomNum, lat, lon)
	}

	// 如果都没配置，防止报错，给一个默认值
	if lat == 0 || lon == 0 {
		// 采用之前硬编码的徐州默认值或你希望的任意值
		lat = 34.212723
		lon = 117.142737
	}

	// 坐标随机抖动 (防封号关键，参考了你的 checkin.go 逻辑)
	if sign.IsGPS == 1 {
		// 生成 -20 到 20 之间的随机微量
		offsetLat := float64(r.Intn(40)-20) * 0.000001
		offsetLon := float64(r.Intn(40)-20) * 0.000001
		lat += offsetLat
		lon += offsetLon
		log.Printf("[%d] GPS Jittered to (Lat: %f, Lon: %f)", randomNum, lat, lon)
	}

	// 格式化为字符串
	latStr := strconv.FormatFloat(lat, 'f', 6, 64)
	lonStr := strconv.FormatFloat(lon, 'f', 6, 64)

	// 构造请求体：双重保险，Body里也放经纬度
	var requestBody string
	if sign.IsGPS == 1 {
		requestBody = fmt.Sprintf(`{"courseId":%d,"signId":%d,"lat":%s,"lon":%s}`, courseId, signId, latStr, lonStr)
	} else {
		// 普通签到不携带经纬度
		requestBody = fmt.Sprintf(`{"courseId":%d,"signId":%d}`, courseId, signId)
	}

	// 创建请求
	data := strings.NewReader(requestBody)
	req, err := http.NewRequest("POST", signInUrl, data)
	if err != nil {
		log.Println(randomNum, "Error creating Signin request:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Set("Openid", openId)
	req.Header.Set("Host", "v18.teachermate.cn")
	req.Header.Set("Content-Type", "application/json")

	// 双重保险：Header 里也放经纬度
	if sign.IsGPS == 1 {
		req.Header.Set("lat", latStr)
		req.Header.Set("lon", lonStr)
	}

	// ================= 发送请求 =================
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error sending Signin request:", err)
		return
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	log.Println(randomNum, "Response:", string(body))

	if strings.Contains(string(body), "你已经签到成功") {
		CoolDownFor5Min(openId, signId)
	}

	if strings.Contains(string(body), "studentRank") {
		var signResult model.SignResultData
		json.Unmarshal(body, &signResult)
		mail_title := courseName + "刚刚签到！"
		mail_content := fmt.Sprintf("【签到No.%d】你是第%d个签到的！该消息仅供参考，签到结果以实际为准。[%s/C%d/S%d/%s]", signResult.SignRank, signResult.StudentRank, courseName, courseId, signId, openId)
		mail.SendEmail(mail_title, mail_content, FindEmailByOpenId(openId))
	}
}

func FindEmailByOpenId(openid string) string {
	email, err := db.RedisGet("wzj:user:" + openid).Result()
	if err != nil {
		log.Println("Error getting value for key:", err)
		return ""
	}
	return email
}

// 设置重复签到，五分钟冷却时间
func CoolDownFor5Min(openId string, signId int) {
	openidSign := fmt.Sprintf("wzj:repeat:%s%d", openId, signId)
	result := db.RedisSet(openidSign, signId, 5*time.Minute)
	if result.Err() != nil {
		log.Println("Error setting key:", result.Err())
		return
	}
}
