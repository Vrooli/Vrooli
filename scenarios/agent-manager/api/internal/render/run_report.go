// Package render renders bounded read-side projections for humans and agents.
package render

import "agent-manager/internal/runreport"

// RunReportText renders discriminators only. Payloads remain behind the
// command-specific drill-downs in Next so a large transcript or diff cannot
// crowd out the facts that route an investigation.
func RunReportText(r *runreport.RunReport) string {
	return runreport.Text(r)
}
