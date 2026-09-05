package main

import (
	"os"
	"testing"
)

// [REQ:KO-SS-001,KO-SS-002,KO-SS-003] Test server creation error handling
func TestNewServerIntegration(t *testing.T) {
	// The scenario stores relational data in SQLite, so no PostgreSQL
	// environment is required to boot. This test previously asserted the
	// opposite — that a missing POSTGRES_* set must fail startup — which was
	// the contract that made the scenario unstartable without a provisioned
	// POSTGRES_PASSWORD. Removing that requirement is the point of the engine
	// migration, so the assertion is inverted rather than deleted.
	t.Run("starts without any postgres configuration", func(t *testing.T) {
		for _, key := range []string{
			"DATABASE_URL", "POSTGRES_URL", "POSTGRES_USER", "POSTGRES_HOST",
			"POSTGRES_PORT", "POSTGRES_PASSWORD", "POSTGRES_DB",
		} {
			t.Setenv(key, "")
			_ = os.Unsetenv(key)
		}
		t.Setenv("API_PORT", "8080")
		// Keep the test's database inside the test's own tree rather than the
		// operator's real data directory.
		t.Setenv("VROOLI_DATA_ROOT", t.TempDir())

		srv, err := NewServer()
		if err != nil {
			t.Fatalf("NewServer() must succeed with no postgres configuration, got: %v", err)
		}
		if srv == nil {
			t.Fatal("NewServer() returned no server and no error")
		}
		if srv.db == nil {
			t.Fatal("NewServer() left the database unset; SQLite needs no credentials and must always open")
		}
		if srv.stores == nil {
			t.Fatal("NewServer() left the domain repositories unset")
		}
		t.Cleanup(func() { _ = srv.db.Close() })
	})
}
