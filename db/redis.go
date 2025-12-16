package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"wzj_signin/config"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

var (
	ctx         = context.Background()
	redisClient *redis.Client
)

func InitRedis() {
	_ = config.Load()

	redisAddress := viper.GetString("redis.address")
	if envAddress := os.Getenv("REDIS_ADDRESS"); envAddress != "" {
		redisAddress = envAddress
	}

	// 连接到 Redis 数据库
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})
	pong, err := redisClient.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Errorf("redis ping failed: %w", err))
	}
	fmt.Println(pong)
}

func RedisSet(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return redisClient.Set(ctx, key, value, expiration)
}

func RedisSetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return redisClient.SetNX(ctx, key, value, expiration)
}

// 后面要接上.Result()
func RedisGet(key string) *redis.StringCmd {
	return redisClient.Get(ctx, key)
}

func RedisExpire(key string, expiration time.Duration) *redis.BoolCmd {
	return redisClient.Expire(ctx, key, expiration)
}

func RedisDel(key string) *redis.IntCmd {
	return redisClient.Del(ctx, key)
}

func RedisGetAllMatchedKeys(pattern string) []string {
	result := []string{}
	iter := redisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		result = append(result, iter.Val())
	}
	if err := iter.Err(); err != nil {
		log.Println("Error scanning keys:", err)
		return nil
	}
	return result
}

func RedisLPush(key string, values ...interface{}) *redis.IntCmd {
	return redisClient.LPush(ctx, key, values...)
}

func RedisRPop(key string) *redis.StringCmd {
	return redisClient.RPop(ctx, key)
}

func RedisLTrim(key string, start, stop int64) *redis.StatusCmd {
	return redisClient.LTrim(ctx, key, start, stop)
}
