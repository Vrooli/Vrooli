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

	planmodel "plan-manager/internal/planmodel"
	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/validation"

	"github.com/stretchr/testify/require"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
)

// --- fakes ---

type fakePlans struct {
	plan internalplans.Plan
	err  error
}

func TestValidationDomainCannotReintroducePrivateCommandOrFreshnessExecution(t *testing.T) {
	entries, err := filepath.Glob("*.go")
	require.NoError(t, err)
	for _, path := range entries {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		content, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		for _, forbidden := range []string{"\"os/exec\"", "CommandRunner", "git diff --numstat"} {
			require.NotContainsf(t, string(content), forbidden, "%s reintroduced displaced lifecycle/freshness logic", path)
		}
	}
}

func TestCurrentValidationCompilerCannotReintroduceProviderCommandWalls(t *testing.T) {
	for _, path := range []string{"checks.go", "service.go", "receipts.go"} {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		for _, forbidden := range []string{
			"git-control-tower baseline collection diff --name",
			"git-control-tower baseline path diff --before",
		} {
			require.NotContainsf(t, string(content), forbidden, "%s reintroduced a caller-owned provider command wall", path)
		}
	}
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
	diffRead   validation.BaselineCollectionDiffRequest
	pathResult validation.BaselinePathDiffResult
	pathErr    error
	pathCalled validation.BaselinePathDiffRequest
	getName    string
	getBranch  string
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

func (f *fakeCollectionClient) GetCollection(_ context.Context, name, branch string) (validation.BaselineCollectionCaptureResult, error) {
	f.getName, f.getBranch = name, branch
	return f.result, f.err
}

func (f *fakeCollectionClient) GetCollectionDiff(_ context.Context, name, branch, operationID string) (validation.BaselineCollectionDiffResult, error) {
	_ = branch
	f.diffRead = validation.BaselineCollectionDiffRequest{Name: name, OperationID: operationID}
	return f.diffResult, f.diffErr
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

type fakeReceiptClient struct {
	intent  *validationv1.ValidationIntent
	receipt *validationv1.ValidationReceipt
}

func (f *fakeReceiptClient) CreateValidation(_ context.Context, intent *validationv1.ValidationIntent) (*validationv1.ValidationReceipt, error) {
	f.intent = intent
	if f.receipt == nil {
		f.receipt = &validationv1.ValidationReceipt{ReceiptId: "receipt-1", LineageId: "lineage-1", State: validationv1.ReceiptState_RECEIPT_STATE_QUEUED, Compatibility: &validationv1.CompatibilityDecision{Kind: validationv1.CompatibilityKind_COMPATIBILITY_KIND_NEW_WORK}}
	}
	return f.receipt, nil
}

func (f *fakeReceiptClient) GetValidation(context.Context, string) (*validationv1.ValidationReceipt, error) {
	return f.receipt, nil
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
	scope := op.PlanID + "\x00" + op.PhaseID + "\x00" + op.ExecutionID + "\x00" + fmt.Sprint(op.ScopeGeneration)
	key := scope + "\x00" + op.IdempotencyKey
	if op.IdempotencyKey != "" {
		if id := s.byKey[key]; id != "" {
			return cloneOperation(s.ops[id]), false, nil
		}
		s.byKey[key] = op.ID
	} else if id := s.byScope[scope]; id != "" && !s.ops[id].Terminal() {
		return cloneOperation(s.ops[id]), false, nil
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
	require.Empty(t, scope.Commands, "legacy per-scenario commands are never rendered for a collection-backed plan")
}

func TestDeriveBaselineScopeDefaultsToPhaseAffectedAreas(t *testing.T) {
	plan := planWith(nil, nil)
	plan.ChangeBoundary.AcceptanceAllow = []string{"scenarios/foo/**", "scenarios/bar/**"}
	plan.RegressionAnchor.BaselineName = "impl"
	plan.Phases = []internalplans.Phase{{
		ID: "phase-1", AffectedAreas: []string{"scenarios/foo/api/handler.go"},
	}}
	scope, err := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}}).DeriveBaselineScope(context.Background(), "p1", "phase-1")
	require.NoError(t, err)
	require.Equal(t, "derived", scope.Provenance)
	require.Contains(t, scope.Locations, "scenarios/foo")
	require.NotContains(t, scope.Locations, "scenarios/bar")
}

