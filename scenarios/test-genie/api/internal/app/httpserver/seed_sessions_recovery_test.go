package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"test-genie/agentmanager"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/projectmeta"
	"github.com/vrooli/api-core/storage"
	seedspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds"
	seedsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds/seeds_v1connect"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/workspace"
)

type seedRecoveryFixture struct {
	root, source, journal, targetURL string
	db                               *database.RoutedDB
	refuseClear                      *atomic.Bool
}

func TestMissingSeedRegistryRefusesRetainedAuthorityAndInitializesOnlyFreshOwner(t *testing.T) {
	for _, retained := range []string{"", "fc3c4af6-ef0c-44db-8bc9-1ebd744b8ff2.json", "backing", "gates"} {
		t.Run("retained="+retained, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "owner")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if retained != "" {
				if err := os.WriteFile(filepath.Join(dir, retained), []byte("retained authority"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			server := &Server{}
			err := server.initializeSeedOwner(dir)
			if server.seedStore != nil {
				defer server.seedStore.release()
			}
			if retained != "" {
				if err == nil {
					t.Fatal("missing registry admitted fresh ownership despite retained state")
				}
				if _, err := os.Stat(filepath.Join(dir, "registry.json")); !os.IsNotExist(err) {
					t.Fatal("refusal overwrote missing registry with guessed empty ownership")
				}
				return
			}
			if err != nil {
				t.Fatal("fresh private owner refused", err)
			}
			info, err := os.Stat(filepath.Join(dir, "registry.json"))
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatal("fresh owner did not durably initialize a private registry", err)
			}
		})
	}
}

func TestSharedSeedAdmissionRejectsOversizedAndMalformedInputsBeforeReservation(t *testing.T) {
	f := newSeedRecoveryFixture(t)
	server := newRecoverySeedServer(t, f)
	before, err := os.ReadFile(filepath.Join(f.journal, "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(f.journal)
	for _, test := range []struct{ name, scenario, path, request string }{
		{"oversized path", "demo", "/" + strings.Repeat("x", 4<<20), "bounded-request"},
		{"oversized request", "demo", f.source, strings.Repeat("x", 201)},
		{"oversized scenario", strings.Repeat("x", 129), f.source, "bounded-request"},
		{"canonical expansion", "demo", strings.Repeat("x", maxSeedSourcePathBytes), "bounded-request"},
		{"NUL path", "demo", f.source + "\x00", "bounded-request"},
		{"multiline request", "demo", f.source, "request\r\nother"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, record, err := server.applySeedSession(t.Context(), test.scenario, test.path, test.request, false)
			if connect.CodeOf(err) != connect.CodeInvalidArgument || record != nil || server.seedOwnerErr != nil {
				t.Fatal("invalid identity acquired or poisoned seed ownership")
			}
		})
	}
	after, err := os.ReadFile(filepath.Join(f.journal, "registry.json"))
	afterEntries, _ := os.ReadDir(f.journal)
	if err != nil || string(after) != string(before) || len(afterEntries) != len(entries) || len(server.seedSessions) != 0 || len(server.seedSessionsByScenario) != 0 {
		t.Fatal("invalid inputs left a reservation, checkpoint or journal growth", err)
	}
}

func TestSeedRegistryByteCapacityPreservesAuthorityAndExistingOwnerCleanup(t *testing.T) {
	f := newSeedRecoveryFixture(t)
	server := newRecoverySeedServer(t, f)
	token, active, err := server.applySeedSession(t.Context(), "demo", f.source, "existing-owner", false)
	if err != nil {
		t.Fatal(err)
	}
	// A legacy retained outcome may use more bytes than today's input bound.
	// Preserve its loadable authority while refusing new admissions at7MiB.
	historyID := "cff8b88a-c381-45df-90d1-cf793306d09a"
	if err := phases.InitializeSeedCheckpoint(server.seedCheckpointPath(historyID), historyID); err != nil {
		t.Fatal(err)
	}
	history := &seedSession{Scenario: "retired", scenarioPath: "/" + strings.Repeat("<", seedRegistryAdmissionBytes/6+100), requestID: "legacy", runID: historyID, Created: time.Now(), retiredAt: time.Now(), ready: make(chan struct{}), prepareErr: errors.New(strings.Repeat("e", seedPrivateErrorMaxBytes*2))}
	close(history.ready)
	server.seedSessionsMu.Lock()
	server.seedSessions["bf8cc4ed-d5d4-4e51-8696-8e1f21dd29b1"] = history
	err = server.persistSeedSessionsLocked()
	server.seedSessionsMu.Unlock()
	if err != nil {
		t.Fatal("loadable legacy state did not fit hard cleanup budget", err)
	}
	registry := filepath.Join(f.journal, "registry.json")
	before, _ := os.ReadFile(registry)
	entries, _ := os.ReadDir(f.journal)
	_, record, err := server.applySeedSession(t.Context(), "other", f.source, "new-byte-capacity", false)
	if connect.CodeOf(err) != connect.CodeResourceExhausted || record != nil || server.seedOwnerErr != nil {
		t.Fatal("byte capacity rejection poisoned valid owner or admitted effects", err)
	}
	after, readErr := os.ReadFile(registry)
	afterEntries, _ := os.ReadDir(f.journal)
	if readErr != nil || string(before) != string(after) || len(entries) != len(afterEntries) || len(server.seedSessions) != 2 || len(server.seedSessionsByScenario) != 1 {
		t.Fatal("effect-free capacity rollback changed retained authority", readErr)
	}
	payload, _ := json.Marshal(map[string]string{"scenarioPath": f.source})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/other/playbooks/seed/apply", strings.NewReader(string(payload))))
	if response.Code != http.StatusTooManyRequests || server.seedOwnerErr != nil {
		t.Fatal("legacy seed surface misclassified capacity as owner corruption", response.Code)
	}
	var disk seedRegistryDisk
	if err := json.Unmarshal(after, &disk); err != nil {
		t.Fatal(err)
	}
	for _, row := range disk.Sessions {
		if row.RunID == historyID && (len(row.PrepareError) != seedPrivateErrorMaxBytes || !strings.HasSuffix(row.PrepareError, " [truncated]")) {
			t.Fatal("retained private error excerpt exceeded its bound")
		}
	}
	priorPath := history.scenarioPath
	history.scenarioPath = "/" + strings.Repeat("<", seedRegistryMaxBytes/6+100)
	server.seedSessionsMu.Lock()
	err = server.persistSeedSessionsLocked()
	server.seedSessionsMu.Unlock()
	history.scenarioPath = priorPath
	after, _ = os.ReadFile(registry)
	if connect.CodeOf(err) != connect.CodeResourceExhausted || string(before) != string(after) {
		t.Fatal("hard write cap replaced loadable authority", err)
	}
	if _, err := server.clearSeedSession(t.Context(), "demo", token); err != nil || !active.Session.CleanupComplete() || server.seedOwnerErr != nil {
		t.Fatal("byte pressure disabled exact cleanup of existing owner", err)
	}
	if _, err := os.Stat(active.Session.Env["VROOLI_STORAGE_ROOT"]); !os.IsNotExist(err) {
		t.Fatal("byte pressure retained backing after exact clear", err)
	}
}

func TestFailedSeedOwnerInitializationNeverRewritesRetainedJournalDuringMaintenance(t *testing.T) {
	for _, kind := range []string{"missing", "corrupt"} {
		t.Run(kind, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "owner")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "retained-checkpoint.json"), []byte("private authority"), 0600); err != nil {
				t.Fatal(err)
			}
			registry := filepath.Join(dir, "registry.json")
			original := []byte("{incomplete private registry")
			if kind == "corrupt" {
				if err := os.WriteFile(registry, original, 0600); err != nil {
					t.Fatal(err)
				}
			}
			server := &Server{}
			server.seedOwnerErr = server.initializeSeedOwner(dir)
			if server.seedStore != nil {
				defer server.seedStore.release()
			}
			if server.seedOwnerErr == nil {
				t.Fatal("fixture did not refuse failed journal initialization")
			}
			server.sweepSeedSessions(t.Context(), time.Now())
			raw, err := os.ReadFile(registry)
			if kind == "missing" {
				if !os.IsNotExist(err) {
					t.Fatal("maintenance replaced missing authority with guessed empty registry")
				}
			} else if err != nil || string(raw) != string(original) {
				t.Fatal("maintenance overwrote failed retained registry", err)
			}
			server.seedSessionsMu.Lock()
			err = server.persistSeedSessionsLocked()
			server.seedSessionsMu.Unlock()
			if err == nil {
				t.Fatal("uninitialized owner was permitted to publish a registry")
			}
		})
	}
}

