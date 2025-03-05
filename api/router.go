package api

import (
	_ "wegugin/api/docs"
	"wegugin/api/handler"
	"wegugin/api/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @title User
// @version 1.0
// @description API Gateway
// BasePath: /
func Router(hand *handler.Handler) *gin.Engine {
	router := gin.New()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	car := router.Group("/v1/car/photo")
	car.Use(middleware.Check)
	car.Use(middleware.CheckPermissionMiddleware(hand.Enforcer))
	{
		car.POST("/:car_id", hand.CreatePhoto)
		car.GET("/:car_id", hand.GetImagesByCar)
		car.DELETE("/:id", hand.DeleteImage)
		car.DELETE("/car/:car_id", hand.DeleteImagesByCarId)
	}

	return router
}
