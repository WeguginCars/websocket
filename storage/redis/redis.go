package redis

import (
	"context"
	"fmt"
	"time"
	"wegugin/config"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

func ConnectDB() *redis.Client {
	conf := config.Load()
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.RDB_ADDRESS,
		Password: conf.Redis.RDB_PASSWORD,
		DB:       0,
	})

	return rdb
}
func StoreUserAsTyping(ctx context.Context, TyperId, UserId string) error {
	rdb := ConnectDB()

	err := rdb.Set(ctx, TyperId, UserId, 2*time.Minute).Err()
	if err != nil {
		return errors.Wrap(err, "failed to set user in Redis")
	}

	return nil
}

func GetStatus(ctx context.Context, TyperId, UserId string) (bool, error) {
	rdb := ConnectDB()
	code, err := rdb.Get(ctx, TyperId).Result()
	if err != nil {
		if err == redis.Nil {
			return false, fmt.Errorf("no status found for UserId: %s", TyperId)
		}
		return false, errors.Wrap(err, "failed to get code from Redis")
	}
	if UserId == code {
		return true, nil
	}
	return false, nil
}

func DeleteStatus(ctx context.Context, TyperId string) error {
	rdb := ConnectDB()
	err := rdb.Del(ctx, TyperId).Err()
	if err != nil {
		return errors.Wrap(err, "failed to delete user status from Redis")
	}
	return nil
}
