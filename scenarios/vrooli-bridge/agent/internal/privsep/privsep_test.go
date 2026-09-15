package privsep_test

import (
	"context"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"vrooli-bridge/agent/internal/privsep"

	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
)

// fakeReporter collects the ProvisionEvents the helper emits.
type fakeReporter struct {
	mu     sync.Mutex
	events []*provisionv1.ProvisionEvent
}

func (f *fakeReporter) Report(_ context.Context, ev *provisionv1.ProvisionEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, &provisionv1.ProvisionEvent{
		OpId: ev.OpId, Kind: ev.Kind, Sequence: ev.Sequence,
		LogChunk: ev.LogChunk, Status: ev.Status, Revision: ev.Revision, ExitCode: ev.ExitCode,
	})
	return nil
}

func (f *fakeReporter) terminal() *provisionv1.ProvisionEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.events) == 0 {
		return nil
	}
	return f.events[len(f.events)-1]
}

func (f *fakeReporter) lastVersion() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	v := ""
	for _, e := range f.events {
		if e.Kind == provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_VERSION {
			v = e.Revision
		}
	}
	return v
}

// fakeStep records the argvs it ran and returns canned exit codes. failOn names
// a token (e.g. "setup") that makes a step fail. failOnce limits the failure to
// the FIRST matching step (modelling a target setup that fails while the
// subsequent rollback setup succeeds — the rollback revision is known-good).
type fakeStep struct {
	mu       sync.Mutex
	argvs    [][]string
	failOn   string
	failOnce bool
	failed   bool
}

func (f *fakeStep) Run(_ context.Context, argv []string, _ string, _ func(string)) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.argvs = append(f.argvs, argv)
	if f.failOn != "" {
		for _, tok := range argv {
			if tok == f.failOn {
				if f.failOnce && f.failed {
					return 0, nil
				}
				f.failed = true
				return 2, nil
			}
		}
	}
	return 0, nil
}

func (f *fakeStep) ran() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]string, len(f.argvs))
	copy(out, f.argvs)
	return out
}

// fakeRevision returns a scripted current revision. It can switch its answer
// after a checkout argv is observed, so a rollback's resulting HEAD differs.
type fakeRevision struct {
	current string
}

func (f *fakeRevision) Current(_ context.Context, _ string) (string, error) { return f.current, nil }

func fixedClock() time.Time { return time.Unix(0, 0).UTC() }

// [REQ:BRG-P0-006] Steps builds a typed, ordered argv plan — fetch, checkout,
// setup — and is the no-shell-path proof: it returns [][]string, never a shell
// string.
func TestSteps_TypedPlan(t *testing.T) {
	steps, err := privsep.Steps("git", "vrooli", "a1b2c3d")
	require.NoError(t, err)
	require.Equal(t, [][]string{
		{"git", "fetch", "--all", "--tags"},
		{"git", "checkout", "a1b2c3d"},
		{"vrooli", "setup"},
	}, steps)
}

// [REQ:BRG-P0-006] A revision token carrying a shell metacharacter is rejected
// before any step runs — a smuggled shell construct can never reach the helper.
func TestSteps_RejectsUnsafeRevision(t *testing.T) {
	for _, bad := range []string{"a; rm -rf /", "$(whoami)", "a && b", "a`id`", "a b"} {
		_, err := privsep.Steps("git", "vrooli", bad)
		require.Error(t, err, "revision %q must be rejected", bad)
	}
	_, err := privsep.Steps("git", "vrooli", "   ")
	require.Error(t, err, "empty revision rejected")
}

// [REQ:BRG-P0-006] A successful provision runs fetch→checkout→setup, reports the
// resulting version, and ends with a clean EXIT(0).
func TestProvision_Success(t *testing.T) {
	step := &fakeStep{}
	rep := &fakeReporter{}
	h := privsep.NewHelper("vrooli", "/work", rep,
		privsep.WithStepRunner(step),
		privsep.WithRevisionResolver(&fakeRevision{current: "rev-B"}),
		privsep.WithClock(fixedClock),
	)
	require.NoError(t, h.Provision(context.Background(), &channelv1.ProvisionCommand{
		OpId: "op-1", TargetRevision: "rev-B",
	}))

	ran := step.ran()
	require.Equal(t, [][]string{
		{"git", "fetch", "--all", "--tags"},
		{"git", "checkout", "rev-B"},
		{"vrooli", "setup"},
	}, ran, "the helper runs a typed argv plan, never a shell string")

	require.Equal(t, "rev-B", rep.lastVersion())
	term := rep.terminal()
	require.Equal(t, provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT, term.Kind)
	require.Equal(t, int32(0), term.ExitCode)
}

