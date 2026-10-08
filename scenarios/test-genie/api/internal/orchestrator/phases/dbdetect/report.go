package dbdetect

import (
	"fmt"
	"strings"
)

// FormatHuman renders the DetectionReport as the block written to the
// playbooks log. Format is locked by the implementation plan.
//
// Example:
//
//	db-detect:
//	  postgres:  required   (manifest:resource[type=postgres])
//	                         + godeps:postgres-driver
//	  redis:     not needed
//	  sqlite:    required   (godeps:sqlite-driver)
//	                         + source:sqlite-tokens ×3
//	                         ! missing-corroboration: source:sqlite-tokens
func (r DetectionReport) FormatHuman() string {
	var b strings.Builder
	b.WriteString("db-detect:\n")
	for _, db := range r.Order {
		res := r.Results[db]
		head := fmt.Sprintf("  %-9s ", db+":")
		if !res.Required {
			b.WriteString(head + "not needed\n")
			if db == "sqlite" {
				b.WriteString("                         ! manifest declares no sqlite resource (expected — sqlite is library-only)\n")
			}
			continue
		}
		decision := "(unknown)"
		if res.Decision != nil {
			decision = "(" + formatEvidence(*res.Decision) + ")"
		}
		b.WriteString(fmt.Sprintf("%srequired   %s\n", head, decision))
		for _, c := range res.Corroborating {
			b.WriteString("                         + " + formatEvidence(c) + "\n")
		}
		for _, cf := range res.Conflicts {
			b.WriteString("                         ! " + cf.Kind + ": " + cf.Detail + "\n")
		}
	}
	return b.String()
}

func formatEvidence(e Evidence) string {
	s := e.Source
	if e.Detail != "" && !strings.Contains(s, e.Detail) {
		s += ":" + e.Detail
	}
	if n := len(e.Locations); n > 1 {
		s += fmt.Sprintf(" ×%d", n)
	}
	return s
}
