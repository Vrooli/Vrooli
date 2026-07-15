package agentopsdiag

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// ownerRef validates the owner selector for the override RPCs: only backlog
// items and initiatives own an override layer.
func (s *Service) ownerRef(sel *apipb.AgentOpsTargetSelector) (opsrunner.TargetRef, error) {
	owner, err := s.targetRef(sel)
	if err != nil {
		return opsrunner.TargetRef{}, err
	}
	if owner.Kind != agentops.TargetBacklogItem && owner.Kind != agentops.TargetInitiative {
		return opsrunner.TargetRef{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("target kind %q owns no binding-override layer (only backlog-item and initiative do)", owner.Kind))
	}
	return owner, nil
}

func (s *Service) requireWriter() error {
	if s.writer == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("binding-override administration is not configured"))
	}
	return nil
}

// ListBindingOverrides returns the raw override documents stored at one
// owner's layer with file-level provenance.
func (s *Service) ListBindingOverrides(_ context.Context, req *connect.Request[apipb.AgentOpsListBindingOverridesRequest]) (*connect.Response[apipb.AgentOpsListBindingOverridesResponse], error) {
	owner, err := s.ownerRef(req.Msg.GetOwner())
	if err != nil {
		return nil, err
	}
	if err := s.requireWriter(); err != nil {
		return nil, err
	}
	stored, err := s.writer.List(owner.Kind, owner.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &apipb.AgentOpsListBindingOverridesResponse{}
	for _, st := range stored {
		resp.Overrides = append(resp.Overrides, &apipb.AgentOpsBindingOverrideDocument{
			Binding:   bindingToProto(st.Binding),
			File:      st.File,
			Revision:  st.Revision,
			UpdatedAt: st.UpdatedAt,
		})
	}
	return connect.NewResponse(resp), nil
}

// overrideApplicableKinds returns the target kinds an owner's override can take
// effect on: a backlog-item override applies only to that item; an initiative
// override applies to the initiative itself and (by inheritance) to its member
// items.
func overrideApplicableKinds(owner agentops.TargetKind) []agentops.TargetKind {
	if owner == agentops.TargetInitiative {
		return []agentops.TargetKind{agentops.TargetInitiative, agentops.TargetBacklogItem}
	}
	return []agentops.TargetKind{agentops.TargetBacklogItem}
}

// PutBindingOverride validates and writes one override at the owner's layer
// (idempotent replace per operation+version). Every check is fail-closed and
// runs BEFORE the write; the write itself is atomic. Binding resolution is
// SNAPSHOT-at-Invoke: the new override affects only operations started
// afterwards.
func (s *Service) PutBindingOverride(_ context.Context, req *connect.Request[apipb.AgentOpsPutBindingOverrideRequest]) (*connect.Response[apipb.AgentOpsPutBindingOverrideResponse], error) {
	owner, err := s.ownerRef(req.Msg.GetOwner())
	if err != nil {
		return nil, err
	}
	if err := s.requireWriter(); err != nil {
		return nil, err
	}
	op := agentops.OperationID(req.Msg.GetOperation())
	version := req.Msg.GetOperationVersion()

	// The operation must be declared in the catalog at the pinned version.
	lc, ok := s.catalog.Contract(op, version)
	if !ok {
		if version == "" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("operation %q is not declared in the catalog", op))
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("operation %q has no authored contract at version %q", op, version))
	}

	// Mode checks are mandatory for writes: no registry means no write, never a
	// blind bind. A disabled veto names no runnable mode intent, but the mode it
	// pins must still be real so re-enabling never resurrects a dangling ref.
	if s.checker == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("mode registry unavailable: refusing to write an unvalidatable binding override"))
	}
	mode, modeRevision := req.Msg.GetMode(), req.Msg.GetModeRevision()
	if !s.checker.RevisionExists(mode, modeRevision) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("mode %q@%s is not registered", mode, modeRevision))
	}
	// The mode must be able to run the operation on at least one target kind
	// where this override can take effect (owner scope ∩ contract-compatible
	// kinds), using the same checks the live path resolves with.
	compatible := false
	for _, kind := range overrideApplicableKinds(owner.Kind) {
		if agentops.CheckOperationTargetCompatibility(op, lc.Contract.TargetRequirements.Capabilities, kind) != nil {
			continue
		}
		if s.checker.ModeCompatible(mode, op, kind) {
			compatible = true
			break
		}
	}
	if !compatible {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("mode %q cannot run operation %q on any target kind this %s override applies to", mode, op, owner.Kind))
	}

	binding, err := opsrunner.BuildOverride(owner.Kind, owner.ID, op, version, mode, modeRevision, req.Msg.GetDisabled())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	stored, err := s.writer.Put(owner.Kind, owner.ID, binding)
	if err != nil {
		// The writer fails closed on schema violations and same-layer ambiguity;
		// both are caller-correctable preconditions, not server faults.
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&apipb.AgentOpsPutBindingOverrideResponse{
		Stored:   bindingToProto(stored.Binding),
		File:     stored.File,
		Revision: stored.Revision,
	}), nil
}

// DeleteBindingOverride removes the owner's override for an operation(+version
// pin). Absence is found=false, not an error. Running operations keep the
// binding they pinned (SNAPSHOT-at-Invoke).
func (s *Service) DeleteBindingOverride(_ context.Context, req *connect.Request[apipb.AgentOpsDeleteBindingOverrideRequest]) (*connect.Response[apipb.AgentOpsDeleteBindingOverrideResponse], error) {
	owner, err := s.ownerRef(req.Msg.GetOwner())
	if err != nil {
		return nil, err
	}
	if err := s.requireWriter(); err != nil {
		return nil, err
	}
	found, err := s.writer.Delete(owner.Kind, owner.ID, agentops.OperationID(req.Msg.GetOperation()), req.Msg.GetOperationVersion())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&apipb.AgentOpsDeleteBindingOverrideResponse{Found: found}), nil
}
