package config

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"app-monitor-api/logger"

	"github.com/docker/docker/client"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
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
			WriteTimeout:    getDurationEnv("API_WRITE_TIMEOUT", 30*time.Second),
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

// InitializeDatabase creates and configures a database connection
func (c *Config) InitializeDatabase() (*sql.DB, error) {
	if c.Database.URL == "" {
		return nil, nil // Database is optional
	}

	db, err := sql.Open("postgres", c.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(c.Database.MaxOpenConns)
	db.SetMaxIdleConns(c.Database.MaxIdleConns)
	db.SetConnMaxLifetime(c.Database.ConnMaxLifetime)

	// Test connection with retries
	if err := c.testDatabaseConnection(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// testDatabaseConnection tests the database connection with exponential backoff
func (c *Config) testDatabaseConnection(db *sql.DB) error {
	var lastErr error
	baseDelay := c.Database.RetryBackoffBase
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}
	maxBackoff := c.Database.RetryBackoffMax
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}
	rand.Seed(time.Now().UnixNano())

	for attempt := 0; attempt < c.Database.MaxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), c.Database.ConnectionTimeout)
		err := db.PingContext(ctx)
		cancel()

		if err == nil {
			logger.Info(fmt.Sprintf("✅ Database connected successfully on attempt %d", attempt+1))
			return nil
		}

		lastErr = err
		logger.Warn(fmt.Sprintf("Database connection attempt %d/%d failed", attempt+1, c.Database.MaxRetries), err)

		if attempt < c.Database.MaxRetries-1 {
			exponentialDelay := time.Duration(math.Min(float64(baseDelay)*math.Pow(2, float64(attempt)), float64(maxBackoff)))
			jitter := time.Duration(rand.Float64() * float64(exponentialDelay) * 0.25)
			wait := exponentialDelay + jitter
			logger.Info(fmt.Sprintf("⏳ Waiting %v before next attempt", wait))
			time.Sleep(wait)
		}
	}

	return fmt.Errorf("database connection failed after %d attempts: %w", c.Database.MaxRetries, lastErr)
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
	// First check for complete URL
	if url := os.Getenv("POSTGRES_URL"); url != "" {
		return url
	}

	// Try to build from individual components
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	if host != "" && port != "" && user != "" && password != "" && dbName != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user, password, host, port, dbName)
	}

	return ""
}

func buildRedisURL() string {
	// First check for complete URL
	if url := os.Getenv("REDIS_URL"); url != "" {
		return url
	}

	// Try to build from individual components
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")

	if host != "" && port != "" {
		return fmt.Sprintf("redis://%s:%s", host, port)
	}

	return ""
}
