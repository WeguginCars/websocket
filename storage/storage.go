package storage

import (
	"wegugin/storage/mongosh"
	"wegugin/storage/repo"

	redisnosql "wegugin/storage/redis"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

type IStorage interface {
	TopCars() repo.ITopCarsStorage
	Redis() repo.IRedisStorage
	CloseRDB() error
}

type databaseStorage struct {
	mdb *mongo.Database
	rdb *redis.Client
}

func NewStorage(mdb *mongo.Database, rdb *redis.Client) IStorage {
	return &databaseStorage{
		mdb: mdb,
		rdb: rdb,
	}
}

func (p *databaseStorage) CloseRDB() error {
	err := p.rdb.Close()
	if err != nil {
		return err
	}
	return nil
}

func (p *databaseStorage) TopCars() repo.ITopCarsStorage {
	return mongosh.NewTopCarsRepository(p.mdb)
}

func (p *databaseStorage) Redis() repo.IRedisStorage {
	return redisnosql.NewRedisRepository(p.rdb)
}
