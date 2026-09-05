package research_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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

// TestRunL3PromptRegistersResearchTools asserts the L3 run's callable toolset:
// the spawned agent-manager task declares the L2 fetch/synthesis endpoint, the
// bounded gather endpoint, fresh search, and the findings curation CLI commands
// as the tools the research run may invoke.
func TestRunL3PromptRegistersResearchTools(t *testing.T) {
	am := &fakeAgentManager{spawnResult: agentmanager.RunResult{RunID: "run-1", TaskID: "task-1", Status: "pending"}}
	svc := research.NewService(research.Deps{AgentManager: am})

	_, err := svc.RunL3(context.Background(), "what changed about X")
	require.NoError(t, err)

	prompt := am.spawnedReq.Prompt
	for _, tool := range []string{
		"web-search research l2",     // L2 fetch + cited synthesis endpoint
		"web-search research gather", // bounded GATHER endpoint
		"web-search search",          // fresh candidate URLs
		"web-search findings supersede",
		"web-search findings flag",
		"web-search findings add",
	} {
		require.Contains(t, prompt, tool, "L3 run must register %q as a callable tool", tool)
	}
}

// TestRunL3EncodesGapIterationBeyondSingleL2 pins the business contract that an
// L3 run is a strict superset of a single L2 call: the run is instructed to
// gather what is already known, then iterate focused L2 sub-queries over what
// the existing findings do NOT already cover (the gaps), then reconcile —
// rather than answering from one synthesis pass.
func TestRunL3EncodesGapIterationBeyondSingleL2(t *testing.T) {
	am := &fakeAgentManager{spawnResult: agentmanager.RunResult{RunID: "run-1", Status: "pending"}}
	svc := research.NewService(research.Deps{AgentManager: am})

	_, err := svc.RunL3(context.Background(), "q")
	require.NoError(t, err)

	prompt := am.spawnedReq.Prompt
	require.Contains(t, prompt, "do not already cover", "research is directed at the GAPS, not a blanket pass")
	require.Contains(t, prompt, "focused sub-query", "gaps are pursued via focused L2 sub-queries")
	require.Contains(t, prompt, "web-search research l2", "a single L2 call is a sub-step of the L3 loop")
	require.Contains(t, prompt, "RECONCILE", "the loop ends with the bounded reconcile post-step")
}

// TestL3PromptMirrorsConfiguredTuning asserts the spawned task prompt reflects
// the service's EFFECTIVE tuning (Deps.GatherCap / Deps.ConfidenceGate /
// Deps.MaxResearchLoops), so an operator override changes what the agent is
// told, not just what the server enforces.
func TestL3PromptMirrorsConfiguredTuning(t *testing.T) {
	am := &fakeAgentManager{spawnResult: agentmanager.RunResult{RunID: "run-1", Status: "pending"}}
	svc := research.NewService(research.Deps{AgentManager: am, GatherCap: 5, ConfidenceGate: 0.9, MaxResearchLoops: 4})

	_, err := svc.RunL3(context.Background(), "q")
	require.NoError(t, err)

	prompt := am.spawnedReq.Prompt
	require.Contains(t, prompt, "caps it at 5 findings", "prompt mirrors the configured gather cap")
	require.Contains(t, prompt, "confidence >= 0.90", "prompt mirrors the configured confidence gate")
	require.Contains(t, prompt, "at most 4 research loops", "prompt mirrors the configured iteration budget")
}

