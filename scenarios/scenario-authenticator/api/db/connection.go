package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

var (
	DB          *sql.DB
	RedisClient *redis.Client
	Ctx         = context.Background()
)

// InitDB initializes the PostgreSQL database connection with exponential backoff
func InitDB(dbURL string) error {
	var err error
	DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("[scenario-authenticator/db] 🔄 Attempting database connection with exponential backoff...")
	log.Printf("[scenario-authenticator/db] 📆 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = DB.Ping()
		if pingErr == nil {
			log.Printf("[scenario-authenticator/db] ✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(time.Now().UnixNano() % int64(jitterRange))
		actualDelay := delay + jitter

		log.Printf("[scenario-authenticator/db] ⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("[scenario-authenticator/db] ⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("[scenario-authenticator/db] 📈 Retry progress:")
			log.Printf("[scenario-authenticator/db]    - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("[scenario-authenticator/db]    - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("[scenario-authenticator/db]    - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		return fmt.Errorf("database connection failed after %d attempts: %w", maxRetries, pingErr)
	}

	log.Println("[scenario-authenticator/db] 🎉 Database connection pool established successfully!")
	return nil
}

// InitRedis initializes the Redis connection with resilience for slow-starting resources
func InitRedis(redisURL string) error {
	if redisURL == "" {
		return fmt.Errorf("redis URL is required")
	}

	var (
		opts *redis.Options
		err  error
	)

	opts, err = redis.ParseURL(redisURL)
	if err != nil {
		addr := redisURL
		if strings.HasPrefix(addr, "redis://") {
			addr = strings.TrimPrefix(addr, "redis://")
		}
		if idx := strings.Index(addr, "/"); idx != -1 {
			addr = addr[:idx]
		}

		opts = &redis.Options{
			Addr:     addr,
			Password: "",
			DB:       0,
		}
	}

	RedisClient = redis.NewClient(opts)

	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second

	log.Println("[scenario-authenticator/redis] 🔄 Attempting Redis connection with exponential backoff...")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = RedisClient.Ping(Ctx).Err()
		if pingErr == nil {
			log.Printf("[scenario-authenticator/redis] ✅ Redis connected successfully on attempt %d", attempt+1)
			return nil
		}

		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		jitterRange := float64(delay) * 0.25
		var jitter time.Duration
		if jitterRange > 0 {
			jitter = time.Duration(time.Now().UnixNano() % int64(jitterRange))
		}
		actualDelay := delay + jitter

		log.Printf("[scenario-authenticator/redis] ⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("[scenario-authenticator/redis] ⏳ Waiting %v before next attempt", actualDelay)

		time.Sleep(actualDelay)
	}

	RedisClient.Close()
	log.Printf("[scenario-authenticator/redis] ❌ Redis connection failed after %d attempts: %v", maxRetries, pingErr)
	return fmt.Errorf("redis connection failed after %d attempts: %w", maxRetries, pingErr)
}

// Close closes database connections
func Close() {
	if DB != nil {
		DB.Close()
	}
	if RedisClient != nil {
		RedisClient.Close()
	}
}
