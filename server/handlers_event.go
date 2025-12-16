package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"wzj_signin/db"
)

// PendingEventHandler returns one pending event (FIFO) for the given openId.
// Events are produced by backend sign-in logic and consumed by the frontend history page.
func PendingEventHandler(c *gin.Context) {
	openId := strings.TrimSpace(c.Param("openId"))
	if openId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "missing openId"})
		return
	}

	val, err := db.RedisRPop("wzj:evt:" + openId).Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusOK, gin.H{"ok": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	val = strings.TrimSpace(val)
	if val == "" {
		c.JSON(http.StatusOK, gin.H{"ok": false})
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(val), &payload); err != nil {
		// If corrupted, just drop it.
		c.JSON(http.StatusOK, gin.H{"ok": false})
		return
	}

	payload["ok"] = true
	c.JSON(http.StatusOK, payload)
}
