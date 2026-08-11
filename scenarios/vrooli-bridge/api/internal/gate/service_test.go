package gate_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"vrooli-bridge/internal/gate"
	"vrooli-bridge/internal/gate/mocks"
	testmocks "vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

func newService(nodes *mocks.FakeNodeLister, presence *mocks.FakePresence, runner *mocks.FakeRunner) (gate.Service, *mocks.FakeRepository) {
	repo := mocks.NewFakeRepository()
	clk := testmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	return gate.NewService(repo, nodes, presence, runner, clk), repo
}

// threeOSFleet is one online, compatible node per OS.
func threeOSFleet() (*mocks.FakeNodeLister, *mocks.FakePresence) {
	nodes := &mocks.FakeNodeLister{Nodes: []gate.NodeRef{
		{ID: "ubuntu-1", OS: "linux", Arch: "amd64"},
		{ID: "mac-1", OS: "darwin", Arch: "arm64"},
		{ID: "win-1", OS: "windows", Arch: "amd64"},
	}}
	presence := &mocks.FakePresence{Online: map[string]bool{"ubuntu-1": true, "mac-1": true, "win-1": true}}
	return nodes, presence
}

func runInput() gate.RunInput {
	return gate.RunInput{
		Actor:          "owner",
		Scenario:       "web-search",
		TargetRevision: "a1b2c3d",
		TargetOSes:     []string{"linux", "darwin", "windows"},
	}
}

// [REQ:BRG-P1-002] RunGate selects one node per target OS and dispatches a
// validation run to each, recording a durable gate that starts PENDING.
func TestRun_FansOutOnePerOS(t *testing.T) {
	nodes, presence := threeOSFleet()
	runner := mocks.NewFakeRunner()
	svc, repo := newService(nodes, presence, runner)

	dec, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)
	require.NotEmpty(t, dec.GateID)
	require.Equal(t, gate.VerdictPending, dec.Verdict)
	require.Len(t, dec.Results, 3)
	require.Equal(t, []string{"mac-1", "ubuntu-1", "win-1"}, runner.DispatchedNodes())

	// The default validation verb is dispatched.
	require.Equal(t, gate.DefaultVerb, runner.Dispatched[0].Verb)

	g, results, err := svc.GetGate(context.Background(), dec.GateID)
	require.NoError(t, err)
	require.Equal(t, 3, g.TotalTargets)
	require.Equal(t, 3, g.Pending)
	for _, r := range results {
		require.Equal(t, gate.OSDispositionPending, r.Disposition)
		require.NotEmpty(t, r.RunID)
	}
	_ = repo
}

// [REQ:BRG-P1-002] A target OS with no eligible node fails the gate — the
// scenario can never be validated on that OS.
func TestRun_NoNodeForOSFailsGate(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []gate.NodeRef{{ID: "ubuntu-1", OS: "linux"}}}
	presence := &mocks.FakePresence{Online: map[string]bool{"ubuntu-1": true}}
	runner := mocks.NewFakeRunner()
	svc, _ := newService(nodes, presence, runner)

	dec, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)
	require.Equal(t, gate.VerdictFailed, dec.Verdict)

	byOS := map[string]gate.OSDisposition{}
	for _, r := range dec.Results {
		byOS[r.OS] = r.Disposition
	}
	require.Equal(t, gate.OSDispositionPending, byOS["linux"])
	require.Equal(t, gate.OSDispositionNoNode, byOS["darwin"])
	require.Equal(t, gate.OSDispositionNoNode, byOS["windows"])
}

// An offline or protocol-incompatible node is NOT an eligible validation target.
func TestRun_OfflineAndFlaggedNodesSkipped(t *testing.T) {
	nodes, presence := threeOSFleet()
	presence.Online["mac-1"] = false                  // offline
	presence.Flagged = map[string]bool{"win-1": true} // protocol drift
	runner := mocks.NewFakeRunner()
	svc, _ := newService(nodes, presence, runner)

	dec, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)
	require.Equal(t, gate.VerdictFailed, dec.Verdict)
	require.Equal(t, []string{"ubuntu-1"}, runner.DispatchedNodes(), "only the eligible OS was dispatched")
}

