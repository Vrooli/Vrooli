package validation

import (
	"context"
	"fmt"
	"strings"

	"ui-health/internal/services/manifestvalidation"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// freshnessClient is the seam the freshness group consumes: the typed
// `vrooli scenario freshness <name> --json` contract (the canonical
// content-hash freshness engine). *vroolicli.Client satisfies it; tests inject
// a fake. Kept as an interface so the handler never shells out directly.
type freshnessClient interface {
	ScenarioFreshness(ctx context.Context, name string, opts ...vroolicli.ScenarioFreshnessOption) (*cliv1.ScenarioFreshnessResponse, error)
}

// freshnessFindings runs the UI-bundle freshness group (check group 4 in the
// consolidated report). It is a STATIC group: it runs on every validation,
// including --static-only, and needs no BAS — it delegates to the canonical
// content-hash freshness engine via the typed vrooli CLI, which folds in file:
// workspace deps (e.g. @vrooli/iframe-bridge), keyed build inputs (NODE_ENV,
// toolchain), and per-file content hashing.
//
// A stale UI bundle is a gating ERROR with the restart remediation, because the
// runtime/render group must not validate a stale build. Only the `ui-bundle`
// check group is consulted; API-binary freshness is not a UI concern.
//
// Resolution errors degrade gracefully (logged, no finding): a freshness-engine
// hiccup must never block UI validation, mirroring smoke's graceful path.
func (h *connectHandler) freshnessFindings(ctx context.Context, scenario, scenarioDir string) []manifestvalidation.Finding {
	client := h.deps.Freshness
	if client == nil {
		client = vroolicli.New()
	}
	resp, err := client.ScenarioFreshness(ctx, scenario, vroolicli.WithFreshnessPath(scenarioDir))
	if err != nil {
		if h.deps.Logger != nil {
			h.deps.Logger.Printf("ui-health freshness: %v (degrading; no freshness finding)", err)
		}
		return nil
	}
	var findings []manifestvalidation.Finding
	for _, chk := range resp.GetChecks() {
		if chk.GetCheckType() != "ui-bundle" || !chk.GetStale() {
			continue
		}
		findings = append(findings, manifestvalidation.Finding{
			Severity:   manifestvalidation.SeverityError,
			Code:       "freshness_ui_bundle_stale",
			Location:   strings.TrimSpace(chk.GetFile()),
			Message:    uiBundleStaleReason(chk.GetTarget(), chk.GetCause(), chk.GetFile()),
			Suggestion: fmt.Sprintf("Rebuild the UI bundle so it reflects current sources: vrooli scenario restart %s", scenario),
		})
	}
	return findings
}

// uiBundleStaleReason renders a human-facing stale reason from a freshness check
// result: "<target> stale (<cause>): <file>", degrading gracefully when the
// cause or offending file is absent.
func uiBundleStaleReason(target, cause, file string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		target = "UI bundle"
	}
	cause = strings.TrimSpace(cause)
	file = strings.TrimSpace(file)
	switch {
	case cause != "" && file != "":
		return fmt.Sprintf("%s stale (%s): %s", target, cause, file)
	case cause != "":
		return fmt.Sprintf("%s stale (%s)", target, cause)
	default:
		return fmt.Sprintf("%s stale", target)
	}
}
