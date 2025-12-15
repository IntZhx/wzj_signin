package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func ServerInfoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"interval": viper.GetInt("app.interval"),
		"delay":    viper.GetInt("app.normal_delay"),
	})
}

func ServerNoticeHandler(c *gin.Context) {
	interval := viper.GetInt("app.interval")
	delay := viper.GetInt("app.normal_delay")
	c.JSON(http.StatusOK, gin.H{
		"notice": fmt.Sprintf("当前配置：查询间隔 %d 秒；GPS/普通签到延迟 %d 秒；二维码签到无延迟（检测到后会弹窗/邮件提醒）", interval, delay),
	})
}
