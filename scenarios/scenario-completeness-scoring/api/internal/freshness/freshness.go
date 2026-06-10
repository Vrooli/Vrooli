// Package freshness labels score output with the tree digest it was
// computed against and per-phase fresh/stale/unknown verdicts.
//
// All semantics live in the shared packages/freshness-go module (digest
// spec, required phase set, verdict logic, suggested-command format) and
// are reused verbatim — this package only orchestrates the calls and
// enriches verdicts with run context for display. No local freshness
// policy may be added here (operator anti-gaming decision: the required
// set is a code-level SSOT, never per-scenario configuration).
package freshness

import (
	"time"

	sharedfresh "github.com/vrooli/freshness-go"
	"github.com/vrooli/freshness-go/runindex"
	"github.com/vrooli/freshness-go/treedigest"
)

// PhaseStatus is one phase's verdict enriched with run context.
type PhaseStatus struct {
	Phase   string
	Verdict string // "fresh" | "stale" | "unknown"

	// Newest evidence considered (fresh: the current-digest passing run;
	// stale: the newest passing run at any digest). Empty when none.
	LastRunID  string
	LastRunAt  time.Time
	LastDigest string
	LastStatus string
}

// Result is the freshness block for one scenario.
type Result struct {
	// Digest is the scenario's current tree digest ("td:..."); empty when
	// computation failed, in which case DigestErr carries the reason and
	// every verdict is "unknown" (a digest failure can never prove
	// freshness).
	Digest    string
	DigestErr string

	Phases []PhaseStatus

	// SuggestedCommand refreshes every non-fresh phase; empty when all
	// verdicts are fresh.
	SuggestedCommand string
}

// Service computes freshness results. The function fields are seams for
// tests; production uses New().
type Service struct {
	ComputeDigest func(scenarioDir string) (string, error)
	LoadRecords   func(scenarioDir string) ([]runindex.RunRecord, error)
}

// New returns a Service bound to the real digest and run-index readers.
func New() *Service {
	return &Service{
		ComputeDigest: treedigest.Compute,
		LoadRecords:   runindex.Load,
	}
}

// Check assembles the freshness block for the scenario rooted at root.
// It never returns an error: digest failures degrade to unknown verdicts
// with DigestErr set, and a missing/unreadable run index simply means no
// evidence (unknown verdicts under a valid digest only when no stamped
// runs exist — sharedfresh.Check semantics).
func (s *Service) Check(scenarioName, root string) Result {
	out := Result{}

	digest, err := s.ComputeDigest(root)
	if err != nil {
		out.DigestErr = err.Error()
	} else {
		out.Digest = digest
	}

	records, err := s.LoadRecords(root)
	if err != nil {
		// Unreadable index = no usable evidence; verdicts fall out as
		// unknown below. The digest (when computed) is still reported.
		records = nil
	}
	runindex.SortNewestFirst(records)

	phases := sharedfresh.RequiredPhases()
	var verdicts []sharedfresh.PhaseVerdict
	if out.Digest == "" {
		// No digest to compare against: every phase is unknown.
		for _, p := range phases {
			verdicts = append(verdicts, sharedfresh.PhaseVerdict{Phase: p, Status: sharedfresh.StatusUnknown})
		}
	} else {
		verdicts = sharedfresh.Check(records, out.Digest, phases).Phases
	}

	byRun := make(map[string]runindex.RunRecord, len(records))
	for _, r := range records {
		byRun[r.RunID] = r
	}

	out.Phases = make([]PhaseStatus, 0, len(verdicts))
	for _, v := range verdicts {
		ps := PhaseStatus{Phase: v.Phase, Verdict: v.Status, LastRunID: v.LastRunID}
		if v.LastRunCompletedAt != "" {
			if t, perr := time.Parse(time.RFC3339, v.LastRunCompletedAt); perr == nil {
				ps.LastRunAt = t
			}
		}
		if r, ok := byRun[v.LastRunID]; ok {
			ps.LastDigest = r.TreeDigest
			if st, found := r.PhaseStatus()[v.Phase]; found {
				ps.LastStatus = st
			}
		}
		out.Phases = append(out.Phases, ps)
	}

	out.SuggestedCommand = sharedfresh.SuggestedCommand(scenarioName, verdicts, true)
	return out
}
