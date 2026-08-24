package main

import (
	"context"
	"database/sql"
	schema "local-info-scout/internal/scout"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
)

var db *sql.DB

// initDB initializes the PostgreSQL database connection with automatic retry and backoff.
// Reads POSTGRES_* environment variables set by the lifecycle system.
func initDB() {
	// Check if database configuration is available through the shared seam.
	dsn, dsnErr := database.ResolvePostgresDSN(os.Getenv)
	if dsnErr != nil || dsn == "" {
		dbLogger.Info("Database configuration not found, persistence disabled", nil)
		return
	}

	var err error
	db, err = database.Connect(context.Background(), database.Config{
		Driver:          "postgres",
		DSN:             dsn,
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		dbLogger.Error("PostgreSQL connection failed, persistence disabled", map[string]interface{}{
			"error": err.Error(),
		})
		db = nil
		return
	}

	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(schema.Schema)); err != nil {
		dbLogger.Error("schema initialization failed", map[string]interface{}{"error": err.Error()})
		_ = db.Close()
		db = nil
		return
	}

	// Set additional pool settings not covered by database.Connect
	db.SetConnMaxIdleTime(1 * time.Minute)

	dbLogger.Info("PostgreSQL connected with connection pooling", map[string]interface{}{
		"max_open":      25,
		"max_idle":      5,
		"max_lifetime":  "5m",
		"max_idle_time": "1m",
	})

}

// createTables reapplies the canonical domain-owned schema for compatibility
// with older tests that call this helper directly.
func createTables() {
	if db == nil {
		return
	}

	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(schema.Schema)); err != nil {
		dbLogger.Error("Failed to apply local-info-scout schema", map[string]interface{}{"error": err.Error()})
	}
}

// savePlaceToDb saves a place to the database
func savePlaceToDb(place Place) error {
	if db == nil {
		return nil
	}

	query := `
    INSERT INTO lis_places (id, name, address, category, lat, lon, rating, price_level, open_now, description)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        address = EXCLUDED.address,
        category = EXCLUDED.category,
        rating = EXCLUDED.rating,
        price_level = EXCLUDED.price_level,
        open_now = EXCLUDED.open_now,
        description = EXCLUDED.description,
        updated_at = CURRENT_TIMESTAMP
    `

	_, err := db.Exec(query, place.ID, place.Name, place.Address, place.Category,
		0.0, 0.0, // We don't have real lat/lon yet
		place.Rating, place.PriceLevel, place.OpenNow, place.Description)

	return err
}

// logSearch logs a search request to the database
func logSearch(req SearchRequest, resultsCount int, cacheHit bool, duration time.Duration) {
	if db == nil {
		return
	}

	query := `
    INSERT INTO lis_search_logs (query, lat, lon, radius, category, results_count, cache_hit, search_time_ms)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `

	_, err := db.Exec(query, req.Query, req.Lat, req.Lon, req.Radius, req.Category,
		resultsCount, cacheHit, duration.Milliseconds())
	if err != nil {
		dbLogger.Error("Failed to log search", map[string]interface{}{
			"error": err.Error(),
			"query": req.Query,
		})
	}
}

// getPopularSearches returns the most popular recent searches
func getPopularSearches() []string {
	if db == nil {
		return []string{}
	}

	query := `
    SELECT query, COUNT(*) as count
    FROM lis_search_logs
    WHERE query IS NOT NULL AND query != ''
        AND created_at > NOW() - INTERVAL '24 hours'
    GROUP BY query
    ORDER BY count DESC
    LIMIT 10
    `

	rows, err := db.Query(query)
	if err != nil {
		dbLogger.Error("Failed to get popular searches", map[string]interface{}{
			"error": err.Error(),
		})
		return []string{}
	}
	defer rows.Close()

	var searches []string
	for rows.Next() {
		var query string
		var count int
		if err := rows.Scan(&query, &count); err == nil {
			searches = append(searches, query)
		}
	}

	return searches
}

// getDB returns the database connection (useful for testing)
func getDB() *sql.DB {
	return db
}
