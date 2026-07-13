package validation_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	planmodel "plan-manager/internal/planmodel"
	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/testutil/mocks"
	"plan-manager/internal/validation"

	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakePlans struct {
	plan internalplans.Plan
	err  error
}

func (f fakePlans) GetPlan(_ context.Context, _ string) (internalplans.Plan, error) {
	return f.plan, f.err
}

type fakeBaselineInventory struct {
	inventory validation.BaselineInventory
	ok        bool
	err       error
}

func (f fakeBaselineInventory) LatestBaselineInventory(context.Context, string) (validation.BaselineInventory, bool, error) {
	return f.inventory, f.ok, f.err
}

type fakeResolver struct {
	resolution internalplans.ReferenceResolution
	err        error
}

func (f fakeResolver) Resolve(_ context.Context, ref internalplans.Reference) (internalplans.Reference, error) {
	if f.err != nil {
		return ref, f.err
	}
	ref.Resolution = f.resolution
	return ref, nil
}

type fakeStaleness struct {
	tier   internalplans.StalenessTier
	factor float64
	err    error
}

type fakeCollectionClient struct {
	result     validation.BaselineCollectionCaptureResult
	err        error
	called     validation.BaselineCollectionCaptureRequest
	calls      int
	diffResult validation.BaselineCollectionDiffResult
	diffErr    error
	diffCalled validation.BaselineCollectionDiffRequest
	pathResult validation.BaselinePathDiffResult
	pathErr    error
	pathCalled validation.BaselinePathDiffRequest
}

func (f *fakeCollectionClient) StartCollectionDiff(_ context.Context, req validation.BaselineCollectionDiffRequest) (validation.BaselineCollectionDiffResult, error) {
	f.diffCalled = req
	return f.diffResult, f.diffErr
}

func (f *fakeCollectionClient) StartCollectionCapture(_ context.Context, req validation.BaselineCollectionCaptureRequest) (validation.BaselineCollectionCaptureResult, error) {
	f.calls++
	f.called = req
	return f.result, f.err
}

func (f *fakeCollectionClient) DiffPathEvidence(_ context.Context, req validation.BaselinePathDiffRequest) (validation.BaselinePathDiffResult, error) {
	f.pathCalled = req
	return f.pathResult, f.pathErr
}

func (f fakeStaleness) Compute(_ context.Context, _ internalplans.Reference) (internalplans.StalenessTier, float64, error) {
	return f.tier, f.factor, f.err
}

type fakeCommandValidator struct {
	results map[string]validation.CommandReferenceResult
	calls   []validation.CommandReferenceRequest
}

type fakeDurableStore struct {
	mu      sync.Mutex
	ops     map[string]validation.ValidationOperation
	byKey   map[string]string
	byScope map[string]string
	results map[string]validation.Result
}

func newFakeDurableStore() *fakeDurableStore {
	return &fakeDurableStore{ops: map[string]validation.ValidationOperation{}, byKey: map[string]string{}, byScope: map[string]string{}, results: map[string]validation.Result{}}
}

func cloneOperation(op validation.ValidationOperation) validation.ValidationOperation {
	data, _ := json.Marshal(op)
	var cloned validation.ValidationOperation
	_ = json.Unmarshal(data, &cloned)
	return cloned
}

func (s *fakeDurableStore) CreateOperation(_ context.Context, op validation.ValidationOperation) (validation.ValidationOperation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := op.PlanID + "\x00" + op.PhaseID + "\x00" + op.IdempotencyKey
	scope := op.PlanID + "\x00" + op.PhaseID
	if id := s.byScope[scope]; id != "" && !s.ops[id].Terminal() {
		return cloneOperation(s.ops[id]), false, nil
	}
	if op.IdempotencyKey != "" {
		if id := s.byKey[key]; id != "" {
			return cloneOperation(s.ops[id]), false, nil
		}
		s.byKey[key] = op.ID
	}
	s.ops[op.ID] = cloneOperation(op)
	s.byScope[scope] = op.ID
	return cloneOperation(op), true, nil
}

func (s *fakeDurableStore) SaveOperation(_ context.Context, op validation.ValidationOperation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ops[op.ID] = cloneOperation(op)
	return nil
}

func (s *fakeDurableStore) GetOperation(_ context.Context, id string) (validation.ValidationOperation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	op, ok := s.ops[id]
	return cloneOperation(op), ok, nil
}

func (s *fakeDurableStore) ListNonTerminalOperations(context.Context) ([]validation.ValidationOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []validation.ValidationOperation
	for _, op := range s.ops {
		if !op.Terminal() {
			out = append(out, cloneOperation(op))
		}
	}
	return out, nil
}

func (s *fakeDurableStore) SaveResult(_ context.Context, result validation.Result) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.results[result.ID]; !exists {
		s.results[result.ID] = result
	}
	return nil
}

func (s *fakeDurableStore) GetResult(_ context.Context, id string) (validation.Result, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, found := s.results[id]
	return result, found, nil
}

func (s *fakeDurableStore) LastResult(_ context.Context, planID, phaseID string) (validation.Result, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest validation.Result
	for _, result := range s.results {
		if result.PlanID == planID && result.PhaseID == phaseID && result.RanAt >= latest.RanAt {
			latest = result
		}
	}
	return latest, latest.ID != "", nil
}

func (f *fakeCommandValidator) ValidateCommandReference(_ context.Context, req validation.CommandReferenceRequest) (validation.CommandReferenceResult, error) {
	f.calls = append(f.calls, req)
	if res, ok := f.results[req.CommandText]; ok {
		return res, nil
	}
	return validation.CommandReferenceResult{Verdict: "unknown", ValidationLevel: "parsed"}, nil
}

func planWith(refs []internalplans.Reference, phases []internalplans.Phase) internalplans.Plan {
	if phases == nil {
		phases = []internalplans.Phase{validationReadyPhase("ph1")}
	} else {
		for i := range phases {
			phases[i] = normalizeValidationPhase(phases[i])
		}
	}
	plan := internalplans.Plan{
		ID:                 "p1",
		Slug:               "p1",
		Title:              "P",
		Purpose:            "Validate a plan.",
		ProblemStatement:   "Validation must report command and quality issues honestly.",
		TargetOutcome:      "Validation verdicts match the requested oracle.",
		Scope:              "Validation service test fixture.",
		TechnicalApproach:  "Use local seams and fakes.",
		ValidationStrategy: "Run validation unit tests.",
		DefinitionOfDone:   "Validation result is deterministic.",
		Constraints:        "NO_CODE_REFS: validation unit fixture has no plan-level connected refs.",
		ChangeBoundary: internalplans.ChangeBoundary{
			AcceptanceAllow: []string{"scenarios/plan-manager/**"},
		},
		RegressionAnchor: internalplans.RegressionAnchor{
			Strategy: internalplans.AnchorStrategyChangeBoundary,
		},
		References: refs,
		RelevantContext: []internalplans.RelevantContextItem{{
			ID:           "ctx-global",
			Kind:         internalplans.RelevantContextNote,
			Scope:        internalplans.RelevantContextScopeGlobal,
			Label:        "NO_CONTEXT: validation unit fixture has no plan-wide setup.",
			Instruction:  "NO_CONTEXT: validation unit fixture has no plan-wide setup.",
			Required:     true,
			RepeatPolicy: internalplans.RelevantContextOncePerExecution,
			Source:       internalplans.RelevantContextSourceAuthored,
			Status:       internalplans.RelevantContextStatusReady,
		}},
		Phases: phases,
	}
	return plan
}

func validationReadyPhase(id string) internalplans.Phase {
	return normalizeValidationPhase(internalplans.Phase{
		ID:              id,
		Title:           "Validation fixture phase",
		Intent:          "Exercise validation behavior.",
		Steps:           []string{"Run validation."},
		Validation:      "go test ./internal/validation",
		Acceptance:      "Validation result matches expectation.",
		RelevantContext: noContextPhaseItem(id),
	})
}