// TestL3PromptBoundsIterationBudget pins the iteration-budget contract: every
// L3 task prompt carries a finite research-loop budget (default 10) and the
// instruction to converge rather than iterate indefinitely. The hard run
// lifecycle bound (timeout/cancellation) is owned by agent-manager; this
// budget is the task-contract half web-search controls.
func TestL3PromptBoundsIterationBudget(t *testing.T) {
	am := &fakeAgentManager{spawnResult: agentmanager.RunResult{RunID: "run-1", Status: "pending"}}
	svc := research.NewService(research.Deps{AgentManager: am})

	_, err := svc.RunL3(context.Background(), "q")
	require.NoError(t, err)

	prompt := am.spawnedReq.Prompt
	require.Contains(t, prompt, "at most 10 research loops", "default budget is 10 loops")
	require.Contains(t, prompt, "emit the brief from what you have", "budget exhaustion converges instead of iterating")
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

// TestReconcileGateConfigurable asserts the supersede/flag boundary is
// configuration-driven (Deps.ConfidenceGate, fed by
// WEB_SEARCH_HIGH_CONFIDENCE_THRESHOLD): with a 0.9 gate, a 0.85 contradiction
// — which would supersede under the 0.75 default — is FLAGGED instead, and an
// out-of-range gate falls back to the compiled default.
func TestReconcileGateConfigurable(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)
	svc := research.NewService(research.Deps{Findings: fsvc, ConfidenceGate: 0.9})

	atGate, err := fsvc.Add(ctx, findings.NewFinding{Claim: "at the configured gate", Confidence: 0.5})
	require.NoError(t, err)
	belowGate, err := fsvc.Add(ctx, findings.NewFinding{Claim: "below the configured gate", Confidence: 0.5})
	require.NoError(t, err)

	results, err := svc.Reconcile(ctx, []research.ReconcileItem{
		{ExistingID: atGate.ID, Confidence: 0.9, Contradicts: true},
		{ExistingID: belowGate.ID, Confidence: 0.85, Contradicts: true}, // above the 0.75 default
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []research.ReconcileResult{
		{ExistingID: atGate.ID, Action: research.ActionSupersede},
		{ExistingID: belowGate.ID, Action: research.ActionFlag},
	}, results)

	// An out-of-range gate (e.g. 1.5) falls back to the compiled default.
	fallback := research.NewService(research.Deps{Findings: fsvc, ConfidenceGate: 1.5})
	defaulted, err := fsvc.Add(ctx, findings.NewFinding{Claim: "default-gated", Confidence: 0.5})
	require.NoError(t, err)
	results, err = fallback.Reconcile(ctx, []research.ReconcileItem{
		{ExistingID: defaulted.ID, Confidence: research.HighConfidenceThreshold, Contradicts: true},
	})
	require.NoError(t, err)
	require.Equal(t, []research.ReconcileResult{{ExistingID: defaulted.ID, Action: research.ActionSupersede}}, results)
}

// TestReconcilePostStepWithinBudget is the OT-P1-003 performance gate for the
// bounded reconcile post-step: curating a typical run's output (10 items — a
// mix of supersedes and flags) against a real SQLite findings store must
// complete within the 30-second budget.
func TestReconcilePostStepWithinBudget(t *testing.T) {
	ctx := context.Background()
	fsvc := newFindingsService(t)
	svc := research.NewService(research.Deps{Findings: fsvc})

	items := make([]research.ReconcileItem, 0, 10)
	for i := 0; i < 10; i++ {
		f, err := fsvc.Add(ctx, findings.NewFinding{Claim: fmt.Sprintf("existing claim %d", i), Confidence: 0.5})
		require.NoError(t, err)
		conf := 0.9 // supersede path
		if i%2 == 1 {
			conf = 0.4 // flag path
		}
		items = append(items, research.ReconcileItem{ExistingID: f.ID, Confidence: conf, Contradicts: true, Reason: "perf fixture"})
	}

	start := time.Now()
	results, err := svc.Reconcile(ctx, items)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Len(t, results, 10)
	require.Less(t, elapsed, 30*time.Second, "reconcile post-step for 10 findings must finish within the 30s budget")
}

// TestReconcileUnavailableWithoutFindings asserts the reconcile path needs a
// wired findings store.
func TestReconcileUnavailableWithoutFindings(t *testing.T) {
	svc := research.NewService(research.Deps{})
	_, err := svc.Reconcile(context.Background(), []research.ReconcileItem{{ExistingID: "x", Contradicts: true}})
	require.Error(t, err)
}
