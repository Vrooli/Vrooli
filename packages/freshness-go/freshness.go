// Package freshness computes per-phase run-freshness verdicts: given the
// recorded test runs for a scenario and the scenario's CURRENT tree digest
// (see the treedigest subpackage), it reports per phase whether passing
// evidence exists against exactly the current byte-state.
//
// This is the pure verdict core extracted from test-genie's
// RunsService.CheckFreshness; test-genie's RPC and any cached status reader
// (scenario-completeness-scoring) share it so "fresh"/"stale"/"unknown" can
// never mean different things in different surfaces.
package freshness

import (
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/freshness-go/runindex"
)

// Per-phase freshness verdict values.
const (
	StatusFresh   = "fresh"
	StatusStale   = "stale"
	StatusUnknown = "unknown"
)

// PhaseVerdict is the freshness verdict for one phase. LastRunID /
// LastRunCompletedAt identify the newest evidence considered: for a fresh
// phase, the current-digest run that passed it; for a stale phase, the newest
// run at ANY digest that passed it (context for "last passed at X").
type PhaseVerdict struct {
	Phase              string
	Status             string
	LastRunID          string
	LastRunCompletedAt string // RFC3339 UTC, "" when no evidence exists
}

// Report is the verdict set for one scenario at one digest.
type Report struct {
	TreeDigest string
	Phases     []PhaseVerdict
}

// Check computes the per-phase verdicts. A phase is fresh iff some run whose
// TreeDigest equals the current digest contains the phase with status passed;
// verdicts are "unknown" when no digest-stamped runs exist at all (pre-digest
// history can never prove staleness). records are expected newest-first (the
// index order — runindex.Load and the test-genie index both provide it).
func Check(records []runindex.RunRecord, currentDigest string, phaseNames []string) Report {
	anyStamped := false
	for _, r := range records {
		if r.TreeDigest != "" {
			anyStamped = true
			break
		}
	}

	out := make([]PhaseVerdict, 0, len(phaseNames))
	for _, name := range phaseNames {
		pv := PhaseVerdict{Phase: name, Status: StatusStale}
		if !anyStamped {
			pv.Status = StatusUnknown
			out = append(out, pv)
			continue
		}
		for _, r := range records { // newest-first: first hit is the newest evidence
			if r.TreeDigest != currentDigest {
				continue
			}
			if status, ok := r.PhaseStatus()[name]; ok && status == runindex.StatusPassed {
				pv.Status = StatusFresh
				pv.LastRunID = r.RunID
				pv.LastRunCompletedAt = formatTime(r.CompletedAt)
				break
			}
		}
		if pv.Status == StatusStale {
			// Context for the advisory message: the newest run (any digest)
			// that passed this phase, so the caller can say "last passed at X".
			for _, r := range records {
				if status, ok := r.PhaseStatus()[name]; ok && status == runindex.StatusPassed {
					pv.LastRunID = r.RunID
					pv.LastRunCompletedAt = formatTime(r.CompletedAt)
					break
				}
			}
		}
		out = append(out, pv)
	}

	return Report{TreeDigest: currentDigest, Phases: out}
}

// RequiredPhases returns the global required-phase set for run-freshness
// checks. It mirrors test-genie's quick preset ("required" always means "what
// a quick run executes"); test-genie's phases.FreshnessRequired() remains
// derived from the preset itself and a guard test there pins the two lists
// equal, so drift is a loud test failure rather than a silent disagreement.
//
// Deliberately a code-level SSOT and NOT configurable per scenario (operator
// anti-gaming decision): a config knob would let an agent delete required
// phases to silence the freshness checker instead of running tests.
func RequiredPhases() []string {
	return []string{"structure", "standards", "docs", "business", "unit"}
}

// NormalizePhases lowercases, trims, and de-duplicates a requested phase
// list, preserving first-seen order.
func NormalizePhases(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, p := range in {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// SuggestedCommand renders the copy-pastable remediation for stale/unknown
// phases. With the default (required) set it is the quick preset; with an
// explicit phase list it names exactly the stale phases. Returns "" when
// every verdict is fresh.
func SuggestedCommand(scenario string, verdicts []PhaseVerdict, defaulted bool) string {
	var stale []string
	for _, v := range verdicts {
		if v.Status != StatusFresh {
			stale = append(stale, v.Phase)
		}
	}
	if len(stale) == 0 {
		return ""
	}
	if defaulted {
		return fmt.Sprintf("test-genie execute %s --preset quick", scenario)
	}
	return fmt.Sprintf("test-genie execute %s %s", scenario, strings.Join(stale, " "))
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
