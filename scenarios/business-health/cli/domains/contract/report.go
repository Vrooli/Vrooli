package contract

import (
	"fmt"
	"sort"
	"strings"

	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/contract"

	"github.com/vrooli/cli-core/cliapp"
)

// renderReport renders the traceability report from an already-fetched
// matrix: summary or markdown. --json carries the proto matrix.
func (h *handlers) renderReport(ctx cliapp.RunContext, msg *contractv1.GetMatrixResponse, format string) error {
	reg := msg.GetRegistry()

	if strings.EqualFold(format, "markdown") {
		var b strings.Builder
		fmt.Fprintf(&b, "# Traceability report — %s\n\n", msg.Scenario)
		fmt.Fprintf(&b, "| OT | Requirement | Status | Evidence | Unproven |\n|---|---|---|---|---|\n")
		for _, r := range msg.Matrix {
			evidence := r.GetEvidence().GetLiveStatus()
			if evidence == "" {
				evidence = "—"
			}
			unproven := ""
			if r.Unproven {
				unproven = "⚠ " + r.UnprovenReason
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", dash(r.OtId), dash(r.RequirementId), dash(r.RequirementStatus), evidence, unproven)
		}
		return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("%s: %d requirements across %d modules, %d operational targets.", msg.Scenario, reg.GetRequirementCount(), reg.GetModuleCount(), reg.GetOperationalTargetCount())},
			ResultsHeading: "Markdown",
			Results:        []string{b.String()},
		})
	}

	unproven := 0
	for _, r := range msg.Matrix {
		if r.Unproven {
			unproven++
		}
	}
	summary := []string{
		fmt.Sprintf("%s: %d requirements / %d modules / %d operational targets.", msg.Scenario, reg.GetRequirementCount(), reg.GetModuleCount(), reg.GetOperationalTargetCount()),
		fmt.Sprintf("Status counts: %s.", formatCounts(reg.GetStatusCounts())),
		fmt.Sprintf("Unproven claims: %d.", unproven),
	}
	if reg.GetStarterTemplate() {
		summary = append(summary, "Registry still carries starter-template residue.")
	}
	if msg.DegradedReason != "" {
		summary = append(summary, "Degraded: "+msg.DegradedReason)
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Rows",
		Results:        []string{fmt.Sprintf("%d matrix rows (use the default table or --format markdown for detail).", len(msg.Matrix))},
	})
}

// renderPhase filters the matrix's validation cells by phase.
func (h *handlers) renderPhase(ctx cliapp.RunContext, msg *contractv1.GetMatrixResponse, phase string) error {
	want := strings.ToLower(strings.TrimSpace(phase))
	if want == "all" {
		want = "" // `--phase all` = every phase (the facade's phase-inspect default)
	}
	var results []string
	for _, r := range msg.Matrix {
		for _, v := range r.Validations {
			if want != "" && !strings.EqualFold(v.Phase, want) {
				continue
			}
			marker := "✓"
			if v.Ref != "" && !v.RefExists {
				marker = "✗ missing ref"
			}
			results = append(results, fmt.Sprintf("%s [%s/%s] %s %s %s", r.RequirementId, dash(v.Phase), v.Type, dash(v.Status), v.Ref, marker))
		}
	}
	if len(results) == 0 {
		results = append(results, "No validations matched.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d validation(s)%s.", len(results), phaseSuffix(want))},
		ResultsHeading: "Validations",
		Results:        results,
	})
}

func phaseSuffix(want string) string {
	if want == "" {
		return " across all phases"
	}
	return " in phase " + want
}

func formatCounts(counts map[string]int32) string {
	if len(counts) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(counts))
	for k, v := range counts {
		parts = append(parts, fmt.Sprintf("%s=%d", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