func normalizeValidationPhase(phase internalplans.Phase) internalplans.Phase {
	if phase.ID == "" {
		phase.ID = "ph1"
	}
	if phase.Title == "" {
		phase.Title = "Validation fixture phase"
	}
	hasNoCode := false
	for _, reminder := range phase.Reminders {
		if strings.Contains(reminder, "NO_CODE_REFS:") || strings.Contains(strings.ToLower(reminder), "no connected code references:") {
			hasNoCode = true
			break
		}
	}
	if len(phase.References) == 0 && !hasNoCode {
		phase.Reminders = append(phase.Reminders, "NO_CODE_REFS: validation unit fixture has no phase refs.")
	}
	return phase
}

func noContextPhaseItem(phaseID string) []internalplans.RelevantContextItem {
	return []internalplans.RelevantContextItem{{
		ID:           "ctx-" + phaseID,
		Kind:         internalplans.RelevantContextNote,
		Scope:        internalplans.RelevantContextScopePhase,
		PhaseID:      phaseID,
		Label:        "NO_CONTEXT: validation unit fixture has no phase setup.",
		Instruction:  "NO_CONTEXT: validation unit fixture has no phase setup.",
		Required:     true,
		RepeatPolicy: internalplans.RelevantContextPhaseEntry,
		Source:       internalplans.RelevantContextSourceAuthored,
		Status:       internalplans.RelevantContextStatusReady,
	}}
}

func requireFindingCode(t *testing.T, findings []validation.CommandFinding, code string) {
	t.Helper()
	for _, finding := range findings {
		for _, got := range finding.IssueCodes {
			if got == code {
				return
			}
		}
	}
	t.Fatalf("finding code %q not found in %#v", code, findings)
}

// --- tests ---

// [REQ:PM-REF-001]
func TestResolveReferencesDegradesWhenResolverDown(t *testing.T) {
	plan := planWith([]internalplans.Reference{
		{Kind: internalplans.ReferenceCode, Target: "a.go"},
		{Kind: internalplans.ReferenceCode, Target: "b.go", Future: true},
	}, nil)
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}}) // nil resolver
	report, err := svc.ResolveReferences(context.Background(), "p1", "")
	require.NoError(t, err)
	require.True(t, report.Degraded, "nil resolver => degraded")
	require.Equal(t, internalplans.ResolutionUnresolved, report.References[0].Resolution)
	require.Equal(t, internalplans.ResolutionFuture, report.References[1].Resolution, "future refs are not flagged as deleted")
}

func TestResolveReferencesResolverError(t *testing.T) {
	plan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "a.go"}}, nil)
	svc := validation.NewService(validation.Deps{
		Plans:    fakePlans{plan: plan},
		Resolver: fakeResolver{err: errors.New("code-facts boom")},
	})
	report, err := svc.ResolveReferences(context.Background(), "p1", "")
	require.NoError(t, err)
	require.True(t, report.Degraded)
	require.Equal(t, internalplans.ResolutionUnresolved, report.References[0].Resolution)
}

// [REQ:PM-STALE-001]
func TestComputeStalenessTiering(t *testing.T) {
	plan := planWith([]internalplans.Reference{
		{Kind: internalplans.ReferenceCode, Target: "a.go"},
		{Kind: internalplans.ReferenceCode, Target: "b.go"},
	}, nil)
	svc := validation.NewService(validation.Deps{
		Plans:     fakePlans{plan: plan},
		Resolver:  fakeResolver{resolution: internalplans.ResolutionResolved},
		Staleness: fakeStaleness{tier: internalplans.StalenessLightlyStale, factor: 0.3},
	})
	report, err := svc.ComputeStaleness(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, internalplans.StalenessLightlyStale, report.Overall, "overall is the worst tier")
	require.InDelta(t, 0.3, report.References[0].ChangeFactor, 0.001)
}

// TestStalenessRefinesFreshToLightlyStaleViaGit pins the lightly-stale tier: a
// reference the existence floor calls FRESH (still present) is upgraded to
// LIGHTLY_STALE when git shows its code changed since the anchor HeadSha — with a
// non-zero change factor — while no change leaves it FRESH.
// [REQ:PM-STALE-001]
func TestStalenessRefinesFreshToLightlyStaleViaGit(t *testing.T) {
	plan := internalplans.Plan{
		ID: "p1", Slug: "p1", Title: "P",
		References:       []internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "a.go"}},
		RegressionAnchor: internalplans.RegressionAnchor{HeadSha: "abc123"},
	}

	t.Run("changed code => lightly stale", func(t *testing.T) {
		svc := validation.NewService(validation.Deps{
			Plans:     fakePlans{plan: plan},
			Resolver:  fakeResolver{resolution: internalplans.ResolutionResolved},
			Staleness: fakeStaleness{tier: internalplans.StalenessFresh}, // floor: present
			Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte("12\t3\ta.go\n"), nil // numstat: 12 added, 3 deleted
			},
		})
		report, err := svc.ComputeStaleness(context.Background(), "p1", "")
		require.NoError(t, err)
		require.Equal(t, internalplans.StalenessLightlyStale, report.Overall)
		require.Greater(t, report.References[0].ChangeFactor, 0.0)
	})

	t.Run("no change => stays fresh", func(t *testing.T) {
		svc := validation.NewService(validation.Deps{
			Plans:     fakePlans{plan: plan},
			Resolver:  fakeResolver{resolution: internalplans.ResolutionResolved},
			Staleness: fakeStaleness{tier: internalplans.StalenessFresh},
			Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return []byte(""), nil // numstat empty => unchanged
			},
		})
		report, err := svc.ComputeStaleness(context.Background(), "p1", "")
		require.NoError(t, err)
		require.Equal(t, internalplans.StalenessFresh, report.Overall)
	})

	t.Run("tool absent => floor fresh stands", func(t *testing.T) {
		svc := validation.NewService(validation.Deps{
			Plans:     fakePlans{plan: plan},
			Resolver:  fakeResolver{resolution: internalplans.ResolutionResolved},
			Staleness: fakeStaleness{tier: internalplans.StalenessFresh},
			Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return nil, validation.ErrToolNotFound
			},
		})
		report, err := svc.ComputeStaleness(context.Background(), "p1", "")
		require.NoError(t, err)
		require.Equal(t, internalplans.StalenessFresh, report.Overall)
	})
}

func TestComputeStalenessUnknownWhenComputerDown(t *testing.T) {
	plan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "a.go"}}, nil)
	svc := validation.NewService(validation.Deps{
		Plans:    fakePlans{plan: plan},
		Resolver: fakeResolver{resolution: internalplans.ResolutionResolved},
		// nil staleness computer
	})
	report, err := svc.ComputeStaleness(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, internalplans.StalenessUnknown, report.Overall)
	require.True(t, report.Degraded)
}

// [REQ:PM-VALID-001]
func TestDeriveBaselineScope(t *testing.T) {
	plan := internalplans.Plan{
		ID:   "p1",
		Slug: "p1",
		References: []internalplans.Reference{
			{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/api/main.go"},
			{Kind: internalplans.ReferenceCode, Target: "scenarios/bar/cli/app.go"},
			{Kind: internalplans.ReferenceCode, Target: "packages/api-core/x.go"},
			{Kind: internalplans.ReferenceCode, Target: "scenarios/baz/x.go", Future: true}, // future excluded
			{Kind: internalplans.ReferenceReq, Target: "OT-P0-001"},                         // non-code excluded
		},
		RegressionAnchor: internalplans.RegressionAnchor{
			BaselineName: "impl",
			Commands:     []string{"git-control-tower baseline diff --scenario foo --name impl --wait --json"},
		},
	}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)

	require.Contains(t, scope.Commands, "git-control-tower baseline diff --scenario foo --name impl --wait --json")
	require.Contains(t, scope.Commands, "git-control-tower baseline diff --scenario bar --name impl --wait --json")
	require.Contains(t, scope.Commands, "git diff --stat", "non-scenario code => repo-level diff")
	require.Contains(t, scope.Locations, "scenarios/foo")
	require.Contains(t, scope.Locations, "repo")
	// Anchor command is deduped, not duplicated.
	foo := 0
	for _, c := range scope.Commands {
		if c == "git-control-tower baseline diff --scenario foo --name impl --wait --json" {
			foo++
		}
	}
	require.Equal(t, 1, foo, "anchor command deduped against derived command")
	require.Len(t, scope.Commands, 3, "one semantic scenario oracle each plus one informational repo diff")
}

