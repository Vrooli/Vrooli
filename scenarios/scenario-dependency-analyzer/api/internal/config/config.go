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

	dbDSN, err := sqliteDSN()
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

func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("scenario-dependency-analyzer")
	if err != nil {
		return "", fmt.Errorf("resolve scenario-dependency-analyzer storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"scenario-dependency-analyzer.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve scenario-dependency-analyzer db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	dbDir := filepath.Clean(filepath.Dir(path))
	if err := os.MkdirAll(dbDir, 0o750); err != nil { // #nosec G703 -- dbDir is resolved by api-core storage or an operator-provided SQLITE_PATH.
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
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
