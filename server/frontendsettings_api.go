package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"wzj_signin/config"
)

type frontendSettingsPayload struct {
	DefaultEmail string `json:"defaultEmail"`
	GpsLabels    []struct {
		Label    string `json:"label"`
		Location string `json:"location"`
	} `json:"gpsLabels"`
}

func GetFrontendSettingsHandler(c *gin.Context) {
	s, err := config.GetFrontendSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func UpdateFrontendSettingsHandler(c *gin.Context) {
	var payload frontendSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求数据格式错误：" + err.Error()})
		return
	}

	out := config.FrontendSettings{
		DefaultEmail: payload.DefaultEmail,
		GpsLabels:    make([]config.FrontendGpsLabel, 0, len(payload.GpsLabels)),
	}
	for _, it := range payload.GpsLabels {
		out.GpsLabels = append(out.GpsLabels, config.FrontendGpsLabel{Label: it.Label, Location: it.Location})
	}

	s, err := config.UpdateFrontendSettings(out)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}
