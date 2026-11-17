package agent_manager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"connectrpc.com/connect"
	"path/filepath"
	"strings"
	"testing"

	vrun "development-toolchain-validator/internal/validation_run"

	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// TestResolveGoldenRoot pins the path-resolution contract that the golden
// seeding depends on: absolute paths pass through unchanged, relative
// paths become absolute (rooted at the repo), and empty is an error.
func TestResolveGoldenRoot(t *testing.T) {
	if _, err := resolveGoldenRoot(""); err == nil {
		t.Fatalf("empty golden path should error")
	}

	abs := filepath.Join(string(filepath.Separator), "tmp", "golden")
	got, err := resolveGoldenRoot(abs)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if got != abs {
		t.Fatalf("abs path = %q, want passthrough %q", got, abs)
	}

	// Relative paths resolve against the repo root. The test runs inside
	// the repo, so the cwd fallback locates it; we only assert the result
	// is absolute and ends with the requested relative path.
	rel := filepath.Join(".vrooli", "generated-goldens", "reference-react-vite")
	got, err = resolveGoldenRoot(rel)
	if err != nil {
		t.Fatalf("relative path: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("relative path did not resolve to absolute: %q", got)
	}
	if !strings.HasSuffix(got, rel) {
		t.Fatalf("resolved %q does not end with %q", got, rel)
	}
}

// TestBuildSkillPrompt verifies the agent prompt embeds the skill body
// when present and falls back to a generic description otherwise.
func TestBuildSkillPrompt(t *testing.T) {
	withContent := buildSkillPrompt(vrun.SandboxedRunSpec{
		SkillID:     "progress",
		GoldenSlug:  "reference-react-vite",
		SkillPrompt: "Advance the operational progress log.",
	})
	if !strings.Contains(withContent, "Advance the operational progress log.") {
		t.Fatalf("prompt missing skill body: %q", withContent)
	}
	if !strings.Contains(withContent, "=== BEGIN SKILL: progress ===") {
		t.Fatalf("prompt missing skill delimiter: %q", withContent)
	}
	if !strings.Contains(withContent, "empty diff is a valid") {
		t.Fatalf("prompt missing convergence framing: %q", withContent)
	}

	fallback := buildSkillPrompt(vrun.SandboxedRunSpec{
		SkillID:    "progress",
		GoldenSlug: "reference-react-vite",
	})
	if !strings.Contains(fallback, "progress") || !strings.Contains(fallback, "manifest evaluation") {
		t.Fatalf("fallback prompt unexpected: %q", fallback)
	}
}

// TestSha256Hex pins the three properties the diff_hash fix relies on:
// identical content hashes identically (so converged skills collapse to
// one hash), different content diverges, and the empty diff hashes to a
// stable constant ("ran, produced no changes").
func TestSha256Hex(t *testing.T) {
	const diffA = "diff --git a/x b/x\n+added\n"
	const diffB = "diff --git a/y b/y\n-removed\n"

	if got := sha256Hex(diffA); got != sha256Hex(diffA) {
		t.Fatalf("same content produced different hashes")
	}
	if sha256Hex(diffA) == sha256Hex(diffB) {
		t.Fatalf("different content collided: %q vs %q", diffA, diffB)
	}

	wantEmpty := func() string {
		sum := sha256.Sum256([]byte(""))
		return hex.EncodeToString(sum[:])
	}()
	if got := sha256Hex(""); got != wantEmpty {
		t.Fatalf("empty hash = %q, want %q", got, wantEmpty)
	}
	// Guard the literal so a future hash-algorithm swap is a deliberate
	// decision, not a silent break of "same diff → same record".
	if wantEmpty != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("empty-string sha256 changed unexpectedly: %q", wantEmpty)
	}
	if len(sha256Hex(diffA)) != 64 {
		t.Fatalf("hash is not 64 hex chars: %q", sha256Hex(diffA))
	}
}

// connectStub is served through the generated AgentManagerService Connect
// handler so these tests exercise the actual generated client wire contract.
type connectStub struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	t *testing.T
}