func TestDeriveBaselineScopeExplicitFullPlanRemainsWide(t *testing.T) {
	plan := planWith(nil, nil)
	plan.ChangeBoundary.AcceptanceAllow = []string{"scenarios/foo/**", "scenarios/bar/**"}
	plan.RegressionAnchor.BaselineName = "impl"
	plan.Phases = []internalplans.Phase{{
		ID: "phase-1", AffectedAreas: []string{"scenarios/foo/api/handler.go"},
		ValidationScope: planmodel.ValidationScope{Mode: planmodel.ValidationScopeFullPlan, Rationale: "cross-scenario contract"},
	}}
	scope, err := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}}).DeriveBaselineScope(context.Background(), "p1", "phase-1")
	require.NoError(t, err)
	require.Equal(t, "explicit", scope.Provenance)
	require.Contains(t, scope.Locations, "scenarios/foo")
	require.Contains(t, scope.Locations, "scenarios/bar")
}

func TestDeriveBaselineScopeWithoutPhaseAreasFallsBackToPlan(t *testing.T) {
	plan := planWith(nil, nil)
	plan.ChangeBoundary.AcceptanceAllow = []string{"scenarios/foo/**"}
	plan.Phases = []internalplans.Phase{{ID: "phase-1"}}
	scope, err := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}}).DeriveBaselineScope(context.Background(), "p1", "phase-1")
	require.NoError(t, err)
	require.Equal(t, "plan", scope.Provenance)
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
func TestDirectValidationRequiresProducerTicket(t *testing.T) {
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: planWith(nil, nil)}})
	_, err := svc.RunValidation(context.Background(), "p1", "")
	var ticketRequired validation.ErrProducerTicketRequired
	require.ErrorAs(t, err, &ticketRequired)
	require.Equal(t, "p1", ticketRequired.PlanID)

	_, _, err = svc.VerifyDefinitionOfDone(context.Background(), "p1")
	require.ErrorAs(t, err, &ticketRequired)
}

func TestSyncBaselineUsesExecutionAdoptedCollectionTicket(t *testing.T) {
	plan := durableValidationPlan()
	plan.BaselineSet = planmodel.BaselineSetIntent{Name: "authored-before", ScenarioTargets: []string{"foo"}}
	collections := &fakeCollectionClient{result: validation.BaselineCollectionCaptureResult{
		Name: "recaptured-before", Branch: "agi", Required: 1, Ready: 1,
		Members: []validation.BaselineCollectionMember{{Scenario: "foo", Required: true, Status: "ready"}},
	}}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: plan}, Collections: collections})

	capture, err := svc.SyncBaseline(context.Background(), "p1", "recaptured-before")
	require.NoError(t, err)
	require.True(t, capture.Captured)
	require.Equal(t, "recaptured-before", capture.BaselineName)
	require.Equal(t, "recaptured-before", collections.getName)
}

func durableValidationPlan() internalplans.Plan {
	plan := planWith(nil, nil)
	plan.ChangeBoundary.AcceptanceAllow = []string{"scenarios/foo/**"}
	plan.RegressionAnchor = internalplans.RegressionAnchor{
		Strategy: internalplans.AnchorStrategyChangeBoundary, BaselineName: "impl",
	}
	plan.BaselineSet = planmodel.BaselineSetIntent{Name: "impl", ScenarioTargets: []string{"foo"}}
	return plan
}

func TestValidationTicketUsesCanonicalReceiptWithoutProducerCommandWall(t *testing.T) { // [REQ:PM-VALID-005]
	store := newFakeDurableStore()
	receipts := &fakeReceiptClient{}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store, Receipts: receipts})
	op, reused, err := svc.StartValidationTicket(context.Background(), validation.ValidationTicketRequest{PlanID: "p1", PhaseID: "", ExecutionID: "execution-1", ScopeGeneration: 3, IdempotencyKey: "canonical"})
	require.NoError(t, err)
	require.False(t, reused)
	require.Equal(t, "receipt-1", op.ID)
	require.Empty(t, op.Children, "Plan Manager must not own producer child commands")
	require.Equal(t, []string{"test-genie", "validation", "wait", "--wait-id", "plan-manager-receipt-1", "receipt-1", "--json"}, op.ProducerWaitArgv)
	require.Equal(t, validationv1.ValidationStrength_VALIDATION_STRENGTH_CERTIFICATION, receipts.intent.GetRequiredStrength())
	require.Equal(t, validationv1.ValidationPurpose_VALIDATION_PURPOSE_CERTIFICATION, receipts.intent.GetPurpose())
	require.Len(t, receipts.intent.GetTargets(), 1)
	require.NotEmpty(t, receipts.intent.GetContentInputs())

	receipts.receipt = &validationv1.ValidationReceipt{ReceiptId: "receipt-1", LineageId: "lineage-1", State: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, AchievedStrength: validationv1.ValidationStrength_VALIDATION_STRENGTH_CERTIFICATION, ReasonCode: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE, Detail: "certified"}
	op, err = svc.SyncValidation(context.Background(), op.ID)
	require.NoError(t, err)
	require.True(t, op.Terminal())
	require.Equal(t, validation.VerdictPass, op.Result.Verdict)
	require.Equal(t, 3, op.Result.ScopeGeneration)
}

