package agentopsdiag

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"connectrpc.com/connect"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// latestContracts returns the highest authored version of every catalog
// operation, ordered by operation id. It is the operation set the per-target
// projections (compatibility, resolved bindings) iterate.
func (s *Service) latestContracts() []opscatalog.LoadedContract {
	byID := map[agentops.OperationID]opscatalog.LoadedContract{}
	for _, lc := range s.catalog.Contracts() {
		byID[lc.Contract.ID] = lc // Contracts() is version-ascending per id
	}
	out := make([]opscatalog.LoadedContract, 0, len(byID))
	for _, lc := range byID {
		out = append(out, lc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Contract.ID < out[j].Contract.ID })
	return out
}

// ListOperationCatalog returns every authored operation contract with its
// pinned revision and compatible target kinds.
func (s *Service) ListOperationCatalog(_ context.Context, _ *connect.Request[apipb.AgentOpsListOperationCatalogRequest]) (*connect.Response[apipb.AgentOpsListOperationCatalogResponse], error) {
	resp := &apipb.AgentOpsListOperationCatalogResponse{}
	for _, lc := range s.catalog.Contracts() {
		entry := &apipb.AgentOpsCatalogEntry{
			Contract: contractToProto(lc.Contract),
			Revision: lc.Revision,
		}
		for _, kind := range agentops.CompatibleTargets(lc.Contract.TargetRequirements.Capabilities) {
			entry.CompatibleTargets = append(entry.CompatibleTargets, targetKindToProto(kind))
		}
		resp.Entries = append(resp.Entries, entry)
	}
	return connect.NewResponse(resp), nil
}

// ListCompatibleModes reports which authored modes can perform which catalog
// operations on a target, with every verdict computed by the same agentops
// checks the live path uses (operation↔target capabilities via
// CheckOperationTargetCompatibility, mode↔operation/target via the
// LivePreparer-backed ModeChecker). A mode is never listed without validated
// verdicts, so this RPC fails closed when the mode registry is unavailable.
func (s *Service) ListCompatibleModes(_ context.Context, req *connect.Request[apipb.AgentOpsListCompatibleModesRequest]) (*connect.Response[apipb.AgentOpsListCompatibleModesResponse], error) {
	target, err := s.targetRef(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	if len(s.modeDefs) == 0 || s.checker == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mode registry unavailable: compatibility verdicts cannot be computed"))
	}

	contracts := s.latestContracts()
	if op := req.Msg.GetOperation(); op != "" {
		filtered := contracts[:0]
		for _, lc := range contracts {
			if string(lc.Contract.ID) == op {
				filtered = append(filtered, lc)
			}
		}
		if len(filtered) == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("operation %q is not declared in the catalog", op))
		}
		contracts = filtered
	}

	modeIDs := make([]string, 0, len(s.modeDefs))
	for id := range s.modeDefs {
		modeIDs = append(modeIDs, id)
	}
	sort.Strings(modeIDs)

	resp := &apipb.AgentOpsListCompatibleModesResponse{}
	for _, id := range modeIDs {
		def := s.modeDefs[id]
		digest, err := agentops.DigestOf(def)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("digest mode %q: %w", id, err))
		}
		entry := &apipb.AgentOpsCompatibleMode{
			Mode:         id,
			ModeRevision: digest, // the registry pins revisions by content
			ModeDigest:   digest,
			TargetKind:   targetKindToProto(agentops.TargetKind(def.Target.Kind)),
		}
		for _, lc := range contracts {
			verdict := &apipb.AgentOpsModeOperationVerdict{
				Operation:        string(lc.Contract.ID),
				OperationVersion: lc.Contract.Version,
			}
			switch compatErr := agentops.CheckOperationTargetCompatibility(lc.Contract.ID, lc.Contract.TargetRequirements.Capabilities, target.Kind); {
			case compatErr != nil:
				verdict.Reason = compatErr.Error()
			case !s.checker.ModeCompatible(id, lc.Contract.ID, target.Kind):
				verdict.Reason = fmt.Sprintf("mode %q targets %q and cannot run operation %q on target kind %q", id, def.Target.Kind, lc.Contract.ID, target.Kind)
			default:
				verdict.Compatible = true
			}
			entry.Verdicts = append(entry.Verdicts, verdict)
		}
		resp.Modes = append(resp.Modes, entry)
	}
	return connect.NewResponse(resp), nil
}

