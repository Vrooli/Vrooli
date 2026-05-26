package baseline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// auditorAdapter backs the structure and rules surfaces. Capture runs a
// scenario-auditor scan and stores the findings as a GCT-local IssueSnapshot;
// Diff re-scans the current tree and classifies the violation-set delta.
//
// Classification follows the same "passing in baseline, failing now =
// regression" rule as test-genie: a violation absent at baseline but present
// now means a file that was clean now violates — a regression.
type auditorAdapter struct {
	surfaceID string
	scanType  string // scenario-auditor scan type ("structure", or "" for all standards)
	auditor   Auditor
	snaps     *SnapshotStore
}

// NewStructureAdapter builds the structure surface adapter (auditor "structure"
// scan).
func NewStructureAdapter(auditor Auditor, snaps *SnapshotStore) SurfaceAdapter {
	return &auditorAdapter{surfaceID: SurfaceStructure, scanType: "structure", auditor: auditor, snaps: snaps}
}

// NewRulesAdapter builds the rules surface adapter (all-standards scan).
func NewRulesAdapter(auditor Auditor, snaps *SnapshotStore) SurfaceAdapter {
	return &auditorAdapter{surfaceID: SurfaceRules, scanType: "", auditor: auditor, snaps: snaps}
}

func (a *auditorAdapter) ID() string { return a.surfaceID }

func (a *auditorAdapter) Available(_ context.Context, _ Target) bool {
	return a.auditor != nil && a.snaps != nil
}

func (a *auditorAdapter) Capture(ctx context.Context, t Target, _ CaptureOptions) (SurfacePointer, error) {
	issues, err := a.auditor.Scan(ctx, t.Scenario, a.scanType)
	if err != nil {
		return SurfacePointer{}, fmt.Errorf("%s capture: scan: %w", a.surfaceID, err)
	}
	id, err := a.snaps.Save(t.RepoID, IssueSnapshot{
		Surface:    a.surfaceID,
		Scenario:   t.Scenario,
		CapturedAt: time.Now().UTC(),
		Issues:     issues,
	})
	if err != nil {
		return SurfacePointer{}, fmt.Errorf("%s capture: save snapshot: %w", a.surfaceID, err)
	}
	raw, _ := json.Marshal(map[string]int{"violations": len(issues)})
	return SurfacePointer{
		SurfaceID:  a.surfaceID,
		Kind:       KindGCTLocalSnapshot,
		Ref:        id,
		CapturedAt: time.Now().UTC(),
		Summary:    raw,
	}, nil
}

func (a *auditorAdapter) Diff(ctx context.Context, t Target, ptr SurfacePointer) (SurfaceDiff, error) {
	base, err := a.snaps.Load(t.RepoID, t.Scenario, ptr.Ref)
	if err != nil {
		return notComparable(a.surfaceID, "baseline snapshot missing: "+err.Error()), nil
	}
	current, err := a.auditor.Scan(ctx, t.Scenario, a.scanType)
	if err != nil {
		return notComparable(a.surfaceID, "current scan failed: "+err.Error()), nil
	}
	return diffIssues(a.surfaceID, base.Issues, current), nil
}

// diffIssues classifies the delta between two violation sets keyed by
// Issue.Key.
func diffIssues(surfaceID string, baseline, current []Issue) SurfaceDiff {
	baseSet := map[string]Issue{}
	for _, i := range baseline {
		baseSet[i.Key] = i
	}
	curSet := map[string]Issue{}
	for _, i := range current {
		curSet[i.Key] = i
	}

	d := SurfaceDiff{SurfaceID: surfaceID, Verdict: VerdictClean}
	for k, iss := range curSet {
		if _, ok := baseSet[k]; ok {
			d.Preexisting = append(d.Preexisting, issueLabel(iss))
		} else {
			d.Regressions = append(d.Regressions, issueLabel(iss))
		}
	}
	for k, iss := range baseSet {
		if _, ok := curSet[k]; !ok {
			d.Cleared = append(d.Cleared, issueLabel(iss))
		}
	}
	sort.Strings(d.Regressions)
	sort.Strings(d.Preexisting)
	sort.Strings(d.Cleared)

	switch {
	case len(d.Regressions) > 0:
		d.Verdict = VerdictRegression
	case len(d.Preexisting) > 0:
		d.Verdict = VerdictPreexisting
	default:
		d.Verdict = VerdictClean
	}
	d.Summary = summarizeVerdict(d)
	return d
}

func issueLabel(i Issue) string {
	if i.FilePath != "" {
		return fmt.Sprintf("%s (%s)", i.Title, i.FilePath)
	}
	if i.Title != "" {
		return i.Title
	}
	return i.Key
}
