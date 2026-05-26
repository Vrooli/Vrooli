package runs

import (
	"time"

	sharedruns "test-genie/internal/shared/runs"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// Per-phase verdict values (also the surface-diff vocabulary in Plan A).
const (
	verdictClean         = "clean"
	verdictRegression    = "regression"
	verdictNewFailure    = "new-failure"
	verdictPreexisting   = "preexisting"
	verdictNotComparable = "not-comparable"
)

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func toRunInfo(r sharedruns.RunRecord) *runspb.RunInfo {
	phases := make([]*runspb.PhaseInfo, 0, len(r.Phases))
	for _, p := range r.Phases {
		phases = append(phases, &runspb.PhaseInfo{
			Name:            p.Name,
			Status:          p.Status,
			DurationSeconds: float64(p.DurationSeconds),
		})
	}
	pins := make([]*runspb.PinInfo, 0, len(r.Pins))
	for _, p := range r.Pins {
		pins = append(pins, &runspb.PinInfo{
			PinnedBy: p.PinnedBy,
			PinnedAt: formatTime(p.PinnedAt),
			Reason:   p.Reason,
		})
	}
	return &runspb.RunInfo{
		RunId:           r.RunID,
		Scenario:        r.Scenario,
		StartedAt:       formatTime(r.StartedAt),
		CompletedAt:     formatTime(r.CompletedAt),
		Status:          r.Status,
		Phases:          phases,
		GitSha:          r.GitSha,
		GitBranch:       r.GitBranch,
		GitDirty:        r.GitDirty,
		GitDirtySummary: r.GitDirtySummary,
		Diagnostics: &runspb.DiagnosticsInfo{
			Video:   r.Diagnostics.Video,
			Console: r.Diagnostics.Console,
			Network: r.Diagnostics.Network,
			Har:     r.Diagnostics.HAR,
			Trace:   r.Diagnostics.Trace,
			Dom:     r.Diagnostics.DOM,
		},
		Pins: pins,
	}
}

func phaseStatusMap(r sharedruns.RunRecord) map[string]string {
	m := make(map[string]string, len(r.Phases))
	for _, p := range r.Phases {
		m[p.Name] = p.Status
	}
	return m
}

// isFailed reports whether a phase status counts as a failure for diffing.
// Empty (absent) is handled by the caller.
func isFailed(status string) bool {
	return status == "failed"
}

// comparePhases classifies each phase between baseline run A and current run B.
// Comparison is phase-level: a phase passing in A and failing in B is a
// regression; failing in B but absent in A is a new failure; failing in both is
// preexisting; failing in A and passing in B is cleared.
func comparePhases(a, b sharedruns.RunRecord, phaseFilter string) *runspb.CompareRunsResponse {
	statusA := phaseStatusMap(a)
	statusB := phaseStatusMap(b)

	// Stable phase ordering: B's phases first (current run), then any A-only.
	seen := make(map[string]bool)
	var order []string
	for _, p := range b.Phases {
		if !seen[p.Name] {
			order = append(order, p.Name)
			seen[p.Name] = true
		}
	}
	for _, p := range a.Phases {
		if !seen[p.Name] {
			order = append(order, p.Name)
			seen[p.Name] = true
		}
	}

	out := make([]*runspb.PhaseDiff, 0, len(order))
	worst := verdictClean
	for _, name := range order {
		if phaseFilter != "" && name != phaseFilter {
			continue
		}
		sa, okA := statusA[name]
		sb, okB := statusB[name]

		diff := &runspb.PhaseDiff{Phase: name, StatusA: sa, StatusB: sb}

		switch {
		case !okB:
			// Phase only present in baseline; nothing to judge in current run.
			diff.Verdict = verdictNotComparable
		case isFailed(sb) && okA && !isFailed(sa) && sa != "":
			diff.Verdict = verdictRegression
			diff.Regressions = []string{name}
		case isFailed(sb) && !okA:
			diff.Verdict = verdictNewFailure
			diff.NewFailures = []string{name}
		case isFailed(sb) && isFailed(sa):
			diff.Verdict = verdictPreexisting
			diff.PreexistingFailures = []string{name}
		case !isFailed(sb) && isFailed(sa):
			diff.Verdict = verdictClean
			diff.ClearedFailures = []string{name}
		default:
			diff.Verdict = verdictClean
		}

		worst = worsen(worst, diff.Verdict)
		out = append(out, diff)
	}

	return &runspb.CompareRunsResponse{Phases: out, Verdict: worst}
}

// worsen returns the more severe of two verdicts for the overall summary.
// Severity (high→low): regression > not-comparable > new-failure > preexisting > clean.
func worsen(current, next string) string {
	return maxRank(current, next, map[string]int{
		verdictClean:         0,
		verdictPreexisting:   1,
		verdictNewFailure:    2,
		verdictNotComparable: 3,
		verdictRegression:    4,
	})
}

func maxRank(a, b string, rank map[string]int) string {
	if rank[b] > rank[a] {
		return b
	}
	return a
}
