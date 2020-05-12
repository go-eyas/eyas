package redis

import (
	"sync"
	"time"

	"github.com/go-eyas/toolkit/log"
	"github.com/go-eyas/toolkit/types"
	"github.com/go-redis/redis"
)

var redisSetMu sync.Mutex

// StringValue redis 返回值
// type StringValue string

// // JSON 将redis值转成指定结构体
// func (val StringValue) JSON(v interface{}) error {
// 	bt := []byte(val)
// 	return util.ToStruct(bt, v)
// }

// // String 将redis值转成字符串
// func (val StringValue) String() string {
// 	return string(val)
// }

func (r *RedisClient) Expire(key string, expiration time.Duration) (bool, error) {
	c := r.Client.Expire(key, expiration)
	return c.Result()
}

// Get 获取字符串值
func (r *RedisClient) Get(key string) (string, error) {
	v, err := r.Client.Get(key).Result()
	if err == redis.Nil {
		err = nil
	}
	return v, err
}

// Set 设置字符串值，有效期默认 24 小时
func (r *RedisClient) Set(key string, value interface{}, expiration ...time.Duration) error {
	redisSetMu.Lock()
	defer redisSetMu.Unlock()
	expire := RedisTTL
	if len(expiration) > 0 {
		expire = expiration[0]
	}
	s := value
	cmd := r.Client.Set(key, s, expire)
	return cmd.Err()
}

// Del 删除键
func (r *RedisClient) Del(keys ...string) error {
	cmd := r.Client.Del(keys...)
	return cmd.Err()
}

// HGet 获取 Hash 的字段值
func (r *RedisClient) HGet(key string, field string) (string, error) {
	cmd := r.Client.HGet(key, field)
	log.Debugf("redis get hash key=%s, field=%s", key, field)
	v, err := cmd.Result()
	if err == redis.Nil {
		return "", nil
	}
	return v, err
}

// HGetAll 获取 Hash 的所有字段
func (r *RedisClient) HGetAll(key string) (map[string]string, error) {
	cmd := r.Client.HGetAll(key)
	log.Debugf("redis get all hash key=%s", key)
	v, err := cmd.Result()
	mp := make(map[string]string)
	if err == redis.Nil {
		return mp, nil
	}
	for k, sv := range v {
		mp[k] = sv
	}
	return mp, err
}

// HSet 设置hash值
func (r *RedisClient) HSet(key, field string, val interface{}, expiration ...time.Duration) error {
	redisSetMu.Lock()
	defer redisSetMu.Unlock()
	cmd := r.Client.HSet(key, field, val)

	expire := RedisTTL
	if len(expiration) > 0 {
		expire = expiration[0]
	}
	r.Expire(key, expire)
	log.Debugf("redis set hash key=%s, field=%s", key, field)
	return cmd.Err()
}

// HDel 删除hash的键
func (r *RedisClient) HDel(key string, field ...string) error {
	k := key
	cmd := r.Client.HDel(k, field...)
	log.Debugf("redis set hash key=%s, field=%s", k, field)
	err := cmd.Err()
	if err != nil {
		return err
	}
	// 是否键全删完了，如果是就清理掉这个key
	length, err := r.Client.HLen(k).Result()
	if err != nil {
		return err
	}
	if length == 0 {
		if err = r.Del(k); err != nil {
			return err
		}
	}

	return nil
}

type Message struct {
	Channel string
	Pattern string
	Payload string
}

// JSON 绑定json对象
func (msg *Message) JSON(v interface{}) error {
	return types.JSONString(msg.Payload).JSON(v)
}

// Sub 监听通道，有数据时触发回调 handler
// example:
// redis.Sub("chat")(func(msg *redis.Message) {
// 	fmt.Printf("receive message: %#v", msg)
// })
func (r *RedisClient) Sub(channel string, handler func(*Message)) {
	pb := r.Client.Subscribe(channel)
	ch := pb.Channel()

	for msg := range ch {
		handler(&Message{msg.Channel, msg.Pattern, msg.Payload})
	}

	defer pb.Close()
}

// Pub 发布事件
// example:
// Redis.Pub("chat", "this is a test message")
func (r *RedisClient) Pub(channel string, msg string) error {
	cmd := r.Client.Publish(channel, msg)
	_, err := cmd.Result()
	return err
}
