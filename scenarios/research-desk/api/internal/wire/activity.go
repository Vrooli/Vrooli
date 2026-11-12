package wire

import (
	"research-desk/internal/home"
	"research-desk/internal/peer"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"
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
func Workstreams(list []home.Workstream) []*sharedv1.Workstream {
	out := make([]*sharedv1.Workstream, 0, len(list))
	for _, w := range list {
		out = append(out, &sharedv1.Workstream{Id: w.ID, Status: w.Status, Showcase: w.Showcase, Title: w.Title, Magnitude: w.Magnitude})
	}
	return out
}

// NeverStarted converts an unknown commission's never-started answer.
func NeverStarted(n home.NeverStarted) *sharedv1.NeverStarted {
	return &sharedv1.NeverStarted{Assertable: n.Assertable, Reason: n.Reason}
}

var readWire = map[home.ReadWhen]sharedv1.ReadWhen{
	home.ReadNextUnit:         sharedv1.ReadWhen_READ_WHEN_NEXT_UNIT,
	home.ReadAfterResume:      sharedv1.ReadWhen_READ_WHEN_AFTER_RESUME,
	home.ReadNotUntilSteering: sharedv1.ReadWhen_READ_WHEN_NOT_UNTIL_STEERING,
	home.ReadNever:            sharedv1.ReadWhen_READ_WHEN_NEVER,
	home.ReadAtStart:          sharedv1.ReadWhen_READ_WHEN_AT_START,
	home.ReadUnknown:          sharedv1.ReadWhen_READ_WHEN_UNKNOWN,
}

// EntryReading converts when a new operator entry will be read.
func EntryReading(r home.EntryReading) *sharedv1.EntryReading {
	return &sharedv1.EntryReading{When: readWire[r.When], Message: r.Message, ReadSince: Time(r.ReadSince)}
}

// ResearchWrite converts a topic's newest research write.
func ResearchWrite(w home.ResearchWrite) *activityv1.ResearchWrite {
	return &activityv1.ResearchWrite{At: Time(w.Time), CommissionId: w.CommissionID, CommissionState: State(w.State), Note: w.Note, Topic: w.Topic}
}

// Access is how the desk reaches Web Console now: the kind,
// the same in a plain sentence, and whether automatic launch is on.
type Access struct {
	Kind      activityv1.WebConsoleAccess
	Said      string
	Automatic bool
}

// AccessKind is the kind of a peer access phrase (peer.AccessCredential or
// peer.AccessAvailability); "" means the desk could not settle its access.
func AccessKind(access string) activityv1.WebConsoleAccess {
	switch access {
	case peer.AccessCredential:
		return activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_SERVICE_CREDENTIAL
	case peer.AccessAvailability:
		return activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_AVAILABILITY_POLICY
	case "":
		return activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_UNAVAILABLE
	}
	return activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_UNSPECIFIED
}

// ReadSets converts measured read sets.
func ReadSets(sets []home.ReadSet) []*sharedv1.ReadSet {
	out := make([]*sharedv1.ReadSet, 0, len(sets))
	for _, set := range sets {
		out = append(out, &sharedv1.ReadSet{Role: set.Role, Files: Files(set.Files), Bytes: set.Bytes, EstimatedTokens: set.EstimatedTokens, Flags: set.Flags})
	}
	return out
}
