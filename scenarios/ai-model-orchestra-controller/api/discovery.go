package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/docker/docker/client"
	"github.com/redis/go-redis/v9"
)

// Initialize database connection with exponential backoff
func initDatabase(logger *log.Logger) (*sql.DB, error) {
	host := os.Getenv("ORCHESTRATOR_HOST")
	if host == "" {
		return nil, fmt.Errorf("ORCHESTRATOR_HOST environment variable is required")
	}

	port := os.Getenv("RESOURCE_PORTS_POSTGRES")
	if port == "" {
		return nil, fmt.Errorf("RESOURCE_PORTS_POSTGRES environment variable is required")
	}

	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		return nil, fmt.Errorf("POSTGRES_USER environment variable is required")
	}

	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD environment variable is required")
	}

	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "orchestrator"
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var db *sql.DB
	var err error
	maxRetries := 10
	backoffBase := time.Second
	maxBackoff := 30 * time.Second
	rand.Seed(time.Now().UnixNano())

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err = sql.Open("postgres", psqlInfo)
		if err != nil {
			return nil, fmt.Errorf("error opening database: %v", err)
		}

		// Test connection
		if err = db.Ping(); err == nil {
			logger.Printf("✅ Connected to PostgreSQL on attempt %d", attempt)

			// Set connection pool settings
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)

			return db, nil
		}

		db.Close()

		if attempt == maxRetries {
			return nil, fmt.Errorf("failed to connect to database after %d attempts: %v", maxRetries, err)
		}

		// Exponential backoff with random jitter
		backoff := time.Duration(math.Pow(2, float64(attempt-1))) * backoffBase
		if backoff > maxBackoff {
			backoff = maxBackoff
		}

		// Add random jitter to prevent thundering herd
		jitterRange := float64(backoff) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := backoff + jitter

		logger.Printf("⚠️  Database connection failed (attempt %d/%d), retrying in %v: %v",
			attempt, maxRetries, actualDelay, err)
		time.Sleep(actualDelay)
	}

	return nil, err
}

// Initialize Redis connection with exponential backoff
func initRedis(logger *log.Logger) (*redis.Client, error) {
	host := os.Getenv("ORCHESTRATOR_HOST")
	if host == "" {
		return nil, fmt.Errorf("ORCHESTRATOR_HOST environment variable is required")
	}

	port := os.Getenv("RESOURCE_PORTS_REDIS")
	if port == "" {
		return nil, fmt.Errorf("RESOURCE_PORTS_REDIS environment variable is required")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%s", host, port),
		Password:        os.Getenv("REDIS_PASSWORD"), // No fallback - empty string if not set
		DB:              0,
		MaxRetries:      3,
		MinRetryBackoff: time.Second,
		PoolSize:        10,
		MinIdleConns:    2,
	})

	maxRetries := 10
	backoffBase := time.Second
	maxBackoff := 30 * time.Second
	rand.Seed(time.Now().UnixNano())

	for attempt := 0; attempt < maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := rdb.Ping(ctx).Result()
		cancel()

		if err == nil {
			logger.Printf("✅ Connected to Redis on attempt %d", attempt+1)
			return rdb, nil
		}

		if attempt == maxRetries-1 {
			return nil, fmt.Errorf("failed to connect to Redis after %d attempts: %v", maxRetries, err)
		}

		// Exponential backoff with real jitter to avoid synchronized retries
		delay := time.Duration(math.Min(float64(backoffBase)*math.Pow(2, float64(attempt)), float64(maxBackoff)))
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := delay + jitter

		logger.Printf("⚠️  Redis connection failed (attempt %d/%d), retrying in %v: %v",
			attempt+1, maxRetries, actualDelay, err)
		time.Sleep(actualDelay)
	}

	return nil, nil
}

