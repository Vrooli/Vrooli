package baseline

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/cli-core/cliutil"
)

// TestMain clears VROOLI_AGENT_IDENTITY_TOKEN for the whole package so a
// `baseline diff` test run inside an agent-manager run (the agent's token leaking
// into the test process) cannot accidentally PARK the agent's own run. Tests that
// exercise the park path opt back in via t.Setenv.
func TestMain(m *testing.M) {
	// These tests consume operator-granted authority. Provide a private stable
	// fixture instead of inheriting whichever operator-state file the developer
	// or test runner currently uses.
	previousIdentity, hadIdentity := os.LookupEnv(cliutil.EnvIdentityToken)
	os.Unsetenv(cliutil.EnvIdentityToken)
	previousStorageRoot, hadStorageRoot := os.LookupEnv("VROOLI_STORAGE_ROOT")
	storageRoot, err := os.MkdirTemp("", "gct-baseline-tests-")
	if err != nil {
		panic(err)
	}
	if err := os.Setenv("VROOLI_STORAGE_ROOT", storageRoot); err != nil {
		panic(err)
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		panic(err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "vrooli-onboarding"})
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(paths.StateDir, 0o755); err != nil {
		panic(err)
	}
	authority := map[string]any{"core": map[string][]string{
		"seed":         {"agent-manager", "git-control-tower", "test-genie"},
		"trusted_base": {"git-control-tower", "test-genie"},
	}}
	raw, err := json.Marshal(authority)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(paths.StateDir, "operator-state.json"), raw, 0o600); err != nil {
		panic(err)
	}

	code := m.Run()
	if hadIdentity {
		_ = os.Setenv(cliutil.EnvIdentityToken, previousIdentity)
	} else {
		_ = os.Unsetenv(cliutil.EnvIdentityToken)
	}
	if hadStorageRoot {
		_ = os.Setenv("VROOLI_STORAGE_ROOT", previousStorageRoot)
	} else {
		_ = os.Unsetenv("VROOLI_STORAGE_ROOT")
	}
	_ = os.RemoveAll(storageRoot)
	os.Exit(code)
}

func TestParkForDiff_NotAgentControlled(t *testing.T) {
	_, parked, err := parkForDiff("web-console", "pre-launch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parked {
		t.Fatal("expected parked=false outside an agent-manager run")
	}
}

// TestParkForDiff_ParksInsideAgentManagerRun proves the gct producer parks with
// the git-control-tower producer key and a "<scenario>/<name>" await key.
func TestParkForDiff_ParksInsideAgentManagerRun(t *testing.T) {
	const runID = "run-uuid-gct"
	const identityToken = "agent-token"
	brokerCalls := withParkTestCredentialBroker(t, identityToken)
	am := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/identity/verify":
			json.NewEncoder(w).Encode(cliutil.VerifyResult{
				Valid:     true,
				Claims:    &cliutil.VerifiedClaims{RunID: runID},
				RunStatus: "running",
			})
		case "/api/v1/runs/" + runID + "/park":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["producer"] != cliutil.ParkProducerGCT {
				t.Errorf("producer = %v, want %q", body["producer"], cliutil.ParkProducerGCT)
			}
			if body["key"] != "web-console/pre-launch" {
				t.Errorf("key = %v, want web-console/pre-launch", body["key"])
			}
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"message": "PARKED — waiting on git-control-tower:web-console/pre-launch",
			})
		default:
			t.Errorf("unexpected AM request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer am.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", am.URL)
	t.Setenv(cliutil.EnvIdentityToken, identityToken)

	park, parked, err := parkForDiff("web-console", "pre-launch")
	if err != nil {
		t.Fatalf("parkForDiff: %v", err)
	}
	if !parked {
		t.Fatal("expected parked=true inside an agent-manager run")
	}
	if park == nil || !strings.Contains(park.Message, "PARKED") {
		t.Fatalf("expected PARKED message, got %+v", park)
	}
	if brokerCalls.Load() == 0 {
		t.Fatal("expected park flow to resolve its ordinary run identity through the isolated broker")
	}
}

func withParkTestCredentialBroker(t *testing.T, identityToken string) *atomic.Int32 {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "gct-park-broker-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmpDir)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	socket := cliutil.AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/credential" || r.URL.Query().Get("purpose") != "ordinary_identity" {
			t.Errorf("unexpected credential broker request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			http.Error(w, "unexpected broker request", http.StatusBadRequest)
			return
		}
		if got := r.Header.Get(cliutil.HeaderAgentIdentityToken); got != identityToken {
			t.Errorf("credential broker received identity match=%t, want true", got == identityToken)
			http.Error(w, "unexpected inherited identity", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"baseline_token": identityToken})
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return &calls
}
