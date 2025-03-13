package handler

import (
	"context"
	"log"
	"net/http"
	"time"
	"wegugin/api/auth"
	"wegugin/genproto/cruds"
	"wegugin/genproto/user"
	"wegugin/model"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) ChatWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	token := c.GetHeader("Authorization")
	if token == "" {
		log.Println("Authorization header is required")
		return
	}

	userID, _, err := auth.GetUserIdFromToken(token)
	if err != nil {
		log.Println("Error getting user ID from token:", err)
		return
	}
	if userID == "" {
		log.Println("User ID is required")
		return
	}

	ctx := context.Background()

	// WebSocket o'qish uchun goroutine ishga tushiramiz
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Println("User disconnected:", err)
				conn.Close()
				return
			}
		}
	}()

	// Xabarlarni userga yuborish
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			messages, err := h.Crud.GetMessagesByUser(ctx, &cruds.GetMessagesByUserRequest{UserId: userID})
			if err != nil {
				log.Println("Error fetching messages:", err)
				return // Loopdan chiqish o'rniga to'liq funktsiyani tugatish
			}
			// Xabarlarni qayta ishlash
			for i := range messages.Groups {
				UserInfo, err := h.User.GetUserById(ctx, &user.UserId{
					Id: messages.Groups[i].UserId,
				})
				if err != nil {
					log.Println("Error getting user info", err)
					continue
				}
				messages.Groups[i].UserName = UserInfo.Name
				messages.Groups[i].UserSurname = UserInfo.Surname
			}
			// WebSocket orqali jo'natish
			if err := conn.WriteJSON(messages); err != nil {
				log.Println("Error writing message:", err)
				return // Xatolik yuz berganda loopdan chiqish
			}
		case <-c.Request.Context().Done():
			// Client ulanishni uzgan holatda
			log.Println("Connection closed by client")
			return
		}
	}
}

// @Summary SendMessage
// @Security ApiKeyAuth
// @Description Send Message
// @Tags MESSAGES
// @Param info body model.SendMessageBody true "info"
// @Success 200 {object} cruds.Message
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /v1/car/message [post]
func (h *Handler) SendMessage(c *gin.Context) {
	token := c.GetHeader("Authorization")
	userId, _, err := auth.GetUserIdFromToken(token)
	if err != nil {
		h.Log.Error("Error getting user id from token", "error", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req model.SendMessageBody

	err = c.ShouldBindJSON(&req)
	if err != nil {
		h.Log.Error("Error binding JSON", "error", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	resp, err := h.Crud.SendMessage(c, &cruds.SendMessageRequest{SenderId: userId, RecipientId: req.RecipientID, Content: req.Content})
	if err != nil {
		h.Log.Error("Error sending message", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error sending message"})
		return
	}
	h.Log.Info("Message sent successfully")
	c.JSON(http.StatusOK, resp)
}

// @Summary MarkMessageAsRead
// @Security ApiKeyAuth
// @Description mark message as read
// @Tags MESSAGES
// @Param message_id path string true "message_id"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /v1/car/message/{message_id} [post]
func (h *Handler) MarkMessageAsRead(c *gin.Context) {
	token := c.GetHeader("Authorization")
	userId, _, err := auth.GetUserIdFromToken(token)
	if err != nil {
		h.Log.Error("Error getting user id from token", "error", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	messageID := c.Param("message_id")
	bl, err := h.Crud.CheckMessageOwnership(c, &cruds.BoolCheckMessage{UserId: userId, MessageId: messageID})
	if err != nil {
		h.Log.Error("Error checking message ownership", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error checking message ownership"})
		return
	}
	if !bl.Result {
		h.Log.Error("User does not own the message")
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "User does not own the message"})
		return
	}

	resp, err := h.Crud.MarkMessageAsRead(c, &cruds.MessageId{Id: messageID})
	if err != nil {
		h.Log.Error("Error marking message as read", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error marking message as read"})
		return
	}
	h.Log.Info("Message marked as read successfully")
	c.JSON(http.StatusOK, resp)
}

// @Summary Delete Message
// @Security ApiKeyAuth
// @Description Delete Message
// @Tags MESSAGES
// @Param message_id path string true "message_id"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /v1/car/message/{message_id} [delete]
func (h *Handler) DeleteMessage(c *gin.Context) {
	token := c.GetHeader("Authorization")
	userId, _, err := auth.GetUserIdFromToken(token)
	if err != nil {
		h.Log.Error("Error getting user id from token", "error", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	messageID := c.Param("message_id")
	bl, err := h.Crud.CheckMessageOwnership(c, &cruds.BoolCheckMessage{UserId: userId, MessageId: messageID})
	if err != nil {
		h.Log.Error("Error checking message ownership", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error checking message ownership"})
		return
	}
	if !bl.Result {
		h.Log.Error("User does not own the message")
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "User does not own the message"})
		return
	}
	_, err = h.Crud.DeleteMessage(c, &cruds.DeleteMessageRequest{Id: messageID})
	if err != nil {
		h.Log.Error("Error deleting message", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error deleting message"})
		return
	}
	h.Log.Info("Message deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Message deleted successfully"})
}
