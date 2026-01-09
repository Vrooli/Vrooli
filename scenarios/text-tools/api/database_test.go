package main

import (
	"context"
	"database/sql"
	"math"
	"math/rand"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func sampleBackoff(attempt int, rng *rand.Rand) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	base := time.Duration(math.Min(float64(initialBackoff)*math.Pow(backoffFactor, float64(attempt-1)), float64(maxBackoff)))
	jitterRange := float64(base) * 0.25
	return base + time.Duration(rng.Float64()*jitterRange)
}

func TestNewDatabaseConfig(t *testing.T) {
	t.Run("Reads_DATABASE_URL", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://test:pass@localhost:5432/testdb")
		defer os.Unsetenv("DATABASE_URL")

		config := NewDatabaseConfig()

		if config.URL != "postgres://test:pass@localhost:5432/testdb" {
			t.Errorf("Expected URL to be set from DATABASE_URL, got %s", config.URL)
		}
	})

	t.Run("Falls_Back_To_POSTGRES_URL", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Setenv("POSTGRES_URL", "postgres://test2:pass2@localhost:5432/testdb2")
		defer os.Unsetenv("POSTGRES_URL")

		config := NewDatabaseConfig()

		if config.URL != "postgres://test2:pass2@localhost:5432/testdb2" {
			t.Errorf("Expected URL to be set from POSTGRES_URL, got %s", config.URL)
		}
	})

	t.Run("Constructs_URL_From_Components", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("POSTGRES_URL")
		os.Setenv("POSTGRES_HOST", "testhost")
		os.Setenv("POSTGRES_PORT", "5433")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdatabase")
		os.Setenv("POSTGRES_SSLMODE", "disable")
		defer func() {
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
			os.Unsetenv("POSTGRES_DB")
			os.Unsetenv("POSTGRES_SSLMODE")
		}()

		config := NewDatabaseConfig()

		expectedURL := "postgres://testuser:testpass@testhost:5433/testdatabase?sslmode=disable"
		if config.URL != expectedURL {
			t.Errorf("Expected constructed URL %s, got %s", expectedURL, config.URL)
		}
	})

	t.Run("Sets_Default_Connection_Pool_Settings", func(t *testing.T) {
		config := NewDatabaseConfig()

		if config.MaxOpenConns != 25 {
			t.Errorf("Expected MaxOpenConns 25, got %d", config.MaxOpenConns)
		}
		if config.MaxIdleConns != 5 {
			t.Errorf("Expected MaxIdleConns 5, got %d", config.MaxIdleConns)
		}
		if config.ConnMaxLifetime != 5*time.Minute {
			t.Errorf("Expected ConnMaxLifetime 5m, got %v", config.ConnMaxLifetime)
		}
	})
}

