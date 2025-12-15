package main

import (
	"time"
	"wzj_signin/config"
	"wzj_signin/db"
	"wzj_signin/server"
	"wzj_signin/service"

	"github.com/spf13/viper"
)

func main() {
	if err := config.Load(); err != nil {
		panic(err)
	}
	db.InitRedis()
	go startTimer()
	server.Start()
}

func startTimer() {
	interval := viper.GetInt("app.interval")
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, openId := range db.RedisGetAllMatchedKeys("wzj:user:*") {
			openId := openId[9:]
			signList, _ := service.GetAllSigns(openId)
			for _, sign := range signList {
				go service.Signin(sign, openId)
			}
		}
	}
}
