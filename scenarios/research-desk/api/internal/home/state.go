package home

import (
	"fmt"
	"time"
)

// State is a commission's observed state (SYSTEM.md §4).
type State string

// The seven states, in the order their rules are tried.
const (
	StateEnded    State = "ended"
	StatePrepared State = "prepared"
	StateUnknown  State = "unknown"
	StateStopped  State = "stopped"
	StateLive     State = "live"
	StatePaused   State = "paused"
	StateLimited  State = "limited"
)

// Running reports whether a supervisor window is open for the state.
func (s State) Running() bool {
	return s == StateLive || s == StatePaused || s == StateLimited
}

// stateInput is everything the state rules read.
// StallAfter flags a live commission with no write for this long (D16). It
// is fixed, not a setting.
const StallAfter = 70 * time.Minute

type stateInput struct {
	Now time.Time
	// Unlisted is why the commission folder cannot be listed; its state is
	// then unknown whatever else is found.
	Unlisted       string
	Discarded      bool
	PacketComplete bool
	HasLaunch      bool
	HasRelated     bool
	Exact          *Match // newest exact session, nil when none
	ProcsKnown     bool
	ProcsReason    string
	WindowOpen     bool
	AllowanceEnd   Time
	// NotStarted is the evidence that nothing started a complete packet
	// (rulings F8); read only for a packet with no session and no launch.
	NotStarted Evidence
}

// notStartedRule opens the rule of a complete packet whose absence of a
// start cannot be proven.
const notStartedRule = "packet complete, but not proven unstarted"

// deriveState applies the state rules; the first that matches wins. It
// returns the state and the rule in words.
func deriveState(in stateInput) (State, string) {
	exact := in.Exact
	switch {
	case in.Unlisted != "":
		return StateUnknown, in.Unlisted
	case in.Discarded:
		return StateEnded, "discarded in LAUNCH.json"
	case exact != nil && exact.GoalStatus == "complete":
		return StateEnded, "goal marked complete"
	case exact != nil && in.AllowanceEnd.Known && in.Now.After(in.AllowanceEnd.Value) && in.ProcsKnown && !in.WindowOpen:
		return StateEnded, "allowance ended and no supervisor window is open"
	case exact == nil && in.PacketComplete && !in.HasRelated && !in.HasLaunch && in.NotStarted.Known:
		return StatePrepared, "packet complete; " + in.NotStarted.Reason
	case exact == nil && in.PacketComplete && !in.HasRelated && !in.HasLaunch:
		return StateUnknown, notStartedRule + ": " + in.NotStarted.Reason
	case exact == nil:
		if in.HasRelated {
			return StateUnknown, "only preparation prompts name this folder; no goal names its charter"
		}
		return StateUnknown, "no session names this commission's charter"
	case !in.ProcsKnown:
		return StateUnknown, in.ProcsReason
	case !in.WindowOpen:
		if !in.AllowanceEnd.Known {
			return StateStopped, "no supervisor window is open; allowance end unknown"
		}
		return StateStopped, "no supervisor window is open; goal not complete; allowance not ended"
	}
	// A window open past the allowance says so (rulings F4).
	past := ""
	if in.AllowanceEnd.Known && in.Now.After(in.AllowanceEnd.Value) {
		past = "; allowance ended " + in.AllowanceEnd.Value.UTC().Format("2006-01-02 15:04Z")
	}
	switch exact.GoalStatus {
	case "active":
		return StateLive, "supervisor window open; goal active" + past
	case "paused":
		return StatePaused, "supervisor window open; goal paused" + past
	case "usage_limited", "budget_limited", "blocked":
		return StateLimited, fmt.Sprintf("supervisor window open; goal %s%s", exact.GoalStatus, past)
	case "":
		return StateUnknown, "supervisor window open but no goal row for its thread"
	default:
		return StateUnknown, fmt.Sprintf("supervisor window open; unrecognised goal status %q", exact.GoalStatus)
	}
}

// ReadWhen is when the supervisor will read a new OPERATOR.md entry
// (rulings P5).
type ReadWhen string

// When a new entry is read.
const (
	ReadNextUnit         ReadWhen = "next_unit"
	ReadAfterResume      ReadWhen = "after_resume"
	ReadNotUntilSteering ReadWhen = "not_until_steering"
	ReadNever            ReadWhen = "never"
	ReadAtStart          ReadWhen = "at_start"
	ReadUnknown          ReadWhen = "unknown"
)

// EntryReading says when a new entry will be read, in a code and a
// sentence, and since when entries have been read.
type EntryReading struct {
	When    ReadWhen
	Message string
	// ReadSince is the newest STATE.md rewrite: a unit rewrites it after
	// reading OPERATOR.md, so entries before it were read.
	ReadSince Time
}

// EntryReading says when the supervisor will read a new OPERATOR.md entry.
func (c *Commission) EntryReading() EntryReading {
	out := EntryReading{ReadSince: Time{Evidence: Unknown("no " + StateFile + ": no unit has run")}}
	if fact, ok := c.File(StateFile); ok {
		out.ReadSince = Time{Value: fact.Modified, Evidence: From(fact.Path, fact.Modified)}
	}
	switch {
	case c.State == StateEnded:
		out.When, out.Message = ReadNever, "this commission has ended; no supervisor will read it"
	case !c.Steering.Known:
		out.When, out.Message = ReadUnknown, "whether the charter tells the scheduler to read "+OperatorFile+" is unknown: "+c.Steering.Reason
	case !c.Steering.Value:
		out.When, out.Message = ReadNotUntilSteering, "the charter does not tell the scheduler to read "+OperatorFile+"; it is read only after steering is enabled"
	case c.State == StateLive:
		out.When, out.Message = ReadNextUnit, "read at the start of the scheduler's next unit"
	case c.State == StatePaused:
		out.When, out.Message = ReadAfterResume, "the goal is paused; read at the first unit after it resumes"
	case c.State == StateLimited:
		out.When, out.Message = ReadAfterResume, "the goal is limited; read at the first unit after the limit lifts"
	case c.State == StatePrepared, c.State == StateStopped:
		out.When, out.Message = ReadAtStart, "no supervisor window is open; read when a supervisor starts on it"
	default:
		out.When, out.Message = ReadUnknown, "the commission's state is unknown, so when it is read is unknown"
	}
	return out
}
