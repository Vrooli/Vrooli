package httpserver

import (
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/workspace"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/projectmeta"
	"github.com/vrooli/api-core/storage"
	seedspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds"
	seedsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/seeds/seeds_v1connect"
)

func TestSeedConnectRetainsOneActualSeedAndItsClearEvidence(t *testing.T) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	source := filepath.Join(scenariosRoot, "demo")
	for _, dir := range []string{filepath.Join(root, ".vrooli"), filepath.Join(source, "api"), filepath.Join(source, "bas", "seeds")} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for path, body := range map[string]string{filepath.Join(root, ".vrooli", "service.json"): `{"mode":"development"}`, filepath.Join(source, "api", "go.mod"): "module seedfixture\n\ngo 1.26.0\nrequire modernc.org/sqlite v1.50.1\n"} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	projectmeta.SetStartDirForTesting(root)
	script := `set -eu
mkdir -p "$VROOLI_STORAGE_ROOT/project" "$TEST_GENIE_SCENARIO_DIR/coverage/runtime"
printf 'seed\n' >> "$TEST_GENIE_SCENARIO_DIR/coverage/runtime/calls.txt"
printf '{"project":"%s/project"}' "$VROOLI_STORAGE_ROOT" > "$TEST_GENIE_SCENARIO_DIR/coverage/runtime/seed-state.json"
`
	if err := os.WriteFile(filepath.Join(source, "bas", "seeds", "seed.sh"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(t.Context(), database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, "CREATE TABLE effects(value TEXT)")
		return err
	})
	paths := storage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")}
	roots := filerouting.New(paths)
	targetMux := http.NewServeMux()
	owner := devrouting.NewService(db, roots)
	devrouting.RegisterWithFileRootsService(targetMux, db, roots, owner)
	target := httptest.NewServer(owner.PinnedTestModeMiddleware(targetMux))
	defer target.Close()
	prior := resolveSeedTargetAPI
	resolveSeedTargetAPI = func(context.Context, string, string) (string, error) { return target.URL, nil }
	t.Cleanup(func() { resolveSeedTargetAPI = prior })
	server := &Server{router: mux.NewRouter(), scenarios: &stubScenarioDirectory{scenarioRoot: scenariosRoot}, logger: log.New(io.Discard, "", 0)}
	server.setupRoutes()
	api := httptest.NewServer(server.Handler())
	defer api.Close()
	client := seedsconnect.NewSeedServiceClient(api.Client(), api.URL)
	request := &seedspb.ApplySeedRequest{Scenario: "demo", RequestId: "one-request", ScenarioPath: source}
	first, err := client.ApplySeed(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	token := first.Msg.GetSessionToken()
	lease := first.Msg.GetEvidence().GetLeaseId()
	defer func() {
		_, _ = client.ClearSeed(context.Background(), connect.NewRequest(&seedspb.ClearSeedRequest{Scenario: "demo", SessionToken: token}))
	}()
	second, err := client.ApplySeed(t.Context(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || second.Msg.GetSessionToken() != token || second.Msg.GetEvidence().GetLeaseId() != lease {
		t.Fatal("apply retry created another seed binding")
	}
	raw, err := os.ReadFile(filepath.Join(source, "coverage", "runtime", "calls.txt"))
	if err != nil || string(raw) != "seed\n" {
		t.Fatalf("seed execution replayed: %q %v", raw, err)
	}
	if _, err := client.ClearSeed(t.Context(), connect.NewRequest(&seedspb.ClearSeedRequest{Scenario: "demo", SessionToken: "unknown"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown token result: %v", err)
	}
	if _, err := client.GetSeed(t.Context(), connect.NewRequest(&seedspb.GetSeedRequest{Scenario: "demo", SessionToken: token})); err != nil {
		t.Fatal("unknown cleanup removed active owner", err)
	}
	if _, err := db.ExecContext(database.WithTestMode(t.Context()), "INSERT INTO effects VALUES ('leased')"); err != nil {
		t.Fatal(err)
	}
	project := first.Msg.GetInitialParams().AsMap()["project"].(string)
	for i := 0; i < 2; i++ {
		result, err := client.ClearSeed(t.Context(), connect.NewRequest(&seedspb.ClearSeedRequest{Scenario: "demo", SessionToken: token}))
		if err != nil {
			t.Fatal(err)
		}
		e := result.Msg.GetEvidence()
		if e.GetState() != seedspb.SeedState_SEED_STATE_CLEARED || e.GetStats().GetTestPoolRequests() != 1 {
			t.Fatalf("clear retry lost actual counters: %v", e)
		}
	}
	if _, err := os.Stat(project); !os.IsNotExist(err) {
		t.Fatalf("backing workspace not retired: %v", err)
	}
	server.seedSessionsMu.Lock()
	active := server.seedSessionsByScenario["demo"]
	server.seedSessionsMu.Unlock()
	if active != "" {
		t.Fatal("retired session retains active ownership")
	}
	// Owner restart must return missing state rather than invent successful cleanup.
	replacement := &seedConnectService{server: &Server{}}
	_, err = replacement.GetSeed(t.Context(), connect.NewRequest(&seedspb.GetSeedRequest{Scenario: "demo", SessionToken: token}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("replacement owner fabricated proof: %v", err)
	}
	if strings.Contains(first.Msg.GetEvidence().String(), token) {
		t.Fatal("public evidence contains bearer token")
	}
}

func TestSeedReservationCoalescesSetupAndDoesNotExposeForeignToken(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0755); err != nil {
		t.Fatal(err)
	}
	priorResolve := resolveSeedTargetAPI
	resolveSeedTargetAPI = func(context.Context, string, string) (string, error) { return "http://127.0.0.1:1", nil }
	t.Cleanup(func() { resolveSeedTargetAPI = priorResolve })
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	priorApply := applyPlaybooksSeed
	applyPlaybooksSeed = func(context.Context, workspace.Environment, io.Writer, bool) (*phases.PlaybooksSeedSession, error) {
		calls.Add(1)
		close(entered)
		<-release
		return &phases.PlaybooksSeedSession{RunID: "held"}, nil
	}
	t.Cleanup(func() { applyPlaybooksSeed = priorApply })
	server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: root}}
	type outcome struct {
		token string
		err   error
	}
	results := make(chan outcome, 2)
	go func() {
		token, _, err := server.applySeedSession(t.Context(), "demo", "", "same", false)
		results <- outcome{token, err}
	}()
	<-entered
	go func() {
		token, _, err := server.applySeedSession(t.Context(), "demo", "", "same", false)
		results <- outcome{token, err}
	}()
	token, _, err := server.applySeedSession(t.Context(), "demo", "", "foreign", false)
	if connect.CodeOf(err) != connect.CodeAlreadyExists || token != "" {
		t.Fatalf("foreign request obtained active authority: token-present=%t err=%v", token != "", err)
	}
	close(release)
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.token == "" || a.token != b.token || calls.Load() != 1 {
		t.Fatalf("setup did not coalesce once: calls=%d errors=%v/%v", calls.Load(), a.err, b.err)
	}
}

func TestSeedCapacityRefusesBeforePreparingMoreResources(t *testing.T) {
	root := t.TempDir()
	server := &Server{scenarios: &stubScenarioDirectory{scenarioRoot: root}, seedSessionsByScenario: map[string]string{"a": "a", "b": "b", "c": "c", "d": "d"}}
	token, record, err := server.applySeedSession(t.Context(), "next", "", "capacity-request", false)
	if connect.CodeOf(err) != connect.CodeResourceExhausted || token != "" || record != nil {
		t.Fatalf("seed capacity did not refuse before effects: record=%v err=%v", record, err)
	}
	if len(server.seedSessions) != 0 || len(server.seedSessionsByScenario) != 4 {
		t.Fatal("capacity refusal changed existing owners")
	}
}
