package server

import (
	"database/sql"
	"log"
	"math"
	"math/rand"
	"time"

	_ "github.com/lib/pq"

	"system-monitor-api/internal/config"
)

func connectDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConnections)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	maxRetries := 10
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	for attempt := 0; attempt < maxRetries; attempt++ {
		err = db.Ping()
		if err == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			return db, nil
		}

		delay := time.Duration(math.Min(float64(500*time.Millisecond)*math.Pow(2, float64(attempt)), float64(30*time.Second)))
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(randSource.Float64() * jitterRange)
		waitTime := delay + jitter

		log.Printf("⚠️  Database connection attempt %d/%d failed: %v", attempt+1, maxRetries, err)
		log.Printf("⏳ Waiting %v before next attempt", waitTime)
		time.Sleep(waitTime)
	}

	return nil, err
}
