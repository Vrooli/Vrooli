package gate

import (
	"context"
	"time"

	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/gate"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/runs"

	"github.com/vrooli/api-core/targetmodel"
	"google.golang.org/protobuf/types/known/timestamppb"

	gatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate"
)

// This file is the single translation point between the proto-free gate domain
// (its seams + DTOs) and the concrete registry / dispatch / runs services and
// the proto wire types. The gate domain never imports a sibling domain or proto;
// these adapters do.

// ---- proto <-> domain translations (api-steer §7) ----

func domainGateToProto(g gate.Gate) *gatev1.Gate {
	return &gatev1.Gate{
		Id:             g.ID,
		Scenario:       g.Scenario,
		TargetRevision: g.TargetRevision,
		Verb:           g.Verb,
		Args:           g.Args,
		Verdict:        verdictToProto(g.Verdict),
		TotalTargets:   int32(g.TotalTargets),
		Passed:         int32(g.Passed),
		Failed:         int32(g.Failed),
		Pending:        int32(g.Pending),
		CreatedAt:      timestamppb.New(g.CreatedAt),
	}
}

func verdictToProto(v gate.GateVerdict) gatev1.GateVerdict {
	switch v {
	case gate.VerdictPending:
		return gatev1.GateVerdict_GATE_VERDICT_PENDING
	case gate.VerdictPassed:
		return gatev1.GateVerdict_GATE_VERDICT_PASSED
	case gate.VerdictFailed:
		return gatev1.GateVerdict_GATE_VERDICT_FAILED
	default:
		return gatev1.GateVerdict_GATE_VERDICT_UNSPECIFIED
	}
}

func dispositionToProto(d gate.OSDisposition) gatev1.OSDisposition {
	switch d {
	case gate.OSDispositionPending:
		return gatev1.OSDisposition_OS_DISPOSITION_PENDING
	case gate.OSDispositionPassed:
		return gatev1.OSDisposition_OS_DISPOSITION_PASSED
	case gate.OSDispositionFailed:
		return gatev1.OSDisposition_OS_DISPOSITION_FAILED
	case gate.OSDispositionAborted:
		return gatev1.OSDisposition_OS_DISPOSITION_ABORTED
	case gate.OSDispositionNoNode:
		return gatev1.OSDisposition_OS_DISPOSITION_NO_NODE
	case gate.OSDispositionDispatchFailed:
		return gatev1.OSDisposition_OS_DISPOSITION_DISPATCH_FAILED
	default:
		return gatev1.OSDisposition_OS_DISPOSITION_UNSPECIFIED
	}
}

func domainResultToProto(r gate.OSResult) *gatev1.OSResult {
	return &gatev1.OSResult{
		Os:          r.OS,
		NodeId:      r.NodeID,
		RunId:       r.RunID,
		Disposition: dispositionToProto(r.Disposition),
		ExitCode:    r.ExitCode,
		Detail:      r.Detail,
	}
}

func domainResultsToProto(results []gate.OSResult) []*gatev1.OSResult {
	out := make([]*gatev1.OSResult, 0, len(results))
	for _, r := range results {
		out = append(out, domainResultToProto(r))
	}
	return out
}

// ---- seam adapters (proto-free domain <-> concrete services) ----

// nodeListerAdapter projects registry nodes down to the gate NodeRef set.
type nodeListerAdapter struct {
	svc registry.Service
}

var _ gate.NodeLister = nodeListerAdapter{}

func (a nodeListerAdapter) ListNodes(ctx context.Context) ([]gate.NodeRef, error) {
	nodes, err := a.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]gate.NodeRef, 0, len(nodes))
	for _, n := range nodes {
		revoked := n.Revoked()
		out = append(out, gate.NodeRef{
			ID: n.ID, OS: n.OS, Arch: n.Arch, Revoked: revoked,
			Target: targetmodel.Target{
				ID: n.ID, Label: n.Name, Platform: "desktop", OS: n.OS, Architecture: n.Arch,
				DeviceKind: "desktop", NodeID: n.ID, Capabilities: append([]string(nil), n.Capabilities...),
				Transport: targetmodel.Transport{Kind: targetmodel.TransportBridge, ID: n.ID, Trust: "bridge", Available: true},
				Available: !revoked, Reason: bridgeTargetReason(revoked, hasScenarioTestScope(n.Scopes)),
				MissingCapability: "bridge dispatch reachability", NextAction: "restore the node channel and protocol compatibility",
				Health:      targetmodel.TargetHealth{Status: "registered"},
				BridgeTrust: &targetmodel.BridgeTrust{Registered: !revoked, DispatchAuthorized: hasScenarioTestScope(n.Scopes)},
				Revoked:     revoked,
			},
		})
	}
	return out, nil
}

func bridgeTargetReason(revoked, dispatchAuthorized bool) string {
	if revoked {
		return targetmodel.ReasonBridgeRevoked
	}
	if !dispatchAuthorized {
		return targetmodel.ReasonBridgeNoDispatchScope
	}
	return targetmodel.ReasonBridgeAuthorizedDesktop
}

func hasScenarioTestScope(scopes []string) bool {
	for _, scope := range scopes {
		if scope == "scenario test" || scope == "scenario test*" {
			return true
		}
	}
	return false
}

// runnerAdapter binds the gate Runner seam to the SHARED dispatch service (which
// enforces the allowlist + per-node scopes + audit) and the runs service (the
// durable run lifecycle), so a gate validation run is dispatched and tracked
// exactly like any other job — gate reimplements neither.
type runnerAdapter struct {
	dispatchSvc dispatch.Service
	runsSvc     runs.Service
}

var _ gate.Runner = runnerAdapter{}

func (a runnerAdapter) Dispatch(ctx context.Context, in gate.DispatchRequest) (string, error) {
	dec, err := a.dispatchSvc.Dispatch(ctx, dispatch.DispatchInput{
		Actor: in.Actor,
		Job: dispatch.Job{
			NodeID:         in.NodeID,
			Scenario:       in.Scenario,
			Verb:           in.Verb,
			Args:           in.Args,
			TimeoutSeconds: in.TimeoutSeconds,
		},
	})
	if err != nil {
		return "", err
	}
	return dec.RunID, nil
}

func (a runnerAdapter) Verdict(ctx context.Context, runID string) (gate.RunVerdict, error) {
	run, _, err := a.runsSvc.Get(ctx, runID)
	if err != nil {
		return gate.RunVerdict{}, err
	}
	return runVerdict(run), nil
}

func (a runnerAdapter) Wait(ctx context.Context, runID string, timeout time.Duration) (gate.RunVerdict, error) {
	run, _, err := a.runsSvc.Wait(ctx, runID, timeout)
	if err != nil {
		return gate.RunVerdict{}, err
	}
	return runVerdict(run), nil
}

// runVerdict projects a durable run onto the gate's proto-free verdict DTO.
func runVerdict(run runs.Run) gate.RunVerdict {
	return gate.RunVerdict{
		Terminal: run.Status.Terminal(),
		Passed:   run.Status == runs.StatusPassed,
		Aborted:  run.Status == runs.StatusAborted,
		ExitCode: run.ExitCode,
		Detail:   run.Status.String(),
	}
}