func TestDeriveBaselineScopeCompilesFleetToOneOraclePerScenario(t *testing.T) { // [REQ:PM-VALID-001]
	allow := make([]string, 0, 22)
	for i := 0; i < 22; i++ {
		allow = append(allow, fmt.Sprintf("scenarios/scenario-%02d/**", i))
	}
	allow = append(allow, "packages/proto/**")
	plan := planWith(nil, nil)
	plan.ChangeBoundary.AcceptanceAllow = allow
	plan.RegressionAnchor.BaselineName = "fixture"
	// The human and JSON projections of the same GCT oracle must collapse.
	plan.RegressionAnchor.Commands = []string{
		"git-control-tower baseline diff --scenario scenario-00 --name fixture --wait",
		"git-control-tower baseline diff --scenario scenario-00 --name fixture --wait --json",
	}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Len(t, scope.Commands, 23, "one oracle per scenario plus one informational repo diff, with no snapshots or render duplicates")
	for _, command := range scope.Commands {
		require.NotContains(t, command, "snapshot status")
	}
}

func TestDeriveBaselineScopeUsesExplicitNarrowPhaseScope(t *testing.T) { // [REQ:PM-VALID-001]
	plan := durableValidationPlan()
	plan.ChangeBoundary.AcceptanceAllow = []string{"scenarios/agent-manager/**", "scenarios/plan-manager/**"}
	plan.Phases = []internalplans.Phase{{
		ID: "phase-3", Title: "Agent Manager work", Intent: "Narrow validation.",
		ValidationScope: planmodel.ValidationScope{Mode: planmodel.ValidationScopeNarrow, Boundary: internalplans.ChangeBoundary{AcceptanceAllow: []string{"scenarios/agent-manager/**"}}},
	}}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "phase-3")
	require.NoError(t, err)
	require.Equal(t, []string{"git-control-tower baseline diff --scenario agent-manager --name impl --wait --json"}, scope.Commands)
}

func TestDeriveBaselineScopeDoesNotFabricateGCTCommandWithoutName(t *testing.T) {
	plan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/x.go"}}, nil)
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Contains(t, scope.Locations, "scenarios/foo")
	require.Empty(t, scope.Commands, "GCT baseline diff requires a verified --name")
}

func TestDeriveBaselineScopeUsesTypedAnchorWithoutReferences(t *testing.T) {
	plan := planWith(nil, nil)
	plan.RegressionAnchor = internalplans.RegressionAnchor{
		Strategy:     "scenario_baseline",
		Scenario:     "plan-manager",
		BaselineName: "hardening",
	}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Contains(t, scope.Locations, "scenarios/plan-manager")
	require.Contains(t, scope.Commands, "git-control-tower baseline diff --scenario plan-manager --name hardening --wait --json")
}

func TestDeriveBaselineScopeUsesHeadAllowlistAnchorWithoutReferences(t *testing.T) {
	plan := planWith(nil, nil)
	plan.RegressionAnchor = internalplans.RegressionAnchor{
		Strategy:       "head_sha_allowlist",
		HeadSha:        "abc123",
		AllowlistPaths: []string{"packages/proto", "scenarios/plan-manager"},
	}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Contains(t, scope.Locations, "repo")
	require.Contains(t, scope.Commands, "git diff --stat abc123 -- packages/proto scenarios/plan-manager")
}

// TestDeriveBaselineScopeFromChangeBoundary proves the change boundary is the
// source of truth: affected scenarios derive from acceptance_allow, non-scenario
// allow globs become a path-scoped INFORMATIONAL diff, and references supplement
// (never under-cover) the boundary scenarios.
func TestDeriveBaselineScopeFromChangeBoundary(t *testing.T) {
	plan := internalplans.Plan{
		ID:   "p1",
		Slug: "p1",
		ChangeBoundary: internalplans.ChangeBoundary{
			AcceptanceAllow: []string{"scenarios/foo/**", "packages/proto/**", "docs/**"},
		},
		References: []internalplans.Reference{
			{Kind: internalplans.ReferenceCode, Target: "scenarios/bar/cli/app.go"}, // supplements boundary
		},
		RegressionAnchor: internalplans.RegressionAnchor{
			Strategy:     internalplans.AnchorStrategyChangeBoundary,
			BaselineName: "impl",
		},
	}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)

	// Boundary scenario foo and reference scenario bar both produce oracle pairs.
	require.Contains(t, scope.Commands, "git-control-tower baseline diff --scenario foo --name impl --wait --json")
	require.Contains(t, scope.Commands, "git-control-tower baseline diff --scenario bar --name impl --wait --json")
	// Non-scenario allow globs become ONE path-scoped informational diff (no sha).
	require.Contains(t, scope.Commands, "git diff --stat -- docs/** packages/proto/**")
	require.Contains(t, scope.Locations, "scenarios/foo")
	require.Contains(t, scope.Locations, "repo")
}

// TestDeriveBaselineScopeDocsOnlyBoundary proves a docs-only boundary yields no
// scenario oracle and only an informational repo/path diff.
func TestDeriveBaselineScopeDocsOnlyBoundary(t *testing.T) {
	plan := internalplans.Plan{
		ID: "p1", Slug: "p1",
		ChangeBoundary: internalplans.ChangeBoundary{AcceptanceAllow: []string{"docs/**"}},
	}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, []string{"repo"}, scope.Locations)
	require.Contains(t, scope.Commands, "git diff --stat -- docs/**")
	for _, c := range scope.Commands {
		require.False(t, strings.HasPrefix(c, "git-control-tower baseline diff"), "docs-only boundary must not fabricate a scenario oracle")
	}
}

// TestDeriveBaselineScopePhaseBoundaryNarrows proves a phase boundary narrows the
// derived scope when a phase id is supplied.
func TestDeriveBaselineScopePhaseBoundaryNarrows(t *testing.T) {
	plan := internalplans.Plan{
		ID: "p1", Slug: "p1",
		ChangeBoundary:   internalplans.ChangeBoundary{AcceptanceAllow: []string{"scenarios/foo/**", "scenarios/bar/**"}},
		RegressionAnchor: internalplans.RegressionAnchor{BaselineName: "impl"},
		Phases: []internalplans.Phase{{
			ID:             "ph1",
			ChangeBoundary: internalplans.ChangeBoundary{AcceptanceAllow: []string{"scenarios/foo/**"}},
		}},
	}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	scope, err := svc.DeriveBaselineScope(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Contains(t, scope.Locations, "scenarios/foo")
	require.NotContains(t, scope.Locations, "scenarios/bar", "phase boundary narrows away the plan's other scenario")
}

// [REQ:PM-VALID-002]
func TestRunValidationVerdicts(t *testing.T) {
	plan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/x.go"}}, nil)
	plan.RegressionAnchor = internalplans.RegressionAnchor{BaselineName: "impl"}

	// PASS: runner returns no error for every command.
	pass := validation.NewService(validation.Deps{
		Plans:  fakePlans{plan: plan},
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) { return []byte("clean"), nil },
	})
	res, err := pass.RunValidation(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictPass, res.Verdict)

	// FAIL: runner errors (non-zero exit).
	fail := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan},
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("diff"), errors.New("exit 1")
		},
	})
	res, err = fail.RunValidation(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictFail, res.Verdict)

	// UNKNOWN: no runner configured — never a fabricated pass.
	unknown := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	res, err = unknown.RunValidation(context.Background(), "p1", "")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictUnknown, res.Verdict)
}

