package research_test

import (
	"context"
	"strings"
	"testing"

	"web-search/internal/findings"
	"web-search/internal/research"
	"web-search/internal/research/agentmanager"

	"github.com/stretchr/testify/require"
)

// TestRunL3SpawnsAndPolls asserts RunL3 spawns an agent-manager run (returning
// the handle) and GetResearchStatus polls it. The spawned task prompt must
// encode the GATHER -> RESEARCH -> RECONCILE loop with the gather step FIRST.
func TestRunL3SpawnsAndPolls(t *testing.T) {
	ctx := context.Background()
	am := &fakeAgentManager{
		spawnResult: agentmanager.RunResult{RunID: "run-1", TaskID: "task-1", Status: "pending"},
		stateByID: map[string]agentmanager.RunState{
			"run-1": {RunID: "run-1", Status: "complete", Summary: "done"},
		},
	}
	svc := research.NewService(research.Deps{AgentManager: am})

	res, err := svc.RunL3(ctx, "what changed about X")
	require.NoError(t, err)
	require.Equal(t, "run-1", res.RunID)
	require.Equal(t, "pending", res.Status)

	// The L3 prompt orders GATHER before RECONCILE (gather-before-reconcile).
	prompt := am.spawnedReq.Prompt
	gatherIdx := strings.Index(prompt, "GATHER")
	researchIdx := strings.Index(prompt, "RESEARCH")
	reconcileIdx := strings.Index(prompt, "RECONCILE")
	require.Greater(t, gatherIdx, -1)
	require.Greater(t, researchIdx, gatherIdx, "research the gap comes after gather")
	require.Greater(t, reconcileIdx, researchIdx, "reconcile is the bounded post-step, last")

	state, err := svc.GetResearchStatus(ctx, "run-1")
	require.NoError(t, err)
	require.Equal(t, "complete", state.Status)
	require.Equal(t, "done", state.Summary)
}

// TestRunL3UnavailableWithoutAgentManager asserts L3 surfaces the upstream
// unavailability rather than panicking when agent-manager is not wired.
func TestRunL3UnavailableWithoutAgentManager(t *testing.T) {
	svc := research.NewService(research.Deps{})
	_, err := svc.RunL3(context.Background(), "q")
	require.Error(t, err)
	_, err = svc.GetResearchStatus(context.Background(), "run-1")
	require.Error(t, err)
}

// TestRunL3PropagatesSpawnError asserts a spawn failure (e.g. agent-manager
// down) is propagated, not swallowed.
func TestRunL3PropagatesSpawnError(t *testing.T) {
	am := &fakeAgentManager{spawnErr: agentmanager.ErrNotAvailable}
	svc := research.NewService(research.Deps{AgentManager: am})
	_, err := svc.RunL3(context.Background(), "q")
	require.ErrorIs(t, err, agentmanager.ErrNotAvailable)
}

// TestReconcileConfidenceGate is the core Phase 8 contract: a HIGH-confidence
// contradiction SUPERSEDES the outdated finding; a LOW-confidence contradiction
// FLAGS it into DISPUTED rather than silently overwriting; a non-contradiction
// is left untouched.
func TestReconcileConfidenceGate(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)
	svc := research.NewService(research.Deps{Findings: fsvc})

	// Three existing findings to reconcile against.
	high, err := fsvc.Add(ctx, findings.NewFinding{Claim: "outdated, strongly refuted", Confidence: 0.5})
	require.NoError(t, err)
	low, err := fsvc.Add(ctx, findings.NewFinding{Claim: "weakly contested", Confidence: 0.5})
	require.NoError(t, err)
	untouched, err := fsvc.Add(ctx, findings.NewFinding{Claim: "still reinforced", Confidence: 0.5})
	require.NoError(t, err)
	replacement, err := fsvc.Add(ctx, findings.NewFinding{Claim: "the new truth", Confidence: 0.95})
	require.NoError(t, err)

	results, err := svc.Reconcile(ctx, []research.ReconcileItem{
		{ExistingID: high.ID, Confidence: 0.9, Contradicts: true, ReplacementID: replacement.ID, Reason: "strong new evidence"},
		{ExistingID: low.ID, Confidence: 0.4, Contradicts: true, Reason: "weak conflicting source"},
		{ExistingID: untouched.ID, Confidence: 0.99, Contradicts: false},
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []research.ReconcileResult{
		{ExistingID: high.ID, Action: research.ActionSupersede},
		{ExistingID: low.ID, Action: research.ActionFlag},
		{ExistingID: untouched.ID, Action: research.ActionNone},
	}, results)

	// High-confidence contradiction was superseded (never silently overwritten).
	gotHigh, err := fsvc.Get(ctx, high.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusSuperseded, gotHigh.Status)
	require.Equal(t, replacement.ID, gotHigh.SupersededBy)

	// Low-confidence contradiction was FLAGGED (disputed), not overwritten.
	gotLow, err := fsvc.Get(ctx, low.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusDisputed, gotLow.Status)
	require.Equal(t, "weak conflicting source", gotLow.DisputeNote)
	require.Equal(t, "weakly contested", gotLow.Claim, "the contested claim text is preserved, not overwritten")

	// Non-contradiction left ACTIVE and untouched.
	gotUntouched, err := fsvc.Get(ctx, untouched.ID)
	require.NoError(t, err)
	require.Equal(t, findings.StatusActive, gotUntouched.Status)
}

// TestReconcileGateThresholdBoundary pins the gate constant's boundary: exactly
// at the threshold ACTS (supersede), just below FLAGS.
func TestReconcileGateThresholdBoundary(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)
	svc := research.NewService(research.Deps{Findings: fsvc})

	atThreshold, err := fsvc.Add(ctx, findings.NewFinding{Claim: "at threshold", Confidence: 0.5})
	require.NoError(t, err)
	belowThreshold, err := fsvc.Add(ctx, findings.NewFinding{Claim: "below threshold", Confidence: 0.5})
	require.NoError(t, err)

	results, err := svc.Reconcile(ctx, []research.ReconcileItem{
		{ExistingID: atThreshold.ID, Confidence: research.HighConfidenceThreshold, Contradicts: true},
		{ExistingID: belowThreshold.ID, Confidence: research.HighConfidenceThreshold - 0.01, Contradicts: true},
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []research.ReconcileResult{
		{ExistingID: atThreshold.ID, Action: research.ActionSupersede},
		{ExistingID: belowThreshold.ID, Action: research.ActionFlag},
	}, results)
}

// TestReconcileUnavailableWithoutFindings asserts the reconcile path needs a
// wired findings store.
func TestReconcileUnavailableWithoutFindings(t *testing.T) {
	svc := research.NewService(research.Deps{})
	_, err := svc.Reconcile(context.Background(), []research.ReconcileItem{{ExistingID: "x", Contradicts: true}})
	require.Error(t, err)
}
