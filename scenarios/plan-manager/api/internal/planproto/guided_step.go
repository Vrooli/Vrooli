package planproto

import (
	planmodel "plan-manager/internal/planmodel"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// GuidedStepToProto maps the shared guided-flow steering model to the wire
// shape used by authoring, execution, and log endpoints.
func GuidedStepToProto(g planmodel.GuidedStep) *sharedv1.GuidedStep {
	return &sharedv1.GuidedStep{
		StepKind:       g.StepKind,
		Title:          g.Title,
		Summary:        g.Summary,
		Instructions:   append([]string(nil), g.Instructions...),
		RequiredInputs: append([]string(nil), g.RequiredInputs...),
		Examples:       append([]string(nil), g.Examples...),
		CommonMistakes: append([]string(nil), g.CommonMistakes...),
		NextActions:    NextActionsToProto(g.NextActions),
		Checklist:      ChecklistToProto(g.Checklist),
	}
}

// ChecklistToProto maps the full-disclosure checklist to the wire shape.
func ChecklistToProto(items []planmodel.ChecklistItem) []*sharedv1.ChecklistItem {
	out := make([]*sharedv1.ChecklistItem, 0, len(items))
	for _, item := range items {
		out = append(out, &sharedv1.ChecklistItem{
			Key:    item.Key,
			Label:  item.Label,
			State:  string(item.State),
			Detail: item.Detail,
		})
	}
	return out
}

// NextActionsToProto maps shared guided-flow actions to proto actions.
func NextActionsToProto(actions []planmodel.NextAction) []*sharedv1.NextAction {
	out := make([]*sharedv1.NextAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, &sharedv1.NextAction{
			Id:                 action.ID,
			Kind:               NextActionKindToProto(action.Kind),
			Label:              action.Label,
			Reason:             action.Reason,
			Argv:               append([]string(nil), action.Argv...),
			ContentPlaceholder: action.ContentPlaceholder,
			BlockedBy:          append([]string(nil), action.BlockedBy...),
		})
	}
	return out
}

// NextActionKindToProto maps the shared guided-flow action kind to the wire enum.
func NextActionKindToProto(kind planmodel.NextActionKind) sharedv1.NextActionKind {
	switch kind {
	case planmodel.NextActionRecommended:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOMMENDED
	case planmodel.NextActionAlternative:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_ALTERNATIVE
	case planmodel.NextActionOptional:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_OPTIONAL
	case planmodel.NextActionRecovery:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOVERY
	default:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_UNSPECIFIED
	}
}
