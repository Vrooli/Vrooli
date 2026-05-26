// Package eligibility hosts the Connect-RPC EligibilityService handler that
// exposes test-genie's routed-test-db eligibility decision to external
// callers (GCT, swarm-manager, the test-genie CLI). It is a thin wrapper
// over internal/eligibility.Checker.
package eligibility

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"

	internalelig "test-genie/internal/eligibility"
	"test-genie/internal/orchestrator/workspace"

	eligpb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility"
)

// Checker is the subset of *internal/eligibility.Checker the service depends
// on. Defined here so tests can inject a stub without touching the auditor
// HTTP path.
type Checker interface {
	Check(ctx context.Context, scenario string, mapping workspace.Mapping) (internalelig.Eligibility, error)
}

// Service implements eligibility_v1connect.EligibilityServiceHandler.
type Service struct {
	checker       Checker
	scenariosRoot string
}

// NewService returns a Service bound to checker. scenariosRoot is used to
// resolve each request's scenario into a workspace.Mapping so the underlying
// scan can address its physical files.
func NewService(checker Checker, scenariosRoot string) *Service {
	return &Service{checker: checker, scenariosRoot: scenariosRoot}
}

// Check runs (or reuses a cached) eligibility decision for the requested
// scenario. Connect-RPC status codes:
//   - InvalidArgument: missing scenario name.
//   - Internal: auditor scan or rule-registry fetch failed.
func (s *Service) Check(ctx context.Context, req *connect.Request[eligpb.CheckRequest]) (*connect.Response[eligpb.CheckResponse], error) {
	name := strings.TrimSpace(req.Msg.GetScenario())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario is required"))
	}

	mapping := workspace.Mapping{
		PhysicalScenarioDir: filepath.Join(s.scenariosRoot, name),
	}

	elig, err := s.checker.Check(ctx, name, mapping)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(toCheckResponse(elig)), nil
}

func toCheckResponse(elig internalelig.Eligibility) *eligpb.CheckResponse {
	resp := &eligpb.CheckResponse{
		Routed: elig.Routed,
	}

	for _, v := range elig.Violations {
		resp.Violations = append(resp.Violations, &eligpb.Violation{
			RuleId:   v.RuleID,
			Severity: v.Severity,
			File:     v.FilePath,
			Line:     uint32(v.LineNumber),
			Excerpt:  v.Title,
		})
	}

	if elig.RuleAssertion != nil && len(elig.RuleAssertion.MissingRules) > 0 {
		missing := append([]string(nil), elig.RuleAssertion.MissingRules...)
		sort.Strings(missing)
		resp.RuleAssertion = &eligpb.RuleAssertion{MissingRules: missing}
	}

	if !elig.Routed {
		resp.DisqualifyingReasons = buildReasons(elig)
	}

	return resp
}

// buildReasons composes the human-readable explanations consumers display in
// CLI output and UI tooltips. One reason per distinct rule_id (deduped) plus
// one reason for missing rules.
func buildReasons(elig internalelig.Eligibility) []string {
	seen := map[string]struct{}{}
	var reasons []string
	for _, v := range elig.Violations {
		key := v.CanonicalRuleID()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		reasons = append(reasons, ruleIDReason(key))
	}
	if elig.RuleAssertion != nil && len(elig.RuleAssertion.MissingRules) > 0 {
		missing := append([]string(nil), elig.RuleAssertion.MissingRules...)
		sort.Strings(missing)
		reasons = append(reasons, fmt.Sprintf("Auditor did not register required routing rules: %s", strings.Join(missing, ", ")))
	}
	if len(reasons) == 0 {
		reasons = []string{"Scenario is not eligible for the routed test-db path."}
	}
	return reasons
}

func ruleIDReason(ruleID string) string {
	switch ruleID {
	case internalelig.RuleRoutedDrivers:
		return "Scenario opens database connections outside the routed driver (rule: routed_database_drivers)."
	case internalelig.RuleRoutedHandleCapture:
		return "Scenario captures a database handle without going through RoutedDB (rule: routed_database_handle_capture)."
	case internalelig.RuleDatabaseBackoff:
		return "Scenario uses sql.Open or other raw connect paths that bypass RoutedDB (rule: database_backoff)."
	default:
		return fmt.Sprintf("Scenario violates routing rule %s.", ruleID)
	}
}
