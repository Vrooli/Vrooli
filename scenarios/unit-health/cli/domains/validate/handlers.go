package validate

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	maturityreport "github.com/vrooli/maturity-go/report"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation/validation_v1connect"
)

type handlers struct {
	client validationconnect.ValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: validationconnect.NewValidationServiceClient(httpClient, baseURL)}
}

func (h *handlers) validateScenario(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{
		Scenario:         scenario,
		Path:             firstFlag(ctx.FlagValues("path")),
		Workspaces:       splitCSV(ctx.FlagValues("workspace")),
		IncludeExecution: ctx.BoolFlag("execution"),
		UseCache:         true,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("validate scenario %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	msg := resp.Msg

	results := findingLines(msg.GetFindings())
	if len(results) == 0 {
		results = append(results, "No test-maturity findings.")
	}

	summary := []string{
		fmt.Sprintf("%s (%s): %d error(s), %d warning(s), %d info(s) across %d workspace(s); maturity %s",
			msg.GetScenario(), msg.GetStatus(),
			msg.GetCounts().GetErrors(), msg.GetCounts().GetWarnings(), msg.GetCounts().GetInfos(),
			msg.GetCounts().GetWorkspaces(), msg.GetMaturity().GetLabel()),
	}
	if reason := strings.TrimSpace(msg.GetDegradedReason()); reason != "" {
		summary = append(summary, "Degraded: "+reason)
	}
	summary = append(summary, workspaceLines(msg.GetWorkspaces())...)
	summary = append(summary, planLines(msg.GetPlan())...)
	summary = append(summary, executionLines(msg.GetCommandResults())...)
	summary = append(summary, coverageLines(msg.GetCoverage())...)
	summary = append(summary, diagnosticLines(msg.GetDiagnostics())...)

	human := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Findings",
		Results:        results,
		RetrievalHints: msg.GetNextSteps(),
	}
	if maturity := maturityreport.BuildMaturityListReport(msg.GetAssessment()); maturity.Summary != nil {
		human.Summary = append(human.Summary, maturity.Summary...)
		human.RetrievalHints = append(human.RetrievalHints, maturity.RetrievalHints...)
	}
	if err := cliapp.RenderProtoList(ctx, msg, human); err != nil {
		return err
	}
	if msg.GetStatus() == "failed" || msg.GetCounts().GetErrors() > 0 {
		return fmt.Errorf("unit-health validation failed with %d error finding(s)", msg.GetCounts().GetErrors())
	}
	return nil
}

// findingLines keeps the first line scan-friendly while preserving the
// policy/projection evidence operators need for remediation.
func findingLines(findings []*validationv1.ValidationFinding) []string {
	results := make([]string, 0, len(findings))
	for _, f := range findings {
		results = append(results, fmt.Sprintf("[%s] %s %s: %s", f.GetSeverity(), f.GetCode(), f.GetFilePath(), f.GetMessage()))
		for _, line := range findingDetailLines(f) {
			results = append(results, "  "+line)
		}
	}
	return results
}

func findingDetailLines(f *validationv1.ValidationFinding) []string {
	if f == nil {
		return nil
	}
	details := []struct {
		label string
		value string
	}{
		{"evidence", f.GetEvidence()},
		{"expected", f.GetExpected()},
		{"observed", f.GetObserved()},
		{"why", f.GetWhyItMatters()},
		{"remediation", f.GetRemediation()},
		{"source", f.GetSourceCommand()},
	}
	lines := make([]string, 0, len(details))
	for _, detail := range details {
		value := strings.TrimSpace(detail.value)
		if value == "" {
			continue
		}
		lines = append(lines, detail.label+": "+value)
	}
	return lines
}

// workspaceLines renders the discovered testable workspaces, their canonical
// framework, and their readiness so operators see what would run and why.
func workspaceLines(workspaces []*validationv1.TestWorkspace) []string {
	if len(workspaces) == 0 {
		return nil
	}
	lines := []string{fmt.Sprintf("Workspaces (%d):", len(workspaces))}
	for _, w := range workspaces {
		line := fmt.Sprintf("  • %s [%s] %s — %s", w.GetId(), w.GetLanguage(), w.GetCanonicalFramework(), w.GetStatus())
		if reason := strings.TrimSpace(w.GetDegradedReason()); reason != "" {
			line += " (" + reason + ")"
		}
		lines = append(lines, line)
	}
	return lines
}

// planLines renders the dry-run execution plan: the bounded commands Unit
// Health would run for each workspace.
func planLines(plan *validationv1.ExecutionPlan) []string {
	if plan == nil {
		return nil
	}
	var lines []string
	if len(plan.GetCommands()) > 0 {
		lines = append(lines, fmt.Sprintf("Test plan (%d command(s)):", len(plan.GetCommands())))
		for _, c := range plan.GetCommands() {
			lines = append(lines, fmt.Sprintf("  • %s: %s (cwd=%s, timeout=%ds)",
				c.GetName(), c.GetCommand(), c.GetWorkingDirectory(), c.GetTimeoutSeconds()))
		}
	}
	if notes := strings.TrimSpace(plan.GetNotes()); notes != "" {
		lines = append(lines, "Plan notes: "+notes)
	}
	return lines
}

// executionLines renders the outcome of any executed test commands so failing,
// hanging, or unrunnable workspaces are visible in human output.
func executionLines(results []*validationv1.CommandResult) []string {
	if len(results) == 0 {
		return nil
	}
	lines := []string{fmt.Sprintf("Execution (%d command(s)):", len(results))}
	for _, r := range results {
		line := fmt.Sprintf("  • %s: %s (exit=%d, %dms)", r.GetName(), r.GetStatus(), r.GetExitCode(), r.GetDurationMs())
		if class := strings.TrimSpace(r.GetFailureClass()); class != "" {
			line += " [" + class + "]"
		}
		lines = append(lines, line)
	}
	return lines
}

// coverageLines renders a per-surface coverage roll-up (covered/total units and
// percent) so operators see hardening depth without a per-file dump.
func coverageLines(targets []*validationv1.CoverageTarget) []string {
	if len(targets) == 0 {
		return nil
	}
	type agg struct {
		covered, total int64
		threshold      float64
	}
	order := []string{}
	bySurface := map[string]*agg{}
	for _, t := range targets {
		key := t.GetSurfaceId()
		if key == "" {
			key = t.GetLanguage()
		}
		a, ok := bySurface[key]
		if !ok {
			a = &agg{threshold: t.GetThreshold()}
			bySurface[key] = a
			order = append(order, key)
		}
		a.covered += t.GetCoveredLines()
		a.total += t.GetTotalLines()
	}
	lines := []string{fmt.Sprintf("Coverage (%d file target(s)):", len(targets))}
	for _, key := range order {
		a := bySurface[key]
		pct := 0.0
		if a.total > 0 {
			pct = float64(a.covered) / float64(a.total) * 100
		}
		line := fmt.Sprintf("  • %s: %.1f%% (%d/%d units)", key, pct, a.covered, a.total)
		if a.threshold > 0 {
			line += fmt.Sprintf(", threshold %.0f%%", a.threshold)
		}
		lines = append(lines, line)
	}
	return lines
}

// diagnosticLines renders flake/runtime/hang diagnostics so operators see the
// likely culprit behind a slow or hung suite.
func diagnosticLines(diagnostics []*validationv1.Diagnostic) []string {
	if len(diagnostics) == 0 {
		return nil
	}
	lines := []string{fmt.Sprintf("Diagnostics (%d):", len(diagnostics))}
	for _, d := range diagnostics {
		line := fmt.Sprintf("  • [%s] %s", d.GetKind(), d.GetMessage())
		if ws := strings.TrimSpace(d.GetWorkspaceId()); ws != "" {
			line += " (" + ws + ")"
		}
		lines = append(lines, line)
	}
	return lines
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func splitCSV(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}
