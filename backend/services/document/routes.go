package document

import (
	"office-file-sharing/backend/internal/handlers"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes registers all document workflows under AuthMiddleware
func RegisterRoutes(g *echo.Group) {
	// Protected routes
	r := g.Group("")
	r.Use(handlers.AuthMiddleware)

	r.POST("/documents", handlers.UploadDocument)
	r.GET("/documents", handlers.GetDocuments)
	r.GET("/documents/:id", handlers.GetDocumentDetails)
	r.GET("/documents/:id/download", handlers.DownloadDocument)
	r.PUT("/documents/:id/replace", handlers.ReplaceDocument)
	r.POST("/documents/:id/action", handlers.DocumentAction)
}
