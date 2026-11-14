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
	return planproto.GuidedStepToProto(g)
}
