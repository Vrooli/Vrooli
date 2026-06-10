package runs

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"test-genie/internal/orchestrator/phases"
	sharedruns "test-genie/internal/shared/runs"

	freshness "github.com/vrooli/freshness-go"
	"github.com/vrooli/freshness-go/treedigest"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// digestFn computes the scenario's current tree digest (seam for tests).
var digestFn = treedigest.Compute

// CheckFreshness reports, per phase, whether some recorded run executed that
// phase (status passed) against the scenario's CURRENT working-tree digest.
// Empty phases default to the global required set (the quick preset —
// phases.FreshnessRequired, a code-level SSOT that is deliberately not
// per-scenario configurable). Read-only. The verdict semantics live in the
// shared freshness-go package; this RPC only resolves inputs and converts the
// report to the wire shape.
func (s *Service) CheckFreshness(ctx context.Context, req *connect.Request[runspb.CheckFreshnessRequest]) (*connect.Response[runspb.CheckFreshnessResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}

	digest, err := digestFn(dir)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("compute tree digest: %w", err))
	}

	records, err := sharedruns.NewIndex(dir).List()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	requested := freshness.NormalizePhases(req.Msg.GetPhases())
	defaulted := len(requested) == 0
	if defaulted {
		requested = phases.FreshnessRequired()
	}

	report := freshness.Check(records, digest, requested)
	resp := toFreshnessResponse(report)
	resp.Scenario = strings.TrimSpace(req.Msg.GetScenario())
	resp.SuggestedCommand = freshness.SuggestedCommand(resp.Scenario, report.Phases, defaulted)
	return connect.NewResponse(resp), nil
}

// toFreshnessResponse converts the shared verdict report to the wire shape.
func toFreshnessResponse(report freshness.Report) *runspb.CheckFreshnessResponse {
	out := make([]*runspb.PhaseFreshness, 0, len(report.Phases))
	for _, v := range report.Phases {
		out = append(out, &runspb.PhaseFreshness{
			Phase:              v.Phase,
			Status:             v.Status,
			LastRunId:          v.LastRunID,
			LastRunCompletedAt: v.LastRunCompletedAt,
		})
	}
	return &runspb.CheckFreshnessResponse{TreeDigest: report.TreeDigest, Phases: out}
}