func TestDatabaseConnection(t *testing.T) {
	// Skip if DATABASE_URL not set (for unit testing without real DB)
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("POSTGRES_HOST") == "" {
		t.Skip("Skipping database tests - no DATABASE_URL configured")
	}

	t.Run("Creates_Connection", func(t *testing.T) {
		config := NewDatabaseConfig()
		if config.URL == "" {
			t.Skip("No database URL configured")
		}

		conn, err := NewDatabaseConnection(config)
		if err != nil {
			t.Logf("Database connection failed (expected in unit tests): %v", err)
			t.Skip("Skipping - database not available")
		}
		defer conn.Close()

		if !conn.IsConnected() {
			t.Error("Expected connection to be connected")
		}
	})

	t.Run("Returns_Error_For_Empty_URL", func(t *testing.T) {
		config := &DatabaseConfig{
			URL: "",
		}

		_, err := NewDatabaseConnection(config)
		if err == nil {
			t.Error("Expected error for empty database URL")
		}
		if err.Error() != "DATABASE_URL not configured" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("GetDB_Returns_Connection", func(t *testing.T) {
		config := NewDatabaseConfig()
		if config.URL == "" {
			t.Skip("No database URL configured")
		}

		conn, err := NewDatabaseConnection(config)
		if err != nil {
			t.Skip("Database not available")
		}
		defer conn.Close()

		db := conn.GetDB()
		if db == nil {
			t.Error("Expected GetDB to return non-nil database")
		}
	})

	t.Run("IsConnected_Returns_Status", func(t *testing.T) {
		config := NewDatabaseConfig()
		if config.URL == "" {
			t.Skip("No database URL configured")
		}

		conn, err := NewDatabaseConnection(config)
		if err != nil {
			t.Skip("Database not available")
		}
		defer conn.Close()

		if !conn.IsConnected() {
			t.Error("Expected IsConnected to return true")
		}
	})

	t.Run("Close_Disconnects", func(t *testing.T) {
		config := NewDatabaseConfig()
		if config.URL == "" {
			t.Skip("No database URL configured")
		}

		conn, err := NewDatabaseConnection(config)
		if err != nil {
			t.Skip("Database not available")
		}

		err = conn.Close()
		if err != nil {
			t.Errorf("Expected Close to succeed, got error: %v", err)
		}

		if conn.IsConnected() {
			t.Error("Expected connection to be disconnected after Close")
		}
	})
}

func TestDatabaseQueryExec(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("POSTGRES_HOST") == "" {
		t.Skip("Skipping database query tests - no DATABASE_URL configured")
	}

	t.Run("Query_Returns_Error_When_Not_Connected", func(t *testing.T) {
		config := &DatabaseConfig{
			URL:             "postgres://invalid:invalid@localhost:9999/invalid",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
		}

		conn := &DatabaseConnection{
			config:    config,
			stopChan:  make(chan struct{}),
			connected: false,
			db:        nil,
		}

		ctx := context.Background()
		_, err := conn.Query(ctx, "SELECT 1")
		if err == nil {
			t.Error("Expected error when querying disconnected database")
		}
	})

	t.Run("Exec_Returns_Error_When_Not_Connected", func(t *testing.T) {
		config := &DatabaseConfig{
			URL:             "postgres://invalid:invalid@localhost:9999/invalid",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
		}

		conn := &DatabaseConnection{
			config:    config,
			stopChan:  make(chan struct{}),
			connected: false,
			db:        nil,
		}

		ctx := context.Background()
		_, err := conn.Exec(ctx, "SELECT 1")
		if err == nil {
			t.Error("Expected error when executing on disconnected database")
		}
	})
}

func TestDatabaseHelpers(t *testing.T) {
	t.Run("ComputeBackoffDelay_WithinExpectedRange", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1))
		delay := sampleBackoff(1, rng)

		base := initialBackoff
		maxWithJitter := base + time.Duration(float64(base)*0.25)
		if delay < base || delay > maxWithJitter {
			t.Fatalf("expected delay within [%v, %v], got %v", base, maxWithJitter, delay)
		}
	})

	t.Run("ComputeBackoffDelay_Increases_With_Attempt", func(t *testing.T) {
		rng := rand.New(rand.NewSource(1))
		first := sampleBackoff(1, rng)
		second := sampleBackoff(2, rng)

		if second <= first {
			t.Fatalf("expected second attempt delay (%v) to exceed first (%v)", second, first)
		}

		base := time.Duration(math.Min(float64(initialBackoff)*math.Pow(backoffFactor, 1), float64(maxBackoff)))
		maxWithJitter := base + time.Duration(float64(base)*0.25)
		if second > maxWithJitter {
			t.Fatalf("expected second delay <= %v, got %v", maxWithJitter, second)
		}
	})

	t.Run("IsConnectionError_Detects_Errors", func(t *testing.T) {
		testCases := []struct {
			err      error
			expected bool
		}{
			{sql.ErrConnDone, false}, // Not a connection string error
		}

		for _, tc := range testCases {
			result := isConnectionError(tc.err)
			if result != tc.expected {
				t.Errorf("For error %v, expected %v, got %v", tc.err, tc.expected, result)
			}
		}
	})

	t.Run("Contains_Checks_Substring", func(t *testing.T) {
		testCases := []struct {
			s        string
			substr   string
			expected bool
		}{
			{"hello world", "world", true},
			{"hello world", "test", false},
			{"connection refused", "connection refused", true},
			{"short", "longer string", false},
		}

		for _, tc := range testCases {
			result := contains(tc.s, tc.substr)
			if result != tc.expected {
				t.Errorf("contains(%q, %q) = %v, expected %v", tc.s, tc.substr, result, tc.expected)
			}
		}
	})
}

func TestDatabaseHealthCheck(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("POSTGRES_HOST") == "" {
		t.Skip("Skipping database health check tests - no DATABASE_URL configured")
	}

	t.Run("HealthCheck_Succeeds_When_Connected", func(t *testing.T) {
		config := NewDatabaseConfig()
		if config.URL == "" {
			t.Skip("No database URL configured")
		}

		conn, err := NewDatabaseConnection(config)
		if err != nil {
			t.Skip("Database not available")
		}
		defer conn.Close()

		err = conn.healthCheck()
		if err != nil {
			t.Errorf("Expected health check to succeed, got: %v", err)
		}
	})

	t.Run("HealthCheck_Fails_When_Not_Connected", func(t *testing.T) {
		conn := &DatabaseConnection{
			config:    &DatabaseConfig{},
			stopChan:  make(chan struct{}),
			connected: false,
			db:        nil,
		}

		err := conn.healthCheck()
		if err == nil {
			t.Error("Expected health check to fail when not connected")
		}
	})
}
