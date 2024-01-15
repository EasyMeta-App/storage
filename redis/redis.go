package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/EasyMeta-App/helper"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Ring

// RedisNil is redis nil
var RedisNil = redis.Nil

// cachePrefix
var cachePrefix = "%s"

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

// Prefix is set prefix
func Prefix(key string) {
	cachePrefix = fmt.Sprintf("%s:%%s", key)
}

// GetPrefix is get prefix
func GetPrefix() string {
	return cachePrefix
}

// GetCacheKey is get cache key
func GetCacheKey(key string) string {
	return fmt.Sprintf(cachePrefix, key)
}

// Expire
func Expire(key string, expiration time.Duration) *redis.BoolCmd {
	return GetRedis().Expire(context.Background(), GetCacheKey(key), expiration)
}

// Set
func Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return GetRedis().Set(context.Background(), GetCacheKey(key), value, expiration)
}

// JsonSet
func JsonSet(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	ctx := context.Background()
	redisObj := GetRedis()
	cacheKey := GetCacheKey(key)
	resp := redisObj.JSONSet(ctx, cacheKey, "$", value)
	if resp.Err() == nil {
		redisObj.Expire(ctx, cacheKey, expiration)
	}
	return resp
}

// IncrBy
func IncrBy(key string, value int64) (int64, error) {
	return GetRedis().IncrBy(context.Background(), GetCacheKey(key), value).Result()
}

// IncrByFloat
func IncrByFloat(key string, value float64) (float64, error) {
	return GetRedis().IncrByFloat(context.Background(), GetCacheKey(key), value).Result()
}

// DecrBy
func DecrBy(key string, value int64) (int64, error) {
	return GetRedis().DecrBy(context.Background(), GetCacheKey(key), value).Result()
}

// Get
func Get(key string) *redis.StringCmd {
	return GetRedis().Get(context.Background(), GetCacheKey(key))
}

// JsonSet
func JsonGet(key string, paths ...string) *redis.JSONCmd {
	return GetRedis().JSONGet(context.Background(), GetCacheKey(key), paths...)
}

// Has
func Has(key string) bool {
	_, err := Get(key).Result()
	return err != redis.Nil
}

// Del
func Del(keys ...string) *redis.IntCmd {
	newKeys := []string{}
	for _, k := range keys {
		newKeys = append(newKeys, GetCacheKey(k))
	}
	return GetRedis().Del(context.Background(), newKeys...)
}

// TTL
func TTL(key string) *redis.DurationCmd {
	return GetRedis().TTL(context.Background(), GetCacheKey(key))
}

// AnyDo
func AnyDo(name string, expiration time.Duration) int {
	ckey := fmt.Sprintf("EveryAnyDo:%s:%s", name, expiration.String())
	icr, err := IncrBy(ckey, 1)
	if err != nil {
		return 0
	}
	if icr == 1 {
		Expire(ckey, expiration)
	}
	return int(icr)
}

// HourDo 每小时可执行次数
func HourDo(name string) int {
	return AnyDo(name, 1*time.Hour)
}

// DayDo 每天可执行次数
func DayDo(name string) int {
	return AnyDo(name, 24*time.Hour)
}

// MonthDo 每月可执行次数
func MonthDo(name string) int {
	return AnyDo(name, 30*24*time.Hour)
}
