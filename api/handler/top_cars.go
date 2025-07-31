package handler

import (
	"net/http"
	"strconv"
	"time"

	"wegugin/api/auth"
	"wegugin/genproto/cruds"
	"wegugin/model"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateTopCar godoc
// @Summary Create a new top car
// @Description Add a new car to top cars list
// @Tags TopCars
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param topcar body CreateTopCarRequest true "Top Car data"
// @Success 201 {object} CreateTopCarResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/topcar [post]
func (h *Handler) CreateTopCar(c *gin.Context) {
	var req CreateTopCarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Token dan user_id olish
	token := c.GetHeader("Authorization")
	userID, _, err := auth.GetUserIdFromToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Invalid token",
		})
		return
	}

	// FinishedAt ni hisoblash
	var finishedAt time.Time
	now := time.Now()
	switch req.Category {
	case "daily":
		finishedAt = now.Add(24 * time.Hour)
	case "weekly":
		finishedAt = now.Add(7 * 24 * time.Hour)
	case "monthly":
		finishedAt = now.Add(30 * 24 * time.Hour)
	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid category. Must be daily, weekly, or monthly",
		})
		return
	}

	topCar := &model.TopCars{
		CarId:      req.CarId,
		UserId:     userID,
		Category:   req.Category,
		CreatedAt:  now,
		FinishedAt: finishedAt,
	}

	resbool, err := h.Crud.CheckCarOwnership(c, &cruds.BoolCheckCar{UserId: userID, CarId: req.CarId})
	if err != nil {
		h.Log.Error("Error checking car ownership", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error checking car ownership"})
		return
	}
	if !resbool.Result {
		h.Log.Error("User does not own the car")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User does not own the car"})
		return
	}
	err = h.Cruds.TopCars().CreateTopCar(c.Request.Context(), topCar)
	if err != nil {
		if err.Error() == "topCar already exists in this category and is still active" {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error: "Car is already in active top list for this category",
			})
			return
		}
		h.Log.Error("Failed to create top car", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to create top car",
		})
		return
	}

	c.JSON(http.StatusCreated, CreateTopCarResponse{
		ID:         topCar.ID.Hex(),
		CarId:      topCar.CarId,
		UserId:     topCar.UserId,
		Category:   topCar.Category,
		CreatedAt:  topCar.CreatedAt,
		FinishedAt: topCar.FinishedAt,
		Message:    "Top car created successfully",
	})
}

// GetTopCars godoc
// @Summary Get list of top cars
// @Description Get filtered list of top cars
// @Tags TopCars
// @Accept json
// @Produce json
// @Param category query string false "Category filter (daily, weekly, monthly)"
// @Param user_id query string false "User ID filter"
// @Param car_id query string false "Car ID filter"
// @Param sort_by query string false "Sort by (finished_at_asc, finished_at_desc, created_at_asc, created_at_desc)"
// @Param show_expired query bool false "Show expired top cars"
// @Param show_deleted query bool false "Show deleted top cars"
// @Param limit query int false "Limit results (default 50)"
// @Param skip query int false "Skip results for pagination"
// @Success 200 {object} GetTopCarsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/topcar [get]
func (h *Handler) GetTopCars(c *gin.Context) {
	filter := model.TopCarsFilter{
		Category:    c.Query("category"),
		UserID:      c.Query("user_id"),
		CarID:       c.Query("car_id"),
		SortBy:      c.Query("sort_by"),
		ShowExpired: c.Query("show_expired") == "true",
		ShowDeleted: c.Query("show_deleted") == "true",
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			filter.Limit = limit
		}
	}

	if skipStr := c.Query("skip"); skipStr != "" {
		if skip, err := strconv.ParseInt(skipStr, 10, 64); err == nil {
			filter.Skip = skip
		}
	}

	topCars, err := h.Cruds.TopCars().GetListOfTopCars(c.Request.Context(), filter)
	if err != nil {
		h.Log.Error("Failed to get top cars", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get top cars",
		})
		return
	}

	// Count uchun
	count, err := h.Cruds.TopCars().CountTopCars(c.Request.Context(), filter)
	if err != nil {
		h.Log.Error("Failed to count top cars", "error", err)
		count = 0
	}

	var response []TopCarResponse
	for _, topCar := range topCars {
		response = append(response, TopCarResponse{
			ID:         topCar.ID.Hex(),
			CarId:      topCar.CarId,
			UserId:     topCar.UserId,
			Category:   topCar.Category,
			CreatedAt:  topCar.CreatedAt,
			FinishedAt: topCar.FinishedAt,
			DeletedAt:  topCar.DeletedAt,
		})
	}

	c.JSON(http.StatusOK, GetTopCarsResponse{
		TopCars: response,
		Total:   count,
		Limit:   filter.Limit,
		Skip:    filter.Skip,
	})
}

