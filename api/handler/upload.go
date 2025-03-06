package handler

import (
	"fmt"
	"net/http"
	"wegugin/api/auth"
	pb "wegugin/genproto/cruds"

	"github.com/gin-gonic/gin"
)

// @Summary CreatePhoto
// @Security ApiKeyAuth
// @Description Upload Car Photo
// @Tags IMAGES
// @Param car_id path string true "car_id"
// @Param file formData file true "UploadMediaForm"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /v1/car/photo/{car_id} [post]
func (h *Handler) CreatePhoto(c *gin.Context) {
	token := c.GetHeader("Authorization")
	userId, _, err := auth.GetUserIdFromToken(token)
	if err != nil {
		h.Log.Error("Error getting user id from token", "error", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	h.Log.Info("UploadProductPhoto called")
	Id := c.Param("car_id")
	if len(Id) == 0 {
		h.Log.Error("car_id is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Car id is required"})
		return
	}
	fmt.Println(userId)
	check, err := h.Crud.CheckCarOwnership(c, &pb.BoolCheckCar{UserId: userId, CarId: Id})
	if err != nil {
		h.Log.Error("Error checking car ownership", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error checking car ownership"})
		return
	}
	if !check.Result {
		h.Log.Error("User does not own the car")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User does not own the car"})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.Log.Error("Error retrieving the file", "error", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Error retrieving the file"})
		return
	}
	fmt.Println(file)
	defer file.Close()
	url, err := h.MINIO.UploadFile("photos", file, header)
	if err != nil {
		h.Log.Error("Error uploading the file to MinIO", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	res, err := h.Crud.AddImage(c, &pb.AddImageRequest{
		CarId:    Id,
		Filename: url,
	})
	if err != nil {
		h.Log.Error("Error creating photo", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error creating photo"})
		return
	}
	h.Log.Info("Photo uploaded successfully")
	c.JSON(http.StatusOK, gin.H{"photo_id": res.Id, "url1": url, "url2": res.Filename})
}

// GetImagesByCar godoc
// @Summary Get Car Photos
// @Description it will Get Car Photos
// @Tags IMAGES
// @Param car_id path string true "car_id"
// @Success 200 {object} []cruds.Image
// @Failure 400 {object} string "Invalid data"
// @Failure 500 {object} string "Server error"
// @Router /v1/car/photo/{car_id} [get]
func (h *Handler) GetImagesByCar(c *gin.Context) {
	h.Log.Info("GetImagesByCar called")
	id := c.Param("car_id")
	if len(id) == 0 {
		h.Log.Error("car_id is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Car id is required"})
		return
	}
	res, err := h.Crud.GetImagesByCar(c, &pb.CarId{CarId: id})
	if err != nil {
		h.Log.Error("Error getting images by car", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error getting images by car"})
		return
	}
	h.Log.Info("Get images by car successfully")
	c.JSON(http.StatusOK, res.Images)
}

// @Summary DeleteImage
// @Security ApiKeyAuth
// @Description Delete Image
// @Tags IMAGES
// @Param id path string true "id"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /v1/car/photo/{id} [delete]
func (h *Handler) DeleteImage(c *gin.Context) {
	token := c.GetHeader("Authorization")
	userId, _, err := auth.GetUserIdFromToken(token)
	if err != nil {
		h.Log.Error("Error getting user id from token", "error", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	h.Log.Info("DeleteImage called")
	id := c.Param("id")
	if len(id) == 0 {
		h.Log.Error("id is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	check, err := h.Crud.CheckCarOwnership(c, &pb.BoolCheckCar{UserId: userId, CarId: id})
	if err != nil {
		h.Log.Error("Error checking car ownership", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error checking car ownership"})
		return
	}
	if !check.Result {
		h.Log.Error("User does not own the car")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User does not own the car"})
		return
	}
	_, err = h.Crud.DeleteImage(c, &pb.ImageId{Id: id})
	if err != nil {
		h.Log.Error("Error deleting image", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error deleting image"})
		return
	}
	h.Log.Info("Image deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}

// @Summary DeleteImagesByCarId
// @Security ApiKeyAuth
// @Description Delete Images By Car Id
// @Tags IMAGES
// @Param car_id path string true "car_id"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /v1/car/photo/car/{car_id} [delete]
func (h *Handler) DeleteImagesByCarId(c *gin.Context) {
	token := c.GetHeader("Authorization")
	userId, _, err := auth.GetUserIdFromToken(token)
	if err != nil {
		h.Log.Error("Error getting user id from token", "error", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	h.Log.Info("DeleteImagesByCarId called")
	id := c.Param("car_id")
	if len(id) == 0 {
		h.Log.Error("car_id is required")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Car id is required"})
		return
	}
	check, err := h.Crud.CheckCarOwnership(c, &pb.BoolCheckCar{UserId: userId, CarId: id})
	if err != nil {
		h.Log.Error("Error checking car ownership", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error checking car ownership"})
		return
	}
	if !check.Result {
		h.Log.Error("User does not own the car")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User does not own the car"})
		return
	}
	_, err = h.Crud.DeleteImagesByCarId(c, &pb.CarId{CarId: id})
	if err != nil {
		h.Log.Error("Error deleting images by car id", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error deleting images by car id"})
		return
	}
	h.Log.Info("Images deleted successfully by car id")
	c.JSON(http.StatusOK, gin.H{"message": "Images deleted successfully by car id"})
}
