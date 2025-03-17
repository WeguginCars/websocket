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

	// WebSocket uchun chat
	router.GET("/v1/messages/ws", hand.ChatWebSocket)
	router.GET("/v1/messages/ws/chat", hand.ChatWebSocketByUserAndId)
	message := router.Group("/v1/car/message")
	{
		message.POST("", middleware.Check, middleware.CheckPermissionMiddleware(hand.Enforcer), hand.SendMessage)
		message.POST("/:message_id", middleware.Check, middleware.CheckPermissionMiddleware(hand.Enforcer), hand.MarkMessageAsRead)
		message.DELETE("/:message_id", middleware.Check, middleware.CheckPermissionMiddleware(hand.Enforcer), hand.DeleteMessage)
		message.POST("/disconnectwebsocket", middleware.Check, middleware.CheckPermissionMiddleware(hand.Enforcer), hand.DisconnectWebSocket)
		message.POST("/store-user-as-typing/:user_id", middleware.Check, middleware.CheckPermissionMiddleware(hand.Enforcer), hand.StoreUserAsTyping)
		message.DELETE("/user-typing", middleware.Check, middleware.CheckPermissionMiddleware(hand.Enforcer), hand.DeleteUserTypingStatus)
	}

	car := router.Group("/v1/car/photo")
	{
		car.POST("/:car_id", middleware.Check, middleware.CheckPermissionMiddleware(hand.Enforcer), hand.CreatePhoto)
		car.GET("/:car_id", hand.GetImagesByCar) // Middleware YO‘Q
		car.DELETE("/:id", middleware.Check, middleware.CheckPermissionMiddleware(hand.Enforcer), hand.DeleteImage)
		car.DELETE("/car/:car_id", middleware.Check, middleware.CheckPermissionMiddleware(hand.Enforcer), hand.DeleteImagesByCarId)
	}

	return router
}
