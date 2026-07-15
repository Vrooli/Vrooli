package agentopsdiag

import (
	"fmt"
	"strings"

	"swarm-manager/internal/agentops"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
)

// The enum wire names follow a fixed prefix + SCREAMING_SNAKE convention derived
// from the kebab-case runtime values, so the generated *_value maps give a
// robust string->enum mapping without a hand-maintained per-value switch.

func enumValue[E ~int32](values map[string]int32, prefix, kebab string) E {
	name := prefix + strings.ToUpper(strings.ReplaceAll(kebab, "-", "_"))
	return E(values[name])
}

func targetKindToProto(kind agentops.TargetKind) domainpb.OperatingModeTargetKind {
	return enumValue[domainpb.OperatingModeTargetKind](domainpb.OperatingModeTargetKind_value, "OPERATING_MODE_TARGET_KIND_", string(kind))
}

func targetKindFromProto(k domainpb.OperatingModeTargetKind) (agentops.TargetKind, bool) {
	switch k {
	case domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_BACKLOG_ITEM:
		return agentops.TargetBacklogItem, true
	case domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_INITIATIVE:
		return agentops.TargetInitiative, true
	case domainpb.OperatingModeTargetKind_OPERATING_MODE_TARGET_KIND_PLAN_EXECUTION:
		return agentops.TargetPlanExecution, true
	default:
		return "", false
	}
}

func bindingLayerToProto(layer agentops.BindingLayer) domainpb.AgentOpsBindingLayer {
	return enumValue[domainpb.AgentOpsBindingLayer](domainpb.AgentOpsBindingLayer_value, "AGENT_OPS_BINDING_LAYER_", string(layer))
}

func workflowStateToProto(state agentops.WorkflowState) domainpb.AgentOpsWorkflowState {
	return enumValue[domainpb.AgentOpsWorkflowState](domainpb.AgentOpsWorkflowState_value, "AGENT_OPS_WORKFLOW_STATE_", string(state))
}

func actionToProto(name agentops.ActionName) domainpb.AgentOpsDomainAction {
	return enumValue[domainpb.AgentOpsDomainAction](domainpb.AgentOpsDomainAction_value, "AGENT_OPS_DOMAIN_ACTION_", string(name))
}

func capabilityToProto(cap agentops.CapabilityID) domainpb.AgentOpsCapability {
	return enumValue[domainpb.AgentOpsCapability](domainpb.AgentOpsCapability_value, "AGENT_OPS_CAPABILITY_", string(cap))
}

func resolvedBindingToProto(b agentops.ResolvedBinding) *domainpb.AgentOpsOperationBinding {
	out := &domainpb.AgentOpsOperationBinding{
		Operation:    string(b.Operation),
		Layer:        bindingLayerToProto(b.Layer),
		Mode:         b.Mode,
		ModeRevision: b.ModeRevision,
	}
	if b.Owner != nil {
		out.Owner = &domainpb.AgentOpsBindingOwner{Kind: b.Owner.Kind, Id: b.Owner.ID}
	}
	return out
}

func provenanceToProto(p agentops.ExecutionProvenance) *domainpb.AgentOpsExecutionProvenance {
	return &domainpb.AgentOpsExecutionProvenance{
		Operation:             string(p.Operation),
		OperationVersion:      p.OperationVersion,
		Binding:               &domainpb.AgentOpsProvenanceBinding{Layer: bindingLayerToProto(p.Binding.Layer), OwnerKind: p.Binding.OwnerKind, OwnerId: p.Binding.OwnerID},
		Mode:                  p.Mode,
		ModeRevision:          p.ModeRevision,
		CompiledModeDigest:    p.CompiledModeDigest,
		PromptCatalogRevision: p.PromptCatalogRevision,
		PromptCatalogDigest:   p.PromptCatalogDigest,
		Target:                &domainpb.AgentOpsProvenanceTarget{Kind: targetKindToProto(p.Target.Kind), Id: p.Target.ID},
		CallerInputDigest:     p.CallerInputDigest,
		PolicyRevision:        p.PolicyRevision,
		WorkflowInstanceId:    p.WorkflowInstanceID,
	}
}