// A revoked node is excluded even when online.
func TestRun_RevokedNodeExcluded(t *testing.T) {
	nodes := &mocks.FakeNodeLister{Nodes: []gate.NodeRef{{ID: "ubuntu-1", OS: "linux", Revoked: true}}}
	presence := &mocks.FakePresence{Online: map[string]bool{"ubuntu-1": true}}
	runner := mocks.NewFakeRunner()
	svc, _ := newService(nodes, presence, runner)

	dec, err := svc.Run(context.Background(), gate.RunInput{
		Actor: "owner", Scenario: "web-search", TargetRevision: "rev", TargetOSes: []string{"linux"},
	})
	require.NoError(t, err)
	require.Equal(t, gate.VerdictFailed, dec.Verdict)
	require.Equal(t, gate.OSDispositionNoNode, dec.Results[0].Disposition)
	require.Empty(t, runner.DispatchedNodes())
}

// [REQ:BRG-P1-002] A dry-run selects + classifies every OS but creates no gate
// and dispatches nothing.
func TestRun_DryRunShortCircuits(t *testing.T) {
	nodes, presence := threeOSFleet()
	runner := mocks.NewFakeRunner()
	svc, repo := newService(nodes, presence, runner)

	in := runInput()
	in.DryRun = true
	dec, err := svc.Run(context.Background(), in)
	require.NoError(t, err)
	require.True(t, dec.DryRun)
	require.Empty(t, dec.GateID)
	require.Len(t, dec.Results, 3)
	require.Empty(t, runner.DispatchedNodes(), "nothing dispatched on a dry-run")

	list, _ := repo.List(context.Background(), gate.ListFilter{})
	require.Empty(t, list, "no gate persisted on a dry-run")
}

// A dispatch rejection (e.g. disallowed verb surfaced by the dispatch domain)
// marks that OS dispatch-failed and fails the gate.
func TestRun_DispatchRejectionFailsOS(t *testing.T) {
	nodes, presence := threeOSFleet()
	runner := mocks.NewFakeRunner()
	runner.FailDispatch["win-1"] = errors.New("verb \"deploy\" not permitted")
	svc, _ := newService(nodes, presence, runner)

	dec, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)
	require.Equal(t, gate.VerdictFailed, dec.Verdict)

	var win gate.OSResult
	for _, r := range dec.Results {
		if r.OS == "windows" {
			win = r
		}
	}
	require.Equal(t, gate.OSDispositionDispatchFailed, win.Disposition)
	require.Contains(t, win.Detail, "deploy")
}

// [REQ:BRG-P1-002] GetGate recomputes the live verdict from the per-OS runs: once
// every run is terminal-green the gate reads PASSED without re-dispatching.
func TestGetGate_RecomputesLiveVerdict(t *testing.T) {
	nodes, presence := threeOSFleet()
	runner := mocks.NewFakeRunner()
	svc, _ := newService(nodes, presence, runner)

	dec, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)

	// All three runs finish green.
	for _, r := range dec.Results {
		runner.SetVerdict(r.RunID, gate.RunVerdict{Terminal: true, Passed: true})
	}

	g, results, err := svc.GetGate(context.Background(), dec.GateID)
	require.NoError(t, err)
	require.Equal(t, gate.VerdictPassed, g.Verdict)
	require.Equal(t, 3, g.Passed)
	for _, r := range results {
		require.Equal(t, gate.OSDispositionPassed, r.Disposition)
	}
}

// [REQ:BRG-P1-002] One OS failing flips the live verdict to FAILED with the
// offending run's exit code + logs surfaced.
func TestGetGate_OneFailingOSFailsGate(t *testing.T) {
	nodes, presence := threeOSFleet()
	runner := mocks.NewFakeRunner()
	svc, _ := newService(nodes, presence, runner)

	dec, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)

	var failingRun string
	for _, r := range dec.Results {
		if r.OS == "windows" {
			failingRun = r.RunID
			runner.SetVerdict(r.RunID, gate.RunVerdict{Terminal: true, Passed: false, ExitCode: 1, Detail: "3 tests failed"})
		} else {
			runner.SetVerdict(r.RunID, gate.RunVerdict{Terminal: true, Passed: true})
		}
	}
	require.NotEmpty(t, failingRun)

	g, results, err := svc.GetGate(context.Background(), dec.GateID)
	require.NoError(t, err)
	require.Equal(t, gate.VerdictFailed, g.Verdict)

	var win gate.OSResult
	for _, r := range results {
		if r.OS == "windows" {
			win = r
		}
	}
	require.Equal(t, gate.OSDispositionFailed, win.Disposition)
	require.Equal(t, int32(1), win.ExitCode)
	require.Contains(t, win.Detail, "tests failed")
	require.Equal(t, failingRun, win.RunID, "the offending run id is surfaced for log drill-in")
}

