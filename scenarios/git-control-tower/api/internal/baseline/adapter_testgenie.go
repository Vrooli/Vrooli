package baseline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// testGenieAdapter backs both the workflows and tests surfaces. Each captures
// a phase-scoped test-genie run, pins it to the baseline, and diffs by running
// the same phases now and asking RunsService.CompareRuns to classify the delta.
//
// Both surfaces run their own phase-scoped run (workflows: playbooks; tests:
// unit/integration/smoke), so a compare with an empty phase filter is already
// scoped to that surface's phases — no cross-surface bleed.
type testGenieAdapter struct {
	surfaceID string
	phases    []string
	fastDiag  string // diagnostics preset for --fast (and for the diff's current run)
	fullDiag  string // diagnostics preset for --full

	exec Executor
	runs RunsClient
}

// NewWorkflowsAdapter builds the workflows surface adapter. --fast captures
// with light diagnostics (no video); --full captures everything.
func NewWorkflowsAdapter(exec Executor, runs RunsClient) SurfaceAdapter {
	return &testGenieAdapter{
		surfaceID: SurfaceWorkflows,
		phases:    []string{"playbooks"},
		fastDiag:  "light",
		fullDiag:  "full",
		exec:      exec,
		runs:      runs,
	}
}

// NewTestsAdapter builds the tests surface adapter (unit/integration/smoke).
// Tests carry no browser diagnostics, so fast and full are both "none".
func NewTestsAdapter(exec Executor, runs RunsClient) SurfaceAdapter {
	return &testGenieAdapter{
		surfaceID: SurfaceTests,
		phases:    []string{"unit", "integration", "smoke"},
		fastDiag:  "none",
		fullDiag:  "none",
		exec:      exec,
		runs:      runs,
	}
}

// NewStructureAdapter builds the structure surface adapter, scoped to
// test-genie's "structure" phase (scenario layout, manifests, and JSON health).
// Structure is a static check with no browser diagnostics.
func NewStructureAdapter(exec Executor, runs RunsClient) SurfaceAdapter {
	return &testGenieAdapter{
		surfaceID: SurfaceStructure,
		phases:    []string{"structure"},
		fastDiag:  "none",
		fullDiag:  "none",
		exec:      exec,
		runs:      runs,
	}
}

// NewRulesAdapter builds the rules surface adapter, scoped to test-genie's
// "standards" phase. test-genie already runs scenario-auditor's standards rules
// inside that phase, so GCT pins the run rather than calling scenario-auditor
// directly (Decision 3 — test-genie owns runs, GCT owns baselines). Running the
// full test-genie suite therefore refreshes rules just like running just the
// rules surface runs only the standards phase — mirroring workflows↔playbooks.
func NewRulesAdapter(exec Executor, runs RunsClient) SurfaceAdapter {
	return &testGenieAdapter{
		surfaceID: SurfaceRules,
		phases:    []string{"standards"},
		fastDiag:  "none",
		fullDiag:  "none",
		exec:      exec,
		runs:      runs,
	}
}

func (a *testGenieAdapter) ID() string { return a.surfaceID }

func (a *testGenieAdapter) Available(_ context.Context, _ Target) bool {
	return a.exec != nil && a.runs != nil
}

// runSummary is the compact per-surface summary stored on the pointer.
type runSummary struct {
	RunID  string        `json:"run_id"`
	Passed int           `json:"passed"`
	Failed int           `json:"failed"`
	Phases []PhaseStatus `json:"phases"`
}

