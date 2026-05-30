// DOC: docs/phases/architecture/README.md#summary-metrics
package phases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	"github.com/vrooli/api-core/discovery"
	audit_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit/audit_v1connect"
	conflicts_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// ArchitectureSummary is the metric rollup persisted to phase pointers.
type ArchitectureSummary struct {
	Scenario   string `json:"scenario"`
	Outcome    string `json:"outcome"`
	Total      int    `json:"total"`
	Blockers   int    `json:"blockers"`
	Errors     int    `json:"errors"`
	Warnings   int    `json:"warnings"`
	Infos      int    `json:"infos"`
	Suppressed int    `json:"suppressed"`
	Authority  string `json:"authority,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
}

func (s ArchitectureSummary) String() string {
	if s.Skipped {
		return "skipped"
	}
	return fmt.Sprintf("%s outcome=%s total=%d (blocker=%d error=%d warn=%d info=%d) suppressed=%d authority=%s",
		s.Scenario, s.Outcome, s.Total, s.Blockers, s.Errors, s.Warnings, s.Infos, s.Suppressed, s.Authority)
}

type architectureRunResult struct {
	shared.RunResult[ArchitectureSummary]
}

var (
	// Seam for tests: resolveArchitectureBaseURL returns the
	// architecture-cartographer base URL. discovery is the only
	// production path; tests override.
	resolveArchitectureBaseURL = func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "architecture-cartographer")
	}
	// Seam for tests: architectureHTTPClient lets tests substitute a fake
	// connect.HTTPClient instead of dialing the real service.
	architectureHTTPClient connect.HTTPClient = &http.Client{Timeout: 120 * time.Second}
	// Seam for tests: runArchitectureAudit invokes the cartographer audit
	// RPC. Tests substitute a fake to avoid a live dependency.
	runArchitectureAudit = func(ctx context.Context, scenario string) (*audit_v1.AuditRunResponse, error) {
		baseURL, err := resolveArchitectureBaseURL(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve architecture-cartographer URL: %w", err)
		}
		if strings.TrimSpace(baseURL) == "" {
			return nil, errors.New("architecture-cartographer base URL is empty — start it via 'vrooli scenario start architecture-cartographer'")
		}
		client := auditconnect.NewAuditServiceClient(architectureHTTPClient, baseURL)
		resp, err := client.Run(ctx, connect.NewRequest(&audit_v1.AuditRunRequest{
			Scenario: scenario,
			// Advisory phase: surface every finding regardless of severity
			// and do not let low domain authority alone flip the outcome —
			// the campaign nudge (not a gate) does the steering.
			AllowLowAuthority: true,
		}))
		if err != nil {
			return nil, err
		}
		return resp.Msg, nil
	}
)

// runArchitecturePhase delegates to architecture-cartographer's read-only
// AuditService.Run RPC and translates the structural-cohesion findings
// (cycles, coupling, convergence, mislocation) into Observations + the
// shared ArchitectureFinding contract (source=ARCHITECTURE).
//
// The phase is ADVISORY (Optional, never hard-fails on findings): it
// preserves the cartographer's graded semantics. Only a transport error or
// the cartographer's own TOOL_ERROR outcome fails the phase. Blocker-level
// findings (import cycles) still surface prominently and drive the Phase-3
// campaign nudge.
func runArchitecturePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if os.Getenv("TEST_GENIE_SKIP_ARCHITECTURE") == "1" {
		return architectureSkipReport(env, "architecture phase disabled via TEST_GENIE_SKIP_ARCHITECTURE", logWriter)
	}

	// The architecture axis is advisory and MUST NEVER gate the suite. An
	// unreachable or unresponsive cartographer is missing optional infra, not a
	// scenario defect, so degrade to an advisory skip rather than failing the
	// phase. Without this, every preset that includes architecture (e.g.
	// comprehensive) would hard-fail whenever architecture-cartographer is not
	// running — including the development-toolchain-validator tool baseline.
	resp, auditErr := runArchitectureAudit(ctx, env.ScenarioName)
	if auditErr != nil {
		reason := fmt.Sprintf(
			"architecture audit skipped — architecture-cartographer unreachable: %v (start it via 'vrooli scenario start architecture-cartographer')",
			auditErr,
		)
		return architectureSkipReport(env, reason, logWriter)
	}

	var summary ArchitectureSummary
	summary.Scenario = env.ScenarioName
	archFindings := architectureArchFindings(env.ScenarioName, resp)

	report := RunPhase(ctx, logWriter, "architecture",
		func() (*architectureRunResult, error) {
			return translateArchitectureResponse(resp), nil
		},
		func(r *architectureRunResult) PhaseResult[shared.Observation] {
			var result shared.RunResult[ArchitectureSummary]
			summaryText := ""
			if r != nil {
				result = r.RunResult
				summary = r.Summary
				summaryText = r.Summary.String()
			}
			return ExtractWithSummary(
				result.Success,
				result.Error,
				result.FailureClass,
				result.Remediation,
				result.Observations,
				"🏛️",
				fmt.Sprintf("Architecture audit completed (%s)", summaryText),
			)
		},
	)

	report.Findings = archFindings
	writePhasePointer(env, "architecture", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Architecture summary: %s", summary.String())
	return report
}

// architectureSkipReport produces the advisory-skip RunReport used when the
// architecture phase is disabled or the cartographer is unreachable. The report
// carries no Err, so the orchestrator records the phase as passed (skipped) and
// the suite is never gated on this advisory axis.
func architectureSkipReport(env workspace.Environment, reason string, logWriter io.Writer) RunReport {
	summary := ArchitectureSummary{Scenario: env.ScenarioName, Skipped: true}
	report := RunReport{
		Observations: []Observation{NewSkipObservation(reason)},
	}
	writePhasePointer(env, "architecture", report, map[string]any{"summary": summary}, logWriter)
	logPhaseStep(logWriter, "Architecture summary: %s", summary.String())
	return report
}

// translateArchitectureResponse converts the audit response into the
// test-genie RunResult shape. The phase is advisory: outcome=FINDINGS does
// NOT fail it; only outcome=TOOL_ERROR (the cartographer couldn't run the
// audit) does.
func translateArchitectureResponse(resp *audit_v1.AuditRunResponse) *architectureRunResult {
	out := &architectureRunResult{}
	if resp == nil {
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		out.Error = errors.New("architecture-cartographer returned an empty audit response")
		return out
	}

	out.Summary = ArchitectureSummary{
		Scenario:   resp.GetScenario(),
		Outcome:    strings.ToLower(strings.TrimPrefix(resp.GetOutcome().String(), "AUDIT_OUTCOME_")),
		Total:      int(resp.GetTotalFindings()),
		Blockers:   int(resp.GetBySeverity()["blocker"]),
		Errors:     int(resp.GetBySeverity()["error"]),
		Warnings:   int(resp.GetBySeverity()["warn"]),
		Infos:      int(resp.GetBySeverity()["info"]),
		Suppressed: int(resp.GetSuppressedFindings()),
	}
	if d := resp.GetDomains(); d != nil {
		out.Summary.Authority = d.GetConfidence()
	}

	out.Observations = append(out.Observations, shared.NewSectionObservation("🏛️", "Architecture"))
	for _, f := range resp.GetFindings() {
		if f == nil {
			continue
		}
		msg := formatArchitectureFinding(f)
		switch f.GetSeverity() {
		case conflicts_v1.Severity_SEVERITY_BLOCKER, conflicts_v1.Severity_SEVERITY_ERROR:
			out.Observations = append(out.Observations, shared.NewErrorObservation(msg))
		case conflicts_v1.Severity_SEVERITY_WARN:
			out.Observations = append(out.Observations, shared.NewWarningObservation(msg))
		default:
			out.Observations = append(out.Observations, shared.NewInfoObservation(msg))
		}
	}

	switch resp.GetOutcome() {
	case audit_v1.AuditOutcome_AUDIT_OUTCOME_TOOL_ERROR:
		out.Success = false
		out.FailureClass = shared.FailureClassSystem
		errMsg := strings.TrimSpace(resp.GetError())
		if errMsg == "" {
			errMsg = "architecture-cartographer reported a tool error with no detail"
		}
		out.Error = fmt.Errorf("architecture audit tool error: %s", errMsg)
		out.Remediation = "Run `architecture-cartographer audit run " + resp.GetScenario() + "` and inspect the failure."
	default:
		// CLEAN or FINDINGS — advisory phase always passes; findings steer
		// via Observations + the campaign nudge.
		out.Success = true
		if out.Summary.Total == 0 {
			out.Observations = append(out.Observations, shared.NewSuccessObservation("No architectural findings detected"))
		} else {
			out.Observations = append(out.Observations, shared.NewWarningObservation(
				fmt.Sprintf("%d architectural finding(s) detected (advisory)", out.Summary.Total)))
		}
	}
	return out
}

func formatArchitectureFinding(f *audit_v1.ConflictSummary) string {
	code := architectureFindingCode(f)
	line := code
	if h := strings.TrimSpace(f.GetHeadline()); h != "" {
		line += ": " + h
	}
	if locs := f.GetLocations(); len(locs) > 0 {
		line += fmt.Sprintf(" [%s]", strings.Join(locs, ", "))
	}
	if doms := f.GetDomains(); len(doms) > 0 {
		line += fmt.Sprintf(" {%s}", strings.Join(doms, ", "))
	}
	return line
}

// architectureFindingCode renders the cartographer type[/subtype] as the
// finding code (the stable-ID input on the test-genie side).
func architectureFindingCode(f *audit_v1.ConflictSummary) string {
	code := strings.TrimSpace(f.GetType())
	if sub := strings.TrimSpace(f.GetSubtype()); sub != "" {
		code += "/" + sub
	}
	return code
}

// architectureSeverityToken maps the cartographer Severity enum into the
// string vocabulary normalizeFindingSeverity understands.
func architectureSeverityToken(s conflicts_v1.Severity) string {
	switch s {
	case conflicts_v1.Severity_SEVERITY_BLOCKER:
		return "blocker"
	case conflicts_v1.Severity_SEVERITY_ERROR:
		return "error"
	case conflicts_v1.Severity_SEVERITY_WARN:
		return "warn"
	case conflicts_v1.Severity_SEVERITY_INFO:
		return "info"
	default:
		return ""
	}
}

// architectureArchFindings maps the audit response's ConflictSummaries into
// the shared ArchitectureFinding contract (source=ARCHITECTURE). The
// stable ID is recomputed via findingid from the SAME inputs the
// cartographer campaign tracker will use on ingest, so re-audits
// reconcile cleanly.
func architectureArchFindings(scenario string, resp *audit_v1.AuditRunResponse) []*architecturev1.ArchitectureFinding {
	if resp == nil {
		return nil
	}
	out := make([]*architecturev1.ArchitectureFinding, 0, len(resp.GetFindings()))
	for _, f := range resp.GetFindings() {
		if f == nil {
			continue
		}
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE,
			architectureFindingCode(f),
			architectureSeverityToken(f.GetSeverity()),
			f.GetHeadline(), "",
			f.GetLocations(), f.GetDomains(),
		))
	}
	return out
}
