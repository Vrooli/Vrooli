package baseline

import (
	"time"

	bl "git-control-tower/internal/baseline"
	"git-control-tower/internal/git"

	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

// rfc3339 formats a time as RFC3339, returning "" for the zero value.
func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func gitToProto(st git.State) *baselinesv1.GitState {
	return &baselinesv1.GitState{
		Sha:           st.Sha,
		Branch:        st.Branch,
		Detached:      st.Detached,
		Dirty:         st.Dirty,
		DirtySummary:  st.DirtySummary,
		CommitMessage: st.CommitMessage,
		CommitAuthor:  st.CommitAuthor,
		CommitDate:    rfc3339(st.CommitDate),
		Sandboxed:     st.Sandboxed,
	}
}

func pointerToProto(p bl.SurfacePointer) *baselinesv1.SurfacePointer {
	return &baselinesv1.SurfacePointer{
		SurfaceId:  p.SurfaceID,
		Kind:       p.Kind,
		Ref:        p.Ref,
		CapturedAt: rfc3339(p.CapturedAt),
		Summary:    string(p.Summary),
	}
}

func manifestToProto(m bl.BaselineManifest) *baselinesv1.BaselineManifest {
	out := &baselinesv1.BaselineManifest{
		Name:          m.Name,
		Scenario:      m.Scenario,
		Branch:        m.Branch,
		CreatedAt:     rfc3339(m.CreatedAt),
		CreatedBy:     m.CreatedBy,
		Git:           gitToProto(m.Git),
		Surfaces:      make(map[string]*baselinesv1.SurfacePointer, len(m.Surfaces)),
		SchemaVersion: int32(m.SchemaVersion),
	}
	for id, ptr := range m.Surfaces {
		out.Surfaces[id] = pointerToProto(ptr)
	}
	if len(m.Skipped) > 0 {
		out.Skipped = make(map[string]string, len(m.Skipped))
		for id, reason := range m.Skipped {
			out.Skipped[id] = reason
		}
	}
	return out
}

// diffResultToProto renders a computed diff into the wire DiffResult message.
func diffResultToProto(res bl.DiffResult) *baselinesv1.DiffResult {
	out := &baselinesv1.DiffResult{
		Baseline:   manifestToProto(res.Manifest),
		CurrentGit: gitToProto(res.CurrentGit),
		Staleness: &baselinesv1.Staleness{
			CommitsSince: int32(res.Staleness.CommitsSince),
			FilesChanged: int32(res.Staleness.FilesChanged),
			LikelyStale:  res.Staleness.LikelyStale,
		},
		Verdict:      string(res.Verdict),
		DirtyWarning: res.DirtyWarning,
	}
	for _, d := range res.Surfaces {
		out.Surfaces = append(out.Surfaces, surfaceDiffToProto(d))
	}
	for _, d := range res.Phases {
		out.Phases = append(out.Phases, phaseDiffToProto(d))
	}
	return out
}

func surfaceDiffToProto(d bl.SurfaceDiff) *baselinesv1.SurfaceDiff {
	return &baselinesv1.SurfaceDiff{
		SurfaceId:   d.SurfaceID,
		Verdict:     string(d.Verdict),
		Regressions: d.Regressions,
		NewFailures: d.NewFailures,
		Preexisting: d.Preexisting,
		Cleared:     d.Cleared,
		Changed:     d.Changed,
		Summary:     d.Summary,
	}
}

func phaseDiffToProto(d bl.PhaseDetail) *baselinesv1.PhaseDiff {
	return &baselinesv1.PhaseDiff{
		Phase:       d.Phase,
		SurfaceId:   d.SurfaceID,
		Verdict:     string(d.Verdict),
		Regressions: d.Regressions,
		NewFailures: d.NewFailures,
		Preexisting: d.Preexisting,
		Cleared:     d.Cleared,
		Summary:     d.Summary,
	}
}