func workflowToProto(w agentops.WorkflowInstance) *domainpb.AgentOpsWorkflowInstance {
	out := &domainpb.AgentOpsWorkflowInstance{
		SchemaVersion:   w.SchemaVersion,
		InstanceId:      w.InstanceID,
		DomainKind:      w.Domain.Kind,
		DomainId:        w.Domain.ID,
		State:           workflowStateToProto(w.State),
		IdempotencyKeys: append([]string(nil), w.IdempotencyKeys...),
		Version:         int32(w.Version),
	}
	if w.Strategy != nil {
		out.Strategy = &domainpb.AgentOpsMemberItemStrategyConfig{
			Strategy: enumValue[domainpb.AgentOpsMemberItemStrategy](domainpb.AgentOpsMemberItemStrategy_value, "AGENT_OPS_MEMBER_ITEM_STRATEGY_", w.Strategy.Name),
		}
	}
	for _, op := range w.Operations {
		out.Operations = append(out.Operations, &domainpb.AgentOpsOperationExecutionRecord{
			Operation: string(op.Operation), ExecutionId: op.ExecutionID, IdempotencyKey: op.IdempotencyKey,
			ProvenanceDigest: op.ProvenanceDigest, State: op.State, Outcome: op.Outcome, RunId: op.RunID,
		})
	}
	for _, d := range w.Decisions {
		out.Decisions = append(out.Decisions, &domainpb.AgentOpsHumanDecision{
			Decision: d.Decision, Actor: d.Actor, AtVersion: int32(d.AtVersion), Note: d.Note,
		})
	}
	for _, t := range w.Timers {
		out.Timers = append(out.Timers, &domainpb.AgentOpsScheduledIntent{
			Intent: t.Intent, Action: actionToProto(t.Action), NotBefore: t.NotBefore,
		})
	}
	for _, a := range w.LegalActions {
		out.LegalActions = append(out.LegalActions, actionToProto(a))
	}
	return out
}

// bindingToProto projects an authored (unresolved) binding document onto the
// wire shape, preserving its layer, owner, version pin, and disabled veto.
func bindingToProto(b agentops.OperationBinding) *domainpb.AgentOpsOperationBinding {
	out := &domainpb.AgentOpsOperationBinding{
		Operation:        string(b.Operation),
		OperationVersion: b.OperationVersion,
		Layer:            bindingLayerToProto(b.Layer),
		Mode:             b.Mode,
		ModeRevision:     b.ModeRevision,
		Disabled:         b.Disabled,
	}
	if b.Owner != nil {
		out.Owner = &domainpb.AgentOpsBindingOwner{Kind: b.Owner.Kind, Id: b.Owner.ID}
	}
	return out
}

// contractToProto projects a loaded operation contract onto the wire shape:
// identity, required capabilities, caller-input specs, declared result fields,
// and closed outcomes.
func contractToProto(oc agentops.OperationContract) *domainpb.AgentOpsOperationContract {
	out := &domainpb.AgentOpsOperationContract{
		Id:          string(oc.ID),
		Version:     oc.Version,
		Summary:     oc.Summary,
		Description: oc.Description,
	}
	for _, c := range oc.TargetRequirements.Capabilities {
		out.RequiredCapabilities = append(out.RequiredCapabilities, capabilityToProto(c))
	}
	for _, in := range oc.Inputs {
		out.Inputs = append(out.Inputs, &domainpb.AgentOpsCallerInput{
			Name: in.Name, Type: in.Type, Required: in.Required,
			Sensitivity: in.Sensitivity, Retention: in.Retention, Description: in.Description,
		})
	}
	for _, f := range oc.Result.Fields {
		rf := &domainpb.AgentOpsResultField{
			Name: f.Name, Type: f.Type, Required: f.Required, Description: f.Description,
		}
		for _, v := range f.Enum {
			rf.Enum = append(rf.Enum, fmt.Sprint(v))
		}
		out.ResultFields = append(out.ResultFields, rf)
	}
	for _, o := range oc.Outcomes {
		out.Outcomes = append(out.Outcomes, &domainpb.AgentOpsOutcome{
			Name: o.Name, Disposition: o.Disposition, Description: o.Description,
		})
	}
	return out
}
