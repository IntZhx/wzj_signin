package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// server.go (Start() 函数内)

func Start() {
	r := gin.Default()
	// ... (保留 c.Header 的 Use 函数不变)

	// 1) 入口：根路径直接到主页（更简洁）
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/home")
	})

	// 2) 友好路由：不带 .html
	r.GET("/home", func(c *gin.Context) { c.File("./static/home.html") })
	r.GET("/home/", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/home") })

	r.GET("/submit", func(c *gin.Context) { c.File("./static/submit.html") })
	r.GET("/submit/", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/submit") })

	r.GET("/history", func(c *gin.Context) { c.File("./static/history.html") })
	r.GET("/history/", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/history") })

	r.GET("/settings", func(c *gin.Context) { c.File("./static/settings.html") })
	r.GET("/settings/", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/settings") })

	// 3) 静态资源（CSS/JS/qr.html 等）
	// 注意：Gin 不允许同时存在 /home 与 /home/*filepath，因此静态资源放到 /static
	r.Static("/static", "./static") // 使用相对路径

	// 4. API 路由保持不变
	r.POST("/register", RegisterOpenIDHandler)
	r.GET("/openids", OpenIdsHandler)
	r.GET("/qr/:signId", QRCodeHandler)
	r.GET("/qrws/start", StartQRCodeWSHandler)
	r.GET("/pendingqr/:openId", PendingQRCodeHandler)
	r.GET("/api/appconfig", GetAppConfigHandler)
	r.POST("/api/appconfig", UpdateAppConfigHandler)
	r.GET("/api/frontendsettings", GetFrontendSettingsHandler)
	r.POST("/api/frontendsettings", UpdateFrontendSettingsHandler)
	r.GET("/serverinfo", ServerInfoHandler)
	r.GET("/notice", ServerNoticeHandler)

	addr := viper.GetString("server.addr")
	if addr == "" {
		if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
			if strings.HasPrefix(port, ":") {
				addr = port
			} else {
				addr = ":" + port
			}
		}
	}
	if addr == "" {
		addr = ":8080"
	}
	r.Run(addr)
}
