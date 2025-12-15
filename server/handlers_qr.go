package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"wzj_signin/db"
	"wzj_signin/qr"
)

func PendingQRCodeHandler(c *gin.Context) {
	openId := c.Param("openId")
	val, err := db.RedisGet("wzj:qr:pending:" + openId).Result()
	if err != nil || strings.TrimSpace(val) == "" {
		c.JSON(http.StatusOK, gin.H{"signId": 0, "courseId": 0})
		return
	}

	// 读到就删除，避免重复弹窗
	_ = db.RedisDel("wzj:qr:pending:" + openId).Err()

	raw := strings.TrimSpace(val)
	courseId := 0
	signId := 0

	// 兼容三种格式："signId" / "courseId,signId" / "courseId:signId"
	sep := ""
	if strings.Contains(raw, ",") {
		sep = ","
	} else if strings.Contains(raw, ":") {
		sep = ":"
	}

	if sep != "" {
		parts := strings.Split(raw, sep)
		if len(parts) >= 2 {
			courseId, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
			signId, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
		}
	} else {
		signId, _ = strconv.Atoi(raw)
	}

	url := "/static/qr.html?sign=" + strconv.Itoa(signId)
	if courseId > 0 {
		url += "&course=" + strconv.Itoa(courseId)
	}
	url += "&v=" + strconv.FormatInt(time.Now().Unix(), 10)

	if signId > 0 {
		c.JSON(http.StatusOK, gin.H{"signId": signId, "courseId": courseId, "url": url})
		return
	}
	c.JSON(http.StatusOK, gin.H{"signId": 0, "courseId": courseId})
}

func QRCodeHandler(c *gin.Context) {
	signId := c.Param("signId")
	qrUrl, err := db.RedisGet("wzj:qr:" + signId).Result()
	if err != nil {
		if err != redis.Nil {
			log.Println("Error getting value for key:", err)
		}
		c.JSON(http.StatusOK, gin.H{"qrUrl": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"qrUrl": qrUrl})
}

// 允许二维码页在打开时主动触发一次 WS 监听（用于服务重启后、或用户较晚打开页面时）
// GET /qrws/start?courseId=1449049&signId=3854920
func StartQRCodeWSHandler(c *gin.Context) {
	courseIdStr := strings.TrimSpace(c.Query("courseId"))
	signIdStr := strings.TrimSpace(c.Query("signId"))

	courseId, _ := strconv.Atoi(courseIdStr)
	signId, _ := strconv.Atoi(signIdStr)
	if courseId <= 0 || signId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "message": "invalid courseId/signId"})
		return
	}

	go qr.InitQrSign(courseId, signId)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