func durableValidationPlan() internalplans.Plan {
	plan := planWith(nil, nil)
	plan.ChangeBoundary.AcceptanceAllow = []string{"scenarios/foo/**"}
	plan.RegressionAnchor = internalplans.RegressionAnchor{
		Strategy: internalplans.AnchorStrategyChangeBoundary, BaselineName: "impl",
	}
	return plan
}

func TestDurableValidationConcurrentIdempotencyRunsOneChildSet(t *testing.T) { // [REQ:PM-VALID-004]
	store := newFakeDurableStore()
	release := make(chan struct{})
	var callsMu sync.Mutex
	var calls int
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store,
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			callsMu.Lock()
			calls++
			callsMu.Unlock()
			<-release
			return []byte(`{"runId":"run-child"}`), nil
		},
	})

	const starters = 12
	ids := make(chan string, starters)
	var wg sync.WaitGroup
	for i := 0; i < starters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op, _, err := svc.StartValidation(context.Background(), "p1", "", "same-request")
			require.NoError(t, err)
			ids <- op.ID
		}()
	}
	wg.Wait()
	close(ids)
	var operationID string
	for id := range ids {
		if operationID == "" {
			operationID = id
		}
		require.Equal(t, operationID, id)
	}
	close(release)
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err := svc.GetValidationOperation(waitCtx, operationID, true)
	require.NoError(t, err)
	require.True(t, op.Terminal())
	require.NotNil(t, op.Result)
	require.NotEmpty(t, op.ResultRef)
	require.Len(t, op.Children, 1)
	for _, child := range op.Children {
		require.Equal(t, validation.ChildTerminal, child.Status)
		require.Equal(t, 1, child.Attempt)
		require.Equal(t, "run-child", child.ExternalID)
	}
	callsMu.Lock()
	require.Equal(t, len(op.Children), calls)
	callsMu.Unlock()
}

func TestBaselineSetFinalValidationUsesTypedFullCollectionDiff(t *testing.T) {
	store := newFakeDurableStore()
	plan := durableValidationPlan()
	plan.BaselineSet = planmodel.BaselineSetIntent{Name: "before", ScenarioTargets: []string{"foo", "bar"}, RepoPaths: []string{"packages/proto/**"}}
	collections := &fakeCollectionClient{diffResult: validation.BaselineCollectionDiffResult{Classification: "clean", Detail: "foo:ready:clean"}}
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan}, Results: store, Operations: store, Collections: collections,
		Runner: func(context.Context, string, ...string) ([]byte, error) {
			return nil, errors.New("legacy runner must not dispatch collection diff")
		},
	})
	op, _, err := svc.StartValidation(context.Background(), "p1", "", "typed-collection")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err = svc.GetValidationOperation(ctx, op.ID, true)
	require.NoError(t, err)
	require.True(t, op.Terminal())
	require.Equal(t, validation.VerdictPass, op.Result.Verdict)
	require.Len(t, op.Children, 1)
	require.Equal(t, validation.ValidationCheckCollectionDiff, op.Children[0].Check.Kind)
	require.Equal(t, []string{"bar", "foo"}, collections.diffCalled.Scenarios)
	require.Equal(t, op.Children[0].ID, collections.diffCalled.OperationID)
}

func TestBaselineSetPhaseValidationUsesOnlyExplicitNarrowScope(t *testing.T) {
	store := newFakeDurableStore()
	plan := durableValidationPlan()
	plan.BaselineSet = planmodel.BaselineSetIntent{Name: "before", ScenarioTargets: []string{"foo", "bar"}}
	plan.Phases = []planmodel.Phase{{ID: "phase-foo", ValidationScope: planmodel.ValidationScope{Mode: planmodel.ValidationScopeNarrow, Boundary: planmodel.ChangeBoundary{AcceptanceAllow: []string{"scenarios/foo/**"}}}}}
	collections := &fakeCollectionClient{diffResult: validation.BaselineCollectionDiffResult{Classification: "clean"}}
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan}, Results: store, Operations: store, Collections: collections,
		Inventories: fakeBaselineInventory{inventory: validation.BaselineInventory{Name: "before", Branch: "agi", ScenarioTargets: []string{"foo"}, PathSnapshots: []validation.BaselinePathSnapshot{{Name: "paths-before", Branch: "agi"}}}, ok: true},
	})
	op, _, err := svc.StartValidation(context.Background(), "p1", "phase-foo", "narrow-collection")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = svc.GetValidationOperation(ctx, op.ID, true)
	require.NoError(t, err)
	require.Equal(t, []string{"foo"}, collections.diffCalled.Scenarios)
}

func TestBaselineSetPhaseValidationRejectsScenarioOutsideCapturedInventory(t *testing.T) {
	plan := durableValidationPlan()
	plan.BaselineSet = planmodel.BaselineSetIntent{Name: "before", ScenarioTargets: []string{"foo"}}
	plan.Phases = []planmodel.Phase{{ID: "phase-bar", ValidationScope: planmodel.ValidationScope{Mode: planmodel.ValidationScopeNarrow, Boundary: planmodel.ChangeBoundary{AcceptanceAllow: []string{"scenarios/bar/**"}}}}}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}, Operations: newFakeDurableStore()})
	_, _, err := svc.StartValidation(context.Background(), "p1", "phase-bar", "out-of-inventory")
	require.ErrorContains(t, err, "outside captured baseline inventory: bar")
}

func TestBaselineSetValidationRunsTypedScopedPathEvidenceSeparatelyFromOracle(t *testing.T) {
	store := newFakeDurableStore()
	plan := durableValidationPlan()
	plan.ChangeBoundary = planmodel.ChangeBoundary{AcceptanceAllow: []string{"packages/proto/**"}}
	plan.BaselineSet = planmodel.BaselineSetIntent{Name: "before", ScenarioTargets: []string{"foo"}, RepoPaths: []string{"packages/proto/**"}}
	collections := &fakeCollectionClient{pathResult: validation.BaselinePathDiffResult{AfterName: "before-after-1", Deltas: 2, Detail: "informational source evidence: 2 scoped path delta(s)"}}
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan}, Results: store, Operations: store, Collections: collections,
		Inventories: fakeBaselineInventory{inventory: validation.BaselineInventory{Name: "before", Branch: "agi", ScenarioTargets: []string{"foo"}, PathSnapshots: []validation.BaselinePathSnapshot{{Name: "paths-before", Branch: "agi"}}}, ok: true},
	})
	op, _, err := svc.StartValidation(context.Background(), "p1", "", "path-evidence")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err = svc.GetValidationOperation(ctx, op.ID, true)
	require.NoError(t, err)
	require.Len(t, op.Children, 2)
	var sourceChild validation.ValidationChild
	for _, child := range op.Children {
		if child.Check.Kind == validation.ValidationCheckPathSnapshotDiff {
			sourceChild = child
		}
	}
	require.Equal(t, validation.ValidationCheckPathSnapshotDiff, sourceChild.Check.Kind)
	require.Equal(t, "paths-before", collections.pathCalled.BeforeName)
	require.Equal(t, "agi", collections.pathCalled.Branch)
	require.Equal(t, []string{"packages/proto/**"}, collections.pathCalled.Paths)
	require.False(t, sourceChild.Oracle)
	require.Equal(t, validation.VerdictUnknown, op.Result.Verdict, "source evidence never creates a behavioral pass")
}

