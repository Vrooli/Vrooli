package wire

import (
	"research-desk/internal/home"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
)

var stateWire = map[home.State]activityv1.CommissionState{
	home.StateEnded:    activityv1.CommissionState_COMMISSION_STATE_ENDED,
	home.StatePrepared: activityv1.CommissionState_COMMISSION_STATE_PREPARED,
	home.StateUnknown:  activityv1.CommissionState_COMMISSION_STATE_UNKNOWN,
	home.StateStopped:  activityv1.CommissionState_COMMISSION_STATE_STOPPED,
	home.StateLive:     activityv1.CommissionState_COMMISSION_STATE_LIVE,
	home.StatePaused:   activityv1.CommissionState_COMMISSION_STATE_PAUSED,
	home.StateLimited:  activityv1.CommissionState_COMMISSION_STATE_LIMITED,
}

// State converts a commission state; an empty state is unspecified.
func State(s home.State) activityv1.CommissionState { return stateWire[s] }

// Workstreams converts mission_check.py's workstream rows.
func Workstreams(list []home.Workstream) []*activityv1.Workstream {
	out := make([]*activityv1.Workstream, 0, len(list))
	for _, w := range list {
		out = append(out, &activityv1.Workstream{Id: w.ID, Status: w.Status, Showcase: w.Showcase, Title: w.Title, Magnitude: w.Magnitude})
	}
	return out
}

// NeverStarted converts an unknown commission's P1 answer.
func NeverStarted(n home.NeverStarted) *activityv1.NeverStarted {
	return &activityv1.NeverStarted{Assertable: n.Assertable, Reason: n.Reason}
}

var readWire = map[home.ReadWhen]activityv1.ReadWhen{
	home.ReadNextUnit:         activityv1.ReadWhen_READ_WHEN_NEXT_UNIT,
	home.ReadAfterResume:      activityv1.ReadWhen_READ_WHEN_AFTER_RESUME,
	home.ReadNotUntilSteering: activityv1.ReadWhen_READ_WHEN_NOT_UNTIL_STEERING,
	home.ReadNever:            activityv1.ReadWhen_READ_WHEN_NEVER,
	home.ReadAtStart:          activityv1.ReadWhen_READ_WHEN_AT_START,
	home.ReadUnknown:          activityv1.ReadWhen_READ_WHEN_UNKNOWN,
}

// EntryReading converts when a new operator entry will be read.
func EntryReading(r home.EntryReading) *activityv1.EntryReading {
	return &activityv1.EntryReading{When: readWire[r.When], Message: r.Message, ReadSince: Time(r.ReadSince)}
}

// ResearchWrite converts a topic's newest research write.
func ResearchWrite(w home.ResearchWrite) *activityv1.ResearchWrite {
	return &activityv1.ResearchWrite{At: Time(w.Time), CommissionId: w.CommissionID, CommissionState: State(w.State), Note: w.Note, Topic: w.Topic}
}