func (a *testGenieAdapter) Capture(ctx context.Context, t Target, opts CaptureOptions) (SurfacePointer, error) {
	diag := a.fullDiag
	if opts.Fast {
		diag = a.fastDiag
	}
	res, err := a.exec.Execute(ctx, t.Scenario, a.phases, diag)
	if err != nil {
		return SurfacePointer{}, fmt.Errorf("%s capture: execute run: %w", a.surfaceID, err)
	}
	if res.RunID == "" {
		return SurfacePointer{}, fmt.Errorf("%s capture: test-genie returned no runID", a.surfaceID)
	}
	if opts.PinnedBy != "" {
		if err := a.runs.PinRun(ctx, t.Scenario, res.RunID, opts.PinnedBy, "baseline:"+a.surfaceID); err != nil {
			return SurfacePointer{}, fmt.Errorf("%s capture: pin run %s: %w", a.surfaceID, res.RunID, err)
		}
	}

	sum := runSummary{RunID: res.RunID, Phases: res.Phases}
	for _, p := range res.Phases {
		switch p.Status {
		case "passed":
			sum.Passed++
		case "failed":
			sum.Failed++
		}
	}
	raw, _ := json.Marshal(sum)
	return SurfacePointer{
		SurfaceID:  a.surfaceID,
		Kind:       KindTestGenieRun,
		Ref:        res.RunID,
		CapturedAt: time.Now().UTC(),
		Summary:    raw,
	}, nil
}

func (a *testGenieAdapter) Diff(ctx context.Context, t Target, ptr SurfacePointer) (SurfaceDiff, error) {
	if ptr.Ref == "" {
		return notComparable(a.surfaceID, "baseline has no run reference"), nil
	}
	// Run the surface's phases against the current working tree, with light
	// diagnostics (diff is a diagnostic, not an archival capture).
	cur, err := a.exec.Execute(ctx, t.Scenario, a.phases, a.fastDiag)
	if err != nil {
		return notComparable(a.surfaceID, "could not run current "+a.surfaceID+": "+err.Error()), nil
	}
	if cur.RunID == "" {
		return notComparable(a.surfaceID, "current run produced no runID"), nil
	}
	cmp, err := a.runs.CompareRuns(ctx, t.Scenario, ptr.Ref, cur.RunID, "")
	if err != nil {
		return notComparable(a.surfaceID, "compare failed: "+err.Error()), nil
	}
	return compareToSurfaceDiff(a.surfaceID, cmp), nil
}

// compareToSurfaceDiff folds a multi-phase CompareResult into one SurfaceDiff.
func compareToSurfaceDiff(surfaceID string, cmp CompareResult) SurfaceDiff {
	d := SurfaceDiff{SurfaceID: surfaceID, Verdict: VerdictClean}
	for _, p := range cmp.Phases {
		d.Verdict = WorseVerdict(d.Verdict, Verdict(p.Verdict))
		d.Regressions = append(d.Regressions, p.Regressions...)
		d.NewFailures = append(d.NewFailures, p.NewFailures...)
		d.Preexisting = append(d.Preexisting, p.Preexisting...)
		d.Cleared = append(d.Cleared, p.Cleared...)
	}
	// Prefer the service-reported overall verdict when present (it uses the
	// same severity ordering as WorseVerdict).
	if cmp.Verdict != "" {
		d.Verdict = WorseVerdict(d.Verdict, Verdict(cmp.Verdict))
	}
	d.Summary = summarizeVerdict(d)
	return d
}

// notComparable builds a not-comparable SurfaceDiff with a reason.
func notComparable(surfaceID, reason string) SurfaceDiff {
	return SurfaceDiff{
		SurfaceID: surfaceID,
		Verdict:   VerdictNotComparable,
		Summary:   reason,
	}
}

// summarizeVerdict renders a one-line human summary for a surface diff.
func summarizeVerdict(d SurfaceDiff) string {
	switch d.Verdict {
	case VerdictClean:
		if len(d.Cleared) > 0 {
			return fmt.Sprintf("no regressions (%d cleared)", len(d.Cleared))
		}
		return "no change"
	case VerdictRegression:
		return fmt.Sprintf("%d regression(s)", len(d.Regressions))
	case VerdictNewFailure:
		return fmt.Sprintf("%d new failure(s) (added by your changes)", len(d.NewFailures))
	case VerdictPreexisting:
		return fmt.Sprintf("%d preexisting failure(s) (inherited)", len(d.Preexisting))
	case VerdictNotComparable:
		return "not comparable"
	}
	return string(d.Verdict)
}