// Initialize Docker client
func initDocker(logger *log.Logger) (*client.Client, error) {
	dockerCli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Printf("⚠️  Docker client initialization failed: %v", err)
		return nil, err
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = dockerCli.Info(ctx)
	if err != nil {
		logger.Printf("⚠️  Docker connection test failed: %v", err)
		return nil, err
	}

	logger.Printf("✅ Connected to Docker daemon")
	return dockerCli, nil
}

// Initialize Ollama client
func initOllama(logger *log.Logger) (*OllamaClient, error) {
	host := os.Getenv("ORCHESTRATOR_HOST")
	if host == "" {
		return nil, fmt.Errorf("ORCHESTRATOR_HOST environment variable is required")
	}

	port := os.Getenv("RESOURCE_PORTS_OLLAMA")
	if port == "" {
		return nil, fmt.Errorf("RESOURCE_PORTS_OLLAMA environment variable is required")
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port)

	ollamaClient := &OllamaClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		CircuitBreaker: &CircuitBreaker{
			maxFailures:  5,                // Open circuit after 5 consecutive failures
			resetTimeout: 60 * time.Second, // Try again after 60 seconds
			state:        Closed,
		},
	}

	// Test connection by fetching models
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := ollamaClient.GetModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama at %s: %v", baseURL, err)
	}

	logger.Printf("✅ Connected to Ollama at %s", baseURL)
	return ollamaClient, nil
}

