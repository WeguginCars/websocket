package handler

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
	"wegugin/api/auth"
	"wegugin/genproto/cruds"
	"wegugin/genproto/user"
	"wegugin/model"
	"wegugin/storage/redis"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var onlineUsers = struct {
	sync.Mutex
	connections map[string]*websocket.Conn
}{connections: make(map[string]*websocket.Conn)}

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

// WebSocket orqali xabarlarni olish va jo‘natish
func (h *Handler) ChatWebSocketByUserAndId(c *gin.Context) {
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

	// Token orqali foydalanuvchi ID olish
	userID, _, err := auth.GetUserIdFromToken(token)
	if err != nil || userID == "" {
		log.Println("Error getting user ID from token:", err)
		return
	}

	// So‘rovdan ikkinchi foydalanuvchi ID sini olish
	secondUserID := c.Query("second_user_id")
	if secondUserID == "" {
		log.Println("Second user ID is required")
		return
	}

	ctx := context.Background()

	// Foydalanuvchini online deb belgilash
	onlineUsers.Lock()
	onlineUsers.connections[userID] = conn
	onlineUsers.Unlock()

	// Xabarlarni o‘qish uchun goroutine
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Println("User disconnected:", err)
				h.disconnectUser(userID)
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
			// Ikki user orasidagi xabarlarni olish
			messages, err := h.Crud.GetMessageByUserAndId(ctx, &cruds.GetMessageByUserAndIdReq{
				FirstUserId:  userID,
				SecondUserId: secondUserID,
			})
			if err != nil {
				log.Println("Error fetching messages:", err)
				return
			}

			// Ikkinchi userning ismi va familyasini olish
			UserInfo, err := h.User.GetUserById(ctx, &user.UserId{
				Id: secondUserID,
			})
			if err != nil {
				log.Println("Error getting user info", err)
				continue
			}

			Istyping, err := redis.GetStatus(c, secondUserID, userID)
			if err != nil {
				log.Println("Error getting typing status", err)
				continue
			}

			// Agar ikkinchi user ham online bo'lsa, `is_user_online = true`
			onlineUsers.Lock()
			_, isUserOnline := onlineUsers.connections[secondUserID]
			onlineUsers.Unlock()

			// Ma’lumotlarni to‘ldirish
			messages.UserId = secondUserID
			messages.UserName = UserInfo.Name
			messages.UserSurname = UserInfo.Surname
			messages.IsUserOnline = isUserOnline
			messages.IsUserTyping = Istyping

			// WebSocket orqali jo‘natish
			if err := conn.WriteJSON(messages); err != nil {
				log.Println("Error writing message:", err)
				h.disconnectUser(userID)
				return
			}
		case <-c.Request.Context().Done():
			// Client ulanishni uzgan holatda
			h.disconnectUser(userID)
			return
		}
	}
}

func (h *Handler) disconnectUser(userID string) {
	onlineUsers.Lock()
	conn, exists := onlineUsers.connections[userID]
	if exists {
		conn.Close() // WebSocket ulanishini yopish
		delete(onlineUsers.connections, userID)
	}
	onlineUsers.Unlock()
	log.Println("User disconnected:", userID)
}

// @Summary DisconnectWebSocket
// @Security ApiKeyAuth
// @Description Disconnect WebSocket
// @Tags MESSAGES
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /v1/car/message/disconnectwebsocket [post]
func (h *Handler) DisconnectWebSocket(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is required"})
		return
	}

	userID, _, err := auth.GetUserIdFromToken(token)
	if err != nil || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// User ulanishini yopish
	h.disconnectUser(userID)
	c.JSON(http.StatusOK, gin.H{"message": "WebSocket disconnected successfully"})
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

// @Summary StoreUserAsTyping
// @Security ApiKeyAuth
// @Description Store User As Typing
// @Tags MESSAGES
// @Param user_id path string true "user_id"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /v1/car/message/store-user-as-typing/{user_id} [post]
func (h *Handler) StoreUserAsTyping(c *gin.Context) {
	h.Log.Info("StoreUserAsTyping called")
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is required"})
		h.Log.Error("StoreUserAsTyping called with invalid authorization header")
		return
	}

	userID, _, err := auth.GetUserIdFromToken(token)
	if err != nil || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		h.Log.Error("StoreUserAsTyping called with invalid user ID")
		return
	}
	targetUserID := c.Param("user_id")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target user ID is required"})
		h.Log.Error("StoreUserAsTyping called with invalid target user ID")
		return
	}
	err = redis.StoreUserAsTyping(c, userID, targetUserID)
	if err != nil {
		h.Log.Error("Error storing user as typing", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error storing user as typing"})
		return
	}
	h.Log.Info("User as typing stored successfully")
	c.JSON(http.StatusOK, gin.H{"message": "User as typing stored successfully"})
}

// @Summary DeleteUserTypingStatus
// @Security ApiKeyAuth
// @Description Delete User Typing Status
// @Tags MESSAGES
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /v1/car/message/user-typing [delete]
func (h *Handler) DeleteUserTypingStatus(c *gin.Context) {
	h.Log.Info("DeleteUserTypingStatus called")
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is required"})
		h.Log.Error("DeleteUserTypingStatus called with invalid authorization header")
		return
	}

	userID, _, err := auth.GetUserIdFromToken(token)
	if err != nil || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		h.Log.Error("DeleteUserTypingStatus called with invalid user ID")
		return
	}
	err = redis.DeleteStatus(c, userID)
	if err != nil {
		h.Log.Error("Error deleting user typing status", "error", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user typing status"})
		return
	}
	h.Log.Info("User typing status deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "User typing status deleted successfully"})
}