// [REQ:BRG-P1-002] WaitGate blocks once and returns the final aggregate verdict
// when every target run is terminal.
func TestWaitGate_BlocksToFinalVerdict(t *testing.T) {
	nodes, presence := threeOSFleet()
	runner := mocks.NewFakeRunner()
	svc, _ := newService(nodes, presence, runner)

	dec, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)
	for _, r := range dec.Results {
		runner.SetVerdict(r.RunID, gate.RunVerdict{Terminal: true, Passed: true})
	}

	g, _, timedOut, err := svc.WaitGate(context.Background(), dec.GateID, time.Minute)
	require.NoError(t, err)
	require.False(t, timedOut)
	require.Equal(t, gate.VerdictPassed, g.Verdict)
}

// WaitGate reports timed_out=true when a target run never settles.
func TestWaitGate_TimesOutWhilePending(t *testing.T) {
	nodes, presence := threeOSFleet()
	runner := mocks.NewFakeRunner()
	svc, _ := newService(nodes, presence, runner)

	dec, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)
	// Only two of three settle; the third stays non-terminal.
	settled := 0
	for _, r := range dec.Results {
		if settled < 2 {
			runner.SetVerdict(r.RunID, gate.RunVerdict{Terminal: true, Passed: true})
			settled++
		}
	}

	g, _, timedOut, err := svc.WaitGate(context.Background(), dec.GateID, time.Minute)
	require.NoError(t, err)
	require.True(t, timedOut)
	require.Equal(t, gate.VerdictPending, g.Verdict)
}

// Structural validation: scenario, target_revision, and at least one OS are
// required.
func TestRun_ValidatesRequiredFields(t *testing.T) {
	nodes, presence := threeOSFleet()
	runner := mocks.NewFakeRunner()
	svc, _ := newService(nodes, presence, runner)

	_, err := svc.Run(context.Background(), gate.RunInput{Scenario: "", TargetRevision: "r", TargetOSes: []string{"linux"}})
	require.ErrorAs(t, err, &gate.ErrInvalidGate{})

	_, err = svc.Run(context.Background(), gate.RunInput{Scenario: "s", TargetRevision: "", TargetOSes: []string{"linux"}})
	require.ErrorAs(t, err, &gate.ErrInvalidGate{})

	_, err = svc.Run(context.Background(), gate.RunInput{Scenario: "s", TargetRevision: "r", TargetOSes: nil})
	require.ErrorAs(t, err, &gate.ErrInvalidGate{})
}

// GetGate on an unknown id returns the typed not-found sentinel.
func TestGetGate_NotFound(t *testing.T) {
	nodes, presence := threeOSFleet()
	svc, _ := newService(nodes, presence, mocks.NewFakeRunner())
	_, _, err := svc.GetGate(context.Background(), "nope")
	require.ErrorAs(t, err, &gate.ErrGateNotFound{})
}

// ListGates returns gates newest-first.
func TestListGates_NewestFirst(t *testing.T) {
	nodes, presence := threeOSFleet()
	runner := mocks.NewFakeRunner()
	svc, _ := newService(nodes, presence, runner)

	first, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)
	second, err := svc.Run(context.Background(), runInput())
	require.NoError(t, err)
	for _, r := range second.Results {
		runner.SetVerdict(r.RunID, gate.RunVerdict{Terminal: true, Passed: true})
	}

	list, err := svc.ListGates(context.Background(), gate.ListFilter{})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, second.GateID, list[0].ID)
	require.Equal(t, first.GateID, list[1].ID)
	require.Equal(t, gate.VerdictPassed, list[0].Verdict, "list must expose the live verdict, not the persisted pending snapshot")
	require.Equal(t, 3, list[0].Passed)
	require.Equal(t, gate.VerdictPending, list[1].Verdict)
}
