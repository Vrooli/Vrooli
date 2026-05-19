package validation_run_test

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	vr "development-toolchain-validator/internal/validation_record"
	vrmocks "development-toolchain-validator/internal/validation_record/mocks"
	vrun "development-toolchain-validator/internal/validation_run"
	vrunmocks "development-toolchain-validator/internal/validation_run/mocks"
	"development-toolchain-validator/internal/testutil/db"
	"development-toolchain-validator/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "development-toolchain-validator/internal/database"
)

func newWorkerDeps(t *testing.T) (vrun.WorkerDeps, vrun.Repository, *vrmocks.FakeRepository) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(vrun.Schema),
	))
	repo := vrun.NewSQLiteRepository(d)
	vrRepo := vrmocks.NewFakeRepository()
	clk := mocks.NewFakeClock(time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC))
	return vrun.WorkerDeps{
		Repo:      repo,
		Records:   vr.NewService(vrRepo, clk),
		AgentMgr:  &vrunmocks.FakeAgentManager{},
		Tools:     &vrunmocks.FakeToolRunner{},
		Sandbox:   &vrunmocks.FakeWorkspaceSandbox{},
		Goldens:   &vrunmocks.FakeGoldenSource{Paths: map[string]string{"ref": "/tmp/ref"}},
		Manifests: &vrunmocks.FakeManifestSource{Manifests: map[[2]string]manifest.Manifest{}},
		Clock:     clk,
		Logger:    log.New(io.Discard, "", 0),
	}, repo, vrRepo
}

