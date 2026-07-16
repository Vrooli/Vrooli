// Package pgtest is the shared PostgreSQL test harness for the API's domain
// tests. It resolves a database from TEST_DATABASE_URL when set, otherwise
// starts a single throwaway postgres:15-alpine testcontainer shared across the
// whole test binary. Each domain test applies the schema(s) it needs via Apply.
package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NewDB returns a connected, pinged *sql.DB for tests. It skips the test (not
// fails) when no database is reachable and testcontainers are disabled, so CI
// without Docker degrades gracefully.
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	// Test-only helper: spinning up an ephemeral container is the correct
	// fallback when TEST_DATABASE_URL is unset, not a production default.
	// vrooli:env:optional
	dbURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dbURL == "" {
		dbURL = startContainer(t)
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// Apply runs each schema provider's SQL against db, failing the test on error.
func Apply(t *testing.T, db *sql.DB, providers ...func() string) {
	t.Helper()
	for _, p := range providers {
		sqlText := strings.TrimSpace(p())
		if sqlText == "" {
			continue
		}
		if _, err := db.Exec(sqlText); err != nil {
			t.Fatalf("apply schema: %v", err)
		}
	}
}

var (
	containerOnce sync.Once
	containerURL  string
	containerErr  error
)

func startContainer(t *testing.T) string {
	t.Helper()
	containerOnce.Do(func() {
		if strings.EqualFold(os.Getenv("TESTCONTAINERS_DISABLED"), "true") {
			containerErr = fmt.Errorf("testcontainers disabled")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		const user, pass, dbName = "testuser", "testpass", "landing_manager_test"
		container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
			ContainerRequest: tc.ContainerRequest{
				Image:        "postgres:15-alpine",
				Env:          map[string]string{"POSTGRES_USER": user, "POSTGRES_PASSWORD": pass, "POSTGRES_DB": dbName},
				ExposedPorts: []string{"5432/tcp"},
				WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
			},
			Started: true,
		})
		if err != nil {
			containerErr = fmt.Errorf("start postgres container: %w", err)
			return
		}
		host, err := container.Host(ctx)
		if err != nil {
			containerErr = fmt.Errorf("container host: %w", err)
			return
		}
		port, err := container.MappedPort(ctx, "5432/tcp")
		if err != nil {
			containerErr = fmt.Errorf("container port: %w", err)
			return
		}
		containerURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port.Port(), dbName)
	})
	if containerErr != nil {
		t.Skipf("no test database available: %v", containerErr)
	}
	return containerURL
}
