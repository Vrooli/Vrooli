package phases

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/projectmeta"
	"github.com/vrooli/api-core/storage"
	seedspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds"
	"test-genie/internal/orchestrator/workspace"
)

func TestSeedOwnerPipelineKeepsPrimaryAndRetainsCleanupCounters(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "seed-demo")
	for _, dir := range []string{filepath.Join(root, ".vrooli"), filepath.Join(scenarioDir, "api"), filepath.Join(scenarioDir, "bas", "seeds")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(`{"mode":"development"}`), 0600); err != nil {
		t.Fatal(err)
	}
	projectmeta.SetStartDirForTesting(root)
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package main\nimport _ \"modernc.org/sqlite\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"), []byte("module seedfixture\n\ngo 1.26.0\n\nrequire modernc.org/sqlite v1.50.1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	script := `set -eu
mkdir -p "$VROOLI_STORAGE_ROOT/project" "$TEST_GENIE_SCENARIO_DIR/coverage/runtime"
printf '{"project":"%s/project"}' "$VROOLI_STORAGE_ROOT" > "$TEST_GENIE_SCENARIO_DIR/coverage/runtime/seed-state.json"
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "bas", "seeds", "seed.sh"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(t.Context(), database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.ExecContext(t.Context(), "CREATE TABLE effects(value TEXT)"); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, "CREATE TABLE effects(value TEXT)")
		return err
	})
	paths := storage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")}
	roots := filerouting.New(paths)
	owner := devrouting.NewService(db, roots)
	mux := http.NewServeMux()
	devrouting.RegisterWithFileRootsService(mux, db, roots, owner)
	mux.HandleFunc("/effect", func(w http.ResponseWriter, r *http.Request) {
		if _, err := db.ExecContext(r.Context(), "INSERT INTO effects VALUES ('leased')"); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		dir, err := roots.Pick(r.Context(), storage.ClassState)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "effect.txt"), []byte("leased"), 0600); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		roots.RecordWrite(r.Context())
		w.WriteHeader(204)
	})
	target := httptest.NewServer(owner.PinnedTestModeMiddleware(mux))
	defer target.Close()
	session, err := ApplyPlaybooksSeed(t.Context(), workspace.Environment{ScenarioName: "seed-demo", ScenarioDir: scenarioDir, AppRoot: root, APIURL: target.URL}, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Cleanup(context.Background()) })
	project, ok := session.SeedState["project"].(string)
	if !ok || project == "" {
		t.Fatalf("actual seed values missing: %v", session.SeedState)
	}
	if _, err := os.Stat(project); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, target.URL+"/effect", nil)
	req.Header.Set(apihttp.TestModeHeader, "1")
	response, err := target.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 204 {
		t.Fatalf("leased effect status %d", response.StatusCode)
	}
	if err := session.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	evidence := session.SeedEvidence()
	if evidence.GetState() != seedspb.SeedState_SEED_STATE_CLEARED || evidence.GetStats().GetTestPoolRequests() != 1 || evidence.GetStats().GetTestRootWrites() != 1 {
		t.Fatalf("cleanup lost authoritative counters: %v", evidence)
	}
	if err := session.Cleanup(t.Context()); err != nil {
		t.Fatal(err)
	}
	if session.SeedEvidence().GetStats().GetTestPoolRequests() != 1 {
		t.Fatal("retry replaced original clear counters")
	}
	if _, err := os.Stat(project); !os.IsNotExist(err) {
		t.Fatalf("seed workspace retained after owner cleanup: %v", err)
	}
	var primary int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM effects").Scan(&primary); err != nil || primary != 0 {
		t.Fatalf("primary SQL changed: rows=%d err=%v", primary, err)
	}
	if _, err := os.Stat(filepath.Join(paths.StateDir, "effect.txt")); !os.IsNotExist(err) {
		t.Fatalf("primary file changed: %v", err)
	}
}
