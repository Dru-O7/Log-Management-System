// Package config manages configuration loading from environment variables
package config

import (
	"os"
)

type Config struct {
	DatabaseURL  string
	JWTSecret    string
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
}

// Load reads and parses configuration fields
func Load() *Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=localhost user=postgres password=postgres dbname=office_files port=5432 sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "my-office-secret-key-12345"
	}

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587"
	}

	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = "no-reply@school.edu"
	}

	return &Config{
		DatabaseURL:  dbURL,
		JWTSecret:    jwtSecret,
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPUser:     smtpUser,
		SMTPPassword: smtpPass,
		SMTPFrom:     smtpFrom,
	}
}
