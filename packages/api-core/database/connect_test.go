package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/retry"
)

// mockDB simulates a sql.DB for testing
type mockDB struct {
	pingErr   error
	pingCount int
}

func (m *mockDB) PingContext(ctx context.Context) error {
	m.pingCount++
	return m.pingErr
}

func (m *mockDB) Close() error {
	return nil
}

// ============================================================================
// PostgreSQL DSN Building Tests
// ============================================================================

func TestBuildPostgresDSN_FromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name: "POSTGRES_URL takes precedence",
			env: map[string]string{
				"POSTGRES_URL":  "postgres://fromurl:pass@urlhost:5432/urldb",
				"DATABASE_URL":  "postgres://other:pass@otherhost:5432/otherdb",
				"POSTGRES_HOST": "componenthost",
			},
			expected: "postgres://fromurl:pass@urlhost:5432/urldb",
		},
		{
			name: "DATABASE_URL as fallback",
			env: map[string]string{
				"DATABASE_URL":  "postgres://dburl:pass@dbhost:5432/dbdb",
				"POSTGRES_HOST": "componenthost",
			},
			expected: "postgres://dburl:pass@dbhost:5432/dbdb",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string { return tc.env[key] }
			dsn, err := buildPostgresDSN(getenv)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dsn != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, dsn)
			}
		})
	}
}

func TestBuildPostgresDSN_FromComponents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name: "all components with password",
			env: map[string]string{
				"POSTGRES_HOST":     "localhost",
				"POSTGRES_PORT":     "5433",
				"POSTGRES_USER":     "vrooli",
				"POSTGRES_PASSWORD": "secret",
				"POSTGRES_DB":       "mydb",
				"POSTGRES_SSLMODE":  "require",
			},
			expected: "postgres://vrooli:secret@localhost:5433/mydb?sslmode=require",
		},
		{
			name: "without password",
			env: map[string]string{
				"POSTGRES_HOST": "localhost",
				"POSTGRES_PORT": "5433",
				"POSTGRES_USER": "vrooli",
				"POSTGRES_DB":   "mydb",
			},
			expected: "postgres://vrooli@localhost:5433/mydb?sslmode=disable",
		},
		{
			name: "default sslmode",
			env: map[string]string{
				"POSTGRES_HOST":     "localhost",
				"POSTGRES_PORT":     "5433",
				"POSTGRES_USER":     "vrooli",
				"POSTGRES_PASSWORD": "pass",
				"POSTGRES_DB":       "mydb",
			},
			expected: "postgres://vrooli:pass@localhost:5433/mydb?sslmode=disable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string { return tc.env[key] }
			dsn, err := buildPostgresDSN(getenv)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dsn != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, dsn)
			}
		})
	}
}

func TestBuildPostgresDSN_MissingRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		env             map[string]string
		expectedMissing []string
	}{
		{
			name:            "all missing",
			env:             map[string]string{},
			expectedMissing: []string{"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_DB"},
		},
		{
			name: "host missing",
			env: map[string]string{
				"POSTGRES_PORT": "5432",
				"POSTGRES_USER": "user",
				"POSTGRES_DB":   "db",
			},
			expectedMissing: []string{"POSTGRES_HOST"},
		},
		{
			name: "port missing",
			env: map[string]string{
				"POSTGRES_HOST": "localhost",
				"POSTGRES_USER": "user",
				"POSTGRES_DB":   "db",
			},
			expectedMissing: []string{"POSTGRES_PORT"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string { return tc.env[key] }
			_, err := buildPostgresDSN(getenv)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			for _, missing := range tc.expectedMissing {
				if !strings.Contains(err.Error(), missing) {
					t.Errorf("expected error to mention %q, got: %v", missing, err)
				}
			}
		})
	}
}

// ============================================================================
// SQLite DSN Building Tests
// ============================================================================

func TestBuildSQLiteDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name:     "SQLITE_PATH",
			env:      map[string]string{"SQLITE_PATH": "/data/app.db"},
			expected: "/data/app.db",
		},
		{
			name:     "SQLITE_DB",
			env:      map[string]string{"SQLITE_DB": "/tmp/test.db"},
			expected: "/tmp/test.db",
		},
		{
			name: "SQLITE_PATH takes precedence",
			env: map[string]string{
				"SQLITE_PATH": "/primary.db",
				"SQLITE_DB":   "/secondary.db",
			},
			expected: "/primary.db",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(key string) string { return tc.env[key] }
			dsn, err := buildSQLiteDSN(getenv)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dsn != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, dsn)
			}
		})
	}
}