func TestBaselineSetValidationPrefersCapturedExecutionInventory(t *testing.T) {
	store := newFakeDurableStore()
	plan := durableValidationPlan()
	plan.BaselineSet = planmodel.BaselineSetIntent{Name: "before", ScenarioTargets: []string{"foo", "bar"}}
	collections := &fakeCollectionClient{diffResult: validation.BaselineCollectionDiffResult{Classification: "clean"}}
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan}, Results: store, Operations: store, Collections: collections,
		Inventories: fakeBaselineInventory{inventory: validation.BaselineInventory{Name: "before", ScenarioTargets: []string{"foo"}}, ok: true},
	})
	op, _, err := svc.StartValidation(context.Background(), "p1", "", "captured-inventory")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = svc.GetValidationOperation(ctx, op.ID, true)
	require.NoError(t, err)
	require.Equal(t, []string{"foo"}, collections.diffCalled.Scenarios)
}

func TestDurableValidationConcurrentUnkeyedStartsCoalesce(t *testing.T) { // [REQ:PM-VALID-004]
	store := newFakeDurableStore()
	release := make(chan struct{})
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store,
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) { <-release; return nil, nil },
	})
	const starters = 8
	ids := make(chan string, starters)
	var wg sync.WaitGroup
	for i := 0; i < starters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op, _, err := svc.StartValidation(context.Background(), "p1", "", "")
			require.NoError(t, err)
			ids <- op.ID
		}()
	}
	wg.Wait()
	close(ids)
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		require.Equal(t, first, id)
	}
	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err := svc.GetValidationOperation(ctx, first, true)
	require.NoError(t, err)
	require.Len(t, op.Children, 1)
	require.Contains(t, op.ScopeFingerprint, "sha256:")
}

func TestDurableValidationCallerCancellationDetachesAndTerminalPersists(t *testing.T) { // [REQ:PM-VALID-004]
	store := newFakeDurableStore()
	release := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store,
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			once.Do(func() { close(started) })
			<-release
			return []byte(`{"run_id":"run-after-detach"}`), nil
		},
	})
	op, _, err := svc.StartValidation(context.Background(), "p1", "", "detach")
	require.NoError(t, err)
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	detached, err := svc.GetValidationOperation(ctx, op.ID, true)
	var attachmentEnded validation.ErrAttachmentEnded
	require.ErrorAs(t, err, &attachmentEnded)
	require.Equal(t, op.ID, attachmentEnded.OperationID)
	require.False(t, detached.Terminal())
	close(release)
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer waitCancel()
	terminal, err := svc.GetValidationOperation(waitCtx, op.ID, true)
	require.NoError(t, err)
	require.True(t, terminal.Terminal())
	require.NotNil(t, terminal.Result)
	stored, ok, err := store.LastResult(context.Background(), "p1", "")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, terminal.ResultRef, stored.ID)
}

