package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// initDB initializes the PostgreSQL database connection with connection pooling
func initDB() {
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "postgres")
	dbname := getEnv("POSTGRES_DB", "local_info_scout")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		dbLogger.Error("PostgreSQL connection failed, persistence disabled", map[string]interface{}{
			"error": err.Error(),
			"host":  host,
			"port":  port,
		})
		db = nil
		return
	}

	// Configure connection pooling for better performance
	// Max open connections: Allow up to 25 concurrent database connections
	db.SetMaxOpenConns(25)
	// Max idle connections: Keep 5 connections in the pool when idle
	db.SetMaxIdleConns(5)
	// Connection max lifetime: Close connections after 5 minutes to prevent stale connections
	db.SetConnMaxLifetime(5 * time.Minute)
	// Connection max idle time: Close idle connections after 1 minute
	db.SetConnMaxIdleTime(1 * time.Minute)

	err = db.Ping()
	if err != nil {
		dbLogger.Error("PostgreSQL ping failed, persistence disabled", map[string]interface{}{
			"error": err.Error(),
			"host":  host,
			"port":  port,
		})
		db = nil
		return
	}

	dbLogger.Info("PostgreSQL connected with connection pooling", map[string]interface{}{
		"host":          host,
		"port":          port,
		"max_open":      25,
		"max_idle":      5,
		"max_lifetime":  "5m",
		"max_idle_time": "1m",
	})

	// Create tables if they don't exist
	createTables()
}

// createTables creates the necessary database tables with lis_ prefix to avoid conflicts
func createTables() {
	if db == nil {
		return
	}

	createPlacesTable := `
    CREATE TABLE IF NOT EXISTS lis_places (
        id VARCHAR(50) PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        address VARCHAR(500),
        category VARCHAR(50),
        lat DOUBLE PRECISION,
        lon DOUBLE PRECISION,
        rating DECIMAL(2,1),
        price_level INTEGER,
        open_now BOOLEAN,
        description TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_lis_places_category ON lis_places(category);
    CREATE INDEX IF NOT EXISTS idx_lis_places_location ON lis_places(lat, lon);
    CREATE INDEX IF NOT EXISTS idx_lis_places_rating ON lis_places(rating);
    `

	createSearchLogsTable := `
    CREATE TABLE IF NOT EXISTS lis_search_logs (
        id SERIAL PRIMARY KEY,
        query TEXT,
        lat DOUBLE PRECISION,
        lon DOUBLE PRECISION,
        radius DOUBLE PRECISION,
        category VARCHAR(50),
        results_count INTEGER,
        cache_hit BOOLEAN,
        search_time_ms INTEGER,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE INDEX IF NOT EXISTS idx_lis_search_logs_created ON lis_search_logs(created_at);
    `

	_, err := db.Exec(createPlacesTable)
	if err != nil {
		dbLogger.Error("Failed to create lis_places table", map[string]interface{}{
			"error": err.Error(),
		})
	}

	_, err = db.Exec(createSearchLogsTable)
	if err != nil {
		dbLogger.Error("Failed to create lis_search_logs table", map[string]interface{}{
			"error": err.Error(),
		})
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
