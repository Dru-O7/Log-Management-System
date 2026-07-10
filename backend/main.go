package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"office-file-sharing/backend/internal/db"
	"office-file-sharing/backend/internal/models"
	"office-file-sharing/backend/services/auth"
	"office-file-sharing/backend/services/document"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

func main() {
	db.InitDB()
	seedData()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Rate limiter specifically for authentication endpoints to prevent brute force
	authRateLimiter := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			// Only rate limit requests matching /api/auth/
			return !strings.HasPrefix(c.Request().URL.Path, "/api/auth/")
		},
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(5.0 / 60.0), // 5 requests per minute
				Burst:     5,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"error": "Too many requests. Please try again later."})
		},
	})

	e.Use(authRateLimiter)

	api := e.Group("/api")

	// Register Modular Services
	auth.RegisterRoutes(api)
	document.RegisterRoutes(api)

	log.Println("Modular Microservices starting on port :8080...")
	log.Fatal(e.Start(":8080"))
}

func seedData() {
	var count int64
	db.DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash default password:", err)
		}
		users := []models.User{
			{Name: "Alice Smith", Email: "alice@office.com", PasswordHash: string(hash)},
			{Name: "Bob Jones", Email: "bob@office.com", PasswordHash: string(hash)},
			{Name: "Charlie Brown", Email: "charlie@office.com", PasswordHash: string(hash)},
		}
		for _, u := range users {
			u.ID = uuid.New()
			db.DB.Create(&u)
		}
		log.Println("Database seeded with test users.")
	} else {
		// Update legacy dummy password hashes to bcrypt hashes
		var dummyUsers []models.User
		db.DB.Where("password_hash = ?", "dummy").Find(&dummyUsers)
		if len(dummyUsers) > 0 {
			hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
			db.DB.Model(&models.User{}).Where("password_hash = ?", "dummy").Update("password_hash", string(hash))
			log.Println("Updated legacy test users with hashed passwords.")
		}
	}
}