func (s *connectStub) ReconcileScenarioProfiles(_ context.Context, req *connect.Request[apipb.ReconcileScenarioProfilesRequest]) (*connect.Response[apipb.ReconcileScenarioProfilesResponse], error) {
	if req.Msg.Scenario != ScenarioName {
		s.t.Errorf("scenario = %q", req.Msg.Scenario)
	}
	return connect.NewResponse(&apipb.ReconcileScenarioProfilesResponse{Scenario: req.Msg.Scenario, Results: []*apipb.ProfileReconcileResult{{ProfileKey: ProfileKey}}, Created: 1}), nil
}
func (s *connectStub) CreateTask(_ context.Context, req *connect.Request[apipb.CreateTaskRequest]) (*connect.Response[apipb.CreateTaskResponse], error) {
	if req.Msg.Task == nil || !filepath.IsAbs(req.Msg.Task.ProjectRoot) || req.Msg.Task.ScopePath != "." {
		s.t.Errorf("unexpected task: %+v", req.Msg.Task)
	}
	return connect.NewResponse(&apipb.CreateTaskResponse{Task: &domainpb.Task{Id: "task-1"}}), nil
}
func (s *connectStub) CreateRun(_ context.Context, req *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
	if req.Msg.TaskId != "task-1" || req.Msg.GetProfileRef().GetProfileKey() != ProfileKey || !req.Msg.GetForce() {
		s.t.Errorf("unexpected run request: %+v", req.Msg)
	}
	return connect.NewResponse(&apipb.CreateRunResponse{Run: &domainpb.Run{Id: "run-1"}}), nil
}
func (s *connectStub) GetRun(_ context.Context, req *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	if req.Msg.RunId != "run-1" {
		s.t.Errorf("run id = %q", req.Msg.RunId)
	}
	return connect.NewResponse(&apipb.GetRunResponse{Run: &domainpb.Run{Id: req.Msg.RunId, Status: domainpb.RunStatus_RUN_STATUS_COMPLETE}}), nil
}
func (s *connectStub) GetRunDiff(_ context.Context, req *connect.Request[apipb.GetRunDiffRequest]) (*connect.Response[apipb.GetRunDiffResponse], error) {
	if req.Msg.RunId != "run-1" {
		s.t.Errorf("diff run id = %q", req.Msg.RunId)
	}
	return connect.NewResponse(&apipb.GetRunDiffResponse{Diff: &domainpb.RunDiff{RunId: req.Msg.RunId, Content: "diff --git a/foo b/foo\n", Files: []*domainpb.FileDiff{{Path: "foo"}}}}), nil
}

func newConnectClient(t *testing.T) *Client {
	t.Helper()
	stub := &connectStub{t: t}
	path, handler := apiconnect.NewAgentManagerServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return New(Options{Resolver: discovery.NewStaticResolver(srv.URL), PollInterval: time.Millisecond})
}

func TestGeneratedConnectClientCoversAgentManagerProcedures(t *testing.T) {
	c := newConnectClient(t)
	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	runID, err := c.StartSandboxedRun(context.Background(), vrun.SandboxedRunSpec{SkillID: "skill", GoldenSlug: "golden", GoldenPath: t.TempDir()})
	if err != nil {
		t.Fatalf("StartSandboxedRun: %v", err)
	}
	if runID != "run-1" {
		t.Fatalf("run id = %q", runID)
	}
	summary, err := c.WaitForTerminal(context.Background(), runID, time.Second)
	if err != nil {
		t.Fatalf("WaitForTerminal: %v", err)
	}
	if summary.DiffHash != sha256Hex("diff --git a/foo b/foo\n") {
		t.Fatalf("diff hash = %q", summary.DiffHash)
	}
	if len(summary.DiffPaths) != 1 || summary.DiffPaths[0].Path != "foo" {
		t.Fatalf("diff paths = %+v", summary.DiffPaths)
	}
}

