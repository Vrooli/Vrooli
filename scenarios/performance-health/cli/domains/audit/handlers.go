package audit

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit"
	auditconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit/audit_v1connect"
)

type handlers struct {
	client auditconnect.AuditServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	// RunAudit synchronously restarts the target in profile build mode (a UI
	// rebuild — minutes) and drives a BAS capture before responding, far beyond
	// the default client timeout. A zero timeout means no client-side deadline;
	// the server bounds the work.
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: auditconnect.NewAuditServiceClient(httpClient, baseURL)}
}

// run orchestrates a profile-mode perf capture of a scenario.
func (h *handlers) run(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.RunAudit(context.Background(), connect.NewRequest(&auditv1.RunAuditRequest{
		Scenario: scenario,
		Path:     firstFlag(ctx.FlagValues("path")),
		Workflow: firstFlag(ctx.FlagValues("workflow")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("run audit for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no audit response")
	}
	msg := resp.Msg
	result := []string{fmt.Sprintf("%s: %s.", msg.GetScenario(), outcomeLabel(msg.GetOutcome()))}
	switch msg.GetOutcome() {
	case auditv1.AuditOutcome_AUDIT_OUTCOME_UNAVAILABLE:
		// Loud banner: the capture mechanism was genuinely absent (BAS
		// unreachable or no browser). A degraded environment, not a pass. The
		// specific cause is in Reason below.
		result = append(result,
			"⚠ CAPTURE UNAVAILABLE — the capture mechanism was absent (browser-automation-studio unreachable, or no browser).",
			"  This is NOT a pass: no trace was produced. See Reason; run where a browser/BAS is available.")
	case auditv1.AuditOutcome_AUDIT_OUTCOME_FAILED:
		// A reachable BAS that produced no trace is a (often transient) capture
		// failure — NOT an environmental impossibility. Say so honestly so a
		// busy-BAS hiccup is not misread as "no browser".
		result = append(result,
			"⚠ CAPTURE FAILED — BAS was reached but did not return a usable trace (often transient under capture load).",
			"  This is NOT a pass. See Reason; re-run the audit to retry.")
	}
	if r := strings.TrimSpace(msg.GetReason()); r != "" {
		result = append(result, "Reason: "+r)
	}
	changes := []string{}
	if a := strings.TrimSpace(msg.GetTraceArtifact()); a != "" {
		changes = append(changes, "trace: "+a)
		// Document the audit → analysis handoff at the point a trace exists.
		changes = append(changes, "next: performance-health analysis analyze --trace "+a)
	}
	if a := strings.TrimSpace(msg.GetWebVitalsArtifact()); a != "" {
		changes = append(changes, "web-vitals: "+a)
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{Result: result, Changes: changes})
}

func outcomeLabel(o auditv1.AuditOutcome) string {
	switch o {
	case auditv1.AuditOutcome_AUDIT_OUTCOME_CAPTURED:
		return "captured"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_SKIPPED:
		return "skipped (not applicable — no UI surface)"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_FAILED:
		return "failed"
	case auditv1.AuditOutcome_AUDIT_OUTCOME_UNAVAILABLE:
		return "unavailable (no browser / BAS unreachable)"
	default:
		return "unspecified"
	}
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
