package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

// Config captures runtime configuration resolved from the environment.
type Config struct {
	Port                       string
	DatabaseDSN                string
	ScenariosDir               string
	InterfaceGraphCacheTTL     time.Duration
	InterfaceGraphBuildTimeout time.Duration
}

// Load reads environment variables (and .env files) to build the Config.
func Load() Config {
	_ = godotenv.Load()

	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	dbDSN, err := storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "scenario-dependency-analyzer"})
	if err != nil {
		log.Fatalf("❌ SQLite configuration failed: %v", err)
	}

	scenariosDir := os.Getenv("VROOLI_SCENARIOS_DIR")
	if scenariosDir == "" {
		scenariosDir = "../.."
	}
	scenariosDir, err = absolutePath(scenariosDir)
	if err != nil {
		log.Fatalf("❌ Scenario directory configuration failed: %v", err)
	}

	return Config{
		Port:                       port,
		DatabaseDSN:                dbDSN,
		ScenariosDir:               scenariosDir,
		InterfaceGraphCacheTTL:     durationFromEnv("INTERFACE_GRAPH_CACHE_TTL", 5*time.Minute),
		InterfaceGraphBuildTimeout: durationFromEnv("INTERFACE_GRAPH_BUILD_TIMEOUT", 90*time.Second),
	}
}

func durationFromEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

// InitDatabase opens the embedded SQLite database with the standard Vrooli pragmas.
func InitDatabase(dsn string) (*database.RoutedDB, error) {
	log.Println("🔄 Opening SQLite database...")

	db, err := database.Open(context.Background(), database.Config{
		Driver:          database.DriverSQLite,
		DSN:             dsn,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	log.Println("🎉 SQLite database opened successfully!")
	return db, nil
}

func absolutePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path %q: %w", path, err)
	}
	return filepath.Clean(abs), nil
}
