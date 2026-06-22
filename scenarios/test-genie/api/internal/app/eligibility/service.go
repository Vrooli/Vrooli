// Package eligibility hosts the Connect-RPC EligibilityService handler that
// exposes test-genie's routed-test-db eligibility decision to external
// callers (GCT, swarm-manager, the test-genie CLI). It is a thin wrapper
// over internal/eligibility.Checker.
package eligibility

import (
	"context"
	"fmt"
	"path/filepath"
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

	for _, f := range elig.BlockingFindings {
		resp.Violations = append(resp.Violations, &eligpb.Violation{
			RuleId:   f.Code,
			Severity: f.Severity,
			File:     f.Location,
			Excerpt:  f.Message,
		})
	}

	if !elig.Routed {
		resp.DisqualifyingReasons = buildReasons(elig)
	}

	return resp
}

// buildReasons composes the human-readable explanations consumers display in
// CLI output and UI tooltips. One reason per distinct storage-health isolation
// finding code (deduped).
func buildReasons(elig internalelig.Eligibility) []string {
	seen := map[string]struct{}{}
	var reasons []string
	for _, f := range elig.BlockingFindings {
		if _, ok := seen[f.Code]; ok {
			continue
		}
		seen[f.Code] = struct{}{}
		reasons = append(reasons, findingReason(f))
	}
	if len(reasons) == 0 {
		reasons = []string{"Scenario is not eligible for the routed test-db path."}
	}
	return reasons
}

func findingReason(f internalelig.IsolationFinding) string {
	switch f.Code {
	case internalelig.CodeRoutedSeamsUnwired:
		return "Scenario has not wired the routed test-DB seams (storage-health: ROUTED_SEAMS_UNWIRED); destructive E2E cannot be isolated."
	case internalelig.CodeStorageIsolationUnverified:
		return "Scenario's API isolation cannot be statically verified (storage-health: STORAGE_ISOLATION_UNVERIFIED, non-Go API); destructive E2E is refused fail-closed."
	default:
		if msg := strings.TrimSpace(f.Message); msg != "" {
			return fmt.Sprintf("Scenario failed storage isolation (%s): %s", f.Code, msg)
		}
		return fmt.Sprintf("Scenario failed storage isolation check %s.", f.Code)
	}
}
