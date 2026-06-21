package analysis

import (
	"fmt"

	mga "github.com/vrooli/maturity-go/assessment"
)

// DeriveFindings turns the per-component table into deterministic findings: a
// component whose average commit time exceeds budgetMs is flagged with
// quantified evidence and its located definition. NO AI — pure thresholding.
//
// A component over budget whose definition couldn't be located still emits a
// finding (metrics + a "definition not located" note in the message), rather
// than being dropped.
func DeriveFindings(components []ComponentTiming, budgetMs float64) []Finding {
	var out []Finding
	for _, c := range components {
		if budgetMs <= 0 || c.AvgMs <= budgetMs {
			continue
		}
		msg := fmt.Sprintf("%s averages %.1fms per commit, over the %.1fms budget", c.Component, c.AvgMs, budgetMs)
		if c.Definition == "" {
			msg += " (definition not located)"
		}
		out = append(out, Finding{
			Code:       "PERF_COMPONENT_COMMIT_OVER_BUDGET",
			Component:  c.Component,
			Definition: c.Definition,
			Message:    msg,
			Evidence:   fmt.Sprintf("count=%d avg=%.1fms max=%.1fms over budget by %.1fms", c.CommitCount, c.AvgMs, c.MaxMs, c.AvgMs-budgetMs),
			Severity:   "warning",
		})
	}
	return out
}

// AssessmentFindings projects deterministic perf findings into the shared
// maturity-go finding shape so they flow through the same finding pipeline as
// readiness (packages/maturity-go/assessment). The located definition becomes
// the finding Location and the quantified evidence is folded into the message;
// AutofixAvailable is false (perf hotspots are not deterministically autofixed).
func AssessmentFindings(findings []Finding) []mga.Finding {
	out := make([]mga.Finding, 0, len(findings))
	for _, f := range findings {
		out = append(out, mga.Finding{
			Code:             f.Code,
			Severity:         f.Severity,
			Title:            f.Component,
			Message:          f.Message + " [" + f.Evidence + "]",
			Location:         f.Definition,
			AutofixAvailable: false,
		})
	}
	return out
}
