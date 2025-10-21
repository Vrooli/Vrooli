package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

const (
	maxRetries     = 10
	initialBackoff = 1 * time.Second
	maxBackoff     = 60 * time.Second
	backoffFactor  = 2.0
	healthCheckInterval = 30 * time.Second
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DatabaseConnection manages database connection with health monitoring
type DatabaseConnection struct {
	config    *DatabaseConfig
	db        *sql.DB
	mu        sync.RWMutex
	connected bool
	stopChan  chan struct{}
}

// NewDatabaseConfig creates a new database configuration from environment
func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		URL:             os.Getenv("DATABASE_URL"),
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

// NewDatabaseConnection creates and initializes a database connection
func NewDatabaseConnection(config *DatabaseConfig) (*DatabaseConnection, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL not configured")
	}

	conn := &DatabaseConnection{
		config:   config,
		stopChan: make(chan struct{}),
	}

	// Attempt initial connection
	if err := conn.connect(); err != nil {
		return nil, err
	}

	// Start health monitoring
	go conn.monitorHealth()

	return conn, nil
}

// connect establishes database connection with exponential backoff
func (conn *DatabaseConnection) connect() error {
	backoff := initialBackoff

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Attempting database connection (attempt %d/%d)...", attempt, maxRetries)

		db, err := sql.Open("postgres", conn.config.URL)
		if err != nil {
			log.Printf("Failed to open database: %v", err)
			if attempt < maxRetries {
				log.Printf("Retrying in %v...", backoff)
				time.Sleep(backoff)
				backoff = calculateBackoff(backoff)
				continue
			}
			return fmt.Errorf("failed to open database after %d attempts: %w", maxRetries, err)
		}

		// Configure connection pool
		db.SetMaxOpenConns(conn.config.MaxOpenConns)
		db.SetMaxIdleConns(conn.config.MaxIdleConns)
		db.SetConnMaxLifetime(conn.config.ConnMaxLifetime)
		db.SetConnMaxIdleTime(conn.config.ConnMaxIdleTime)

		// Test the connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			log.Printf("Failed to ping database: %v", err)
			db.Close()
			if attempt < maxRetries {
				log.Printf("Retrying in %v...", backoff)
				time.Sleep(backoff)
				backoff = calculateBackoff(backoff)
				continue
			}
			return fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
		}

		// Set schema search path
		if _, err := db.Exec("SET search_path TO text_tools, public"); err != nil {
			log.Printf("Warning: Could not set search path: %v", err)
		}

		// Store connection
		conn.mu.Lock()
		conn.db = db
		conn.connected = true
		conn.mu.Unlock()

		log.Println("Successfully connected to database")
		return nil
	}

	return fmt.Errorf("database connection failed after %d attempts", maxRetries)
}

// monitorHealth continuously monitors database health and reconnects if needed
func (conn *DatabaseConnection) monitorHealth() {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := conn.healthCheck(); err != nil {
				log.Printf("Database health check failed: %v", err)
				conn.handleDisconnection()
			}
		case <-conn.stopChan:
			return
		}
	}
}

// healthCheck performs a database health check
func (conn *DatabaseConnection) healthCheck() error {
	conn.mu.RLock()
	db := conn.db
	isConnected := conn.connected
	conn.mu.RUnlock()

	if !isConnected || db == nil {
		return fmt.Errorf("database not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

// handleDisconnection handles database disconnection and attempts reconnection
func (conn *DatabaseConnection) handleDisconnection() {
	conn.mu.Lock()
	conn.connected = false
	if conn.db != nil {
		conn.db.Close()
		conn.db = nil
	}
	conn.mu.Unlock()

	log.Println("Database disconnected, attempting to reconnect...")

	// Attempt reconnection with exponential backoff
	backoff := initialBackoff
	for {
		select {
		case <-conn.stopChan:
			return
		default:
			if err := conn.connect(); err != nil {
				log.Printf("Reconnection failed: %v", err)
				log.Printf("Retrying in %v...", backoff)
				time.Sleep(backoff)
				backoff = calculateBackoff(backoff)
			} else {
				log.Println("Successfully reconnected to database")
				return
			}
		}
	}
}

// GetDB returns the database connection (may be nil if not connected)
func (conn *DatabaseConnection) GetDB() *sql.DB {
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.db
}

// IsConnected returns whether the database is currently connected
func (conn *DatabaseConnection) IsConnected() bool {
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.connected
}

// Close closes the database connection and stops monitoring
func (conn *DatabaseConnection) Close() error {
	close(conn.stopChan)

	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.db != nil {
		err := conn.db.Close()
		conn.db = nil
		conn.connected = false
		return err
	}

	return nil
}

// Query executes a query with automatic retry on connection failure
func (conn *DatabaseConnection) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	db := conn.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil && isConnectionError(err) {
		// Trigger reconnection
		go conn.handleDisconnection()
		return nil, fmt.Errorf("database connection lost: %w", err)
	}

	return rows, err
}

// Exec executes a statement with automatic retry on connection failure
func (conn *DatabaseConnection) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	db := conn.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil && isConnectionError(err) {
		// Trigger reconnection
		go conn.handleDisconnection()
		return nil, fmt.Errorf("database connection lost: %w", err)
	}

	return result, err
}

// Helper functions

func calculateBackoff(current time.Duration) time.Duration {
	next := time.Duration(float64(current) * backoffFactor)
	if next > maxBackoff {
		return maxBackoff
	}
	return next
}

func isConnectionError(err error) bool {
	// Check for common connection error patterns
	errStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"invalid connection",
	}

	for _, pattern := range connectionErrors {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}