// [REQ:BRG-P0-006] Idempotent re-provision: running the same target twice yields
// the same command plan and the same end state (the node stays at the target).
// `vrooli setup` is itself idempotent, so re-running converges.
func TestProvision_IdempotentReRun(t *testing.T) {
	run := func() [][]string {
		step := &fakeStep{}
		rep := &fakeReporter{}
		h := privsep.NewHelper("vrooli", "/work", rep,
			privsep.WithStepRunner(step),
			privsep.WithRevisionResolver(&fakeRevision{current: "rev-B"}),
			privsep.WithClock(fixedClock),
		)
		require.NoError(t, h.Provision(context.Background(), &channelv1.ProvisionCommand{OpId: "op", TargetRevision: "rev-B"}))
		require.Equal(t, "rev-B", rep.lastVersion())
		require.Equal(t, int32(0), rep.terminal().ExitCode)
		return step.ran()
	}
	first := run()
	second := run()
	require.Equal(t, first, second, "re-provisioning to the same revision runs the same converging plan")
}

// [REQ:BRG-P0-006] Rollback on failed setup: when `vrooli setup` fails and a
// rollback revision is available, the helper checks out the rollback revision,
// re-runs setup, reports the rollback revision as the resulting version, and
// exits non-zero (the control plane records ROLLED_BACK).
func TestProvision_RollbackOnFailedSetup(t *testing.T) {
	step := &fakeStep{failOn: "setup", failOnce: true}
	rep := &fakeReporter{}
	// After the (failed) target setup and a rollback checkout, HEAD is rev-A.
	h := privsep.NewHelper("vrooli", "/work", rep,
		privsep.WithStepRunner(step),
		privsep.WithRevisionResolver(&fakeRevision{current: "rev-A"}),
		privsep.WithClock(fixedClock),
	)
	require.NoError(t, h.Provision(context.Background(), &channelv1.ProvisionCommand{
		OpId: "op-1", TargetRevision: "rev-B", RollbackRevision: "rev-A",
	}))

	ran := step.ran()
	// Expect the target plan (fetch, checkout rev-B, setup [fails]) then the
	// rollback plan (fetch, checkout rev-A, setup).
	require.Contains(t, flatten(ran), "git checkout rev-A", "the helper checked out the rollback revision")
	require.Equal(t, "rev-A", rep.lastVersion(), "the node is reported back on the rollback revision")
	term := rep.terminal()
	require.Equal(t, provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT, term.Kind)
	require.NotEqual(t, int32(0), term.ExitCode, "a rolled-back op exits non-zero")
}

// [REQ:BRG-P0-006] A first provision (no rollback revision) that fails setup is a
// DEGRADED failure: there is nothing to roll back to, so the helper does not
// attempt a rollback and exits non-zero.
func TestProvision_DegradedFailureNoRollback(t *testing.T) {
	step := &fakeStep{failOn: "setup"}
	rep := &fakeReporter{}
	h := privsep.NewHelper("vrooli", "/work", rep,
		privsep.WithStepRunner(step),
		privsep.WithRevisionResolver(&fakeRevision{current: "rev-B"}),
		privsep.WithClock(fixedClock),
	)
	require.NoError(t, h.Provision(context.Background(), &channelv1.ProvisionCommand{
		OpId: "op-1", TargetRevision: "rev-B",
	}))

	require.NotContains(t, flatten(step.ran()), "git checkout rev-A", "no rollback is attempted with no rollback revision")
	require.NotEqual(t, int32(0), rep.terminal().ExitCode)
}

// [REQ:BRG-P0-006] PRIVILEGE SEPARATION (structural): the non-privileged job
// runner package (internal/exec) does NOT import the privileged provisioning
// helper (internal/privsep), and vice-versa. Neither can call into the other —
// only the channel's ProvisionCommand handler invokes privsep — so a runner job
// has no in-process path to escalate to provisioning.
func TestPrivilegeSeparation_NoCrossImport(t *testing.T) {
	assertNoImport(t, "../exec", "vrooli-bridge/agent/internal/privsep")
	assertNoImport(t, "../privsep", "vrooli-bridge/agent/internal/exec")
}

// assertNoImport parses every non-test .go file in dir and fails if any imports
// banned.
func assertNoImport(t *testing.T, dir, banned string) {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ImportsOnly)
	require.NoError(t, err)
	for _, pkg := range pkgs {
		for path, file := range pkg.Files {
			for _, imp := range file.Imports {
				require.NotContains(t, imp.Path.Value, banned,
					"%s imports %s — the two trust tiers must not share an in-process call path", path, banned)
			}
		}
	}
}

func flatten(argvs [][]string) []string {
	out := make([]string, 0, len(argvs))
	for _, a := range argvs {
		out = append(out, strings.Join(a, " "))
	}
	return out
}