func TestBuildSummaryNoDiffLeavesHashEmpty(t *testing.T) {
	// A successful Connect response with no diff is a valid missing diff.
	stub := &emptyDiffStub{connectStub: connectStub{t: t}}
	path, handler := apiconnect.NewAgentManagerServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(Options{Resolver: discovery.NewStaticResolver(srv.URL)})
	summary, err := c.buildSummary(context.Background(), &domainpb.Run{Id: "run-1"})
	if err != nil {
		t.Fatalf("buildSummary: %v", err)
	}
	if summary.DiffHash != "" {
		t.Fatalf("DiffHash = %q, want empty", summary.DiffHash)
	}
}

type emptyDiffStub struct{ connectStub }

func (s *emptyDiffStub) GetRunDiff(context.Context, *connect.Request[apipb.GetRunDiffRequest]) (*connect.Response[apipb.GetRunDiffResponse], error) {
	return connect.NewResponse(&apipb.GetRunDiffResponse{}), nil
}

type getRunErrorStub struct {
	connectStub
	err   error
	calls int
}

func (s *getRunErrorStub) GetRun(context.Context, *connect.Request[apipb.GetRunRequest]) (*connect.Response[apipb.GetRunResponse], error) {
	s.calls++
	return nil, s.err
}

func TestGeneratedConnectClientRetriesTransientErrorsOnly(t *testing.T) {
	for _, tc := range []struct {
		name            string
		code            connect.Code
		wantCalls       int
		wantUnavailable bool
	}{
		{name: "unavailable retries", code: connect.CodeUnavailable, wantCalls: 3, wantUnavailable: true},
		{name: "invalid argument does not retry", code: connect.CodeInvalidArgument, wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &getRunErrorStub{connectStub: connectStub{t: t}, err: connect.NewError(tc.code, errors.New("stub failure"))}
			path, handler := apiconnect.NewAgentManagerServiceHandler(stub)
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			srv := httptest.NewServer(mux)
			defer srv.Close()
			c := New(Options{Resolver: discovery.NewStaticResolver(srv.URL), MaxAttempts: 3})
			_, err := c.getRun(context.Background(), "run")
			if err == nil {
				t.Fatal("expected error")
			}
			if stub.calls != tc.wantCalls {
				t.Fatalf("calls = %d, want %d", stub.calls, tc.wantCalls)
			}
			var unavailable vrun.ErrDependencyUnavailable
			if got := errors.As(err, &unavailable); got != tc.wantUnavailable {
				t.Fatalf("dependency unavailable = %v, want %v (err %v)", got, tc.wantUnavailable, err)
			}
		})
	}
}

type approvalPendingStub struct {
	connectStub
	calls int
}

func (s *approvalPendingStub) CreateRun(context.Context, *connect.Request[apipb.CreateRunRequest]) (*connect.Response[apipb.CreateRunResponse], error) {
	s.calls++
	err := connect.NewError(connect.CodeFailedPrecondition, errors.New("approval is pending"))
	detail, detailErr := connect.NewErrorDetail(&apipb.AuthorizationApprovalPending{Subject: "development-toolchain-validator", Detail: "operator approval pending"})
	if detailErr != nil {
		return nil, detailErr
	}
	err.AddDetail(detail)
	return nil, err
}

func TestCreateRunReturnsTypedApprovalPendingOutcomeWithoutRetry(t *testing.T) {
	stub := &approvalPendingStub{connectStub: connectStub{t: t}}
	path, handler := apiconnect.NewAgentManagerServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := New(Options{Resolver: discovery.NewStaticResolver(srv.URL), MaxAttempts: 3})
	_, err := c.createRun(context.Background(), &apipb.CreateRunRequest{})
	var pending *vrun.AuthorizationApprovalPendingError
	if !errors.As(err, &pending) {
		t.Fatalf("error = %T %v, want typed approval-pending outcome", err, err)
	}
	if pending.Subject != "development-toolchain-validator" || pending.Detail != "operator approval pending" {
		t.Fatalf("pending = %+v", pending)
	}
	if stub.calls != 1 {
		t.Fatalf("CreateRun calls = %d, want 1", stub.calls)
	}
	var unavailable vrun.ErrDependencyUnavailable
	if errors.As(err, &unavailable) {
		t.Fatalf("approval outcome misclassified as dependency unavailable: %v", err)
	}
}
