package repo

import (
	"context"
	"wegugin/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type IStorage interface {
	TopCars() ITopCarsStorage
	Close()
}

type ITopCarsStorage interface {
	CreateTopCar(ctx context.Context, topCar *model.TopCars) error
	GetListOfTopCars(ctx context.Context, filter model.TopCarsFilter) ([]*model.TopCars, error)
	DeleteFinishedTopCars(ctx context.Context) (int64, error)
	DeleteByID(ctx context.Context, id primitive.ObjectID) error
	DeleteByUserID(ctx context.Context, userID string) (int64, error)
	DeleteByCarID(ctx context.Context, carID string) (int64, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.TopCars, error)
	UpdateTopCar(ctx context.Context, id primitive.ObjectID, updateData bson.M) error
	CountTopCars(ctx context.Context, filter model.TopCarsFilter) (int64, error)
}

type IRedisStorage interface {
	StoreUserAsTyping(ctx context.Context, TyperId, UserId string) error
	GetStatus(ctx context.Context, TyperId, UserId string) (bool, error)
	DeleteStatus(ctx context.Context, TyperId string) error
}
