package auth

import (
	"office-file-sharing/backend/internal/handlers"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers the authentication and user directory routes
func RegisterRoutes(g *echo.Group) {
	g.POST("/auth/login", handlers.Login)
	g.POST("/auth/signup", handlers.Signup)
	g.GET("/users", handlers.GetUsers)
}
