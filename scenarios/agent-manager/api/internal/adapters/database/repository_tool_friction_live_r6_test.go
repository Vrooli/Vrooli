//go:build livecheck

package database

import (
	"strings"
	"testing"

	"agent-manager/internal/opportunities"
	"agent-manager/internal/toolfriction"
	"agent-manager/internal/usageattribution"
)

// reviewRound6 prints what review round 6 found: blocking siblings named by
// the polling advice (1, 2), overlapping quota levers (3), verify commands
// (4), the ranking (5), the calls-saved basis of opportunities (6) and the
// identity roll-up's units (7).
func reviewRound6(t *testing.T, report toolfriction.Report, found opportunities.Report, recs usageattribution.Report) {
	named := map[string][]string{}
	defer func() {
		for sibling, paths := range named {
			t.Logf("R6-1 sibling named: %s for %v", sibling, paths)
		}
	}()
	for _, p := range report.Profiles {
		if p.CommandPath == "vrooli scenario status" {
			t.Logf("R6-2 scenario status: %d polled, %d after a start of the same target, %d after a start naming no target, %d with no blocking sibling", p.Polled, p.Steered, p.Unnamed, p.Polled-p.Steered)
		}
	}
	for i, f := range report.Findings {
		switch {
		case f.Kind == toolfriction.KindPolling:
			t.Logf("R6-1 polling #%d %s: %d polled calls, %d steered, siblings %v | %s", i+1, f.CommandPath, f.Cost, f.Steered, f.Blocking, f.Description)
			for _, sibling := range f.Blocking {
				named[sibling] = append(named[sibling], f.CommandPath)
			}
		case f.RootCause == "identity":
			t.Logf("R6-7 identity #%d: %s", i+1, f.Description)
			for _, s := range f.Shares {
				t.Logf("R6-7 identity share: %s", s.Sentence())
			}
		}
	}
	for _, a := range recs.Actions {
		switch {
		case a.Kind == toolfriction.KindPolling:
			t.Logf("R6-1 action #%d %s | %s", a.Rank, a.Title, a.Text)
		case a.Source == usageattribution.SourceLever:
			runs := 0
			for _, ref := range a.Evidence {
				if ref.Kind == "run" {
					runs++
				}
			}
			t.Logf("R6-3 lever #%d %s | recoverable %.1f | %d evidence runs | %s", a.Rank, a.Title, a.Recoverable, runs, a.Rationale)
			t.Logf("R6-4 verify #%d: %s", a.Rank, a.Verify)
		}
	}
	for _, line := range recs.Coverage {
		if strings.Contains(line, "lever") {
			t.Logf("R6-3 coverage: %s", line)
		}
	}
	for i, a := range recs.Actions {
		if i >= 20 {
			break
		}
		t.Logf("R6-5 #%d [%s %s] %s | %s", a.Rank, a.Source, a.Kind, a.Title, a.ScoreBasis)
		if a.Kind == toolfriction.KindShapeGuess {
			t.Logf("R6-5 #%d text: %s", a.Rank, a.Text)
		}
	}
	for _, o := range found.Opportunities {
		t.Logf("R6-6 %s %s: %d occurrences in %d sessions, %d %s (%s) | %s", o.Kind, o.Subject, o.Occurrences, o.Sessions, o.CallsSaved, o.SavedLabel, o.SavedBasis, o.Action)
	}
}
