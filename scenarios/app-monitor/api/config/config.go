package config

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"app-monitor-api/logger"

	"github.com/docker/docker/client"
	_ "github.com/lib/pq" // register postgres driver with database/sql
	"github.com/redis/go-redis/v9"
	"github.com/vrooli/api-core/database"
	coreRedis "github.com/vrooli/api-core/redis"

	monitorSchema "app-monitor-api/internal/monitor"
)

// Config holds all application configuration
type Config struct {
	API          APIConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	Docker       DockerConfig
	Orchestrator OrchestratorConfig
}

// APIConfig holds API server configuration
type APIConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL               string
	MaxOpenConns      int
	MaxIdleConns      int
	ConnMaxLifetime   time.Duration
	ConnectionTimeout time.Duration
	MaxRetries        int
	RetryBackoffBase  time.Duration
	RetryBackoffMax   time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL             string
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

// DockerConfig holds Docker configuration
type DockerConfig struct {
	Host       string
	APIVersion string
	TLSVerify  bool
	CertPath   string
}

// OrchestratorConfig holds orchestrator configuration
type OrchestratorConfig struct {
	StatusURL string
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() (*Config, error) {
	cfg := &Config{
		API: APIConfig{
			Port:            os.Getenv("API_PORT"),
			ReadTimeout:     getDurationEnv("API_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getDurationEnv("API_WRITE_TIMEOUT", 90*time.Second),
			ShutdownTimeout: getDurationEnv("API_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Database: DatabaseConfig{
			URL:               buildPostgresURL(),
			MaxOpenConns:      getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:      getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime:   getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnectionTimeout: getDurationEnv("DB_CONNECTION_TIMEOUT", 30*time.Second),
			MaxRetries:        getIntEnv("DB_MAX_RETRIES", 10),
			RetryBackoffBase:  getDurationEnv("DB_RETRY_BACKOFF_BASE", 1*time.Second),
			RetryBackoffMax:   getDurationEnv("DB_RETRY_BACKOFF_MAX", 30*time.Second),
		},
		Redis: RedisConfig{
			URL:             buildRedisURL(),
			MaxRetries:      getIntEnv("REDIS_MAX_RETRIES", 3),
			MinRetryBackoff: getDurationEnv("REDIS_MIN_RETRY_BACKOFF", 8*time.Millisecond),
			MaxRetryBackoff: getDurationEnv("REDIS_MAX_RETRY_BACKOFF", 512*time.Millisecond),
			DialTimeout:     getDurationEnv("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:     getDurationEnv("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:    getDurationEnv("REDIS_WRITE_TIMEOUT", 3*time.Second),
		},
		Docker: DockerConfig{
			Host:       getEnv("DOCKER_HOST", ""),
			APIVersion: getEnv("DOCKER_API_VERSION", ""),
			TLSVerify:  getBoolEnv("DOCKER_TLS_VERIFY", false),
			CertPath:   getEnv("DOCKER_CERT_PATH", ""),
		},
		Orchestrator: OrchestratorConfig{
			StatusURL: getEnv("ORCHESTRATOR_STATUS_URL", ""),
		},
	}

	// Validate required configuration
	if cfg.API.Port == "" {
		return nil, fmt.Errorf("API_PORT or PORT environment variable is required")
	}

	return cfg, nil
}

// InitializeDatabase creates and configures a database connection with automatic retry and backoff.
func (c *Config) InitializeDatabase() (*sql.DB, error) {
	if c.Database.URL == "" {
		return nil, nil // Database is optional
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:          "postgres",
		DSN:             c.Database.URL,
		MaxOpenConns:    c.Database.MaxOpenConns,
		MaxIdleConns:    c.Database.MaxIdleConns,
		ConnMaxLifetime: c.Database.ConnMaxLifetime,
	})
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(monitorSchema.Schema)); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database schema initialization failed: %w", err)
	}

	logger.Info("✅ Database connected successfully")
	return db, nil
}

// InitializeRedis creates and configures a Redis client
func (c *Config) InitializeRedis() (*redis.Client, error) {
	if c.Redis.URL == "" {
		return nil, nil // Redis is optional
	}

	opts, err := redis.ParseURL(c.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Apply additional configuration
	opts.MaxRetries = c.Redis.MaxRetries
	opts.MinRetryBackoff = c.Redis.MinRetryBackoff
	opts.MaxRetryBackoff = c.Redis.MaxRetryBackoff
	opts.DialTimeout = c.Redis.DialTimeout
	opts.ReadTimeout = c.Redis.ReadTimeout
	opts.WriteTimeout = c.Redis.WriteTimeout

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), c.Redis.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("✅ Redis connected successfully")
	return client, nil
}

// InitializeDocker creates and configures a Docker client
func (c *Config) InitializeDocker() (*client.Client, error) {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	if c.Docker.Host != "" {
		opts = append(opts, client.WithHost(c.Docker.Host))
	}

	if c.Docker.APIVersion != "" {
		opts = append(opts, client.WithVersion(c.Docker.APIVersion))
	}

	dockerClient, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := dockerClient.Ping(ctx); err != nil {
		dockerClient.Close()
		return nil, fmt.Errorf("failed to connect to Docker: %w", err)
	}

	logger.Info("✅ Docker connected successfully")
	return dockerClient, nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func buildPostgresURL() string {
	url, err := database.ResolvePostgresDSN(os.Getenv)
	if err == nil {
		return url
	}
	return ""
}

func buildRedisURL() string {
	config, err := coreRedis.Resolve(os.Getenv)
	if err == nil {
		return "redis://" + config.Addr
	}
	return ""
}
