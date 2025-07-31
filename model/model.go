package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SendMessageBody struct {
	RecipientID string `json:"recipient_id" binding:"required"`
	Content     string `json:"content" binding:"required"`
}

type TopCars struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CarId      string             `bson:"car_id" json:"car_id"`
	UserId     string             `bson:"user_id" json:"user_id"`
	Category   string             `bson:"category" json:"category"` // "daily", "weekly", "monthly"
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	FinishedAt time.Time          `bson:"finished_at" json:"finished_at"`
	DeletedAt  *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Filter options for GetListOfTopCars
type TopCarsFilter struct {
	Category    string `json:"category,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	CarID       string `json:"car_id,omitempty"`
	SortBy      string `json:"sort_by,omitempty"`      // "finished_at_asc", "finished_at_desc", "created_at_asc", "created_at_desc"
	ShowExpired bool   `json:"show_expired,omitempty"` // false by default
	ShowDeleted bool   `json:"show_deleted,omitempty"` // false by default
	Limit       int64  `json:"limit,omitempty"`        // default 50
	Skip        int64  `json:"skip,omitempty"`         // for pagination
}