// Initialize database schema
func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS model_metrics (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		model_name VARCHAR(255) NOT NULL UNIQUE,
		request_count INTEGER DEFAULT 0,
		success_count INTEGER DEFAULT 0,
		error_count INTEGER DEFAULT 0,
		avg_response_time_ms FLOAT DEFAULT 0,
		current_load FLOAT DEFAULT 0,
		memory_usage_mb FLOAT DEFAULT 0,
		last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		healthy BOOLEAN DEFAULT TRUE
	);

	CREATE TABLE IF NOT EXISTS orchestrator_requests (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		request_id VARCHAR(255) UNIQUE NOT NULL,
		task_type VARCHAR(100) NOT NULL,
		selected_model VARCHAR(255) NOT NULL,
		fallback_used BOOLEAN DEFAULT FALSE,
		response_time_ms INTEGER,
		success BOOLEAN DEFAULT TRUE,
		error_message TEXT,
		resource_pressure FLOAT,
		cost_estimate FLOAT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS system_resources (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		memory_available_gb FLOAT,
		memory_free_gb FLOAT,
		memory_total_gb FLOAT,
		cpu_usage_percent FLOAT,
		swap_used_percent FLOAT,
		recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_model_metrics_name ON model_metrics(model_name);
	CREATE INDEX IF NOT EXISTS idx_orchestrator_requests_model ON orchestrator_requests(selected_model);
	CREATE INDEX IF NOT EXISTS idx_system_resources_recorded ON system_resources(recorded_at);
	`

	_, err := db.Exec(schema)
	return err
}

// Get system resource metrics
func getSystemMetrics() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to GB
	totalMem := float64(m.Sys) / 1024 / 1024 / 1024
	freeMem := float64(m.Frees) / 1024 / 1024 / 1024
	usedMem := totalMem - freeMem

	memoryPressure := 0.0
	if totalMem > 0 {
		memoryPressure = usedMem / totalMem
	}

	return map[string]interface{}{
		"memoryPressure":    memoryPressure,
		"availableMemoryGb": freeMem,
		"totalMemoryGb":     totalMem,
		"usedMemoryGb":      usedMem,
		"cpuUsage":          runtime.NumGoroutine(), // Simplified CPU metric
	}
}

// Background monitoring with health checks and reconnection
func startSystemMonitoring(app *AppState) {
	metricsTicker := time.NewTicker(30 * time.Second) // Store metrics every 30 seconds
	healthTicker := time.NewTicker(10 * time.Second)  // Health check every 10 seconds
	defer metricsTicker.Stop()
	defer healthTicker.Stop()

	app.Logger.Printf("🔄 Started background system monitoring with health checks")

	for {
		select {
		case <-healthTicker.C:
			// Check database health and reconnect if needed
			go checkAndReconnectDatabase(app)
			// Check Redis health and reconnect if needed
			go checkAndReconnectRedis(app)

		case <-metricsTicker.C:
			app.DBMutex.RLock()
			db := app.DB
			app.DBMutex.RUnlock()

			if db != nil {
				metrics := getSystemMetrics()

				if err := storeSystemResources(
					db,
					metrics["availableMemoryGb"].(float64),
					metrics["availableMemoryGb"].(float64), // Using available as free for simplicity
					metrics["totalMemoryGb"].(float64),
					metrics["cpuUsage"].(float64),
					0.0, // swap usage - would need to implement proper swap monitoring
				); err != nil {
					app.Logger.Printf("⚠️  Failed to store system resources: %v", err)
				}
			}
		}
	}
}

// Check database health and reconnect if needed
func checkAndReconnectDatabase(app *AppState) {
	app.DBMutex.Lock()
	defer app.DBMutex.Unlock()

	// Skip if already reconnecting
	if app.DBReconnecting {
		return
	}

	// Test current connection
	if app.DB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := app.DB.PingContext(ctx); err == nil {
			app.DBLastHealthCheck = time.Now()
			return // Connection is healthy
		}

		app.Logger.Printf("⚠️  Database health check failed, attempting reconnection")
		app.DB.Close()
		app.DB = nil
	}

	// Mark as reconnecting
	app.DBReconnecting = true
	defer func() { app.DBReconnecting = false }()

	// Attempt reconnection with exponential backoff
	maxRetries := 5
	backoffBase := time.Second
	rand.Seed(time.Now().UnixNano())

	for attempt := 1; attempt <= maxRetries; attempt++ {
		newDB, err := initDatabase(app.Logger)
		if err == nil {
			app.DB = newDB
			app.DBLastHealthCheck = time.Now()
			app.Logger.Printf("✅ Database reconnected successfully on attempt %d", attempt)

			// Reinitialize schema if needed
			if err := initSchema(app.DB); err != nil {
				app.Logger.Printf("⚠️  Schema reinitialization failed: %v", err)
			}
			return
		}

		if attempt < maxRetries {
			baseDelay := time.Duration(math.Pow(2, float64(attempt-1))) * backoffBase
			if baseDelay > 30*time.Second {
				baseDelay = 30 * time.Second
			}
			jitter := time.Duration(rand.Float64() * float64(baseDelay) * 0.25)
			delay := baseDelay + jitter
			app.Logger.Printf("⚠️  Database reconnection failed (attempt %d/%d), retrying in %v",
				attempt, maxRetries, delay)
			time.Sleep(delay)
		}
	}

	app.Logger.Printf("❌ Database reconnection failed after %d attempts", maxRetries)
}

// Check Redis health and reconnect if needed
func checkAndReconnectRedis(app *AppState) {
	app.Mutex.Lock()
	defer app.Mutex.Unlock()

	if app.Redis == nil {
		// Attempt to connect if Redis is nil
		newRedis, err := initRedis(app.Logger)
		if err == nil {
			app.Redis = newRedis
			app.Logger.Printf("✅ Redis connected successfully")
		}
		return
	}

	// Test current connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := app.Redis.Ping(ctx).Err(); err != nil {
		app.Logger.Printf("⚠️  Redis health check failed: %v, attempting reconnection", err)
		app.Redis.Close()

		// Attempt reconnection
		newRedis, err := initRedis(app.Logger)
		if err == nil {
			app.Redis = newRedis
			app.Logger.Printf("✅ Redis reconnected successfully")
		} else {
			app.Redis = nil
			app.Logger.Printf("❌ Redis reconnection failed: %v", err)
		}
	}
}
