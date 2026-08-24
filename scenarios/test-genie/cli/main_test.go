package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	"test-genie/cli/execute"
	"test-genie/cli/remediate"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runs_v1connect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// executeFakeRuns captures the StartRun request the execute command sends over
// the durable Connect surface and completes the run with a successful stream.
type executeFakeRuns struct {
	runs_v1connect.UnimplementedRunsServiceHandler
	mu      sync.Mutex
	started *runspb.StartRunRequest
}

func (f *executeFakeRuns) StartRun(_ context.Context, req *connect.Request[runspb.StartRunRequest]) (*connect.Response[runspb.StartRunResponse], error) {
	f.mu.Lock()
	f.started = req.Msg
	f.mu.Unlock()
	return connect.NewResponse(&runspb.StartRunResponse{RunId: "20260101-000000-abcd1234", Target: req.Msg.GetTarget()}), nil
}

func (f *executeFakeRuns) FollowRun(_ context.Context, _ *connect.Request[runspb.FollowRunRequest], stream *connect.ServerStream[runspb.RunEvent]) error {
	for _, ev := range []*runspb.RunEvent{
		{Event: "run_started", RunId: "20260101-000000-abcd1234"},
		{Event: "run_completed", Success: true, Verdict: "PASS"},
	} {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	return nil
}

// newExecuteTestServer mounts the REST plan-preview endpoint plus the Connect
// RunsService the durable execute path drives.
func newExecuteTestServer(t *testing.T, planJSON string, fake *executeFakeRuns) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":"ok","service":"test-genie"}`)
	})
	mux.HandleFunc("/api/v1/executions/plan", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, planJSON)
	})
	path, handler := runs_v1connect.NewRunsServiceHandler(fake)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestExecuteAcceptsPositionalPhases(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "0") // always follow inline
	fake := &executeFakeRuns{}
	srv := newExecuteTestServer(t,
		`{"scenarioName":"demo","phases":[{"name":"unit","estimatedDurationSeconds":1,"timeoutSeconds":60},{"name":"storage","estimatedDurationSeconds":2,"timeoutSeconds":120}],"summary":{"phaseCount":2,"estimatedDurationSeconds":3,"timeoutSeconds":180}}`,
		fake)

	t.Setenv("TEST_GENIE_API_BASE", srv.URL)
	app := newTestApp(t)
	if err := app.Run([]string{"execute", "demo", "unit", "storage"}); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if got := fake.started.GetPhases(); !reflect.DeepEqual(got, []string{"unit", "storage"}) {
		t.Fatalf("expected phases [unit storage], got %v", got)
	}
}

func TestExecuteSendsExplicitScenarioPath(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "0")
	scenarioPath := filepath.Join(t.TempDir(), "scenarios", "demo")
	logicalRepoRoot := t.TempDir()
	fake := &executeFakeRuns{}
	srv := newExecuteTestServer(t,
		`{"scenarioName":"demo","phases":[{"name":"unit","estimatedDurationSeconds":1,"timeoutSeconds":60}],"summary":{"phaseCount":1,"estimatedDurationSeconds":1,"timeoutSeconds":60}}`,
		fake)

	t.Setenv("TEST_GENIE_API_BASE", srv.URL)
	app := newTestApp(t)
	if err := app.Run([]string{"execute", "demo", "unit", "--scenario-path", scenarioPath, "--logical-repo-root", logicalRepoRoot, "--logical-scenario-relpath", "scenarios/demo"}); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if got := fake.started.GetScenarioPath(); got != scenarioPath {
		t.Fatalf("expected scenarioPath %q, got %q", scenarioPath, got)
	}
	if got := fake.started.GetLogicalRepoRoot(); got != logicalRepoRoot {
		t.Fatalf("expected logicalRepoRoot %q, got %q", logicalRepoRoot, got)
	}
	if got := fake.started.GetLogicalScenarioRelPath(); got != "scenarios/demo" {
		t.Fatalf("expected logicalScenarioRelPath scenarios/demo, got %q", got)
	}
}

func TestExecuteAllPhaseSkipsExplicitList(t *testing.T) {
	t.Setenv("TEST_GENIE_AUTOBACKGROUND_SECONDS", "0")
	fake := &executeFakeRuns{}
	srv := newExecuteTestServer(t,
		`{"scenarioName":"demo","phases":[{"name":"structure","estimatedDurationSeconds":1,"timeoutSeconds":60},{"name":"standards","estimatedDurationSeconds":1,"timeoutSeconds":60}],"summary":{"phaseCount":2,"estimatedDurationSeconds":2,"timeoutSeconds":120}}`,
		fake)

	t.Setenv("TEST_GENIE_API_BASE", srv.URL)
	app := newTestApp(t)
	if err := app.Run([]string{"execute", "demo", "all"}); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if got := fake.started.GetPhases(); len(got) != 0 {
		t.Fatalf("expected phases omitted for 'all', got %v", got)
	}
}

func TestConfigureSetsValues(t *testing.T) {
	app := newTestApp(t)
	if err := app.Run([]string{"configure", "api_base", "http://example.com"}); err != nil {
		t.Fatalf("configure api_base: %v", err)
	}
	if app.core.Config.APIBase != "http://example.com" {
		t.Fatalf("expected api base saved")
	}
	if err := app.Run([]string{"configure", "token", "secret"}); err != nil {
		t.Fatalf("configure token: %v", err)
	}
	if app.core.Config.Token != "secret" {
		t.Fatalf("expected token saved")
	}
}

