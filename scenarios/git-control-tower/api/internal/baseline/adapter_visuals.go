package baseline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// visualsAdapter backs the visuals surface. Capture reuses GCT's existing
// visual snapshot machinery; Diff is metadata-level only (page set + screenshot
// count) — pixel-level comparison stays in the UI (Plan B), per scope.
type visualsAdapter struct {
	vis VisualClient
}

// NewVisualsAdapter builds the visuals surface adapter.
func NewVisualsAdapter(vis VisualClient) SurfaceAdapter {
	return &visualsAdapter{vis: vis}
}

func (a *visualsAdapter) ID() string { return SurfaceVisuals }

func (a *visualsAdapter) Available(_ context.Context, _ Target) bool {
	return a.vis != nil
}

func (a *visualsAdapter) Capture(ctx context.Context, t Target, _ CaptureOptions) (SurfacePointer, error) {
	snap, err := a.vis.Capture(ctx, t.RepoID, t.RepoDir, t.Scenario)
	if err != nil {
		return SurfacePointer{}, fmt.Errorf("visuals capture: %w", err)
	}
	raw, _ := json.Marshal(map[string]any{
		"snapshot_id": snap.SnapshotID,
		"screenshots": snap.ScreenshotCount,
		"pages":       len(snap.Pages),
	})
	return SurfacePointer{
		SurfaceID:  SurfaceVisuals,
		Kind:       KindGCTLocalSnapshot,
		Ref:        snap.SnapshotID,
		CapturedAt: time.Now().UTC(),
		Summary:    raw,
	}, nil
}

func (a *visualsAdapter) Diff(ctx context.Context, t Target, ptr SurfacePointer) (SurfaceDiff, error) {
	base, ok, err := a.vis.Get(ctx, t.RepoID, t.Scenario, ptr.Ref)
	if err != nil {
		return notComparable(SurfaceVisuals, "load baseline snapshot: "+err.Error()), nil
	}
	if !ok {
		return notComparable(SurfaceVisuals, "baseline visual snapshot missing"), nil
	}
	current, err := a.vis.Capture(ctx, t.RepoID, t.RepoDir, t.Scenario)
	if err != nil {
		return notComparable(SurfaceVisuals, "current capture failed: "+err.Error()), nil
	}
	return diffVisuals(base, current), nil
}

// diffVisuals compares two visual snapshots at the metadata level. A page that
// existed at baseline but is no longer captured is a regression; new pages or
// changed screenshot counts are drift (new-failure — inspect side-by-side in
// the UI); identical structure is clean.
func diffVisuals(baseline, current VisualSnapshot) SurfaceDiff {
	d := SurfaceDiff{SurfaceID: SurfaceVisuals, Verdict: VerdictClean}

	basePages := map[string]bool{}
	for _, p := range baseline.Pages {
		basePages[p] = true
	}
	curPages := map[string]bool{}
	for _, p := range current.Pages {
		curPages[p] = true
	}
	for p := range basePages {
		if !curPages[p] {
			d.Regressions = append(d.Regressions, "page no longer captured: "+p)
		}
	}
	for p := range curPages {
		if !basePages[p] {
			d.NewFailures = append(d.NewFailures, "new page: "+p)
		}
	}
	if len(d.Regressions) == 0 && len(d.NewFailures) == 0 && baseline.ScreenshotCount != current.ScreenshotCount {
		d.NewFailures = append(d.NewFailures, fmt.Sprintf("screenshot count changed: %d → %d (inspect in UI)", baseline.ScreenshotCount, current.ScreenshotCount))
	}
	sort.Strings(d.Regressions)
	sort.Strings(d.NewFailures)

	switch {
	case len(d.Regressions) > 0:
		d.Verdict = VerdictRegression
	case len(d.NewFailures) > 0:
		d.Verdict = VerdictNewFailure
	default:
		d.Verdict = VerdictClean
	}
	d.Summary = summarizeVerdict(d)
	return d
}