func TestWorker_SkillRun_HappyPath(t *testing.T) {
	deps, repo, vrRepo := newWorkerDeps(t)
	deps.AgentMgr = &vrunmocks.FakeAgentManager{
		StartRunID: "amr-1",
		WaitResult: vrun.RunSummary{TokensUsed: 100, CostUSDMicro: 200},
	}
	deps.Manifests = &vrunmocks.FakeManifestSource{Manifests: map[[2]string]manifest.Manifest{
		{"plan-skill", "ref"}: {SkillID: "plan-skill", GoldenSlug: "ref", WildcardAllowed: true},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Queue a run, then process directly via ClaimNextQueued path.
	clk := deps.Clock
	r := vrun.Run{
		ID: "r-skill", TupleKind: vr.TupleKindSkill, SubjectID: "plan-skill", GoldenSlug: "ref",
		Status: vrun.StatusQueued, CreatedAt: clk.Now(),
	}
	require.NoError(t, repo.Create(ctx, r))

	w := vrun.NewWorker(deps, vrun.WorkerConfig{PollInterval: 50 * time.Millisecond})
	go w.Run(ctx)
	w.Notify()

	waitFor(t, func() bool {
		got, err := repo.Get(context.Background(), "r-skill")
		return err == nil && got.Status == vrun.StatusTerminal
	}, 3*time.Second)

	got, err := repo.Get(context.Background(), "r-skill")
	require.NoError(t, err)
	require.Equal(t, vr.VerdictPass, got.TerminalVerdict)
	require.Equal(t, "amr-1", got.AgentManagerRunID)

	// A ValidationRecord was appended.
	res, err := deps.Records.List(context.Background(), vr.ListFilter{}, 10, "")
	require.NoError(t, err)
	require.Len(t, res.Records, 1)
	require.Equal(t, vr.VerdictPass, res.Records[0].Verdict)
	_ = vrRepo // silence unused for tests that don't peek
}

func TestWorker_SkillRun_UnexpectedMutation(t *testing.T) {
	deps, repo, _ := newWorkerDeps(t)
	deps.AgentMgr = &vrunmocks.FakeAgentManager{
		StartRunID: "amr-1",
		WaitResult: vrun.RunSummary{
			DiffPaths: []manifest.DiffFile{{Path: "secrets/x"}},
		},
	}
	deps.Manifests = &vrunmocks.FakeManifestSource{Manifests: map[[2]string]manifest.Manifest{
		{"plan-skill", "ref"}: {SkillID: "plan-skill", GoldenSlug: "ref", AllowedPaths: []string{"src/**"}},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clk := deps.Clock
	require.NoError(t, repo.Create(ctx, vrun.Run{
		ID: "r-mut", TupleKind: vr.TupleKindSkill, SubjectID: "plan-skill", GoldenSlug: "ref",
		Status: vrun.StatusQueued, CreatedAt: clk.Now(),
	}))

	w := vrun.NewWorker(deps, vrun.WorkerConfig{PollInterval: 50 * time.Millisecond})
	go w.Run(ctx)
	w.Notify()

	waitFor(t, func() bool {
		got, err := repo.Get(context.Background(), "r-mut")
		return err == nil && got.Status == vrun.StatusTerminal
	}, 3*time.Second)
	got, err := repo.Get(context.Background(), "r-mut")
	require.NoError(t, err)
	require.Equal(t, vr.VerdictUnexpectedMutation, got.TerminalVerdict)
}

func TestWorker_SkillRun_AgentManagerUnavailable(t *testing.T) {
	deps, repo, _ := newWorkerDeps(t)
	deps.AgentMgr = &vrunmocks.FakeAgentManager{
		StartErr: vrun.ErrDependencyUnavailable{Dependency: "agent-manager", Reason: "scenario not running"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clk := deps.Clock
	require.NoError(t, repo.Create(ctx, vrun.Run{
		ID: "r-down", TupleKind: vr.TupleKindSkill, SubjectID: "plan-skill", GoldenSlug: "ref",
		Status: vrun.StatusQueued, CreatedAt: clk.Now(),
	}))

	w := vrun.NewWorker(deps, vrun.WorkerConfig{PollInterval: 50 * time.Millisecond})
	go w.Run(ctx)
	w.Notify()

	waitFor(t, func() bool {
		got, err := repo.Get(context.Background(), "r-down")
		return err == nil && got.Status == vrun.StatusTerminal
	}, 3*time.Second)
	got, err := repo.Get(context.Background(), "r-down")
	require.NoError(t, err)
	require.Equal(t, vr.VerdictRunFailure, got.TerminalVerdict)
	require.Contains(t, got.ErrorMessage, "scenario not running")
}

func TestWorker_ToolRun_HappyPath(t *testing.T) {
	deps, repo, _ := newWorkerDeps(t)
	now := deps.Clock.Now()
	deps.Tools = &vrunmocks.FakeToolRunner{Result: vrun.ToolResult{
		Succeeded: true, StartedAt: now, EndedAt: now.Add(time.Second),
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, repo.Create(ctx, vrun.Run{
		ID: "r-tool", TupleKind: vr.TupleKindTool, SubjectID: "scenario-auditor", GoldenSlug: "ref",
		Status: vrun.StatusQueued, CreatedAt: now,
	}))

	w := vrun.NewWorker(deps, vrun.WorkerConfig{PollInterval: 50 * time.Millisecond})
	go w.Run(ctx)
	w.Notify()

	waitFor(t, func() bool {
		got, err := repo.Get(context.Background(), "r-tool")
		return err == nil && got.Status == vrun.StatusTerminal
	}, 3*time.Second)
	got, err := repo.Get(context.Background(), "r-tool")
	require.NoError(t, err)
	require.Equal(t, vr.VerdictPass, got.TerminalVerdict)
}

func TestWorker_ToolRun_Failure(t *testing.T) {
	deps, repo, _ := newWorkerDeps(t)
	deps.Tools = &vrunmocks.FakeToolRunner{Err: errors.New("binary exited 1")}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	require.NoError(t, repo.Create(ctx, vrun.Run{
		ID: "r-toolfail", TupleKind: vr.TupleKindTool, SubjectID: "scenario-auditor", GoldenSlug: "ref",
		Status: vrun.StatusQueued, CreatedAt: deps.Clock.Now(),
	}))

	w := vrun.NewWorker(deps, vrun.WorkerConfig{PollInterval: 50 * time.Millisecond})
	go w.Run(ctx)
	w.Notify()

	waitFor(t, func() bool {
		got, err := repo.Get(context.Background(), "r-toolfail")
		return err == nil && got.Status == vrun.StatusTerminal
	}, 3*time.Second)
	got, err := repo.Get(context.Background(), "r-toolfail")
	require.NoError(t, err)
	require.Equal(t, vr.VerdictToolFailure, got.TerminalVerdict)
}

// waitFor polls cond until it returns true or timeout elapses. Fails
// the test if the deadline expires.
func waitFor(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("waitFor: condition did not become true within %s", timeout)
}
