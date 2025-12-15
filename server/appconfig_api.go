package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"wzj_signin/config"
)

type appConfigUpdatePayload struct {
	NormalDelay int `json:"normal_delay"`
	Mail        struct {
		Enabled  bool   `json:"enabled"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		From     string `json:"from"`
	} `json:"mail"`
}

func GetAppConfigHandler(c *gin.Context) {
	cfg, err := config.GetForUI()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func UpdateAppConfigHandler(c *gin.Context) {
	var payload appConfigUpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式错误：" + err.Error()})
		return
	}

	updated := config.AppConfig{
		NormalDelay: payload.NormalDelay,
		Mail: config.MailConfig{
			Enabled:  payload.Mail.Enabled,
			Host:     payload.Mail.Host,
			Port:     payload.Mail.Port,
			Username: payload.Mail.Username,
			Password: payload.Mail.Password,
			From:     payload.Mail.From,
		},
	}

	// Minimal validation (avoid obviously wrong values)
	if updated.NormalDelay < 0 || updated.NormalDelay > 600 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "延迟时间范围不合法（0-600 秒）"})
		return
	}
	if updated.Mail.Port < 0 || updated.Mail.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮件端口范围不合法（0-65535）"})
		return
	}

	cfg, err := config.UpdateFromUI(updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}
