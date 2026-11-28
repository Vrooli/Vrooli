package phases

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/projectmeta"
	"github.com/vrooli/api-core/storage"
	seedspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds"
	"test-genie/internal/orchestrator/phases/isolation"
	"test-genie/internal/orchestrator/workspace"
)

type partialSeedIsolation struct {
	result *isolation.Result
	err    error
}

func TestCheckpointWriteCapPreservesLoadableCleanupAuthority(t *testing.T) {
	dir := t.TempDir()
	runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
	checkpoint := &seedCheckpoint{Version: 1, RunID: runID, Cleared: true, path: filepath.Join(dir, runID+".json")}
	if err := checkpoint.save(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(checkpoint.path)
	checkpoint.SeedState = map[string]any{"oversized": strings.Repeat("x", seedCheckpointMaxBytes)}
	if err := checkpoint.save(); !errors.Is(err, errSeedCheckpointCapacity) {
		t.Fatal("unloadable checkpoint was not refused", err)
	}
	after, _ := os.ReadFile(checkpoint.path)
	if string(before) != string(after) {
		t.Fatal("oversized checkpoint replaced prior cleanup authority")
	}
	if _, err := RecoverPlaybooksSeed(checkpoint.path); err != nil {
		t.Fatal("preserved authority is not loadable", err)
	}
}

func TestEncodedSeedStateExpansionFailsBeforePublicationAndCleansExactBacking(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("actual native seed owner is qualified on Linux")
	}
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "encoded-state")
	for _, path := range []string{filepath.Join(root, ".vrooli"), filepath.Join(source, "api"), filepath.Join(source, "bas", "seeds")} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		filepath.Join(root, ".vrooli", "service.json"): `{"mode":"development"}`,
		filepath.Join(source, "api", "go.mod"):         "module encodedfixture\n\ngo 1.26.0\nrequire modernc.org/sqlite v1.50.1\n",
		filepath.Join(source, "bas", "seeds", "seed.sh"): `set -eu
mkdir -p "$TEST_GENIE_SCENARIO_DIR/coverage/runtime"
{ printf '{"payload":"'; head -c 700000 /dev/zero | tr '\0' '<'; printf '"}'; } > "$TEST_GENIE_SCENARIO_DIR/coverage/runtime/seed-state.json"
`,
	}
	for path, value := range files {
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	projectmeta.SetStartDirForTesting(root)
	db, err := database.Open(t.Context(), database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(root, "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, "CREATE TABLE effects(value TEXT)")
		return err
	})
	roots := filerouting.New(storage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")})
	owner := devrouting.NewService(db, roots)
	mux := http.NewServeMux()
	devrouting.RegisterWithFileRootsService(mux, db, roots, owner)
	target := httptest.NewServer(owner.PinnedTestModeMiddleware(mux))
	defer target.Close()
	journal := filepath.Join(root, "owner")
	if err := os.Mkdir(journal, 0700); err != nil {
		t.Fatal(err)
	}
	runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
	path := filepath.Join(journal, runID+".json")
	if err := InitializeSeedCheckpoint(path, runID); err != nil {
		t.Fatal(err)
	}
	session, err := ApplyPlaybooksSeed(WithSeedPersistence(t.Context(), path, runID), workspace.Environment{ScenarioName: "encoded-state", ScenarioDir: source, AppRoot: root, APIURL: target.URL}, nil, false)
	if session == nil || err == nil || !strings.Contains(err.Error(), "encoded seed state exceeds") {
		t.Fatal("encoded expansion did not fail setup before publishing seed values", err)
	}
	if len(session.SeedState) != 0 || !session.CleanupComplete() {
		t.Fatal("invalid producer state poisoned session cleanup or escaped as seed values")
	}
	if _, err := os.Stat(session.Env["VROOLI_STORAGE_ROOT"]); !os.IsNotExist(err) {
		t.Fatal("invalid producer state retained exact private backing", err)
	}
	if evidence := session.SeedEvidence(); evidence.GetStats() == nil || evidence.GetState() != seedspb.SeedState_SEED_STATE_FAILED || db.HasTestPool() {
		t.Fatal("failed producer did not retain exact acknowledged target clear")
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > seedCheckpointMaxBytes {
		t.Fatal("failed producer left unloadable cleanup authority", err)
	}
	recovered, err := RecoverPlaybooksSeed(path)
	if err != nil || !recovered.CleanupComplete() {
		t.Fatal("failed producer cleanup cannot be recovered", err)
	}
}

func TestReplacementReconstructsInterruptedSQLitePreparationWithoutTargetInstall(t *testing.T) {
	for _, boundary := range []string{"unmarked staging", "partial marker", "published", "foreign final"} {
		t.Run(boundary, func(t *testing.T) {
			journal := filepath.Join(t.TempDir(), "owner")
			runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
			backing := filepath.Join(journal, "backing")
			stage := filepath.Join(backing, ".staging", runID)
			final := filepath.Join(backing, "test-genie-playbooks-"+runID)
			if err := os.MkdirAll(filepath.Dir(stage), 0700); err != nil {
				t.Fatal(err)
			}
			path := stage
			marker := ""
			switch boundary {
			case "partial marker":
				marker = runID[:8]
			case "published":
				path, marker = final, runID
			case "foreign final":
				path = final
			}
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
			if marker != "" {
				if err := os.WriteFile(filepath.Join(path, ".seed-owner"), []byte(marker), 0600); err != nil {
					t.Fatal(err)
				}
			}
			checkpoint := &seedCheckpoint{Version: 1, RunID: runID, Cleared: true, path: filepath.Join(journal, runID+".json"), Allocations: []isolation.Allocation{{Kind: "sqlite", RunID: runID, Root: final, StagingRoot: stage}}}
			if err := checkpoint.save(); err != nil {
				t.Fatal(err)
			}
			session, err := RecoverPlaybooksSeed(checkpoint.path)
			if err != nil {
				t.Fatal("replacement failed to reconstruct the exact preparation plan", err)
			}
			err = session.Cleanup(t.Context())
			if boundary == "foreign final" {
				if err == nil || session.CleanupComplete() {
					t.Fatal("replacement guessed ownership of an unmarked final root")
				}
				if _, err := os.Stat(final); err != nil {
					t.Fatal("replacement changed the foreign final root", err)
				}
				return
			}
			if err != nil || !session.CleanupComplete() {
				t.Fatal("replacement did not retire known private preparation", err)
			}
			for _, path := range []string{stage, final} {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("interrupted preparation survived replacement", err)
				}
			}
		})
	}
}

func (p partialSeedIsolation) Prepare(context.Context) (*isolation.Result, error) {
	return p.result, p.err
}

func TestEarlySeedFailureRetainsExactBackingAndStickyCleanupFailure(t *testing.T) {
	for _, boundary := range []string{"partial allocation", "migration", "clientless install"} {
		t.Run(boundary, func(t *testing.T) {
			root := t.TempDir()
			apiDir := filepath.Join(root, "api")
			if err := os.MkdirAll(apiDir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module fixture\n\ngo 1.26.0\nrequire modernc.org/sqlite v1.50.1\n"), 0600); err != nil {
				t.Fatal(err)
			}
			backing := filepath.Join(t.TempDir(), "private.db")
			if err := os.WriteFile(backing, nil, 0600); err != nil {
				t.Fatal(err)
			}
			attempts := 0
			result := &isolation.Result{RunID: "partial-owned-lease", Env: map[string]string{"PLAYBOOKS_SQLITE_DSN": backing}, Cleanup: func(context.Context) error {
				attempts++
				if attempts == 1 {
					return errors.New("injected backing deletion failure")
				}
				return os.Remove(backing)
			}}
			var prepareErr error
			if boundary == "partial allocation" {
				prepareErr = errors.New("redis allocation failed after sqlite")
			}
			if boundary == "migration" {
				migrations := filepath.Join(root, "bas", "seeds", "migrations")
				if err := os.MkdirAll(migrations, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(migrations, "001.sql"), []byte("THIS IS NOT SQL"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			previous := isolationManagerFactory
			isolationManagerFactory = func(isolation.Config) isolationProvider { return partialSeedIsolation{result, prepareErr} }
			defer func() { isolationManagerFactory = previous }()
			session, err := ApplyPlaybooksSeed(t.Context(), workspace.Environment{ScenarioName: "fixture", ScenarioDir: root, AppRoot: root}, nil, false)
			if session == nil || err == nil || !strings.Contains(err.Error(), "injected backing deletion failure") {
				t.Fatalf("early failure lost cleanup owner/error: %v %v", session, err)
			}
			if session.CleanupComplete() {
				t.Fatal("failed cleanup advertised retirement")
			}
			if _, err := os.Stat(backing); err != nil {
				t.Fatal("failed cleanup lost backing", err)
			}
			if session.SeedEvidence().GetState() != seedspb.SeedState_SEED_STATE_FAILED {
				t.Fatal("failed setup became healthy")
			}
			if err := session.Cleanup(t.Context()); err == nil {
				t.Fatal("retry erased prior cleanup failure")
			}
			if !session.CleanupComplete() || attempts != 2 {
				t.Fatalf("cleanup retry failed to retire exact allocation: attempts=%d", attempts)
			}
			if _, err := os.Stat(backing); !os.IsNotExist(err) {
				t.Fatalf("backing remains after successful retry: %v", err)
			}
			_ = session.Cleanup(t.Context())
			if attempts != 2 {
				t.Fatal("retired backing deletion replayed")
			}
		})
	}
}

func TestSQLiteSeedMigrationHonorsSetupDeadline(t *testing.T) {
	dir := t.TempDir()
	migration := filepath.Join(dir, "bounded.sql")
	// Deliberately finite: prove cancellation loss without a runaway query.
	query := "WITH RECURSIVE r(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM r WHERE x<5000000) SELECT sum(x) FROM r;"
	if err := os.WriteFile(migration, []byte(query), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := applySQLiteMigrations(ctx, workspace.Environment{ScenarioDir: dir}, map[string]string{"PLAYBOOKS_SQLITE_DSN": filepath.Join(dir, "private.db")}, []string{migration}, nil)
	elapsed := time.Since(start)
	t.Logf("migration elapsed=%s setupContext=%v migrationError=%v", elapsed, ctx.Err(), err)
	if ctx.Err() != nil && err == nil {
		t.Fatal("SQLite migration reported success after losing setup authority; execution did not receive the setup deadline")
	}
	if elapsed > time.Second {
		t.Fatal("SQLite migration exceeded cancellation budget")
	}
}

func TestCheckpointRejectsTrailingCorruption(t *testing.T) {
	for _, suffix := range []string{" trailing-corruption", " {}", " null", " \n\t"} {
		t.Run(suffix, func(t *testing.T) {
			dir := t.TempDir()
			runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
			path := filepath.Join(dir, runID+".json")
			if err := InitializeSeedCheckpoint(path, runID); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			raw := append(before, []byte(suffix)...)
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			session, err := RecoverPlaybooksSeed(path)
			if strings.TrimSpace(suffix) == "" {
				if err != nil || session == nil {
					t.Fatal("valid trailing whitespace refused", err)
				}
			} else if err == nil || session != nil {
				t.Fatal("trailing malformed/second document admitted as cleanup authority", err)
			}
			after, err := os.ReadFile(path)
			if err != nil || string(raw) != string(after) {
				t.Fatal("reader rewrote rejected authority", err)
			}
		})
	}
}

// setupBudgetIsolation observes the owner budget without native dispatch.
type setupBudgetIsolation func(context.Context) (*isolation.Result, error)

func (f setupBudgetIsolation) Prepare(ctx context.Context) (*isolation.Result, error) { return f(ctx) }

func TestSeedPreparationBudgetPreservesConfiguredScriptTimeout(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".vrooli"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "testing.json"), []byte(`{"playbooks":{"seeds":{"timeout_ms":75}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	previous := isolationManagerFactory
	defer func() { isolationManagerFactory = previous }()
	isolationManagerFactory = func(cfg isolation.Config) isolationProvider {
		return setupBudgetIsolation(func(ctx context.Context) (*isolation.Result, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal("preparation has no owner deadline")
			}
			remaining := time.Until(deadline)
			if remaining < seedAllocationAllowance || remaining > seedAllocationAllowance+75*time.Millisecond {
				t.Fatalf("script timeout was redefined instead of adding bounded preparation: %v", remaining)
			}
			if cfg.Timeout != seedAllocationAllowance {
				t.Fatal("allocation cap changed")
			}
			return nil, errors.New("effect-free budget observation")
		})
	}
	_, err := ApplyPlaybooksSeed(t.Context(), workspace.Environment{ScenarioName: "budget", ScenarioDir: root, AppRoot: root}, nil, false)
	if err == nil || !strings.Contains(err.Error(), "effect-free budget observation") {
		t.Fatal(err)
	}
}

func TestSeedMigrationCancellationRetiresExactBackingBeforeRoutingOrDispatch(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{filepath.Join(root, "api"), filepath.Join(root, "bas", "seeds", "migrations")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module fixture\n\ngo 1.26.0\nrequire modernc.org/sqlite v1.50.1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bas", "seeds", "migrations", "001.sql"), []byte("WITH RECURSIVE r(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM r WHERE x<5000000) SELECT sum(x) FROM r;"), 0600); err != nil {
		t.Fatal(err)
	}
	backing := filepath.Join(root, "private.db")
	previous := isolationManagerFactory
	defer func() { isolationManagerFactory = previous }()
	cleanups := 0
	isolationManagerFactory = func(cfg isolation.Config) isolationProvider {
		return partialSeedIsolation{result: &isolation.Result{RunID: cfg.RunID, Env: map[string]string{"PLAYBOOKS_SQLITE_DSN": backing}, Cleanup: func(ctx context.Context) error {
			if ctx.Err() != nil {
				t.Fatal("canceled setup context prevented exact cleanup")
			}
			cleanups++
			return os.Remove(backing)
		}}}
	}
	var calls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "unexpected routing installation", 503)
	}))
	defer target.Close()
	journal := filepath.Join(root, "owner")
	if err := os.Mkdir(journal, 0700); err != nil {
		t.Fatal(err)
	}
	runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
	path := filepath.Join(journal, runID+".json")
	if err := InitializeSeedCheckpoint(path, runID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	session, err := ApplyPlaybooksSeed(WithSeedPersistence(ctx, path, runID), workspace.Environment{ScenarioName: "fixture", ScenarioDir: root, AppRoot: root, APIURL: target.URL}, nil, false)
	if session == nil || err == nil || !strings.Contains(err.Error(), "migrations") || ctx.Err() == nil {
		t.Fatal("setup did not fail at the canceled SQLite migration", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("setup ignored caller cancellation")
	}
	if calls.Load() != 0 || cleanups != 1 || !session.CleanupComplete() {
		t.Fatal("failed migration dispatched or lost exact cleanup", calls.Load(), cleanups)
	}
	if _, err := os.Stat(backing); !os.IsNotExist(err) {
		t.Fatal("private backing retained", err)
	}
	recovered, err := RecoverPlaybooksSeed(path)
	if err != nil || !recovered.CleanupComplete() || recovered.recovery.Operation != nil || recovered.recovery.RoutingAttempted {
		t.Fatal("canceled setup published routing/dispatch or unloadable cleanup", err)
	}
}

func TestCheckpointOnlyRetirementRefusesForeignAndAliasedAuthority(t *testing.T) {
	for _, kind := range []string{"absent", "canonical", "nonempty", "malformed", "public", "symlink", "hardlink", "directory", "public parent", "symlink parent"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Chmod(dir, 0700); err != nil {
				t.Fatal(err)
			}
			runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
			path := filepath.Join(dir, runID+".json")
			if kind != "absent" {
				if err := InitializeSeedCheckpoint(path, runID); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "public parent":
				if err := os.Chmod(dir, 0755); err != nil {
					t.Fatal(err)
				}
			case "symlink parent":
				alias := dir + "-alias"
				if err := os.Symlink(dir, alias); err != nil {
					t.Fatal(err)
				}
				defer os.Remove(alias)
				path = filepath.Join(alias, runID+".json")

			case "nonempty":
				c := &seedCheckpoint{Version: 1, RunID: runID, Cleared: true, SeedState: map[string]any{"foreign": "state"}, path: path}
				if err := c.save(); err != nil {
					t.Fatal(err)
				}
			case "malformed":
				if err := os.WriteFile(path, []byte("foreign authority"), 0600); err != nil {
					t.Fatal(err)
				}
			case "public":
				if err := os.Chmod(path, 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(path, path+".foreign"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".foreign", path); err != nil {
					t.Fatal(err)
				}
			case "hardlink":
				if err := os.Link(path, path+".foreign"); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadFile(path)
			info, _ := os.Lstat(path)
			err := RetireSeedCheckpointReservation(path, runID)
			if kind == "absent" || kind == "canonical" {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					t.Fatal("exact empty reservation not retired", err)
				}
				return
			}
			after, afterErr := os.Lstat(path)
			if err == nil || afterErr != nil || !os.SameFile(info, after) {
				t.Fatal("foreign/aliased checkpoint was changed", err, afterErr)
			}
			raw, _ := os.ReadFile(path)
			if string(raw) != string(before) {
				t.Fatal("refused authority bytes changed")
			}
		})
	}
}