// resolutionErrorCode maps a fail-closed binding resolution error onto the
// typed per-operation code the wire contract documents.
func resolutionErrorCode(err error) string {
	switch {
	case errors.Is(err, agentops.ErrNoBinding):
		return "no-binding"
	case errors.Is(err, agentops.ErrInvalidOverride):
		return "invalid-override"
	case errors.Is(err, agentops.ErrDeletedRevision):
		return "deleted-revision"
	case errors.Is(err, agentops.ErrIncompatibleMode):
		return "incompatible-mode"
	default:
		return "internal"
	}
}

// GetResolvedBindings resolves the winning binding for every catalog operation
// compatible with the target, with contributing layers for inherited-vs-
// overridden rendering. Per-operation resolution failures are typed results,
// never transport errors.
func (s *Service) GetResolvedBindings(ctx context.Context, req *connect.Request[apipb.AgentOpsGetResolvedBindingsRequest]) (*connect.Response[apipb.AgentOpsGetResolvedBindingsResponse], error) {
	target, err := s.targetRef(req.Msg.GetTarget())
	if err != nil {
		return nil, err
	}
	resp := &apipb.AgentOpsGetResolvedBindingsResponse{}
	for _, lc := range s.latestContracts() {
		op := lc.Contract.ID
		if agentops.CheckOperationTargetCompatibility(op, lc.Contract.TargetRequirements.Capabilities, target.Kind) != nil {
			continue
		}
		row := &apipb.AgentOpsResolvedOperationBinding{
			Operation:        string(op),
			OperationVersion: lc.Contract.Version,
		}

		// Contributing layers: the authored system default plus every in-scope
		// override document. Listing is best-effort per operation — a malformed
		// override surfaces through the resolver's typed error below. The lookup
		// pins the row's contract version: shipped system bindings pin an exact
		// operation_version, and a version-agnostic lookup would miss them (the
		// same trap ops_reroute.go documents for the live Invoke path).
		if sys, ok := s.catalog.SystemBindingFor(op, lc.Contract.Version); ok {
			row.Contributions = append(row.Contributions, &apipb.AgentOpsBindingContribution{Binding: bindingToProto(sys.Binding)})
		}
		if s.overrides != nil {
			if ovs, ovErr := s.overrides.OverridesFor(ctx, target, op); ovErr == nil {
				for _, b := range ovs {
					row.Contributions = append(row.Contributions, &apipb.AgentOpsBindingContribution{Binding: bindingToProto(b)})
				}
			}
		}

		// Resolve at the row's exact contract version, mirroring what a live
		// Invoke pins. An empty OperationVersion here would fail closed with
		// no-binding for every operation whose system default pins a version —
		// which is all of the shipped catalog.
		res, resErr := s.resolver.Resolve(ctx, opsrunner.InvokeRequest{Target: target, Operation: op, OperationVersion: lc.Contract.Version})
		if resErr != nil {
			row.Error = resolutionErrorCode(resErr)
			row.ErrorMessage = resErr.Error()
		} else {
			row.Resolved = true
			row.Binding = resolvedBindingToProto(res.Binding)
			row.PolicyId = res.PolicyID
			row.PolicyRevision = res.PolicyRevision
			for _, c := range row.Contributions {
				if c.GetBinding().GetLayer() == row.GetBinding().GetLayer() {
					c.Winning = true
				}
			}
		}
		resp.Operations = append(resp.Operations, row)
	}
	return connect.NewResponse(resp), nil
}
