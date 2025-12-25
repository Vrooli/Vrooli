package config

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
)

// Config captures runtime configuration resolved from the environment.
type Config struct {
	Port         string
	DatabaseURL  string
	ScenariosDir string
}

// Load reads environment variables (and .env files) to build the Config.
func Load() Config {
	godotenv.Load()

	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Missing database configuration. Provide DATABASE_URL or POSTGRES_* variables")
		}

		dbURL = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName,
		)
	}

	scenariosDir := os.Getenv("VROOLI_SCENARIOS_DIR")
	if scenariosDir == "" {
		scenariosDir = "../.."
	}

	return Config{
		Port:         port,
		DatabaseURL:  dbURL,
		ScenariosDir: scenariosDir,
	}
}

// InitDatabase opens a PostgreSQL connection with automatic retry and backoff.
func InitDatabase(dbURL string) (*sql.DB, error) {
	log.Println("🔄 Attempting database connection with exponential backoff...")

	db, err := database.Connect(context.Background(), database.Config{
		Driver:          "postgres",
		DSN:             dbURL,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	log.Println("🎉 Database connection pool established successfully!")
	return db, nil
}
