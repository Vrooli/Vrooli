package runs

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"test-genie/internal/orchestrator/phases"
	sharedruns "test-genie/internal/shared/runs"
	"test-genie/internal/shared/treedigest"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// Freshness status values.
const (
	freshnessFresh   = "fresh"
	freshnessStale   = "stale"
	freshnessUnknown = "unknown"
)

// digestFn computes the scenario's current tree digest (seam for tests).
var digestFn = treedigest.Compute

// CheckFreshness reports, per phase, whether some recorded run executed that
// phase (status passed) against the scenario's CURRENT working-tree digest.
// Empty phases default to the global required set (the quick preset —
// phases.FreshnessRequired, a code-level SSOT that is deliberately not
// per-scenario configurable). Read-only.
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

	requested := normalizePhases(req.Msg.GetPhases())
	defaulted := len(requested) == 0
	if defaulted {
		requested = phases.FreshnessRequired()
	}

	resp := checkFreshness(records, digest, requested)
	resp.Scenario = strings.TrimSpace(req.Msg.GetScenario())
	resp.SuggestedCommand = suggestedCommand(resp.Scenario, resp.GetPhases(), defaulted)
	return connect.NewResponse(resp), nil
}

func normalizePhases(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, p := range in {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// checkFreshness computes the per-phase verdicts. A phase is fresh iff some
// run whose TreeDigest equals the current digest contains the phase with
// status passed; verdicts are "unknown" when no digest-stamped runs exist at
// all (pre-digest history can never prove staleness). records are expected
// newest-first (the index order).
func checkFreshness(records []sharedruns.RunRecord, currentDigest string, phaseNames []string) *runspb.CheckFreshnessResponse {
	anyStamped := false
	for _, r := range records {
		if r.TreeDigest != "" {
			anyStamped = true
			break
		}
	}

	out := make([]*runspb.PhaseFreshness, 0, len(phaseNames))
	for _, name := range phaseNames {
		pf := &runspb.PhaseFreshness{Phase: name, Status: freshnessStale}
		if !anyStamped {
			pf.Status = freshnessUnknown
			out = append(out, pf)
			continue
		}
		for _, r := range records { // newest-first: first hit is the newest evidence
			if r.TreeDigest != currentDigest {
				continue
			}
			if status, ok := phaseStatusMap(r)[name]; ok && status == sharedruns.StatusPassed {
				pf.Status = freshnessFresh
				pf.LastRunId = r.RunID
				pf.LastRunCompletedAt = formatTime(r.CompletedAt)
				break
			}
		}
		if pf.Status == freshnessStale {
			// Context for the advisory message: the newest run (any digest)
			// that passed this phase, so the caller can say "last passed at X".
			for _, r := range records {
				if status, ok := phaseStatusMap(r)[name]; ok && status == sharedruns.StatusPassed {
					pf.LastRunId = r.RunID
					pf.LastRunCompletedAt = formatTime(r.CompletedAt)
					break
				}
			}
		}
		out = append(out, pf)
	}

	return &runspb.CheckFreshnessResponse{TreeDigest: currentDigest, Phases: out}
}

// suggestedCommand renders the copy-pastable remediation for stale/unknown
// phases. With the default (required) set it is the quick preset; with an
// explicit phase list it names exactly the stale phases.
func suggestedCommand(scenario string, verdicts []*runspb.PhaseFreshness, defaulted bool) string {
	var stale []string
	for _, v := range verdicts {
		if v.GetStatus() != freshnessFresh {
			stale = append(stale, v.GetPhase())
		}
	}
	if len(stale) == 0 {
		return ""
	}
	if defaulted {
		return fmt.Sprintf("test-genie execute %s --preset quick", scenario)
	}
	return fmt.Sprintf("test-genie execute %s %s", scenario, strings.Join(stale, " "))
}