func TestBuildSQLiteDSN_Missing(t *testing.T) {
	t.Parallel()

	getenv := func(key string) string { return "" }
	_, err := buildSQLiteDSN(getenv)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "SQLITE_PATH") {
		t.Errorf("expected error to mention SQLITE_PATH, got: %v", err)
	}
}

// ============================================================================
// Pool Settings Tests
// ============================================================================

func TestResolvePoolSettings_FromEnv(t *testing.T) {
	t.Parallel()

	cfg := Config{
		EnvGetter: func(key string) string {
			if key == "POSTGRES_POOL_SIZE" {
				return "50"
			}
			return ""
		},
	}

	resolvePoolSettings(&cfg)

	if cfg.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns: expected 50, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 10 { // 50/5
		t.Errorf("MaxIdleConns: expected 10, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("ConnMaxLifetime: expected 5m, got %v", cfg.ConnMaxLifetime)
	}
}

func TestResolvePoolSettings_Defaults(t *testing.T) {
	t.Parallel()

	cfg := Config{
		EnvGetter: func(key string) string { return "" },
	}

	resolvePoolSettings(&cfg)

	if cfg.MaxOpenConns != 25 {
		t.Errorf("MaxOpenConns: expected default 25, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 5 { // 25/5
		t.Errorf("MaxIdleConns: expected 5, got %d", cfg.MaxIdleConns)
	}
}

func TestResolvePoolSettings_MinIdleConns(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MaxOpenConns: 5, // 5/5 = 1, but minimum is 2
		EnvGetter:    func(key string) string { return "" },
	}

	resolvePoolSettings(&cfg)

	if cfg.MaxIdleConns != 2 {
		t.Errorf("MaxIdleConns: expected minimum 2, got %d", cfg.MaxIdleConns)
	}
}

func TestResolvePoolSettings_ExplicitOverrides(t *testing.T) {
	t.Parallel()

	cfg := Config{
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime: 10 * time.Minute,
		EnvGetter: func(key string) string {
			return "999" // Should be ignored
		},
	}

	resolvePoolSettings(&cfg)

	if cfg.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns: expected 100, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 20 {
		t.Errorf("MaxIdleConns: expected 20, got %d", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 10*time.Minute {
		t.Errorf("ConnMaxLifetime: expected 10m, got %v", cfg.ConnMaxLifetime)
	}
}

// ============================================================================
// Connect Tests
// ============================================================================

func TestConnect_BuildsDSNFromEnv(t *testing.T) {
	t.Parallel()

	var capturedDSN string
	env := map[string]string{
		"POSTGRES_HOST":     "testhost",
		"POSTGRES_PORT":     "5432",
		"POSTGRES_USER":     "testuser",
		"POSTGRES_PASSWORD": "testpass",
		"POSTGRES_DB":       "testdb",
		"POSTGRES_SSLMODE":  "require",
	}

	cfg := Config{
		Driver:    DriverPostgres,
		EnvGetter: func(key string) string { return env[key] },
		Opener: func(driver, dsn string) (*sql.DB, error) {
			capturedDSN = dsn
			// Return a real *sql.DB from sql.Open with a fake driver
			// This won't actually connect but satisfies the type
			return sql.Open("postgres", dsn)
		},
		Retry: &retry.Config{
			MaxAttempts: 1,
			Sleeper:     func(d time.Duration) {},
		},
	}

	// This will fail on Ping (no real DB), but we can check the DSN was built correctly
	Connect(context.Background(), cfg)

	expected := "postgres://testuser:testpass@testhost:5432/testdb?sslmode=require"
	if capturedDSN != expected {
		t.Fatalf("expected DSN %q, got %q", expected, capturedDSN)
	}
}

func TestConnect_UsesExplicitDSN(t *testing.T) {
	t.Parallel()

	var capturedDSN string
	cfg := Config{
		Driver: DriverPostgres,
		DSN:    "postgres://explicit:pass@host:5432/db",
		EnvGetter: func(key string) string {
			return "should-not-be-used"
		},
		Opener: func(driver, dsn string) (*sql.DB, error) {
			capturedDSN = dsn
			return sql.Open("postgres", dsn)
		},
		Retry: &retry.Config{
			MaxAttempts: 1,
			Sleeper:     func(d time.Duration) {},
		},
	}

	Connect(context.Background(), cfg)

	if capturedDSN != "postgres://explicit:pass@host:5432/db" {
		t.Fatalf("expected explicit DSN, got %q", capturedDSN)
	}
}

func TestConnect_RetriesOnPingFailure(t *testing.T) {
	t.Parallel()

	attempts := 0
	failUntil := 3

	env := map[string]string{
		"POSTGRES_HOST": "localhost",
		"POSTGRES_PORT": "5432",
		"POSTGRES_USER": "user",
		"POSTGRES_DB":   "db",
	}

	cfg := Config{
		Driver:    DriverPostgres,
		EnvGetter: func(key string) string { return env[key] },
		Opener: func(driver, dsn string) (*sql.DB, error) {
			attempts++
			if attempts < failUntil {
				// Simulate open succeeding but ping failing
				// We return a DB that will fail on Ping
				return nil, errors.New("connection refused")
			}
			return sql.Open("postgres", dsn)
		},
		Retry: &retry.Config{
			MaxAttempts: 5,
			BaseDelay:   time.Millisecond,
			Sleeper:     func(d time.Duration) {},
		},
	}

	// Will ultimately fail (no real DB) but should retry
	Connect(context.Background(), cfg)

	if attempts < failUntil {
		t.Fatalf("expected at least %d attempts, got %d", failUntil, attempts)
	}
}

func TestConnect_UnsupportedDriver(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Driver: "unsupported-driver",
	}

	_, err := Connect(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for unsupported driver")
	}
	if !strings.Contains(err.Error(), "unsupported-driver") {
		t.Errorf("expected error to mention driver name, got: %v", err)
	}
}

func TestConnect_LogsRetries(t *testing.T) {
	t.Parallel()

	var logs []string
	env := map[string]string{
		"POSTGRES_HOST": "localhost",
		"POSTGRES_PORT": "5432",
		"POSTGRES_USER": "user",
		"POSTGRES_DB":   "db",
	}

	cfg := Config{
		Driver:    DriverPostgres,
		EnvGetter: func(key string) string { return env[key] },
		Opener: func(driver, dsn string) (*sql.DB, error) {
			return nil, errors.New("connection refused")
		},
		Retry: &retry.Config{
			MaxAttempts: 3,
			BaseDelay:   time.Millisecond,
			Sleeper:     func(d time.Duration) {},
		},
		Logger: func(format string, args ...interface{}) {
			logs = append(logs, format)
		},
	}

	Connect(context.Background(), cfg)

	// Should have logged retry attempts
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 log messages, got %d", len(logs))
	}
}

func TestConnect_DefaultsToPostgres(t *testing.T) {
	t.Parallel()

	var capturedDriver string
	cfg := Config{
		DSN: "postgres://user:pass@host:5432/db", // Explicit DSN to avoid env lookup
		Opener: func(driver, dsn string) (*sql.DB, error) {
			capturedDriver = driver
			return sql.Open(driver, dsn)
		},
		Retry: &retry.Config{
			MaxAttempts: 1,
			Sleeper:     func(d time.Duration) {},
		},
	}

	Connect(context.Background(), cfg)

	if capturedDriver != DriverPostgres {
		t.Fatalf("expected default driver %q, got %q", DriverPostgres, capturedDriver)
	}
}

// ============================================================================
// MustConnect Tests
// ============================================================================

func TestMustConnect_Panics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic, got none")
		}
	}()

	cfg := Config{
		Driver: "unsupported",
	}

	MustConnect(context.Background(), cfg)
}

// ============================================================================
// Driver Auto-detection Tests
// ============================================================================

func TestBuildDSNFromEnv_UnknownDriver(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Driver:    "mysql",
		EnvGetter: func(key string) string { return "" },
	}

	_, err := buildDSNFromEnv(cfg)
	if err == nil {
		t.Fatal("expected error for unknown driver")
	}
	if !strings.Contains(err.Error(), "mysql") {
		t.Errorf("expected error to mention driver, got: %v", err)
	}
	if !strings.Contains(err.Error(), "provide DSN explicitly") {
		t.Errorf("expected error to suggest explicit DSN, got: %v", err)
	}
}
