package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testContainerOnce    sync.Once
	testContainerURL     string
	testContainerCleanup func()
	testContainerInitErr error
)

func setupTestDB(t *testing.T) *sql.DB {
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

	ensureSchema(db)

	// Clean tables for test isolation
	cleanTables(t, db)

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

func cleanTables(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"metrics_history", "probe_results", "recovery_events", "routes"} {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			t.Fatalf("Failed to clean table %s: %v", table, err)
		}
	}
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

func writeServiceJSON(t *testing.T, dir string, uiPort int) {
	t.Helper()
	svc := map[string]any{
		"ports": map[string]any{
			"ui": map[string]any{
				"port":    uiPort,
				"env_var": "UI_PORT",
			},
		},
	}
	data, _ := json.Marshal(svc)
	if err := os.WriteFile(filepath.Join(dir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
}

// seedTestRoute inserts a route for testing and returns it.
func seedTestRoute(t *testing.T, db *sql.DB, subdomain, scenario string, port int) Route {
	t.Helper()
	svc := NewRouteService(db)
	route, err := svc.Create(RouteInput{
		Subdomain:    subdomain,
		ScenarioName: scenario,
		LocalPort:    port,
		PublicURL:    fmt.Sprintf("https://%s.example.com", subdomain),
	})
	if err != nil {
		t.Fatalf("seedTestRoute: %v", err)
	}
	return *route
}
