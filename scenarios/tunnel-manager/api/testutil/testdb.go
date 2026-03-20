package testutil

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
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"tunnel-manager/store"
)

var (
	testContainerOnce    sync.Once
	testContainerURL     string
	testContainerCleanup func()
	testContainerInitErr error
)

func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dbURL == "" {
		dbURL = startTestContainerDB(t)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	if err := store.EnsureSchema(db); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	// Clean tables for test isolation
	CleanTables(t, db)

	return db
}

func startTestContainerDB(t *testing.T) string {
	t.Helper()

	testContainerOnce.Do(func() {
		if strings.EqualFold(os.Getenv("TESTCONTAINERS_DISABLED"), "true") {
			testContainerInitErr = fmt.Errorf("testcontainers explicitly disabled")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		container, err := postgres.Run(ctx,
			"postgres:15-alpine",
			postgres.WithDatabase("tunnel_manager_test"),
			postgres.WithUsername("test"),
			postgres.WithPassword("test"),
			tc.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second),
			),
		)
		if err != nil {
			testContainerInitErr = fmt.Errorf("start postgres container: %w", err)
			return
		}

		connStr, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			testContainerInitErr = fmt.Errorf("get connection string: %w", err)
			return
		}

		testContainerURL = connStr
		testContainerCleanup = func() {
			_ = container.Terminate(context.Background())
		}
	})

	if testContainerInitErr != nil {
		t.Fatalf("Failed to start test container: %v", testContainerInitErr)
	}
	// Don't register per-test cleanup — the Ryuk container handles cleanup,
	// and registering per-test would terminate the container after the first test finishes.
	return testContainerURL
}

func CleanTables(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"metrics_history", "probe_results", "recovery_events", "routes"} {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			t.Fatalf("Failed to clean table %s: %v", table, err)
		}
	}
}