func TestBuildAPIBaseOptionsUsesPortEnv(t *testing.T) {
	app := newTestApp(t)
	t.Setenv("TEST_GENIE_API_PORT", "4567")
	base := cliutil.DetermineAPIBase(app.core.APIBaseOptions())
	if base != "http://localhost:4567" {
		t.Fatalf("expected base from port env, got %s", base)
	}
}

func TestRemediateCommandSendsEvidenceSelectors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			fmt.Fprintf(w, `{"status":"ok","service":"test-genie"}`)
			return
		case "/api/v1/scenarios/demo/remediation/jobs":
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		if !bytes.Contains(body, []byte(`"sourceExecutionId":"00000000-0000-0000-0000-000000000001"`)) || !bytes.Contains(body, []byte(`"findingIds":["afid:1"]`)) || !bytes.Contains(body, []byte(`"roleRef":"code.default"`)) {
			t.Fatalf("expected remediation selectors in payload, got %s", string(body))
		}
		fmt.Fprintf(w, `{"id":"1","scenario":"demo","status":"running"}`)
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"remediate", "demo", "--execution", "00000000-0000-0000-0000-000000000001", "--findings", "afid:1", "--role", "code.default"}); err != nil {
		t.Fatalf("remediate failed: %v", err)
	}
}

func TestRemediateLifecycleCommandsUseDurableJobEndpoints(t *testing.T) {
	requests := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"ok","service":"test-genie"}`)
			return
		}
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/scenarios/demo/remediation/jobs":
			fmt.Fprint(w, `{"items":[{"id":"job-1","scenario":"demo","status":"failed"}],"count":1}`)
		case "/api/v1/scenarios/demo/remediation/jobs/job-1":
			fmt.Fprint(w, `{"id":"job-1","scenario":"demo","status":"failed","selectedFindingIds":["afid:1"],"attempts":[{"kind":"launch","state":"failed"}]}`)
		case "/api/v1/scenarios/demo/remediation/jobs/job-1/retry":
			fmt.Fprint(w, `{"id":"job-1","scenario":"demo","status":"running"}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)
	for _, args := range [][]string{{"remediate", "list", "demo"}, {"remediate", "show", "demo", "job-1"}, {"remediate", "retry", "demo", "job-1"}} {
		if err := app.Run(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if got, want := requests, []string{"GET /api/v1/scenarios/demo/remediation/jobs", "GET /api/v1/scenarios/demo/remediation/jobs/job-1", "POST /api/v1/scenarios/demo/remediation/jobs/job-1/retry"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("requests = %v, want %v", got, want)
	}
}

func TestRunTestsCommandSendsType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			fmt.Fprintf(w, `{"status":"ok","service":"test-genie"}`)
			return
		case "/api/v1/scenarios/demo/run-tests":
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		if !bytes.Contains(body, []byte(`"type":"phased"`)) {
			t.Fatalf("expected type in payload, got %s", string(body))
		}
		if !bytes.Contains(body, []byte(`"paths":["api/foo.go"]`)) {
			t.Fatalf("expected paths in payload, got %s", string(body))
		}
		if !bytes.Contains(body, []byte(`"playbooks":["bas/cases/run.json"]`)) {
			t.Fatalf("expected playbooks in payload, got %s", string(body))
		}
		if !bytes.Contains(body, []byte(`"filter":"UserTest"`)) {
			t.Fatalf("expected filter in payload, got %s", string(body))
		}
		fmt.Fprintf(w, `{"type":"phased","status":"ok","command":{"command":["echo"],"workingDir":"."}}`)
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"run-tests", "demo", "--type", "phased", "--path", "api/foo.go", "--playbook", "bas/cases/run.json", "--filter", "UserTest"}); err != nil {
		t.Fatalf("run-tests failed: %v", err)
	}
}

func TestStatusCommandRequestsHealth(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprintf(w, `{"status":"ok","service":"test-genie","version":"1.0","dependencies":{"db":"up"}}`)
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"status"}); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected two health calls, got %d", calls)
	}
}

func TestExecuteCommandHelpDoesNotCallAPI(t *testing.T) {
	app := newTestApp(t)
	output := captureStdout(t, func() {
		if err := app.Run([]string{"execute", "--help"}); err != nil {
			t.Fatalf("execute help failed: %v", err)
		}
	})
	if !strings.Contains(output, execute.UsageLine) {
		t.Fatalf("expected execute help output, got %s", output)
	}
}

func TestRemediateCommandHelpDoesNotCallAPI(t *testing.T) {
	app := newTestApp(t)
	output := captureStdout(t, func() {
		if err := app.Run([]string{"remediate", "create", "--help"}); err != nil {
			t.Fatalf("remediate help failed: %v", err)
		}
	})
	if !strings.Contains(output, remediate.UsageLine) {
		t.Fatalf("expected remediate help output, got %s", output)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	temp := t.TempDir()
	t.Setenv("HOME", temp)
	t.Setenv("TEST_GENIE_CONFIG_DIR", temp)
	t.Setenv("TEST_GENIE_API_TOKEN", "test-token")
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}