// GetTopCarByID godoc
// @Summary Get top car by ID
// @Description Get a specific top car by its ID
// @Tags TopCars
// @Accept json
// @Produce json
// @Param id path string true "Top Car ID"
// @Success 200 {object} TopCarResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/topcar/{id} [get]
func (h *Handler) GetTopCarByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid ID format",
		})
		return
	}

	topCar, err := h.Cruds.TopCars().GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error: "Top car not found",
			})
			return
		}
		h.Log.Error("Failed to get top car by ID", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to get top car",
		})
		return
	}

	c.JSON(http.StatusOK, TopCarResponse{
		ID:         topCar.ID.Hex(),
		CarId:      topCar.CarId,
		UserId:     topCar.UserId,
		Category:   topCar.Category,
		CreatedAt:  topCar.CreatedAt,
		FinishedAt: topCar.FinishedAt,
		DeletedAt:  topCar.DeletedAt,
	})
}

// DeleteTopCarByID godoc
// @Summary Delete top car by ID
// @Description Delete a specific top car by its ID
// @Tags TopCars
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Top Car ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/topcar/{id} [delete]
func (h *Handler) DeleteTopCarByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid ID format",
		})
		return
	}

	err = h.Cruds.TopCars().DeleteByID(c.Request.Context(), id)
	if err != nil {
		h.Log.Error("Failed to delete top car by ID", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to delete top car",
		})
		return
	}

	c.JSON(http.StatusOK, MessageResponse{
		Message: "Top car deleted successfully",
	})
}

// DeleteTopCarsByUserID godoc
// @Summary Delete top cars by user ID
// @Description Delete all top cars belonging to a specific user
// @Tags TopCars
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param user_id path string true "User ID"
// @Success 200 {object} DeleteResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/topcar/user/{user_id} [delete]
func (h *Handler) DeleteTopCarsByUserID(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "User ID is required",
		})
		return
	}

	deletedCount, err := h.Cruds.TopCars().DeleteByUserID(c.Request.Context(), userID)
	if err != nil {
		h.Log.Error("Failed to delete top cars by user ID", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to delete top cars",
		})
		return
	}

	c.JSON(http.StatusOK, DeleteResponse{
		Message:      "Top cars deleted successfully",
		DeletedCount: deletedCount,
	})
}

// DeleteTopCarsByCarID godoc
// @Summary Delete top cars by car ID
// @Description Delete all top cars for a specific car
// @Tags TopCars
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param car_id path string true "Car ID"
// @Success 200 {object} DeleteResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/topcar/car/{car_id} [delete]
func (h *Handler) DeleteTopCarsByCarID(c *gin.Context) {
	carID := c.Param("car_id")
	if carID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Car ID is required",
		})
		return
	}

	deletedCount, err := h.Cruds.TopCars().DeleteByCarID(c.Request.Context(), carID)
	if err != nil {
		h.Log.Error("Failed to delete top cars by car ID", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to delete top cars",
		})
		return
	}

	c.JSON(http.StatusOK, DeleteResponse{
		Message:      "Top cars deleted successfully",
		DeletedCount: deletedCount,
	})
}

// Request/Response structures
type CreateTopCarRequest struct {
	CarId    string `json:"car_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Category string `json:"category" binding:"required" example:"daily,weekly,monthly"`
}

type CreateTopCarResponse struct {
	ID         string    `json:"id" example:"507f1f77bcf86cd799439011"`
	CarId      string    `json:"car_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	UserId     string    `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174001"`
	Category   string    `json:"category" example:"daily"`
	CreatedAt  time.Time `json:"created_at" example:"2024-01-01T12:00:00Z"`
	FinishedAt time.Time `json:"finished_at" example:"2024-01-02T12:00:00Z"`
	Message    string    `json:"message" example:"Top car created successfully"`
}

type TopCarResponse struct {
	ID         string     `json:"id" example:"507f1f77bcf86cd799439011"`
	CarId      string     `json:"car_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	UserId     string     `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174001"`
	Category   string     `json:"category" example:"daily"`
	CreatedAt  time.Time  `json:"created_at" example:"2024-01-01T12:00:00Z"`
	FinishedAt time.Time  `json:"finished_at" example:"2024-01-02T12:00:00Z"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" example:"2024-01-03T12:00:00Z"`
}

type GetTopCarsResponse struct {
	TopCars []TopCarResponse `json:"top_cars"`
	Total   int64            `json:"total" example:"100"`
	Limit   int64            `json:"limit" example:"50"`
	Skip    int64            `json:"skip" example:"0"`
}

type DeleteResponse struct {
	Message      string `json:"message" example:"Top cars deleted successfully"`
	DeletedCount int64  `json:"deleted_count" example:"5"`
}

type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Error description"`
}
