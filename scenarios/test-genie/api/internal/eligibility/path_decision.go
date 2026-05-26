package eligibility

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Path identifies which path the playbooks phase chose for a run.
//
// PathRouted means the in-place routed e2e path was taken. Every other value
// records *why* the fallback (restart) path was chosen.
type Path string

const (
	// PathRouted: the scenario qualified and the routed e2e path ran.
	PathRouted Path = "routed"

	// PathFallbackRules: scenario-auditor reported violations of one of the
	// routing-eligibility rules. Violations field is populated.
	PathFallbackRules Path = "fallback_rules"

	// PathFallbackPreflight: eligibility passed but a routed-path pre-flight
	// check failed (DSN extraction or routing-service client resolution).
	// PreflightFailure identifies which one.
	PathFallbackPreflight Path = "fallback_preflight"

	// PathFallbackAuditorUnreachable: the scenario-auditor scan could not be
	// completed (auditor down, network error, scan failed). We treat
	// "auditor can't verify" as "scenario is not certifiably eligible."
	PathFallbackAuditorUnreachable Path = "fallback_auditor_unreachable"

	// PathFallbackForcedEnv: operator set TEST_GENIE_FORCE_FALLBACK=1.
	PathFallbackForcedEnv Path = "fallback_forced_env"

	// PathFallbackProductionMode: the target scenario is running in
	// production mode (or otherwise has the dev-only RoutingService
	// disabled); the routed surface is not available.
	PathFallbackProductionMode Path = "fallback_production_mode"
)

// PreflightFailure identifies which routed pre-flight check failed.
type PreflightFailure string

const (
	PreflightFailureNone               PreflightFailure = ""
	PreflightFailureNoDSN              PreflightFailure = "no_test_dsn"
	PreflightFailureRoutingUnreachable PreflightFailure = "routing_unreachable"
)

// RuleAssertion records the routing-rule IDs the auditor scan did not include
// (either the rule isn't registered, or it's disabled). A non-nil assertion
// disqualifies the scenario from the routed path: we can't certify routing-
// rule compliance from a scan that didn't run them.
type RuleAssertion struct {
	MissingRules []string
}

// PathDecision is the consolidated record of which path the playbooks phase
// took for a given run and why. It is the single source of truth for the
// structured log block emitted at the top of each playbooks run.
type PathDecision struct {
	Path             Path
	Reason           string
	Violations       []ViolationExcerpt
	RuleAssertion    *RuleAssertion
	PreflightFailure PreflightFailure

	// RoutingBaseURL, LeaseID, DSNDriver are populated for routed runs so the
	// log block can name the active install.
	RoutingBaseURL string
	LeaseID        string
	DSNDriver      string
}

// IsRouted reports whether the playbooks phase will execute the in-place
// routed path for this decision.
func (d PathDecision) IsRouted() bool { return d.Path == PathRouted }

// rulesListResponse is the subset of GET /api/v1/rules we care about.
type rulesListResponse struct {
	Rules map[string]struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	} `json:"rules"`
}

// FetchRegisteredRules returns the set of rule IDs currently registered AND
// enabled in scenario-auditor. It is the seam used by AssertRulesObserved.
// Tests override this to return a canned set without touching HTTP.
var FetchRegisteredRules = func(ctx context.Context, baseURL string) (map[string]struct{}, error) {
	body, err := RequestJSON(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/rules", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch scenario-auditor rules: %w", err)
	}
	var resp rulesListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode scenario-auditor /rules response: %w", err)
	}
	enabled := make(map[string]struct{}, len(resp.Rules))
	for id, info := range resp.Rules {
		if !info.Enabled {
			continue
		}
		enabled[id] = struct{}{}
	}
	return enabled, nil
}

// AssertRulesObserved checks that every id in ids is present in the supplied
// registered-and-enabled set. Returns a non-nil *RuleAssertion listing the
// missing IDs (sorted, deduplicated) when any are absent.
func AssertRulesObserved(registered map[string]struct{}, ids ...string) *RuleAssertion {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(ids))
	var missing []string
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		if _, ok := registered[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return &RuleAssertion{MissingRules: missing}
}