func TestServerConstructorMaintenanceAndRepeatedReplacementPreserveFailedSeedAuthority(t *testing.T) {
	for _, kind := range []string{"missing", "corrupt", "partially valid", "trailing malformed", "second document", "trailing null"} {
		t.Run(kind, func(t *testing.T) {
			f := newSeedRecoveryFixture(t)
			t.Setenv("VROOLI_STORAGE_ROOT", f.root)
			dir, err := defaultSeedOwnerDir()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
			token := "329417d0-1984-4b6c-b05c-27ecc597d126"
			if err := phases.InitializeSeedCheckpoint(filepath.Join(dir, runID+".json"), runID); err != nil {
				t.Fatal(err)
			}
			registry := filepath.Join(dir, "registry.json")
			original := []byte("{incomplete private registry")
			if kind == "partially valid" {
				original, err = json.Marshal(seedRegistryDisk{Version: 1, Sessions: []seedRecordDisk{{Token: token, RunID: runID, Scenario: "demo", ScenarioPath: f.source, Created: time.Now()}, {Token: "invalid"}}})
				if err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "trailing malformed":
				original = []byte(`{"version":1,"sessions":[]} trailing-corruption`)
			case "second document":
				original = []byte(`{"version":1,"sessions":[]} {}`)
			case "trailing null":
				original = []byte(`{"version":1,"sessions":[]} null`)
			}
			if kind != "missing" {
				if err := os.WriteFile(registry, original, 0600); err != nil {
					t.Fatal(err)
				}
			}
			manager := runmanager.New(&stubSuiteExecutor{}, f.root)
			defer manager.Shutdown()
			for replacement := 0; replacement < 2; replacement++ {
				server, err := New(Config{Port: "0"}, Dependencies{DB: f.db, Executions: &fakeExecutionHistory{}, ExecutionPlanner: &fakeExecutionPlanner{}, RunManager: manager, Scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Join(f.root, "scenarios")}, PhaseCatalog: &orchestrator.SuiteOrchestrator{}, AgentService: agentmanager.NewAgentService(agentmanager.Config{}), Logger: log.New(io.Discard, "", 0)})
				if err != nil || server.seedOwnerErr == nil || server.seedStore == nil || server.seedStore.validated {
					t.Fatal("actual constructor admitted incomplete seed authority", err)
				}
				// Start uses this exact background loop. A canceled fixture context
				// ends it immediately after its unconditional initial sweep.
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				server.startSeedMaintenance(ctx)
				server.seedMaintenance.Wait()
				raw, readErr := os.ReadFile(registry)
				if (kind == "missing" && !os.IsNotExist(readErr)) || (kind != "missing" && (readErr != nil || string(raw) != string(original))) {
					t.Fatal("constructor/maintenance changed failed authority before replacement", readErr)
				}
				if _, err := server.observeSeedSession("demo", token); connect.CodeOf(err) != connect.CodeUnavailable {
					t.Fatal("incomplete owner observed or renewed a partially loaded session", err)
				}
				if _, err := server.clearSeedSession(t.Context(), "demo", token); connect.CodeOf(err) != connect.CodeUnavailable {
					t.Fatal("incomplete owner cleared a partially loaded session", err)
				}
				server.seedStore.release()
			}
		})
	}
}