func TestValidationTicketDerivesStableReceiptIdempotencyFromCompiledScope(t *testing.T) { // [REQ:PM-VALID-005]
	store := newFakeDurableStore()
	receipts := &fakeReceiptClient{}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store, Receipts: receipts})

	request := validation.ValidationTicketRequest{PlanID: "p1", ExecutionID: "execution-1", ScopeGeneration: 3}
	first, _, err := svc.StartValidationTicket(context.Background(), request)
	require.NoError(t, err)
	firstKey := receipts.intent.GetIdempotencyKey()
	require.Regexp(t, `^plan-manager:[0-9a-f]{64}$`, firstKey)

	_, _, err = svc.StartValidationTicket(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, firstKey, receipts.intent.GetIdempotencyKey(), "a retry of the same compiled scope must attach")

	request.ScopeGeneration++
	_, _, err = svc.StartValidationTicket(context.Background(), request)
	require.NoError(t, err)
	require.NotEqual(t, firstKey, receipts.intent.GetIdempotencyKey(), "changed scope must create distinct validation work")
	require.NotEmpty(t, first.ID)
}

func TestValidationTicketRequiresCanonicalReceiptService(t *testing.T) { // [REQ:PM-VALID-004]
	store := newFakeDurableStore()
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store})
	_, _, err := svc.StartValidation(context.Background(), "p1", "", "missing-receipt-service")
	require.ErrorContains(t, err, "canonical Test Genie receipt service is unavailable")
}

func TestExecutionBoundValidationProjectsAdoptedInventoryIntoReceipt(t *testing.T) { // [REQ:PM-VALID-005]
	store := newFakeDurableStore()
	receipts := &fakeReceiptClient{}
	plan := durableValidationPlan()
	plan.BaselineSet = planmodel.BaselineSetIntent{Name: "authored-before", ScenarioTargets: []string{"foo", "bar"}}
	svc := validation.NewService(validation.Deps{
		Plans: fakePlans{plan: plan}, Operations: store, Results: store, Receipts: receipts,
		Inventories: fakeBaselineInventory{inventory: validation.BaselineInventory{Name: "recaptured-before", ScenarioTargets: []string{"foo"}, Complete: true}, ok: true},
	})
	op, _, err := svc.StartValidationTicket(context.Background(), validation.ValidationTicketRequest{PlanID: "p1", ExecutionID: "e1", IdempotencyKey: "captured-inventory"})
	require.NoError(t, err)
	require.Empty(t, op.Children)
	require.Equal(t, "recaptured-before", receipts.intent.GetCallerAttributes()["baseline_name"])
	require.Len(t, receipts.intent.GetTargets(), 1)
	require.Equal(t, "foo", receipts.intent.GetTargets()[0].GetId())
}

func TestValidationTicketHasNoPlanManagerProducerBudgetOrCommands(t *testing.T) { // [REQ:PM-VALID-004]
	store := newFakeDurableStore()
	receipts := &fakeReceiptClient{}
	svc := validation.NewService(validation.Deps{Plans: fakePlans{plan: durableValidationPlan()}, Results: store, Operations: store, Receipts: receipts})
	op, _, err := svc.StartValidation(context.Background(), "p1", "", "receipt-only")
	require.NoError(t, err)
	require.Zero(t, op.QueueBudgetSeconds)
	require.Zero(t, op.ExecutionBudgetSeconds)
	require.Zero(t, op.TransportWaitBudgetSeconds)
	require.Zero(t, op.RecommendedWaitSeconds)
	require.Empty(t, op.Children)
	require.Empty(t, op.Result.CommandsRun)
	require.Equal(t, "test-genie", op.ProducerWaitArgv[0])
}

func TestRunValidationIncludesCommandReferenceFindings(t *testing.T) {
	t.Skip("direct validation was intentionally replaced by producer-owned tickets; command-reference checks run during authoring")
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
	t.Skip("direct validation was intentionally replaced by producer-owned tickets; plan-quality checks run during authoring")
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
	t.Skip("direct validation was intentionally replaced by producer-owned tickets; plan-quality checks run during authoring")
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
	t.Skip("direct validation was intentionally replaced by producer-owned tickets; plan-quality checks run during authoring")
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
	t.Skip("direct validation was intentionally replaced by producer-owned tickets; plan-quality checks run during authoring")
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
	t.Skip("direct validation was intentionally replaced by producer-owned tickets; reference checks run during authoring")
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
	t.Skip("direct validation was intentionally replaced by producer-owned tickets; reference checks run during authoring")
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
