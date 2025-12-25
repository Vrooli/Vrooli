package server

import (
	"context"
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"

	"system-monitor-api/internal/config"
)

func connectDatabase(cfg *config.Config) (*sql.DB, error) {
	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConnections)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	log.Printf("✅ Database connected successfully")
	return db, nil
}
