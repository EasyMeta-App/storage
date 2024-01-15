package redis

import (
	"strconv"
	"strings"

	"github.com/EasyMeta-App/helper"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Ring

// 创建一个 redis 连接
func GetRedis() *redis.Ring {
	if redisClient != nil {
		return redisClient
	}
	redisDB, _ := strconv.Atoi(helper.Config("REDIS_DB"))
	redisServers := helper.Config("REDIS_SERVERS")
	redisAddrs := map[string]string{}
	for _, v := range strings.Split(redisServers, ",") {
		k := helper.MD5(v)
		redisAddrs[k] = strings.TrimSpace(v)
	}
	redisClient = redis.NewRing(&redis.RingOptions{
		Addrs:    redisAddrs,
		Password: helper.Config("REDIS_PASSWORD"), // no password set
		DB:       redisDB,                         // use default DB
	})
	return redisClient
}