func TestValidatedSeedOwnerCleanupConvergesAfterTransientJournalWriteFailure(t *testing.T) {
	f := newSeedRecoveryFixture(t)
	server := newRecoverySeedServer(t, f)
	token, record, err := server.applySeedSession(t.Context(), "demo", f.source, "transient-journal-write", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(f.journal, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(f.journal, 0700)
	if _, err := server.observeSeedSession("demo", token); err == nil || server.seedOwnerErr == nil {
		t.Fatal("fixture did not retain a real registry write failure")
	}
	if !server.seedStore.validated {
		t.Fatal("transient publication failure invalidated known cleanup authority")
	}
	if err := os.Chmod(f.journal, 0700); err != nil {
		t.Fatal(err)
	}
	server.sweepSeedSessions(t.Context(), time.Now().Add(seedConsumerTTL+time.Second))
	awaitSeedRetirement(t, server, token, record.Session.SeedState["project"].(string))
}

func TestPreparingSeedPublishesCompleteSessionUnderSnapshotLock(t *testing.T) {
	f := newSeedRecoveryFixture(t)
	server := newRecoverySeedServer(t, f)
	previous := applyPlaybooksSeed
	entered, finish := make(chan struct{}), make(chan struct{})
	applyPlaybooksSeed = func(context.Context, workspace.Environment, io.Writer, bool) (*phases.PlaybooksSeedSession, error) {
		close(entered)
		<-finish
		return &phases.PlaybooksSeedSession{}, nil
	}
	t.Cleanup(func() { applyPlaybooksSeed = previous })
	done := make(chan error, 1)
	go func() {
		_, _, err := server.applySeedSession(t.Context(), "demo", f.source, "atomic-session-publication", false)
		done <- err
	}()
	<-entered
	server.seedSessionsMu.Lock()
	record := server.seedSessions[server.seedSessionsByScenario["demo"]]
	server.seedSessionsMu.Unlock()
	// Hold the same snapshot lock used by journal publication while the external
	// apply finishes. Neither Session nor its cleanup capability may appear
	// before the complete preparation result can be published atomically.
	record.mu.Lock()
	close(finish)
	persisted := make(chan error, 1)
	go func() {
		server.seedSessionsMu.Lock()
		defer server.seedSessionsMu.Unlock()
		persisted <- server.persistSeedSessionsLocked()
	}()
	time.Sleep(20 * time.Millisecond)
	publishedEarly := record.Session != nil
	record.mu.Unlock()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-persisted; err != nil {
		t.Fatal(err)
	}
	if publishedEarly {
		t.Fatal("preparing seed published its session outside the owner snapshot lock")
	}
	if record.Session == nil || record.Session.CleanupRef == "" || record.consumerDeadline.IsZero() {
		t.Fatal("prepared seed did not publish its complete consumer and cleanup identity")
	}
}

func newSeedRecoveryFixture(t *testing.T) seedRecoveryFixture {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "demo")
	for _, dir := range []string{filepath.Join(root, ".vrooli"), filepath.Join(source, "api"), filepath.Join(source, "bas", "seeds")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{filepath.Join(root, ".vrooli", "service.json"): `{"mode":"development"}`, filepath.Join(source, "api", "go.mod"): "module seedfixture\n\ngo 1.26.0\nrequire modernc.org/sqlite v1.50.1\n", filepath.Join(source, "bas", "seeds", "seed.sh"): `set -eu
mkdir -p "$VROOLI_STORAGE_ROOT/project" "$TEST_GENIE_SCENARIO_DIR/coverage/runtime"
printf '{"project":"%s/project"}' "$VROOLI_STORAGE_ROOT" > "$TEST_GENIE_SCENARIO_DIR/coverage/runtime/seed-state.json"
`}
	for p, body := range files {
		if err := os.WriteFile(p, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	projectmeta.SetStartDirForTesting(root)
	db, err := database.Open(t.Context(), database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(root, "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, "CREATE TABLE effects(value TEXT)")
		return err
	})
	roots := filerouting.New(storage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")})
	owner := devrouting.NewService(db, roots)
	mux := http.NewServeMux()
	devrouting.RegisterWithFileRootsService(mux, db, roots, owner)
	refuseClear := &atomic.Bool{}
	pinned := owner.PinnedTestModeMiddleware(mux)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if refuseClear.Load() && strings.HasSuffix(r.URL.Path, "/ClearTestPool") {
			http.Error(w, "fixture clear unavailable", http.StatusServiceUnavailable)
			return
		}
		pinned.ServeHTTP(w, r)
	}))
	t.Cleanup(target.Close)
	return seedRecoveryFixture{root: root, source: source, journal: filepath.Join(root, "seed-owner"), targetURL: target.URL, db: db, refuseClear: refuseClear}
}
func newRecoverySeedServer(t *testing.T, f seedRecoveryFixture) *Server {
	t.Helper()
	previous := resolveSeedTargetAPI
	resolveSeedTargetAPI = func(context.Context, string, string) (string, error) { return f.targetURL, nil }
	t.Cleanup(func() { resolveSeedTargetAPI = previous })
	server := &Server{router: mux.NewRouter(), scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Join(f.root, "scenarios")}, logger: log.New(io.Discard, "", 0)}
	if err := server.initializeSeedOwner(f.journal); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(server.seedStore.release)
	server.setupRoutes()
	return server
}
func awaitSeedRetirement(t *testing.T, server *Server, token, project string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		server.seedSessionsMu.Lock()
		active := server.seedSessionsByScenario["demo"]
		server.seedSessionsMu.Unlock()
		if active == "" {
			if _, err := os.Stat(project); !os.IsNotExist(err) {
				t.Fatalf("retired owner retained backing: %v", err)
			}
			record, err := server.findSeedSession("demo", token)
			if err != nil {
				t.Fatal(err)
			}
			if !record.Session.CleanupComplete() {
				t.Fatal("slot released before backing retirement")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("seed owner did not retire abandoned exact backing within bound")
}
func TestSeedConsumerExpiryRetiresAliveOwnerAndCannotBeRevived(t *testing.T) {
	f := newSeedRecoveryFixture(t)
	server := newRecoverySeedServer(t, f)
	token, record, err := server.applySeedSession(t.Context(), "demo", f.source, "consumer-loss", false)
	if err != nil {
		t.Fatal(err)
	}
	project := record.Session.SeedState["project"].(string)
	// Advance only the maintenance observation; native effects and owner remain
	// real. Natural 60-second expiration is separately operated against the API.
	server.sweepSeedSessions(t.Context(), time.Now().Add(seedConsumerTTL+time.Second))
	awaitSeedRetirement(t, server, token, project)
	observed, err := server.observeSeedSession("demo", token)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Session.SeedEvidence().GetState() != seedspb.SeedState_SEED_STATE_FAILED {
		t.Fatal("late consumer revived an expired lease")
	}
	_, next, err := server.applySeedSession(t.Context(), "demo", f.source, "independent-later-validation", false)
	if err != nil {
		t.Fatal("retired capacity did not admit independent validation", err)
	}
	if next.Session.RunID == record.Session.RunID {
		t.Fatal("independent validation reused retired backing")
	}
	_, _ = server.clearSeedSession(t.Context(), "demo", next.Session.CleanupRef)
}

type crashSeedResult struct{ Token, RunID, Project string }

func TestSeedOwnerCrashHelper(t *testing.T) {
	if os.Getenv("TG_SEED_CRASH_HELPER") != "1" {
		return
	}
	root := os.Getenv("TG_SEED_ROOT")
	f := seedRecoveryFixture{root: root, source: filepath.Join(root, "scenarios", "demo"), journal: os.Getenv("TG_SEED_JOURNAL"), targetURL: os.Getenv("TG_SEED_TARGET")}
	projectmeta.SetStartDirForTesting(root)
	server := newRecoverySeedServer(t, f)
	api := httptest.NewServer(server.Handler())
	client := seedsconnect.NewSeedServiceClient(api.Client(), api.URL)
	response, err := client.ApplySeed(t.Context(), connect.NewRequest(&seedspb.ApplySeedRequest{Scenario: "demo", ScenarioPath: f.source, RequestId: "crash-owner"}))
	if err != nil {
		t.Fatal(err)
	}
	result := crashSeedResult{response.Msg.GetSessionToken(), response.Msg.GetEvidence().GetLeaseId(), response.Msg.GetInitialParams().AsMap()["project"].(string)}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("TG_SEED_RESULT"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	select {}
}
func TestSeedOwnerReplacementRecoversActualKilledProcessWithoutReplayingSeed(t *testing.T) {
	f := newSeedRecoveryFixture(t)
	resultPath := filepath.Join(f.root, "result.json")
	child := exec.Command(os.Args[0], "-test.run=^TestSeedOwnerCrashHelper$", "-test.timeout=30s")
	child.Env = append(os.Environ(), "TG_SEED_CRASH_HELPER=1", "TG_SEED_ROOT="+f.root, "TG_SEED_JOURNAL="+f.journal, "TG_SEED_TARGET="+f.targetURL, "TG_SEED_RESULT="+resultPath)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	var result crashSeedResult
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(resultPath)
		if err == nil && json.Unmarshal(raw, &result) == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if result.Token == "" {
		t.Fatal("child seed owner did not acquire actual lease")
	}
	if _, err := f.db.ExecContext(database.WithTestMode(t.Context()), "INSERT INTO effects VALUES ('before-owner-loss')"); err != nil {
		t.Fatal(err)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	replacement := newRecoverySeedServer(t, f)
	record, err := replacement.findSeedSession("demo", result.Token)
	if err != nil {
		t.Fatal(err)
	}
	if record.Session.RunID != result.RunID || record.Session.SeedEvidence().GetState() != seedspb.SeedState_SEED_STATE_FAILED {
		t.Fatal("replacement did not retain exact failed ownership")
	}
	replacement.sweepSeedSessions(t.Context(), time.Now())
	awaitSeedRetirement(t, replacement, result.Token, result.Project)
	evidence := record.Session.SeedEvidence()
	if evidence.GetStats().GetTestPoolRequests() != 1 {
		t.Fatalf("replacement lost actual clear counters: %v", evidence)
	}
	if evidence.GetState() != seedspb.SeedState_SEED_STATE_FAILED {
		t.Fatal("owner replacement fabricated validation success")
	}
	// Same identity reattaches to the failed outcome, never launches another seed.
	token, retry, err := replacement.applySeedSession(t.Context(), "demo", f.source, "crash-owner", false)
	if err == nil || token != result.Token || retry.Session.RunID != result.RunID {
		t.Fatal("replacement replayed abandoned validation")
	}
}

func TestSeedOwnerReplacementStopsRunningSeedBeforeBackingRetirement(t *testing.T) {
	f := newSeedRecoveryFixture(t)
	script := `set -eu
mkdir -p "$VROOLI_STORAGE_ROOT/project" "$TEST_GENIE_SCENARIO_DIR/coverage/runtime"
sleep 120 >/dev/null 2>&1 &
printf '%s' "$!" > "$TEST_GENIE_SCENARIO_DIR/coverage/runtime/child.pid"
wait
`
	if err := os.WriteFile(filepath.Join(f.source, "bas", "seeds", "seed.sh"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	resultPath := filepath.Join(f.root, "never-completed.json")
	child := exec.Command(os.Args[0], "-test.run=^TestSeedOwnerCrashHelper$", "-test.timeout=30s")
	child.Env = append(os.Environ(), "TG_SEED_CRASH_HELPER=1", "TG_SEED_ROOT="+f.root, "TG_SEED_JOURNAL="+f.journal, "TG_SEED_TARGET="+f.targetURL, "TG_SEED_RESULT="+resultPath)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = child.Process.Kill(); _ = child.Wait() }()
	var disk seedRegistryDisk
	var checkpoint struct {
		Env       map[string]string
		Operation json.RawMessage
	}
	pid := ""
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(filepath.Join(f.journal, "registry.json"))
		if err == nil && json.Unmarshal(raw, &disk) == nil && len(disk.Sessions) == 1 {
			phase, err := os.ReadFile(filepath.Join(f.journal, disk.Sessions[0].RunID+".json"))
			if err == nil && json.Unmarshal(phase, &checkpoint) == nil && len(checkpoint.Operation) > 0 {
				raw, err := os.ReadFile(filepath.Join(f.source, "coverage", "runtime", "child.pid"))
				if err == nil {
					pid = string(raw)
					break
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	if pid == "" || len(disk.Sessions) != 1 {
		t.Fatal("running contained seed was not witnessed")
	}
	if _, err := os.Stat("/proc/" + pid); err != nil {
		t.Fatal("seed descendant was not actually running", err)
	}
	f.refuseClear.Store(true)
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	replacement := newRecoverySeedServer(t, f)
	replacement.sweepSeedSessions(t.Context(), time.Now())
	project := filepath.Join(checkpoint.Env["VROOLI_STORAGE_ROOT"], "project")
	var retiredNative struct {
		Operation struct {
			Quiescent  bool   `json:"quiescent"`
			Released   bool   `json:"released"`
			WorkerPath string `json:"worker_path"`
			InputPath  string `json:"input_path"`
			GrantPath  string `json:"grant_path"`
		}
	}

	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(filepath.Join(f.journal, disk.Sessions[0].RunID+".json"))
		if err == nil && json.Unmarshal(raw, &retiredNative) == nil && retiredNative.Operation.Quiescent && retiredNative.Operation.Released {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !retiredNative.Operation.Quiescent || !retiredNative.Operation.Released {
		t.Fatal("replacement did not retire the exact native seed independently of target clear")
	}
	for _, path := range []string{retiredNative.Operation.WorkerPath, retiredNative.Operation.InputPath, retiredNative.Operation.GrantPath} {
		if path == "" {
			t.Fatal("native owner did not retain exact private file identities")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("target clear outage retained worker image or credential input")
		}
	}
	if _, err := os.Stat(project); err != nil {
		t.Fatal("unacknowledged target clear deleted backing", err)
	}
	owned, err := replacement.findSeedSession("demo", disk.Sessions[0].Token)
	if err != nil {
		t.Fatal(err)
	}
	if owned.Session.CleanupComplete() {
		t.Fatal("unacknowledged target clear advertised backing retirement")
	}
	f.refuseClear.Store(false)
	for time.Now().Before(deadline) {
		owned.mu.Lock()
		cleaning := owned.cleaning
		owned.mu.Unlock()
		if !cleaning {
			break
		}
		time.Sleep(time.Millisecond)
	}
	replacement.sweepSeedSessions(t.Context(), time.Now())
	awaitSeedRetirement(t, replacement, disk.Sessions[0].Token, project)
	if raw, err := os.ReadFile("/proc/" + pid + "/stat"); err == nil {
		fields := strings.Fields(string(raw))
		if len(fields) < 3 || (fields[2] != "Z" && fields[2] != "X") {
			t.Fatal("seed descendant survived owner replacement backing retirement")
		}
	}
	if _, err := os.Stat(resultPath); !os.IsNotExist(err) {
		t.Fatal("interrupted seed request fabricated completion")
	}
}

func TestRegistryRejectsTrailingCorruptionWithoutRewrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "owner")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"version":1,"sessions":[]} trailing-corruption`)
	path := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	server := &Server{}
	err := server.initializeSeedOwner(dir)
	if server.seedStore != nil {
		defer server.seedStore.release()
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	t.Logf("initializeError=%v validated=%v bytesPreserved=%v beforeBytes=%d afterBytes=%d", err, server.seedStore != nil && server.seedStore.validated, string(raw) == string(after), len(raw), len(after))
	if err == nil || string(raw) != string(after) {
		t.Fatal("malformed retained authority was admitted and rewritten instead of preserved")
	}
}

func TestRetirementPublicationFailureConvergesCapacity(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "owner")
	server := &Server{}
	if err := server.initializeSeedOwner(dir); err != nil {
		t.Fatal(err)
	}
	defer server.seedStore.release()
	runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
	token := "a9a89022-d958-4b1a-99f5-38e445378908"
	source := filepath.Join(dir, "scenarios", "demo")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	path := server.seedCheckpointPath(runID)
	if err := phases.InitializeSeedCheckpoint(path, runID); err != nil {
		t.Fatal(err)
	}
	session, err := phases.RecoverPlaybooksSeed(path)
	if err != nil {
		t.Fatal(err)
	}
	record := &seedSession{Scenario: "demo", scenarioPath: source, requestID: "retirement-publication-failure", runID: runID, Created: time.Now(), consumerDeadline: time.Now().Add(-time.Second), consumerLost: true, prepareErr: errors.New("original failed validation"), Session: session, ready: make(chan struct{})}
	close(record.ready)
	server.seedSessions[token] = record
	server.seedSessionsByScenario["demo"] = token
	if err := server.persistSeedSessionsLocked(); err != nil {
		t.Fatal(err)
	}
	registry := filepath.Join(dir, "registry.json")
	saved := filepath.Join(dir, "registry.saved")
	if err := os.Rename(registry, saved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(registry, 0700); err != nil {
		t.Fatal(err)
	}
	_, err = server.clearSeedSession(t.Context(), "demo", token)
	if err == nil || !record.Session.CleanupComplete() || record.retiredAt.IsZero() {
		t.Fatal("fixture did not hit failed registry publication after exact cleanup", err)
	}
	// Retention must preserve the still-reserved cleanup handle even while
	// publication remains unavailable beyond the ordinary retention interval.
	server.sweepSeedSessions(t.Context(), time.Now().Add(seedSessionRetention+time.Minute))
	if len(server.seedSessions) != 1 || len(server.seedSessionsByScenario) != 1 {
		t.Fatal("pending durable retirement lost its cleanup handle or reservation")
	}
	if err := os.Remove(registry); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(saved, registry); err != nil {
		t.Fatal(err)
	}
	// The real five-second maintenance path now sees writable publication.
	server.sweepSeedSessions(t.Context(), time.Now().Add(time.Minute))
	t.Logf("after recovery sweep: activeSlots=%d retainedRecords=%d", len(server.seedSessionsByScenario), len(server.seedSessions))
	if server.seedOwnerErr != nil || len(server.seedSessionsByScenario) != 0 {
		t.Fatal("writable validated owner did not durably retire and release its exact reservation", server.seedOwnerErr)
	}
	// Later admission must be available, without permitting replay of the old key.
	server.scenarios = &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}
	previous := resolveSeedTargetAPI
	resolveSeedTargetAPI = func(context.Context, string, string) (string, error) {
		return "", errors.New("effect-free independent admission fixture")
	}
	defer func() { resolveSeedTargetAPI = previous }()
	later, _, laterErr := server.applySeedSession(t.Context(), "demo", source, "independent-later", false)
	if later == "" || later == token || laterErr == nil || !strings.Contains(laterErr.Error(), "effect-free independent") {
		t.Fatal("later independent reservation was not admitted after retirement", laterErr)
	}
	same, _, sameErr := server.applySeedSession(t.Context(), "demo", source, "retirement-publication-failure", false)
	if same != token || sameErr == nil || !strings.Contains(sameErr.Error(), "original failed validation") {
		t.Fatal("old request did not reattach to its retired exact owner", sameErr)
	}
	// Also cover retention: pruning must not erase the only cleanup handle
	// while leaving its active capacity reservation permanently occupied.
	server.sweepSeedSessions(t.Context(), time.Now().Add(seedSessionRetention+time.Minute))
	t.Logf("after retention sweep: activeSlots=%d retainedRecords=%d", len(server.seedSessionsByScenario), len(server.seedSessions))
	if len(server.seedSessionsByScenario) != 0 {
		t.Fatal("completed exact cleanup never released capacity after registry writes recovered; retention erased its handle")
	}
}

func TestFailedPreparationRetirementPublicationKeepsRetryableReservation(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "scenarios", "demo")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}}
	if err := server.initializeSeedOwner(filepath.Join(dir, "owner")); err != nil {
		t.Fatal(err)
	}
	defer server.seedStore.release()
	registry := filepath.Join(server.seedStore.dir, "registry.json")
	saved := registry + ".saved"
	previous := resolveSeedTargetAPI
	resolveSeedTargetAPI = func(context.Context, string, string) (string, error) {
		if err := os.Rename(registry, saved); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(registry, 0700); err != nil {
			t.Fatal(err)
		}
		return "", errors.New("original pre-effect setup failure")
	}
	defer func() { resolveSeedTargetAPI = previous }()
	token, record, err := server.applySeedSession(t.Context(), "demo", source, "failed-setup", false)
	if err == nil || token == "" || record.Session != nil || server.seedSessionsByScenario["demo"] != token {
		t.Fatal("failed publication released uncommitted setup retirement", err)
	}
	server.sweepSeedSessions(t.Context(), time.Now().Add(seedSessionRetention+time.Minute))
	if server.seedSessions[token] != record || server.seedSessionsByScenario["demo"] != token {
		t.Fatal("failed setup lost its only pending retirement handle")
	}
	if err := os.Remove(registry); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(saved, registry); err != nil {
		t.Fatal(err)
	}
	server.sweepSeedSessions(t.Context(), time.Now().Add(seedSessionRetention+time.Minute))
	if len(server.seedSessionsByScenario) != 0 || server.seedOwnerErr != nil || server.seedSessions[token] != record {
		t.Fatal("effect-free failed setup did not durably retire before prune", server.seedOwnerErr)
	}
	var disk seedRegistryDisk
	raw, err := os.ReadFile(registry)
	if err != nil || json.Unmarshal(raw, &disk) != nil || len(disk.Sessions) != 1 || disk.Sessions[0].RetiredAt.IsZero() || !strings.Contains(disk.Sessions[0].PrepareError, "original pre-effect setup failure") {
		t.Fatal("retirement publication erased original failed setup", err)
	}
	server.sweepSeedSessions(t.Context(), time.Now().Add(seedSessionRetention+time.Minute))
	if len(server.seedSessions) != 0 {
		t.Fatal("durably retired failed setup did not converge retention")
	}
}

func TestPendingSeedReservationRefusalConvergesOnlyAfterExactRetirement(t *testing.T) {
	for _, cut := range []string{"checkpoint ack", "remove", "remove sync ack", "retirement publication"} {
		t.Run(cut, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "scenarios", "demo")
			if err := os.MkdirAll(source, 0700); err != nil {
				t.Fatal(err)
			}
			server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}}
			if err := server.initializeSeedOwner(filepath.Join(dir, "owner")); err != nil {
				t.Fatal(err)
			}
			defer server.seedStore.release()
			reserve, retire, resolve := reserveSeedCheckpoint, retireSeedCheckpointReservation, resolveSeedTargetAPI
			defer func() {
				reserveSeedCheckpoint = reserve
				retireSeedCheckpointReservation = retire
				resolveSeedTargetAPI = resolve
			}()
			reserveSeedCheckpoint = func(path, runID string) error {
				if err := reserve(path, runID); err != nil {
					return err
				}
				return errors.New("original reservation acknowledgement failure")
			}
			resolved := false
			resolveSeedTargetAPI = func(context.Context, string, string) (string, error) {
				resolved = true
				return "", errors.New("effect-free independent admission probe")
			}
			_, record, err := server.applySeedSession(t.Context(), "demo", source, "uncertain-reservation", false)
			if err == nil || record != nil || server.seedOwnerErr == nil || server.seedStore.pending == nil || resolved {
				t.Fatal("failed reservation lost exact pending owner or dispatched", err)
			}
			pending := server.seedStore.pending
			path := server.seedCheckpointPath(pending.Record.RunID)
			registry := filepath.Join(server.seedStore.dir, "registry.json")
			saved := registry + ".saved"
			if cut == "retirement publication" {
				if err := os.Rename(registry, saved); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(registry, 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				retireSeedCheckpointReservation = func(path, runID string) error {
					if cut == "remove sync ack" {
						if err := retire(path, runID); err != nil {
							return err
						}
					}
					return errors.New("injected exact retirement refusal")
				}
			}
			server.sweepSeedSessions(t.Context(), time.Now())
			if server.seedOwnerErr == nil || server.seedStore.pending != pending || pending.Record.PrepareError != "original reservation acknowledgement failure" {
				t.Fatal("unrelated registry write erased pending authority or original error")
			}
			_, _, _ = server.applySeedSession(t.Context(), "demo", source, "must-stay-refused", false)
			if resolved {
				t.Fatal("pending checkpoint admitted target work")
			}
			if cut == "retirement publication" {
				if err := os.Remove(registry); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(saved, registry); err != nil {
					t.Fatal(err)
				}
			}
			reserveSeedCheckpoint = reserve
			retireSeedCheckpointReservation = retire
			server.sweepSeedSessions(t.Context(), time.Now())
			if server.seedStore.pending != nil || server.seedOwnerErr != nil || len(server.seedSessionsByScenario) != 0 {
				t.Fatal("exact pending retirement did not durably converge", server.seedOwnerErr)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("pending checkpoint remains", err)
			}
			token, prior, err := server.applySeedSession(t.Context(), "demo", source, "uncertain-reservation", false)
			if token != pending.Record.Token || prior == nil || err == nil || !strings.Contains(err.Error(), "original reservation acknowledgement failure") || resolved {
				t.Fatal("failed reservation replayed or lost original outcome", err)
			}
			token, _, err = server.applySeedSession(t.Context(), "demo", source, "later-independent", false)
			if token == "" || token == pending.Record.Token || !resolved || err == nil || !strings.Contains(err.Error(), "effect-free independent admission probe") {
				t.Fatal("verified convergence did not permit independent work", err)
			}
		})
	}
}

func TestPendingRetirementCannotClearUnrelatedAuthorityRefusal(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "scenarios", "demo")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}}
	if err := server.initializeSeedOwner(filepath.Join(dir, "owner")); err != nil {
		t.Fatal(err)
	}
	defer server.seedStore.release()
	reserve := reserveSeedCheckpoint
	defer func() { reserveSeedCheckpoint = reserve }()
	reserveSeedCheckpoint = func(path, runID string) error {
		if err := reserve(path, runID); err != nil {
			return err
		}
		return errors.New("reservation acknowledgement failure")
	}
	_, _, _ = server.applySeedSession(t.Context(), "demo", source, "pending", false)
	if server.seedStore.pending == nil {
		t.Fatal("fixture did not retain pending reservation")
	}
	unrelated := errors.New("unrelated authority refusal requires its own proof")
	server.seedOwnerErr = unrelated
	server.sweepSeedSessions(t.Context(), time.Now())
	if server.seedStore.pending != nil || server.seedOwnerErr != unrelated {
		t.Fatal("exact pending convergence cleared unrelated refusal")
	}
	server.seedSessionsMu.Lock()
	err := server.persistSeedSessionsLocked()
	server.seedSessionsMu.Unlock()
	if err != nil || server.seedOwnerErr != unrelated {
		t.Fatal("registry publication cleared unrelated refusal", err)
	}
	if _, record, err := server.applySeedSession(t.Context(), "demo", source, "later", false); connect.CodeOf(err) != connect.CodeUnavailable || record != nil {
		t.Fatal("unrelated refusal admitted work", err)
	}
}

func TestPendingSeedReservationCapacityRefusesBeforeCheckpointOrJournalGrowth(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "scenarios", "demo")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}}
	if err := server.initializeSeedOwner(filepath.Join(dir, "owner")); err != nil {
		t.Fatal(err)
	}
	defer server.seedStore.release()
	runID := "cff8b88a-c381-45df-90d1-cf793306d09a"
	if err := phases.InitializeSeedCheckpoint(server.seedCheckpointPath(runID), runID); err != nil {
		t.Fatal(err)
	}
	history := &seedSession{Scenario: "history", scenarioPath: "/" + strings.Repeat("<", seedRegistryAdmissionBytes/6+100), requestID: "legacy", runID: runID, Created: time.Now(), retiredAt: time.Now(), ready: make(chan struct{})}
	close(history.ready)
	server.seedSessions["bf8cc4ed-d5d4-4e51-8696-8e1f21dd29b1"] = history
	if err := server.persistSeedSessionsLocked(); err != nil {
		t.Fatal(err)
	}
	registry := filepath.Join(server.seedStore.dir, "registry.json")
	before, _ := os.ReadFile(registry)
	entries, _ := os.ReadDir(server.seedStore.dir)
	previous := reserveSeedCheckpoint
	defer func() { reserveSeedCheckpoint = previous }()
	calls := 0
	reserveSeedCheckpoint = func(string, string) error { calls++; return errors.New("checkpoint creation must not occur") }
	_, record, err := server.applySeedSession(t.Context(), "demo", source, "capacity", false)
	after, _ := os.ReadFile(registry)
	afterEntries, _ := os.ReadDir(server.seedStore.dir)
	if connect.CodeOf(err) != connect.CodeResourceExhausted || record != nil || calls != 0 || server.seedOwnerErr != nil || server.seedStore.pending != nil || string(before) != string(after) || len(entries) != len(afterEntries) {
		t.Fatal("capacity rejection changed authority or created checkpoint", err)
	}
}

// This helper is only a private filesystem journal owner. It never initializes
// an API, allocates resources, calls a target or submits a native unit.
func TestPendingSeedReservationCrashHelper(t *testing.T) {
	root := os.Getenv("TG_PENDING_CRASH_ROOT")
	if root == "" {
		return
	}
	source := filepath.Join(root, "scenarios", "demo")
	server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}}
	if err := server.initializeSeedOwner(filepath.Join(root, "owner")); err != nil {
		t.Fatal(err)
	}
	token := "329417d0-1984-4b6c-b05c-27ecc597d126"
	runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
	record := &seedSession{Scenario: "demo", Created: time.Now(), requestID: "crash-reservation", scenarioPath: source, runID: runID, ready: make(chan struct{})}
	server.seedStore.pending = &seedPendingReservation{State: seedPendingCheckpointState, Record: seedRecordDisk{Token: token, RunID: runID, Scenario: "demo", ScenarioPath: source, RequestID: record.requestID, Created: record.Created}}
	cut := os.Getenv("TG_PENDING_CRASH_CUT")
	if cut != "before descriptor" {
		if err := server.persistSeedSessionsLocked(); err != nil {
			t.Fatal(err)
		}
	}
	if cut != "before descriptor" && cut != "after descriptor" {
		if err := phases.InitializeSeedCheckpoint(server.seedCheckpointPath(runID), runID); err != nil {
			t.Fatal(err)
		}
		switch cut {
		case "checkpoint ack":
			_ = server.failSeedReservationLocked(errors.New("original checkpoint acknowledgement failure"))
		case "transfer":
			server.seedStore.pending = nil
			server.seedSessions[token] = record
			server.seedSessionsByScenario["demo"] = token
			if err := server.persistSeedSessionsLocked(); err != nil {
				t.Fatal(err)
			}
		case "removed before sync":
			if err := server.seedStore.root.Remove(runID + ".json"); err != nil {
				t.Fatal(err)
			}
		case "removed after sync":
			if err := phases.RetireSeedCheckpointReservation(server.seedCheckpointPath(runID), runID); err != nil {
				t.Fatal(err)
			}
		case "pending retirement":
			_ = server.failSeedReservationLocked(errors.New("original checkpoint acknowledgement failure"))
			if err := server.reconcileSeedReservationLocked(); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(root, "cut-ready"), []byte(cut), 0600); err != nil {
		t.Fatal(err)
	}
	<-t.Context().Done()
}

func TestPendingSeedReservationReplacementRecoversActualKilledJournalOwner(t *testing.T) {
	for _, cut := range []string{"before descriptor", "after descriptor", "checkpoint created", "checkpoint ack", "transfer", "removed before sync", "removed after sync", "pending retirement"} {
		t.Run(cut, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "scenarios", "demo")
			if err := os.MkdirAll(source, 0700); err != nil {
				t.Fatal(err)
			}
			child := exec.Command(os.Args[0], "-test.run=^TestPendingSeedReservationCrashHelper$", "-test.timeout=30s")
			child.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "TG_PENDING_CRASH_ROOT=" + root, "TG_PENDING_CRASH_CUT=" + cut}
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = child.Process.Kill()
				if child.ProcessState == nil {
					_ = child.Wait()
				}
			}()
			deadline := time.Now().Add(5 * time.Second)
			ready := false
			for time.Now().Before(deadline) {
				if raw, err := os.ReadFile(filepath.Join(root, "cut-ready")); err == nil && string(raw) == cut {
					ready = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !ready {
				t.Fatal("private child did not reach the exact persistence cut")
			}
			if err := child.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = child.Wait()
			replacement := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}}
			if err := replacement.initializeSeedOwner(filepath.Join(root, "owner")); err != nil {
				t.Fatal("replacement refused an exact owned checkpoint-only plan", err)
			}
			defer replacement.seedStore.release()
			resolve := resolveSeedTargetAPI
			defer func() { resolveSeedTargetAPI = resolve }()
			calls := 0
			resolveSeedTargetAPI = func(context.Context, string, string) (string, error) {
				calls++
				return "", errors.New("effect-free independent admission probe")
			}
			if replacement.seedStore.pending != nil {
				if _, _, err := replacement.applySeedSession(t.Context(), "demo", source, "early-independent", false); connect.CodeOf(err) != connect.CodeUnavailable || calls != 0 {
					t.Fatal("replacement admitted before exact pending retirement", err)
				}
			}
			replacement.sweepSeedSessions(t.Context(), time.Now())
			replacement.seedMaintenance.Wait()
			if replacement.seedStore.pending != nil || replacement.seedOwnerErr != nil || len(replacement.seedSessionsByScenario) != 0 {
				t.Fatal("replacement did not converge exact owned reservation", replacement.seedOwnerErr)
			}
			token := "329417d0-1984-4b6c-b05c-27ecc597d126"
			runID := "31c832dd-4761-4f80-aa9c-d43da34173df"
			if cut != "transfer" {
				if _, err := os.Stat(replacement.seedCheckpointPath(runID)); !os.IsNotExist(err) {
					t.Fatal("effect-free checkpoint remains after replacement", err)
				}
			}
			if cut != "before descriptor" {
				same, record, err := replacement.applySeedSession(t.Context(), "demo", source, "crash-reservation", false)
				if same != token || record == nil || err == nil || calls != 0 {
					t.Fatal("replacement replayed interrupted reservation", err)
				}
			}
			later, _, err := replacement.applySeedSession(t.Context(), "demo", source, "later-independent", false)
			if later == "" || later == token || calls != 1 || err == nil || !strings.Contains(err.Error(), "effect-free independent admission probe") {
				t.Fatal("verified replacement cleanup did not permit independent work", err)
			}
		})
	}
}

func TestUnsupportedPendingSeedAuthorityRemainsUntouchedAcrossReplacement(t *testing.T) {
	for _, kind := range []string{"state", "duplicate token", "duplicate run", "retired", "oversized request", "nil run", "nil token", "active capacity", "retained capacity"} {
		t.Run(kind, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "owner")
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
			row := seedRecordDisk{Token: "329417d0-1984-4b6c-b05c-27ecc597d126", RunID: "31c832dd-4761-4f80-aa9c-d43da34173df", Scenario: "demo", ScenarioPath: "/private/demo", RequestID: "pending", Created: time.Now()}
			pending := &seedPendingReservation{State: seedPendingCheckpointState, Record: row}
			disk := seedRegistryDisk{Version: 1, PendingReservation: pending}
			switch kind {
			case "nil run":
				pending.Record.RunID = "00000000-0000-0000-0000-000000000000"
			case "nil token":
				pending.Record.Token = "00000000-0000-0000-0000-000000000000"
			case "active capacity", "retained capacity":
				count := activeSeedSessionLimit
				if kind == "retained capacity" {
					count = seedSessionLimit
				}
				for i := 0; i < count; i++ {
					other := row
					other.Token = fmt.Sprintf("10000000-0000-4000-8000-%012d", i+1)
					other.RunID = fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1)
					other.Scenario = fmt.Sprintf("scenario-%d", i)
					if kind == "retained capacity" {
						other.RetiredAt = time.Now()
						other.Pruning = true
					} else if err := phases.InitializeSeedCheckpoint(filepath.Join(dir, other.RunID+".json"), other.RunID); err != nil {
						t.Fatal(err)
					}
					disk.Sessions = append(disk.Sessions, other)
				}
			case "state":
				pending.State = "unsupported-future-stage"
			case "duplicate token":
				other := row
				other.RunID = "cff8b88a-c381-45df-90d1-cf793306d09a"
				disk.Sessions = []seedRecordDisk{other}
			case "duplicate run":
				other := row
				other.Token = "bf8cc4ed-d5d4-4e51-8696-8e1f21dd29b1"
				disk.Sessions = []seedRecordDisk{other}
			case "retired":
				pending.Record.RetiredAt = time.Now()
			case "oversized request":
				pending.Record.RequestID = strings.Repeat("x", 201)
			}
			raw, err := json.Marshal(disk)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "registry.json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			for attempt := 0; attempt < 2; attempt++ {
				server := &Server{}
				server.seedOwnerErr = server.initializeSeedOwner(dir)
				if server.seedOwnerErr == nil || (server.seedStore != nil && server.seedStore.validated) {
					t.Fatal("unsupported pending authority was admitted")
				}
				server.sweepSeedSessions(t.Context(), time.Now())
				after, err := os.ReadFile(path)
				if err != nil || string(raw) != string(after) {
					t.Fatal("unsupported pending authority was rewritten", err)
				}
				if server.seedStore != nil {
					server.seedStore.release()
				}
			}
		})
	}
}

func TestPendingSeedReservationErrorExcerptRemainsLoadableAtUTF8Boundary(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}}
	if err := server.initializeSeedOwner(filepath.Join(root, "owner")); err != nil {
		t.Fatal(err)
	}
	reserve := reserveSeedCheckpoint
	defer func() { reserveSeedCheckpoint = reserve }()
	reserveSeedCheckpoint = func(path, runID string) error {
		if err := reserve(path, runID); err != nil {
			return err
		}
		return errors.New(strings.Repeat("x", seedPrivateErrorMaxBytes-1) + "€ and more")
	}
	_, _, _ = server.applySeedSession(t.Context(), "demo", source, "utf8-failure", false)
	if server.seedStore.pending == nil {
		t.Fatal("fixture did not retain pending error")
	}
	excerpt := server.seedStore.pending.Record.PrepareError
	if len(excerpt) > seedPrivateErrorMaxBytes || !strings.HasSuffix(excerpt, " [truncated]") {
		t.Fatal("pending error excerpt violated its bound")
	}
	server.seedStore.release()
	replacement := &Server{}
	if err := replacement.initializeSeedOwner(filepath.Join(root, "owner")); err != nil {
		t.Fatal("writer produced an unloadable pending excerpt", err)
	}
	defer replacement.seedStore.release()
	if replacement.seedStore.pending == nil || replacement.seedStore.pending.Record.PrepareError != excerpt {
		t.Fatal("pending error changed across replacement")
	}
	replacement.sweepSeedSessions(t.Context(), time.Now())
	if replacement.seedStore.pending != nil || replacement.seedOwnerErr != nil {
		t.Fatal("bounded pending error prevented exact convergence", replacement.seedOwnerErr)
	}
}

func TestPendingSeedReservationCountsTowardHardEncodedRegistryBound(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "owner")
	server := &Server{}
	if err := server.initializeSeedOwner(dir); err != nil {
		t.Fatal(err)
	}
	defer server.seedStore.release()
	token := "bf8cc4ed-d5d4-4e51-8696-8e1f21dd29b1"
	history := &seedSession{Scenario: "history", scenarioPath: "/" + strings.Repeat("x", seedRegistryMaxBytes-2048), requestID: "legacy", runID: "cff8b88a-c381-45df-90d1-cf793306d09a", Created: time.Now(), retiredAt: time.Now(), pruning: true, ready: make(chan struct{})}
	close(history.ready)
	server.seedSessions[token] = history
	if err := server.persistSeedSessionsLocked(); err != nil {
		t.Fatal("bounded prior state did not fit", err)
	}
	path := filepath.Join(dir, "registry.json")
	before, _ := os.ReadFile(path)
	server.seedStore.pending = &seedPendingReservation{State: seedPendingCheckpointState, Record: seedRecordDisk{Token: "329417d0-1984-4b6c-b05c-27ecc597d126", RunID: "31c832dd-4761-4f80-aa9c-d43da34173df", Scenario: "demo", ScenarioPath: "/private/demo", RequestID: "pending", Created: time.Now(), PrepareError: strings.Repeat("e", seedPrivateErrorMaxBytes)}}
	if err := server.persistSeedSessionsLocked(); !errors.Is(err, errSeedRegistryCapacity) {
		t.Fatal("pending descriptor escaped hard encoded byte limit", err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("oversized pending authority replaced bounded registry")
	}
}

func TestPendingForeignCheckpointRefusesMaintenanceAndRepeatedReplacement(t *testing.T) {
	for _, kind := range []string{"nonempty", "symlink", "directory"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "scenarios", "demo")
			if err := os.MkdirAll(source, 0700); err != nil {
				t.Fatal(err)
			}
			journal := filepath.Join(root, "owner")
			server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}}
			if err := server.initializeSeedOwner(journal); err != nil {
				t.Fatal(err)
			}
			reserve := reserveSeedCheckpoint
			defer func() { reserveSeedCheckpoint = reserve }()
			reserveSeedCheckpoint = func(path, runID string) error {
				if err := reserve(path, runID); err != nil {
					return err
				}
				return errors.New("original acknowledgement failure")
			}
			_, _, _ = server.applySeedSession(t.Context(), "demo", source, "foreign-pending", false)
			if server.seedStore.pending == nil {
				t.Fatal("fixture did not retain pending authority")
			}
			path := server.seedCheckpointPath(server.seedStore.pending.Record.RunID)
			switch kind {
			case "nonempty":
				if err := os.WriteFile(path, []byte("foreign checkpoint contents"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Rename(path, path+".foreign"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".foreign", path); err != nil {
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
			info, _ := os.Lstat(path)
			raw, _ := os.ReadFile(path)
			server.seedStore.release()
			for attempt := 0; attempt < 2; attempt++ {
				replacement := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: filepath.Dir(source)}}
				if err := replacement.initializeSeedOwner(journal); err != nil {
					t.Fatal("valid pending plan could not retain its refusal", err)
				}
				replacement.sweepSeedSessions(t.Context(), time.Now())
				if replacement.seedStore.pending == nil || replacement.seedOwnerErr == nil || replacement.seedStore.pending.Record.PrepareError != "original acknowledgement failure" {
					t.Fatal("foreign artifact lost pending owner/refusal or original error")
				}
				after, err := os.Lstat(path)
				afterRaw, _ := os.ReadFile(path)
				if err != nil || !os.SameFile(info, after) || string(raw) != string(afterRaw) {
					t.Fatal("replacement modified foreign pending artifact", err)
				}
				if _, record, err := replacement.applySeedSession(t.Context(), "demo", source, "later", false); connect.CodeOf(err) != connect.CodeUnavailable || record != nil {
					t.Fatal("foreign pending owner admitted work", err)
				}
				replacement.seedStore.release()
			}
		})
	}
}

func seedSemanticIdentityRows(t *testing.T) (string, seedRecordDisk, seedRecordDisk) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "owner")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	old := seedRecordDisk{Token: "329417d0-1984-4b6c-b05c-27ecc597d126", RunID: "31c832dd-4761-4f80-aa9c-d43da34173df", Scenario: "demo", ScenarioPath: "/private/demo", RequestID: "request", Created: time.Now(), RetiredAt: time.Now(), Pruning: true, PrepareError: "first retained failure"}
	next := old
	next.Token = "bf8cc4ed-d5d4-4e51-8696-8e1f21dd29b1"
	next.RunID = "cff8b88a-c381-45df-90d1-cf793306d09a"
	next.RetiredAt = time.Time{}
	next.Pruning = false
	next.PrepareError = "second retained failure"
	return dir, old, next
}

func TestSeedRegistrySemanticConflictsRefuseBeforeRecoveryAndPreserveAuthority(t *testing.T) {
	for _, kind := range []string{"pending same request", "pending other checkout", "pending active scenario", "pending ancient created", "normal same request", "normal other checkout", "normal active scenario", "normal duplicate run", "noncanonical scenario"} {
		t.Run(kind, func(t *testing.T) {
			dir, old, next := seedSemanticIdentityRows(t)
			disk := seedRegistryDisk{Version: 1, Sessions: []seedRecordDisk{old}, PendingReservation: &seedPendingReservation{State: seedPendingCheckpointState, Record: next}}
			expected := "duplicate"
			switch kind {
			case "pending other checkout":
				disk.PendingReservation.Record.ScenarioPath = "/private/other"
			case "pending active scenario":
				disk.Sessions[0].RetiredAt = time.Time{}
				disk.Sessions[0].Pruning = false
				disk.PendingReservation.Record.RequestID = "different"
			case "pending ancient created":
				disk.PendingReservation.Record.Created = time.Now().Add(-2 * seedSessionRetention)
			case "normal same request", "normal other checkout", "normal active scenario", "normal duplicate run":
				next.RetiredAt = time.Now()
				next.Pruning = true
				disk.PendingReservation = nil
				if kind == "normal other checkout" {
					next.ScenarioPath = "/private/other"
				}
				if kind == "normal active scenario" {
					disk.Sessions[0].RetiredAt = time.Time{}
					disk.Sessions[0].Pruning = false
					next.RetiredAt = time.Time{}
					next.Pruning = false
					next.RequestID = "different"
				}
				if kind == "normal duplicate run" {
					next.RunID = old.RunID
					next.RequestID = "different"
				}
				disk.Sessions = append(disk.Sessions, next)
			case "noncanonical scenario":
				disk.Sessions[0].Scenario = " demo "
				expected = "invalid"
			}
			raw, err := json.Marshal(disk)
			if err != nil {
				t.Fatal(err)
			}
			registry := filepath.Join(dir, "registry.json")
			if err := os.WriteFile(registry, raw, 0600); err != nil {
				t.Fatal(err)
			}
			// If recovery starts before semantic validation, this checkpoint yields a
			// different parse failure. It must stay unopened as cleanup authority.
			checkpoint := filepath.Join(dir, old.RunID+".json")
			marker := []byte("private checkpoint must not be recovered for invalid document")
			if err := os.WriteFile(checkpoint, marker, 0600); err != nil {
				t.Fatal(err)
			}
			info, _ := os.Lstat(checkpoint)
			for attempt := 0; attempt < 2; attempt++ {
				server := &Server{}
				server.seedOwnerErr = server.initializeSeedOwner(dir)
				if server.seedOwnerErr == nil || !strings.Contains(server.seedOwnerErr.Error(), expected) || server.seedStore == nil || server.seedStore.validated || server.seedStore.pending != nil || len(server.seedSessions) != 0 {
					t.Fatal("semantic conflict exposed cleanup authority before whole-document validation", server.seedOwnerErr)
				}
				server.sweepSeedSessions(t.Context(), time.Now())
				if _, err := server.findSeedSession("demo", old.Token); connect.CodeOf(err) != connect.CodeUnavailable {
					t.Fatal("failed semantic owner exposed retained cleanup", err)
				}
				after, err := os.ReadFile(registry)
				contents, _ := os.ReadFile(checkpoint)
				current, _ := os.Lstat(checkpoint)
				if err != nil || string(after) != string(raw) || string(contents) != string(marker) || !os.SameFile(info, current) {
					t.Fatal("failed semantic owner changed retained authority", err)
				}
				server.seedStore.release()
			}
		})
	}
}

func TestSeedRequestBindingRetentionExactBoundary(t *testing.T) {
	now := time.Now()
	retired := now.Add(-seedSessionRetention)
	for _, test := range []struct {
		name    string
		at      time.Time
		expired bool
	}{{"before", now.Add(-time.Nanosecond), false}, {"exact", now, false}, {"after", now.Add(time.Nanosecond), true}} {
		t.Run(test.name, func(t *testing.T) {
			if seedOutcomeExpired(retired, test.at) != test.expired || seedOutcomeExpired(time.Time{}, test.at) {
				t.Fatal("retention expiration boundary changed")
			}
			_, old, next := seedSemanticIdentityRows(t)
			old.RetiredAt = retired
			old.Created = retired.Add(-time.Minute)
			disk := seedRegistryDisk{Version: 1, Sessions: []seedRecordDisk{old}, PendingReservation: &seedPendingReservation{State: seedPendingCheckpointState, Record: next}}
			err := validateSeedRegistryIdentity(disk, test.at)
			if (err == nil) != test.expired {
				t.Fatal("loader request binding differs from Apply/maintenance expiration", err)
			}
		})
	}
}

func TestSeedRegistryAllowsDistinctRequestsAndExpiredLegacyHistory(t *testing.T) {
	for _, kind := range []string{"distinct retired request", "legacy retained path", "legacy missing request", "raw request bytes", "other scenario same request", "expired same request", "expired other checkout"} {
		t.Run(kind, func(t *testing.T) {
			dir, old, next := seedSemanticIdentityRows(t)
			switch kind {
			case "distinct retired request":
				old.RequestID = "older-request"
			case "legacy retained path", "legacy missing request":
				old.RequestID = "older-request"
				old.ScenarioPath = "/" + strings.Repeat("x", maxSeedSourcePathBytes+1)
				if kind == "legacy missing request" {
					old.RequestID = ""
				}
			case "raw request bytes":
				old.RequestID = " request "
			case "other scenario same request":
				old.Scenario = "other-scenario"
			case "expired same request", "expired other checkout":
				old.RetiredAt = time.Now().Add(-seedSessionRetention - time.Minute)
				old.Created = old.RetiredAt.Add(-time.Minute)
				old.ScenarioPath = "/" + strings.Repeat("x", maxSeedSourcePathBytes+1)
			}
			if kind == "expired other checkout" {
				next.ScenarioPath = "/private/other"
			}
			raw, err := json.Marshal(seedRegistryDisk{Version: 1, Sessions: []seedRecordDisk{old}, PendingReservation: &seedPendingReservation{State: seedPendingCheckpointState, Record: next}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "registry.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: "/private"}}
			if err := server.initializeSeedOwner(dir); err != nil {
				t.Fatal("legitimate retained history was refused", err)
			}
			defer server.seedStore.release()
			if len(server.seedSessions) != 1 || server.seedStore.pending == nil {
				t.Fatal("valid history or pending identity was dropped")
			}
			server.sweepSeedSessions(t.Context(), time.Now())
			if server.seedStore.pending != nil || server.seedOwnerErr != nil {
				t.Fatal("valid distinct/expired history blocked exact pending convergence", server.seedOwnerErr)
			}
			for i := 0; i < 128; i++ {
				token, record, err := server.applySeedSession(t.Context(), next.Scenario, next.ScenarioPath, next.RequestID, false)
				if token != next.Token || record == nil || err == nil || !strings.Contains(err.Error(), "second retained failure") {
					t.Fatal("same current request did not retain one exact failed owner", err)
				}
			}
			token, _, err := server.applySeedSession(t.Context(), next.Scenario, "/private/different", next.RequestID, false)
			if token != "" || connect.CodeOf(err) != connect.CodeAlreadyExists {
				t.Fatal("current request checkout binding was not retained", err)
			}
		})
	}
}

func TestSeedRegistryPreservesUnboundLegacyRetiredHistory(t *testing.T) {
	dir, old, pending := seedSemanticIdentityRows(t)
	old.RequestID = ""
	second := old
	second.Token = "a9a89022-d958-4b1a-99f5-38e445378908"
	second.RunID = "a00d6d42-2205-4cdb-b1b7-c5c64a17ad1e"
	raw, err := json.Marshal(seedRegistryDisk{Version: 1, Sessions: []seedRecordDisk{old, second}, PendingReservation: &seedPendingReservation{State: seedPendingCheckpointState, Record: pending}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: "/private"}}
	if err := server.initializeSeedOwner(dir); err != nil {
		t.Fatal("unbound legacy history became a false request-binding refusal", err)
	}
	defer server.seedStore.release()
	if len(server.seedSessions) != 2 {
		t.Fatal("unbound legacy cleanup outcomes were dropped")
	}
	server.sweepSeedSessions(t.Context(), time.Now())
	token, record, err := server.applySeedSession(t.Context(), pending.Scenario, pending.ScenarioPath, pending.RequestID, false)
	if server.seedStore.pending != nil || server.seedOwnerErr != nil || len(server.seedSessions) != 3 || token != pending.Token || record == nil || err == nil {
		t.Fatal("legacy history changed current request ownership or exact convergence", err)
	}
}
