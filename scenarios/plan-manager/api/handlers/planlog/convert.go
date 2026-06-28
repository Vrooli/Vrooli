package planlog

import (
	internalplanlog "plan-manager/internal/planlog"
	"plan-manager/internal/planproto"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// This file is the only translation point between the proto wire types
// (vrooli.plan_manager.v1.log + .shared) and the log domain vocabulary. The
// domain layer never imports proto (api-steer §7).

func entryToProto(e internalplanlog.Entry) *sharedv1.LogEntry {
	return planproto.LogEntryToProto(e)
}

func entriesToProto(entries []internalplanlog.Entry) []*sharedv1.LogEntry {
	return planproto.LogEntriesToProto(entries)
}

func summaryToProto(s internalplanlog.Summary) *sharedv1.LogSummary {
	return planproto.LogSummaryToProto(s)
}

func guidedStepToProto(g internalplanlog.GuidedStep) *sharedv1.GuidedStep {
	return &sharedv1.GuidedStep{
		StepKind:       g.StepKind,
		Title:          g.Title,
		Summary:        g.Summary,
		Instructions:   append([]string(nil), g.Instructions...),
		RequiredInputs: append([]string(nil), g.RequiredInputs...),
		Examples:       append([]string(nil), g.Examples...),
		CommonMistakes: append([]string(nil), g.CommonMistakes...),
		NextActions:    nextActionsToProto(g.NextActions),
	}
}

func nextActionsToProto(actions []internalplanlog.NextAction) []*sharedv1.NextAction {
	out := make([]*sharedv1.NextAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, &sharedv1.NextAction{
			Id:                 action.ID,
			Kind:               nextActionKindToProto(action.Kind),
			Label:              action.Label,
			Reason:             action.Reason,
			Argv:               append([]string(nil), action.Argv...),
			ContentPlaceholder: action.ContentPlaceholder,
			BlockedBy:          append([]string(nil), action.BlockedBy...),
		})
	}
	return out
}

func nextActionKindToProto(kind internalplanlog.NextActionKind) sharedv1.NextActionKind {
	switch kind {
	case internalplanlog.NextActionRecommended:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOMMENDED
	case internalplanlog.NextActionAlternative:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_ALTERNATIVE
	case internalplanlog.NextActionOptional:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_OPTIONAL
	case internalplanlog.NextActionRecovery:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_RECOVERY
	default:
		return sharedv1.NextActionKind_NEXT_ACTION_KIND_UNSPECIFIED
	}
}
