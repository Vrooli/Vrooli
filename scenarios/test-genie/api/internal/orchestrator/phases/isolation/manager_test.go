package isolation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShouldRetainFromEnv(t *testing.T) {
	t.Setenv("TEST_GENIE_PLAYBOOKS_RETAIN", "1")
	if !ShouldRetainFromEnv() {
		t.Fatal("expected retain flag to honor environment override")
	}

	t.Setenv("TEST_GENIE_PLAYBOOKS_RETAIN", "0")
	if ShouldRetainFromEnv() {
		t.Fatal("expected retain flag to be false when env is not 1")
	}
}

func TestNewManagerAppliesDefaultTimeout(t *testing.T) {
	manager := NewManager(Config{})
	if manager.cfg.Timeout != 90*time.Second {
		t.Fatalf("expected default timeout of 90s, got %s", manager.cfg.Timeout)
	}
}

func TestBuildDBNameSanitizesAndTruncates(t *testing.T) {
	name := buildDBName("My Scenario!", strings.Repeat("run-id-", 20))
	if !strings.HasPrefix(name, "tg_pb_") {
		t.Fatalf("expected database name prefix, got %q", name)
	}
	if len(name) > 60 {
		t.Fatalf("expected database name to be truncated to 60 chars, got %d", len(name))
	}
	if strings.ContainsAny(name, " !") {
		t.Fatalf("expected database name to be sanitized, got %q", name)
	}
}

func TestMergeAndSanitize(t *testing.T) {
	merged := merge(map[string]string{"A": "1", "B": "2"}, map[string]string{"B": "override", "C": "3"})
	if merged["A"] != "1" || merged["B"] != "override" || merged["C"] != "3" {
		t.Fatalf("unexpected merged env map: %#v", merged)
	}

	if got := sanitize(" Test Genie/Playbooks "); got != "test_genie_playbooks" {
		t.Fatalf("expected sanitize to normalize punctuation and whitespace, got %q", got)
	}
}

func TestStartSQLiteProvisioning(t *testing.T) {
	manager := NewManager(Config{
		// A display name, not a slug: the manager must normalize it before
		// handing it to the storage seam, which requires a scenario identifier.
		ScenarioName:  "Test Genie",
		RequireSQLite: true,
		SQLiteEnvVars: []string{"APP_SQLITE_PATH"},
	})

	result, cleanup, err := manager.startSQLite(context.Background(), "run-id")
	if err != nil {
		t.Fatalf("startSQLite returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected sqlite start result")
	}

	// Isolation is expressed as a storage ROOT, not a database path. The root
	// is scenario-agnostic, so every scenario beneath it still resolves its own
	// separate file — which is why a leaked value isolates instead of colliding.
	root := result.env["VROOLI_STORAGE_ROOT"]
	if root == "" {
		t.Fatalf("expected an isolated storage root: %#v", result.env)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("expected the isolated storage root to exist: %v", err)
	}

	// A scenario-scoped variable is still supplied: its name carries its owner,
	// so it cannot capture a sibling's database.
	scoped := result.env["APP_SQLITE_PATH"]
	if scoped == "" {
		t.Fatalf("expected the declared scenario-scoped variable to be set: %#v", result.env)
	}
	if !strings.HasPrefix(scoped, root) {
		t.Fatalf("declared path %q is not inside the isolated root %q", scoped, root)
	}
	if filepath.Ext(scoped) != ".db" {
		t.Fatalf("expected a .db file, got %s", scoped)
	}
	if scoped != result.info.Endpoint {
		t.Fatalf("the reported endpoint %q disagrees with the provisioned path %q", result.info.Endpoint, scoped)
	}

	// The generic pair must NOT be injected. Injecting it is what let a single
	// value redirect every scenario that inherited it.
	for _, key := range []string{"SQLITE_PATH", "SQLITE_DB"} {
		if v, ok := result.env[key]; ok {
			t.Fatalf("%s must not be injected (got %q): a generic database path is "+
				"inherited by every child process and redirects siblings", key, v)
		}
	}

	if err := cleanup(context.Background()); err != nil {
		t.Fatalf("cleanup returned error: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected the isolated storage root to be removed, got err=%v", err)
	}
}
