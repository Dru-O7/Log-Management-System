package message

import (
	"office-file-sharing/backend/internal/shared/middleware"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(g *echo.Group, handler *Handler, jwtSecret []byte) {
	messages := g.Group("/messages", middleware.AuthMiddleware(jwtSecret))

	messages.POST("", handler.SendMessage)
	messages.GET("/inbox", handler.GetInbox)
	messages.GET("/sent", handler.GetSent)
	messages.GET("/drafts", handler.GetDrafts)
	messages.POST("/drafts", handler.SaveDraft)
	messages.DELETE("/drafts/:id", handler.DeleteDraft)
	messages.GET("/trash", handler.GetTrash)
	messages.GET("/unread-count", handler.GetUnreadCount)
	messages.GET("/search-users", handler.SearchUsers)
	messages.GET("/by-email", handler.GetUserByEmail)
	messages.GET("/:id", handler.GetMessageDetails)
	messages.PATCH("/:id/read", handler.ToggleReadStatus)
	messages.DELETE("/:id", handler.SoftDeleteMessage)
	messages.POST("/:id/restore", handler.RestoreMessage)
}