func TestDurableValidationRestartRecoversQueuedCheckpoint(t *testing.T) { // [REQ:PM-VALID-004]
	store := newFakeDurableStore()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	command := "git-control-tower baseline diff --scenario foo --name impl --wait --json"
	queued := validation.ValidationOperation{
		ID: "op-restart", PlanID: "p1", Status: validation.OperationQueued, QueuedAt: now,
		ExecutionBudgetSeconds: 60, RecommendedWaitSeconds: 60,
		Children: []validation.ValidationChild{{
			ID: "op-restart:1", Command: command, Oracle: true,
			Status: validation.ChildQueued, QueuedAt: now,
		}},
		Result: &validation.Result{PlanID: "p1", CommandsRun: []string{command}},
	}
	_, created, err := store.CreateOperation(context.Background(), queued)
	require.NoError(t, err)
	require.True(t, created)
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store,
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"runId":"run-recovered"}`), nil
		},
	})
	require.NoError(t, svc.RecoverPending(context.Background()))
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err := svc.GetValidationOperation(waitCtx, queued.ID, true)
	require.NoError(t, err)
	require.True(t, op.Terminal())
	require.Equal(t, 1, op.Attempt)
	require.Equal(t, 1, op.Children[0].Attempt)
	require.Equal(t, "run-recovered", op.Children[0].ExternalID)
}

func TestDurableValidationQueueResidenceDoesNotConsumeExecutionBudget(t *testing.T) { // [REQ:PM-VALID-004]
	store := newFakeDurableStore()
	clock := mocks.NewFakeClock(time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC))
	queuedAt := clock.Now().UTC().Format(time.RFC3339Nano)
	clock.Advance(10*time.Minute - time.Second) // near the former shared command ceiling
	command := "git-control-tower baseline diff --scenario foo --name impl --wait --json"
	queued := validation.ValidationOperation{
		ID: "op-queue", PlanID: "p1", Status: validation.OperationQueued, QueuedAt: queuedAt,
		QueueBudgetSeconds: 600, ExecutionBudgetSeconds: 5,
		Children: []validation.ValidationChild{{ID: "op-queue:1", Command: command, Oracle: true, Status: validation.ChildQueued, QueuedAt: queuedAt}},
		Result:   &validation.Result{ID: "op-queue:result", PlanID: "p1", CommandsRun: []string{command}},
	}
	_, _, err := store.CreateOperation(context.Background(), queued)
	require.NoError(t, err)
	remaining := make(chan time.Duration, 1)
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store, Clock: clock,
		Runner: func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			remaining <- time.Until(deadline)
			return []byte(`{"runId":"run-after-queue"}`), nil
		},
	})
	require.NoError(t, svc.RecoverPending(context.Background()))
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err := svc.GetValidationOperation(waitCtx, queued.ID, true)
	require.NoError(t, err)
	require.True(t, op.Terminal())
	require.Greater(t, <-remaining, 4*time.Second)
}

func TestDurableValidationBoundsConcurrentScenarioChildren(t *testing.T) { // [REQ:PM-VALID-004]
	store := newFakeDurableStore()
	plan := durableValidationPlan()
	plan.ChangeBoundary.AcceptanceAllow = []string{
		"scenarios/a/**", "scenarios/b/**", "scenarios/c/**",
		"scenarios/d/**", "scenarios/e/**", "scenarios/f/**",
	}
	release := make(chan struct{})
	started := make(chan struct{}, 12)
	var mu sync.Mutex
	inFlight, maximum := 0, 0
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan}, Results: store, Operations: store,
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			mu.Lock()
			inFlight++
			if inFlight > maximum {
				maximum = inFlight
			}
			mu.Unlock()
			started <- struct{}{}
			<-release
			mu.Lock()
			inFlight--
			mu.Unlock()
			return []byte(`{"runId":"run-bounded"}`), nil
		},
	})
	op, _, err := svc.StartValidation(context.Background(), "p1", "", "bounded")
	require.NoError(t, err)
	<-started
	mu.Lock()
	require.Equal(t, 1, maximum, "one claimed child per operation leaves capacity for competing operations")
	mu.Unlock()
	close(release)
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err = svc.GetValidationOperation(waitCtx, op.ID, true)
	require.NoError(t, err)
	require.True(t, op.Terminal())
	require.Len(t, op.Children, 6)
	mu.Lock()
	require.LessOrEqual(t, maximum, 4)
	mu.Unlock()
}

func TestDurableValidationUnavailableOracleCannotPublishPass(t *testing.T) { // [REQ:PM-VALID-004]
	store := newFakeDurableStore()
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store,
		Runner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if strings.HasPrefix(strings.Join(args, " "), "baseline diff ") {
				return nil, validation.ErrToolNotFound
			}
			return []byte(`{"runId":"snapshot-ready"}`), nil
		},
	})
	op, _, err := svc.StartValidation(context.Background(), "p1", "", "unknown")
	require.NoError(t, err)
	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	op, err = svc.GetValidationOperation(waitCtx, op.ID, true)
	require.NoError(t, err)
	require.True(t, op.Terminal())
	require.NotNil(t, op.Result)
	require.Equal(t, validation.VerdictUnknown, op.Result.Verdict)
}

// TestCaptureBaselineShellsGCTFromAnchorIntent proves CaptureBaseline derives the
// snapshot command from the typed anchor intent and degrades honestly when the
// intent is a placeholder or git-control-tower is absent.
func TestCaptureBaselineShellsGCTFromAnchorIntent(t *testing.T) {
	ctx := context.Background()

	// Healthy: anchor scenario + name set, runner present → captured.
	plan := planWith(nil, nil)
	plan.RegressionAnchor = internalplans.RegressionAnchor{Scenario: "plan-manager", BaselineName: "impl-baseline"}
	var calls []string
	captured := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan},
		Runner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, name+" "+strings.Join(args, " "))
			if len(args) >= 3 && args[0] == "baseline" && args[1] == "snapshot" && args[2] == "status" {
				return []byte(`{"status":"ready","baseline":{"schemaVersion":2,"run":{"runId":"run-v2"}}}`), nil
			}
			return []byte("snapshot started"), nil
		},
	})
	cap1, err := captured.CaptureBaseline(ctx, "p1")
	require.NoError(t, err)
	require.True(t, cap1.Captured)
	require.Equal(t, "plan-manager", cap1.Scenario)
	require.Equal(t, "impl-baseline", cap1.BaselineName)
	require.Equal(t, "run-v2", cap1.RunID)
	require.Equal(t, 2, cap1.SchemaVersion)
	require.Len(t, calls, 2)
	require.Contains(t, calls[0], "git-control-tower baseline snapshot --scenario plan-manager --name impl-baseline")
	require.Contains(t, calls[1], "git-control-tower baseline snapshot status --scenario plan-manager --name impl-baseline --wait --json")

	// Placeholder scenario → honest degradation, never a fabricated capture.
	placeholder := planWith(nil, nil)
	placeholder.RegressionAnchor = internalplans.RegressionAnchor{Scenario: "<scenario>", BaselineName: "x"}
	deg := validation.NewService(validation.Deps{
		Plans:  fakePlans{plan: placeholder},
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) { return nil, nil },
	})
	cap2, err := deg.CaptureBaseline(ctx, "p1")
	require.NoError(t, err)
	require.False(t, cap2.Captured)
	require.Contains(t, cap2.Detail, "placeholder")

	// No runner configured → honest degradation (git-control-tower unavailable).
	noRunner := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	cap3, err := noRunner.CaptureBaseline(ctx, "p1")
	require.NoError(t, err)
	require.False(t, cap3.Captured)
	require.Contains(t, cap3.Detail, "git-control-tower unavailable")
}

func TestCaptureBaselineUsesTypedCollectionForBaselineSet(t *testing.T) {
	plan := planWith(nil, nil)
	plan.BaselineSet = planmodel.BaselineSetIntent{
		Name: "before-all", ScenarioTargets: []string{"git-control-tower", "plan-manager"}, RepoPaths: []string{"packages/proto/**"},
		CapturePolicy: planmodel.BaselineCapturePolicyExecutionStart, Compatibility: planmodel.BaselineSetCompatibilityCurrent,
	}
	client := &fakeCollectionClient{result: validation.BaselineCollectionCaptureResult{Name: "before-all", Required: 2, Ready: 2}}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}, Collections: client})
	capture, err := svc.CaptureBaseline(context.Background(), "p1")
	require.NoError(t, err)
	require.True(t, capture.Captured)
	require.Equal(t, "before-all", capture.BaselineName)
	require.Equal(t, 1, client.calls)
	require.Equal(t, []string{"git-control-tower", "plan-manager"}, client.called.Scenarios)
	require.Equal(t, []string{"packages/proto/**"}, client.called.RepoPaths)

	client.result = validation.BaselineCollectionCaptureResult{Name: "before-all", Required: 2, Ready: 1, Pending: 1}
	capture, err = svc.CaptureBaseline(context.Background(), "p1")
	require.NoError(t, err)
	require.False(t, capture.Captured)
	require.Contains(t, capture.Detail, "coverage incomplete")
}

func TestCaptureBaselineRejectsLegacySurfaceManifestWithoutV2RunAnchor(t *testing.T) {
	ctx := context.Background()
	plan := planWith(nil, nil)
	plan.RegressionAnchor = internalplans.RegressionAnchor{Scenario: "audio-tools", BaselineName: "stt-scaling-analysis-plan"}
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan},
		Runner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if len(args) >= 3 && args[0] == "baseline" && args[1] == "snapshot" && args[2] == "status" {
				return []byte(`{"status":"ready","baseline":{"surfaces":{},"skipped":{"tests":"test-genie unreachable","rules":"api-core discovery failed"}}}`), nil
			}
			return []byte("snapshot started"), nil
		},
	})

	cap, err := svc.CaptureBaseline(ctx, "p1")
	require.NoError(t, err)
	require.False(t, cap.Captured)
	require.Empty(t, cap.RunID)
	require.Contains(t, cap.Detail, "V2 run anchor")
	require.Contains(t, cap.Detail, "schema=0")
}

func TestCaptureBaselineRejectsUnverifiableGCTBaseline(t *testing.T) {
	ctx := context.Background()
	plan := planWith(nil, nil)
	plan.RegressionAnchor = internalplans.RegressionAnchor{Scenario: "audio-tools", BaselineName: "stt-scaling-analysis-plan"}
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan},
		Runner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if len(args) >= 3 && args[0] == "baseline" && args[1] == "snapshot" && args[2] == "status" {
				return nil, errors.New("connect: unavailable")
			}
			return []byte("snapshot started"), nil
		},
	})

	cap, err := svc.CaptureBaseline(ctx, "p1")
	require.NoError(t, err)
	require.False(t, cap.Captured)
	require.Contains(t, cap.Detail, "finalization failed")
	require.Contains(t, cap.Detail, "not yet a usable oracle")
}

func TestRunValidationIncludesCommandReferenceFindings(t *testing.T) {
	plan := planWith(nil, []internalplans.Phase{{
		ID:         "ph1",
		Steps:      []string{"Validate command references."},
		Validation: "plan-manager validate run p1 --phase ph1",
		Acceptance: "Invalid command references are reported.",
		Intent: strings.Join([]string{
			"Run `cli:vrooli scenario test cli-health`.",
			"Fix `cli:knowledge-observatory docs healt cli-health`.",
			"Document `cli[future]:future-tool launch`.",
		}, "\n"),
		RelevantContext: []internalplans.RelevantContextItem{{
			Kind:         internalplans.RelevantContextCommand,
			Reason:       "Find relevant setup actions.",
			Instruction:  "Discover candidate actions before editing.",
			Command:      "prompt-manager discover plan-manager --type all",
			RepeatPolicy: internalplans.RelevantContextPhaseEntry,
		}},
	}})
	validator := &fakeCommandValidator{results: map[string]validation.CommandReferenceResult{
		"vrooli scenario test cli-health": {
			Verdict:         "partial",
			ValidationLevel: "command_exists",
			Issues:          []validation.CommandIssue{{Code: "argument_schema_unavailable", Message: "arguments unavailable"}},
		},
		"knowledge-observatory docs healt cli-health": {
			Verdict:         "invalid",
			ValidationLevel: "owner_identified",
			Issues:          []validation.CommandIssue{{Code: "unknown_command", Message: "command path was not found"}},
			Suggestions:     []string{"knowledge-observatory docs health"},
			Guidance:        []string{"Fix the command to a current catalog command."},
		},
		"prompt-manager discover plan-manager --type all": {
			Verdict:         "valid",
			ValidationLevel: "argument_shape_validated",
		},
	}}
	svc := validation.NewService(validation.Deps{
		Plans:    fakePlans{plan: plan},
		Commands: validator,
	})

	res, err := svc.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictFail, res.Verdict)
	require.Len(t, res.CommandFindings, 2, "future refs and valid structured context commands should not become findings")
	require.Equal(t, []string{"unknown_command"}, res.CommandFindings[1].IssueCodes)
	require.Equal(t, []string{"knowledge-observatory docs health"}, res.CommandFindings[1].Suggestions)
	require.Equal(t, []string{"Fix the command to a current catalog command."}, res.CommandFindings[1].Guidance)
	require.Len(t, validator.calls, 3)
	require.Equal(t, "prompt-manager discover plan-manager --type all", validator.calls[2].CommandText)
	require.Contains(t, res.Detail, "command reference validation")
}

func TestRunValidationFailsMalformedRelevantContextStructure(t *testing.T) {
	plan := planWith(nil, []internalplans.Phase{{
		ID:         "ph1",
		Intent:     "Implement the change",
		Steps:      []string{"Run the setup command."},
		Validation: "go test ./internal/validation",
		Acceptance: "Validation reports malformed context.",
		RelevantContext: []internalplans.RelevantContextItem{{
			Kind:     internalplans.RelevantContextCommand,
			Required: true,
			Command:  "vrooli help",
		}},
	}})
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})

	res, err := svc.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictFail, res.Verdict)
	require.Contains(t, res.Detail, "relevant context structure validation")
	require.Contains(t, res.Detail, "required context item has no repeat policy")
	require.Contains(t, res.Detail, "command/search context item has no reason")
	require.Contains(t, res.Detail, "command/search context item has no instruction")
	require.NotEmpty(t, res.CommandFindings)
	require.Contains(t, res.CommandFindings[0].IssueCodes, "missing_repeat_policy")
}

func TestRunValidationRequiresPhaseContextOrExplicitNoContext(t *testing.T) {
	noContext := planWith(nil, []internalplans.Phase{{
		ID:         "ph1",
		Intent:     "Implement the change",
		Steps:      []string{"Implement the change."},
		Validation: "go test ./internal/validation",
		Acceptance: "Validation passes.",
	}})
	fail := validation.NewService(validation.Deps{Plans: fakePlans{plan: noContext}})
	res, err := fail.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictFail, res.Verdict)
	require.Contains(t, res.Detail, "phase has no relevant context")

	explicit := planWith(nil, []internalplans.Phase{{
		ID:         "ph1",
		Intent:     "Trivial metadata update",
		Steps:      []string{"Update generated labels."},
		Validation: "go test ./internal/validation",
		Acceptance: "Generated labels remain consistent.",
		Reminders:  []string{"NO_CONTEXT: phase only updates generated labels."},
	}})
	pass := validation.NewService(validation.Deps{Plans: fakePlans{plan: explicit}})
	res, err = pass.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.NotEqual(t, validation.VerdictFail, res.Verdict)
	require.NotContains(t, res.Detail, "phase has no relevant context")
}

func TestRunValidationFlagsPlanQualityGaps(t *testing.T) {
	thin := planWith(nil, []internalplans.Phase{{
		ID:     "ph1",
		Intent: "Only a title and intent survived import.",
		RelevantContext: []internalplans.RelevantContextItem{{
			Kind:         internalplans.RelevantContextNote,
			Required:     true,
			RepeatPolicy: internalplans.RelevantContextPhaseEntry,
			Instruction:  "NO_CONTEXT: fixture focuses on phase quality.",
		}},
	}})
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: thin}})

	res, err := svc.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictFail, res.Verdict)
	require.Contains(t, res.Detail, "plan quality validation")
	requireFindingCode(t, res.CommandFindings, "phase_missing_steps")
	requireFindingCode(t, res.CommandFindings, "phase_missing_validation")
	requireFindingCode(t, res.CommandFindings, "phase_missing_acceptance")
}

func TestRunValidationFlagsMalformedMigratedContextQuality(t *testing.T) {
	plan := planWith(nil, []internalplans.Phase{{
		ID:         "ph1",
		Intent:     "Repair migrated setup.",
		Steps:      []string{"Inspect setup context."},
		Validation: "plan-manager validate run p1 --phase ph1",
		Acceptance: "Malformed migrated setup is reported.",
		RelevantContext: []internalplans.RelevantContextItem{
			{
				Kind:         internalplans.RelevantContextDoc,
				Source:       internalplans.RelevantContextSourceMigrated,
				Required:     true,
				RepeatPolicy: internalplans.RelevantContextPhaseEntry,
				Command:      "sed -n '1,220p' sed -n '1,260p' docs/concepts/PLAN-MODEL.md",
				Target:       "docs/concepts/PLAN-MODEL.md",
			},
			{
				Kind:         internalplans.RelevantContextNote,
				Source:       internalplans.RelevantContextSourceMigrated,
				Required:     true,
				RepeatPolicy: internalplans.RelevantContextPhaseEntry,
				Label:        "```bash",
				Instruction:  "Load or inspect this context before implementation work.",
			},
		},
	}})
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})

	res, err := svc.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictFail, res.Verdict)
	requireFindingCode(t, res.CommandFindings, "migrated_context_malformed_sed")
	requireFindingCode(t, res.CommandFindings, "migrated_context_markdown_fence")
}

