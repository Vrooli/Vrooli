package smoke

import (
	"context"
	"fmt"
	"strings"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// freshnessChecker reports whether a scenario's UI bundle is stale, backed by
// the canonical content-hash freshness engine. It is the seam that replaced
// smoke's mtime-only heuristic: the real implementation shells the typed
// `vrooli scenario freshness <name> --json` contract, which folds in file:
// workspace deps (e.g. @vrooli/iframe-bridge), keyed build inputs (NODE_ENV,
// toolchain), and per-file content hashing — all false-negatives the old
// directory-mtime walk missed.
type freshnessChecker interface {
	// UIBundleStale returns whether the scenario's ui-bundle artifact is stale,
	// a human reason (cause + offending file) when it is, and any error from
	// resolving freshness. An error is non-fatal to the caller: smoke logs it and
	// proceeds (graceful degradation — never block a render on an infra hiccup).
	UIBundleStale(ctx context.Context, scenarioName, scenarioDir string) (stale bool, reason string, err error)
}

// cliFreshnessChecker is the production freshnessChecker: it consumes the typed
// `vrooli scenario freshness` CLI contract via the shared Vrooli CLI client and
// reports staleness for the ui-bundle check group only (smoke is a UI concern;
// API binary freshness is not its business).
type cliFreshnessChecker struct {
	client *vroolicli.Client
}

func newCLIFreshnessChecker() cliFreshnessChecker {
	return cliFreshnessChecker{client: vroolicli.New()}
}

func (c cliFreshnessChecker) UIBundleStale(ctx context.Context, scenarioName, scenarioDir string) (bool, string, error) {
	resp, err := c.client.ScenarioFreshness(ctx, scenarioName, vroolicli.WithFreshnessPath(scenarioDir))
	if err != nil {
		return false, "", err
	}
	for _, chk := range resp.GetChecks() {
		if chk.GetCheckType() != "ui-bundle" || !chk.GetStale() {
			continue
		}
		return true, uiBundleStaleReason(chk.GetTarget(), chk.GetCause(), chk.GetFile()), nil
	}
	return false, "", nil
}

// uiBundleStaleReason renders a human-facing stale reason from a freshness check
// result: "<target> stale (<cause>): <file>", degrading gracefully when the
// cause or offending file is absent.
func uiBundleStaleReason(target, cause, file string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		target = "UI bundle"
	}
	switch {
	case strings.TrimSpace(cause) != "" && strings.TrimSpace(file) != "":
		return fmt.Sprintf("%s stale (%s): %s", target, cause, file)
	case strings.TrimSpace(cause) != "":
		return fmt.Sprintf("%s stale (%s)", target, cause)
	default:
		return fmt.Sprintf("%s stale", target)
	}
}
