package redis

import (
	"context"
	"fmt"
	"time"
	"wegugin/config"
	"wegugin/storage/repo"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

func ConnectRDB() *redis.Client {
	conf := config.Load()
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.RDB_ADDRESS,
		Password: conf.Redis.RDB_PASSWORD,
		DB:       0,
	})

	return rdb
}

type RedisRepository struct {
	Rdb *redis.Client
}

func NewRedisRepository(rdb *redis.Client) repo.IRedisStorage {
	return &RedisRepository{Rdb: rdb}
}

func (s RedisRepository) StoreUserAsTyping(ctx context.Context, TyperId, UserId string) error {

	err := s.Rdb.Set(ctx, TyperId, UserId, 2*time.Minute).Err()
	if err != nil {
		return errors.Wrap(err, "failed to set user in Redis")
	}

	return nil
}

func (s RedisRepository) GetStatus(ctx context.Context, TyperId, UserId string) (bool, error) {
	code, err := s.Rdb.Get(ctx, TyperId).Result()
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

func (s RedisRepository) DeleteStatus(ctx context.Context, TyperId string) error {
	err := s.Rdb.Del(ctx, TyperId).Err()
	if err != nil {
		return errors.Wrap(err, "failed to delete user status from Redis")
	}
	return nil
}