func TestRunValidationChecksRelevantContextReferences(t *testing.T) {
	plan := planWith(nil, []internalplans.Phase{{
		ID:         "ph1",
		Intent:     "Implement the change",
		Steps:      []string{"Resolve context references."},
		Validation: "plan-manager validate run p1 --phase ph1",
		Acceptance: "Context references are resolved or reported.",
		RelevantContext: []internalplans.RelevantContextItem{
			{
				Kind:         internalplans.RelevantContextDoc,
				Target:       "docs/concepts/PLAN-MODEL.md",
				Required:     true,
				RepeatPolicy: internalplans.RelevantContextPhaseEntry,
			},
			{
				Kind:         internalplans.RelevantContextCodeRef,
				Target:       "missing.go",
				Required:     true,
				RepeatPolicy: internalplans.RelevantContextPhaseEntry,
			},
		},
	}})
	svc := validation.NewService(validation.Deps{
		Plans:    fakePlans{plan: plan},
		Resolver: fakeResolver{resolution: internalplans.ResolutionResolved},
	})
	res, err := svc.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.NotEqual(t, validation.VerdictFail, res.Verdict)

	missing := validation.NewService(validation.Deps{
		Plans:    fakePlans{plan: plan},
		Resolver: fakeResolver{resolution: internalplans.ResolutionMissing},
	})
	res, err = missing.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictFail, res.Verdict)
	require.Contains(t, res.Detail, "relevant context reference validation")
	require.Len(t, res.CommandFindings, 2)
	require.Equal(t, []string{"context_reference_unresolved"}, res.CommandFindings[0].IssueCodes)
	require.Equal(t, "phase.ph1.relevant_context[0].target", res.CommandFindings[0].Location)
}

func TestRunValidationUnknownWhenRelevantContextReferenceResolverUnavailable(t *testing.T) {
	plan := planWith(nil, []internalplans.Phase{{
		ID:         "ph1",
		Intent:     "Implement the change",
		Steps:      []string{"Resolve context references."},
		Validation: "plan-manager validate run p1 --phase ph1",
		Acceptance: "Resolver outage is reported as unknown.",
		RelevantContext: []internalplans.RelevantContextItem{{
			Kind:         internalplans.RelevantContextReqRef,
			Target:       "PM-CTX-001",
			Required:     true,
			RepeatPolicy: internalplans.RelevantContextPhaseEntry,
		}},
	}})
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})
	res, err := svc.RunValidation(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Equal(t, validation.VerdictUnknown, res.Verdict)
	require.Contains(t, res.Detail, "relevant context reference validation unknown")
	require.Len(t, res.CommandFindings, 1)
	require.Equal(t, "reference resolver unavailable", res.CommandFindings[0].Message)
}

