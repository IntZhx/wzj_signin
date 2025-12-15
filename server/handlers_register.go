package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"wzj_signin/db"
	"wzj_signin/model"
	"wzj_signin/service"
)

func RegisterOpenIDHandler(c *gin.Context) {
	var registerOpenIdData model.RegisterOpenIdData
	if err := c.ShouldBindJSON(&registerOpenIdData); err != nil {
		log.Println("Error binding JSON:", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式错误：" + err.Error()})
		return
	}

	openId := registerOpenIdData.OpenId
	value := registerOpenIdData.Value
	location := registerOpenIdData.Location

	// OpenID 设定 4 小时过期
	result := db.RedisSet("wzj:user:"+openId, value, 4*time.Hour)
	if result.Err() != nil {
		log.Println("Error setting wzj:user key:", result.Err())
		return
	}

	// 保存用户自定义经纬度（0 表示永不过期）
	if location != "" {
		err := db.RedisSet("wzj:gps:"+openId, location, 0).Err()
		if err != nil {
			log.Println("Error setting wzj:gps key:", err)
		} else {
			log.Println("Location saved for", openId, ":", location)
		}
	}

	// 验证 OpenID 并返回结果
	_, err := service.GetAllSigns(openId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "你提供的OpenId无效，请重新检查。"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "OpenId添加到监控池成功!"})
}

func OpenIdsHandler(c *gin.Context) {
	keys := db.RedisGetAllMatchedKeys("wzj:user:*")
	openIds := make([]string, 0, len(keys))
	for _, k := range keys {
		if strings.HasPrefix(k, "wzj:user:") {
			id := strings.TrimPrefix(k, "wzj:user:")
			if strings.TrimSpace(id) != "" {
				openIds = append(openIds, id)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"openIds": openIds, "count": len(openIds), "keys": keys})
}
