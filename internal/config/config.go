package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// AppConfig holds all the environment variables needed for the engine.
type AppConfig struct {
	DatabaseURL string
	Port        string
	JWTSecret   string // NEW

}

// Load reads the .env file and extracts the variables.
func Load() *AppConfig {
	// The SRE: We ignore the error here on purpose!
	// In local dev, godotenv loads the .env file.
	// In production (Docker/AWS), the .env file won't exist. AWS injects the variables directly into the OS.
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// If there is no database URL, the app mathematically cannot function.
		// log.Fatal kills the server instantly.
		log.Fatal("FATAL: DATABASE_URL environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		// If no port is specified, we safely default to 8080.
		port = "8080"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable is required")
	}
	return &AppConfig{
		DatabaseURL: dbURL,
		Port:        port,
	}
}