// TestVerdictHonestyByExitClass pins the corrected verdict model: a missing tool
// is UNKNOWN (not FAIL — git-control-tower being absent must not look like a
// regression), a baseline-diff exit 2 ("not comparable") is UNKNOWN, an exit 1 is
// FAIL, and an informational-only command set (a bare repo-level diff with no
// oracle) is UNKNOWN even when the command exits 0 — never a false PASS.
func TestVerdictHonestyByExitClass(t *testing.T) {
	scenarioPlan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/x.go"}}, nil)
	scenarioPlan.RegressionAnchor = internalplans.RegressionAnchor{BaselineName: "impl"}
	repoOnlyPlan := planWith([]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "packages/api-core/x.go"}}, nil)

	cases := []struct {
		name   string
		plan   internalplans.Plan
		runner validation.CommandRunner
		want   validation.Verdict
	}{
		{
			name:   "tool absent => UNKNOWN not FAIL",
			plan:   scenarioPlan,
			runner: func(context.Context, string, ...string) ([]byte, error) { return nil, validation.ErrToolNotFound },
			want:   validation.VerdictUnknown,
		},
		{
			name: "baseline exit 2 (not comparable) => UNKNOWN",
			plan: scenarioPlan,
			runner: func(context.Context, string, ...string) ([]byte, error) {
				return nil, validation.CommandExitError{Code: 2}
			},
			want: validation.VerdictUnknown,
		},
		{
			name: "baseline exit 1 (regression) => FAIL",
			plan: scenarioPlan,
			runner: func(context.Context, string, ...string) ([]byte, error) {
				return nil, validation.CommandExitError{Code: 1}
			},
			want: validation.VerdictFail,
		},
		{
			name:   "informational-only repo diff (exit 0, no oracle) => UNKNOWN",
			plan:   repoOnlyPlan,
			runner: func(context.Context, string, ...string) ([]byte, error) { return []byte("M x.go"), nil },
			want:   validation.VerdictUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: tc.plan}, Runner: tc.runner})
			res, err := svc.RunValidation(context.Background(), "p1", "")
			require.NoError(t, err)
			require.Equal(t, tc.want, res.Verdict)
		})
	}
}

// TestVerifyDoDDerivesFromReferences pins the authoring→DoD fix: a wizard-authored
// plan carries the anchor as captured prose with NO explicit commands, yet DoD
// must still verify against a real oracle derived from the plan's connected code
// (it used to always degrade to UNKNOWN/not-met).
// [REQ:PM-VALID-002]
func TestVerifyDoDDerivesFromReferences(t *testing.T) {
	wizardPlan := internalplans.Plan{
		ID:    "p1",
		Slug:  "p1",
		Title: "Wizard plan",
		References: []internalplans.Reference{
			{Kind: internalplans.ReferenceCode, Target: "scenarios/foo/api/main.go"},
		},
		// Captured prose anchor, NO commands — exactly what the authoring wizard writes.
		RegressionAnchor: internalplans.RegressionAnchor{Strategy: "captured", BaselineName: "impl"},
	}
	var ran []string
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: wizardPlan},
		Runner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			ran = append(ran, name)
			return nil, nil // oracle exits 0 => DoD met
		},
	})
	res, ok, err := svc.VerifyDefinitionOfDone(context.Background(), "p1")
	require.NoError(t, err)
	require.True(t, ok, "DoD derives an oracle from references when the anchor has no commands")
	require.Equal(t, validation.VerdictPass, res.Verdict)
	require.NotEmpty(t, ran, "a baseline command was actually derived and run")
	require.Contains(t, res.CommandsRun, "git-control-tower baseline diff --scenario foo --name impl --wait --json")
}

func TestVerifyDefinitionOfDone(t *testing.T) {
	plan := internalplans.Plan{
		ID:               "p1",
		Slug:             "p1",
		RegressionAnchor: internalplans.RegressionAnchor{Commands: []string{"git-control-tower baseline diff --scenario foo --name impl --wait --json"}},
	}

	// DoD met: oracle exits 0.
	met := validation.NewService(validation.Deps{
		Plans:  fakePlans{plan: plan},
		Runner: func(_ context.Context, _ string, _ ...string) ([]byte, error) { return nil, nil },
	})
	res, ok, err := met.VerifyDefinitionOfDone(context.Background(), "p1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, validation.VerdictPass, res.Verdict)

	// Anchor unavailable => UNKNOWN, not met (never a false pass).
	noAnchor := internalplans.Plan{ID: "p1", Slug: "p1", RegressionAnchor: internalplans.RegressionAnchor{Unavailable: true}}
	degraded := validation.NewService(validation.Deps{Plans: fakePlans{plan: noAnchor}})
	res, ok, err = degraded.VerifyDefinitionOfDone(context.Background(), "p1")
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, validation.VerdictUnknown, res.Verdict)
}

func TestBaselineSetDefinitionOfDoneRequiresCheckpointAndUsesTypedFullCollectionDiff(t *testing.T) {
	plan := durableValidationPlan()
	plan.BaselineSet = planmodel.BaselineSetIntent{Name: "before", ScenarioTargets: []string{"foo", "bar"}}
	store := newFakeDurableStore()
	collections := &fakeCollectionClient{diffResult: validation.BaselineCollectionDiffResult{Classification: "clean", Detail: "foo:ready:clean, bar:ready:clean"}}
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan}, Results: store, Operations: store, Collections: collections,
		Inventories: fakeBaselineInventory{inventory: validation.BaselineInventory{Name: "before", ScenarioTargets: []string{"foo", "bar"}, Complete: true}, ok: true},
		Runner:      func(context.Context, string, ...string) ([]byte, error) { return nil, nil },
	})
	res, ok, err := svc.VerifyDefinitionOfDone(context.Background(), "p1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, validation.VerdictPass, res.Verdict)
	require.Equal(t, []string{"bar", "foo"}, collections.diffCalled.Scenarios)

	incomplete := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan}, Results: store, Operations: store, Collections: collections,
		Inventories: fakeBaselineInventory{inventory: validation.BaselineInventory{Name: "before", ScenarioTargets: []string{"foo", "bar"}}, ok: true},
	})
	res, ok, err = incomplete.VerifyDefinitionOfDone(context.Background(), "p1")
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, validation.VerdictUnknown, res.Verdict)
}

func TestPhaseScopedReferencesAndNotFound(t *testing.T) {
	plan := planWith(
		[]internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "plan-level.go"}},
		[]internalplans.Phase{{ID: "ph1", References: []internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "phase.go", Future: true}}}},
	)
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}})

	report, err := svc.ResolveReferences(context.Background(), "p1", "ph1")
	require.NoError(t, err)
	require.Len(t, report.References, 1)
	require.Equal(t, "phase.go", report.References[0].Target)

	_, err = svc.ResolveReferences(context.Background(), "p1", "missing")
	require.Error(t, err)
}

func TestFileResolverAndExistenceStaleness(t *testing.T) {
	// FileResolver over a fake stat: a present file => resolved/fresh; an absent
	// one => missing/definitely-stale (the moved/deleted tier).
	present := map[string]bool{filepath.Join("/repo", "exists.go"): true}
	stat := func(path string) (os.FileInfo, error) {
		if present[path] {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}
	r := validation.FileResolver{Root: "/repo", Stat: stat}

	resolved, err := r.Resolve(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "exists.go"})
	require.NoError(t, err)
	require.Equal(t, internalplans.ResolutionResolved, resolved.Resolution)

	missing, err := r.Resolve(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "gone.go"})
	require.NoError(t, err)
	require.Equal(t, internalplans.ResolutionMissing, missing.Resolution)

	// REQ references aren't filesystem-resolvable; pass through unspecified.
	req, err := r.Resolve(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceReq, Target: "OT-1"})
	require.NoError(t, err)
	require.Equal(t, internalplans.ResolutionUnspecified, req.Resolution)

	s := validation.NewExistenceStaleness(r)
	tier, factor, err := s.Compute(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "gone.go"})
	require.NoError(t, err)
	require.Equal(t, internalplans.StalenessDefinitelyStale, tier)
	require.InDelta(t, 1.0, factor, 0.001)

	freshTier, _, err := s.Compute(context.Background(), internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "exists.go"})
	require.NoError(t, err)
	require.Equal(t, internalplans.StalenessFresh, freshTier)
}
