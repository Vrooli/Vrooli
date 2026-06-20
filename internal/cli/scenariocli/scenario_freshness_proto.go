package scenariocli

import (
	"fmt"
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// FreshnessRequest is the parsed `vrooli scenario freshness` invocation.
type FreshnessRequest struct {
	Name    string
	Path    string
	JSON    bool
	Explain bool
}

// FreshnessResponse couples the lifecycle freshness report with the --explain
// flag so the renderer can choose between a summary and the full breakdown. JSON
// output is always the complete typed contract regardless of --explain.
type FreshnessResponse struct {
	Report  lifecycle.FreshnessReport
	Explain bool
}

// ScenarioFreshnessResponseProto maps the lifecycle freshness report onto its
// typed cli/v1 wire contract.
func ScenarioFreshnessResponseProto(report lifecycle.FreshnessReport) *cliv1.ScenarioFreshnessResponse {
	checks := make([]*cliv1.ScenarioFreshnessCheck, 0, len(report.Checks))
	for _, c := range report.Checks {
		checks = append(checks, &cliv1.ScenarioFreshnessCheck{
			CheckType: c.CheckType,
			Target:    c.Target,
			Stale:     c.Stale,
			Cause:     c.Cause,
			File:      c.File,
		})
	}
	deps := make([]*cliv1.ScenarioFreshnessDependency, 0, len(report.Dependencies))
	for _, d := range report.Dependencies {
		deps = append(deps, &cliv1.ScenarioFreshnessDependency{
			Name:   d.Name,
			Policy: d.Policy,
		})
	}
	return &cliv1.ScenarioFreshnessResponse{
		Success:      true,
		Scenario:     report.Scenario,
		Stale:        report.Stale,
		Checks:       checks,
		Dependencies: deps,
	}
}

func writeScenarioFreshnessJSON(w io.Writer, report lifecycle.FreshnessReport) error {
	return cliout.WriteProtoJSON(w, ScenarioFreshnessResponseProto(report))
}

// RenderFreshnessResponse prints the freshness verdict. JSON emits the typed
// proto contract; human output shows the overall verdict and any stale checks,
// expanding to every check plus resolved dependency policies under --explain.
func RenderFreshnessResponse(w io.Writer, format cliout.Format, resp FreshnessResponse) error {
	if format == cliout.FormatJSON {
		return writeScenarioFreshnessJSON(w, resp.Report)
	}

	report := resp.Report
	overall := "fresh"
	if report.Stale {
		overall = "stale"
	}
	if _, err := fmt.Fprintf(w, "Scenario %s is %s\n", report.Scenario, overall); err != nil {
		return err
	}

	for _, c := range report.Checks {
		if !resp.Explain && !c.Stale {
			continue
		}
		state := "fresh"
		if c.Stale {
			state = "stale"
		}
		line := fmt.Sprintf("- [%s] %s: %s", c.CheckType, c.Target, state)
		if c.Stale {
			if c.File != "" {
				line += fmt.Sprintf(" (%s: %s)", c.Cause, c.File)
			} else if c.Cause != "" {
				line += fmt.Sprintf(" (%s)", c.Cause)
			}
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	if resp.Explain && len(report.Dependencies) > 0 {
		if _, err := fmt.Fprintln(w, "Dependency freshness policies:"); err != nil {
			return err
		}
		for _, d := range report.Dependencies {
			if _, err := fmt.Fprintf(w, "- %s: %s\n", d.Name, d.Policy); err != nil {
				return err
			}
		}
	}
	return nil
}
