package mongosh

import (
	"context"
	"errors"
	"time"

	"wegugin/model"
	"wegugin/storage/repo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TopCarsRepository struct {
	Coll *mongo.Collection
}

func NewTopCarsRepository(db *mongo.Database) repo.ITopCarsStorage {
	return &TopCarsRepository{Coll: db.Collection("topcars")}
}

func (r *TopCarsRepository) CreateTopCar(ctx context.Context, topCar *model.TopCars) error {
	// Avval shu car allaqachon active top carlar orasida borligini tekshirish
	existsFilter := bson.M{
		"car_id":      topCar.CarId,
		"category":    topCar.Category,
		"finished_at": bson.M{"$gt": time.Now()}, // hali muddati o'tmagan
		"deleted_at":  nil,                       // o'chirilmagan
	}

	count, err := r.Coll.CountDocuments(ctx, existsFilter)
	if err != nil {
		return err
	}

	if count > 0 {
		return errors.New("topCar already exists in this category and is still active")
	}

	if topCar.CreatedAt.IsZero() {
		topCar.CreatedAt = time.Now()
	}

	result, err := r.Coll.InsertOne(ctx, topCar)
	if err != nil {
		return err
	}

	topCar.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetListOfTopCars - filterlar bilan top carlar ro'yxatini olish
func (r *TopCarsRepository) GetListOfTopCars(ctx context.Context, filter model.TopCarsFilter) ([]*model.TopCars, error) {
	// Filter yaratish
	mongoFilter := bson.M{}

	// Deleted itemlarni ko'rsatmaslik (agar show_deleted false bo'lsa)
	if !filter.ShowDeleted {
		mongoFilter["deleted_at"] = nil
	}

	// Expired itemlarni ko'rsatmaslik (agar show_expired false bo'lsa)
	if !filter.ShowExpired {
		mongoFilter["finished_at"] = bson.M{"$gt": time.Now()}
	}

	// Category filter
	if filter.Category != "" {
		mongoFilter["category"] = filter.Category
	}

	// User ID filter
	if filter.UserID != "" {
		mongoFilter["user_id"] = filter.UserID
	}

	// Car ID filter
	if filter.CarID != "" {
		mongoFilter["car_id"] = filter.CarID
	}

	// Sort options
	var sort bson.D
	switch filter.SortBy {
	case "finished_at_asc":
		sort = bson.D{{Key: "finished_at", Value: 1}}
	case "finished_at_desc":
		sort = bson.D{{Key: "finished_at", Value: -1}}
	case "created_at_asc":
		sort = bson.D{{Key: "created_at", Value: 1}}
	case "created_at_desc":
		sort = bson.D{{Key: "created_at", Value: -1}}
	default:
		// Default: muddati tezroq tugaydiganlar birinchi
		sort = bson.D{{Key: "finished_at", Value: 1}}
	}

	// Limit va Skip
	if filter.Limit == 0 {
		filter.Limit = 50 // default limit
	}

	opts := options.Find().
		SetSort(sort).
		SetLimit(filter.Limit).
		SetSkip(filter.Skip)

	cursor, err := r.Coll.Find(ctx, mongoFilter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var topCars []*model.TopCars
	if err = cursor.All(ctx, &topCars); err != nil {
		return nil, err
	}

	return topCars, nil
}

// DeleteFinishedTopCars - muddati o'tgan top carlarni o'chirish
func (r *TopCarsRepository) DeleteFinishedTopCars(ctx context.Context) (int64, error) {
	filter := bson.M{
		"finished_at": bson.M{"$lt": time.Now()},
		"deleted_at":  nil,
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
		},
	}

	result, err := r.Coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

// DeleteByID - ID bo'yicha o'chirish
func (r *TopCarsRepository) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
		},
	}

	_, err := r.Coll.UpdateOne(ctx, filter, update)
	return err
}

// DeleteByUserID - User ID bo'yicha barcha top carlarni o'chirish
func (r *TopCarsRepository) DeleteByUserID(ctx context.Context, userID string) (int64, error) {
	filter := bson.M{
		"user_id":    userID,
		"deleted_at": nil,
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
		},
	}

	result, err := r.Coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

// DeleteByCarID - Car ID bo'yicha barcha top carlarni o'chirish
func (r *TopCarsRepository) DeleteByCarID(ctx context.Context, carID string) (int64, error) {
	filter := bson.M{
		"car_id":     carID,
		"deleted_at": nil,
	}

	update := bson.M{
		"$set": bson.M{
			"deleted_at": time.Now(),
		},
	}

	result, err := r.Coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

// GetByID - ID bo'yicha bitta top car olish
func (r *TopCarsRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.TopCars, error) {
	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}

	var topCar model.TopCars
	err := r.Coll.FindOne(ctx, filter).Decode(&topCar)
	if err != nil {
		return nil, err
	}

	return &topCar, nil
}

// UpdateTopCar - top carni yangilash
func (r *TopCarsRepository) UpdateTopCar(ctx context.Context, id primitive.ObjectID, updateData bson.M) error {
	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}

	update := bson.M{
		"$set": updateData,
	}

	_, err := r.Coll.UpdateOne(ctx, filter, update)
	return err
}

// CountTopCars - top carlar sonini hisoblash
func (r *TopCarsRepository) CountTopCars(ctx context.Context, filter model.TopCarsFilter) (int64, error) {
	mongoFilter := bson.M{}

	if !filter.ShowDeleted {
		mongoFilter["deleted_at"] = nil
	}

	if !filter.ShowExpired {
		mongoFilter["finished_at"] = bson.M{"$gt": time.Now()}
	}

	if filter.Category != "" {
		mongoFilter["category"] = filter.Category
	}

	if filter.UserID != "" {
		mongoFilter["user_id"] = filter.UserID
	}

	if filter.CarID != "" {
		mongoFilter["car_id"] = filter.CarID
	}

	return r.Coll.CountDocuments(ctx, mongoFilter)
}
