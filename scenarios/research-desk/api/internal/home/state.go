package home

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// State is a commission's observed state (SYSTEM.md §4).
type State string

// The eight states, in the order their rules are tried.
const (
	StateEnded    State = "ended"
	StatePrepared State = "prepared"
	StateUnknown  State = "unknown"
	StateStopped  State = "stopped"
	StateLive     State = "live"
	StatePaused   State = "paused"
	StateLimited  State = "limited"
	// StateHalted is a goal Codex blocked because its last turn ended on an
	// error (Halt says which); a deliberate usage or budget limit is limited.
	StateHalted State = "halted"
)

// Running reports whether a manager window is open for the state.
func (s State) Running() bool {
	return s == StateLive || s == StatePaused || s == StateLimited || s == StateHalted
}

// stateInput is everything the state rules read.
// StallAfter flags a live commission with no write for this long. It
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
	// NotStarted is the evidence that nothing started a complete packet;
	// read only for a packet with no session and no launch.
	NotStarted Evidence
	// Halt is why a blocked goal stopped, when its last turn ended on an
	// error; HaltUnknown says why that could not be read.
	Halt        *Halt
	HaltUnknown string
}

// StoppedOrEnded opens the rule of a commission with no window open and no
// known allowance: whether it stopped early or ran its course cannot be told.
const StoppedOrEnded = "stopped or ended (allowance unknown)"

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
		return StateEnded, "allowance ended and no manager window is open"
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
			return StateStopped, StoppedOrEnded + ": no manager window is open"
		}
		return StateStopped, "no manager window is open; goal not complete; allowance not ended"
	}
	// A window open past the allowance says so.
	past := ""
	if in.AllowanceEnd.Known && in.Now.After(in.AllowanceEnd.Value) {
		past = "; allowance ended " + in.AllowanceEnd.Value.UTC().Format("2006-01-02 15:04Z")
	}
	switch exact.GoalStatus {
	case "active":
		return StateLive, "manager window open; goal active" + past
	case "paused":
		return StatePaused, "manager window open; goal paused" + past
	case "blocked":
		if in.Halt != nil {
			return StateHalted, "manager window open; goal blocked: " + in.Halt.Said() + past
		}
		why := ""
		if in.HaltUnknown != "" {
			why = " (" + in.HaltUnknown + ")"
		}
		return StateLimited, "manager window open; goal blocked" + why + past
	case "usage_limited", "budget_limited":
		return StateLimited, fmt.Sprintf("manager window open; goal %s%s", exact.GoalStatus, past)
	case "":
		return StateUnknown, "manager window open but no goal row for its thread"
	default:
		return StateUnknown, fmt.Sprintf("manager window open; unrecognised goal status %q", exact.GoalStatus)
	}
}

// ReadWhen is when the supervisor will read a new OPERATOR.md entry.
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
		out.When, out.Message = ReadNever, "this commission has ended; no manager will read it"
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
	case c.State == StateHalted:
		out.When, out.Message = ReadAfterResume, "the goal halted on an error; read at the first unit after it resumes"
	case c.State == StatePrepared, c.State == StateStopped:
		out.When, out.Message = ReadAtStart, "no manager window is open; read when a manager starts on it"
	default:
		out.When, out.Message = ReadUnknown, "the commission's state is unknown, so when it is read is unknown"
	}
	return out
}

// StateGrace is how long a window may run before its STATE.md must exist
// (the charter's first step writes it).
const StateGrace = time.Hour

// stateGoalLine is the one header line a window's STATE.md records its goal in: `Goal: <status>` alone on a
// line (the charter's first step writes it from get_goal; its last writes `Goal: paused` or `Goal: ended`). Only the
// header, the text before the first `## ` heading, is read, so a per-card `status:` line or a sentence that
// begins "Goal:" is never taken for it.
var stateGoalLine = regexp.MustCompile(`(?m)^Goal: ([a-z_]+)[ \t]*$`)

func stateHeader(text string) string {
	if i := strings.Index("\n"+text, "\n## "); i >= 0 {
		return ("\n" + text)[:i]
	}
	return text
}

// DeadAfter is how long a window may run with its manager doing nothing past start-up before it is called dead.
const DeadAfter = time.Hour

// startupWrites is how long after the goal starts the manager's first reads and writes are still start-up.
const startupWrites = 15 * time.Minute

// deadWindow says a running window did nothing but start: DeadAfter has passed and nothing in its folder (bar the
// launch record) or its run folders was written more than startupWrites after the goal started. A manager
// overwrites STATE.md at every review and owners write run folders, so a live window keeps this moving. The
// session's transcript is not counted: a manager that only talks for hours is the case this catches (two such
// windows, 10 h and 2.5 h, landed nothing on 2026-11-19 and 11-20).
func (c *Commission) deadWindow(now time.Time) (hours float64, dead bool) {
	if !c.Started.Known || now.Sub(c.Started.Value) <= DeadAfter {
		return 0, false
	}
	cutoff := c.Started.Value.Add(startupWrites)
	for _, f := range c.Files {
		if f.Name != LaunchFile && f.Modified.After(cutoff) {
			return 0, false
		}
	}
	for _, run := range c.Runs {
		if run.LastWrite.Known && run.LastWrite.Value.After(cutoff) {
			return 0, false
		}
	}
	return now.Sub(c.Started.Value).Hours(), true
}

// windowFlags names where a running window's own record disagrees with what the desk observes: a window dead an
// hour in (the manager never got past start-up), no STATE.md an hour after the window started, a `Goal:` line
// that says active while the goal is not, or one that says the window is over while the goal is still active. A
// window past its allowance is not flagged for a missing STATE.md, but a dead one is flagged for as long as its
// goal is open: someone must stop or relaunch it. A flag is a line in `now` and the summary's `flags` for the
// steward, never a state change.
func (c *Commission) windowFlags(now time.Time) []string {
	if !c.State.Running() {
		return nil
	}
	var halted []string
	if c.Halt != nil {
		halted = c.Halt.flags(now)
	}
	return append(halted, c.recordFlags(now)...)
}

// recordFlags are windowFlags' disagreements between the window's own record
// and what the desk observes.
func (c *Commission) recordFlags(now time.Time) []string {
	state, ok := c.File(StateFile)
	if hours, dead := c.deadWindow(now); dead {
		noState := ""
		if !ok {
			noState = " and no " + StateFile
		}
		return []string{fmt.Sprintf("dead window: %.1f h after the goal started, nothing in its folder or run folders was written past its first %d minutes%s; stop it or relaunch it", hours, int(startupWrites.Minutes()), noState)}
	}
	if !ok {
		overAllowance := c.AllowanceEnd.Known && now.After(c.AllowanceEnd.Value)
		if c.Started.Known && now.Sub(c.Started.Value) > StateGrace && !overAllowance {
			return []string{fmt.Sprintf("no %s %.1f h after the window started; the charter's first step writes it", StateFile, now.Sub(c.Started.Value).Hours())}
		}
		return nil
	}
	if c.Session == nil || c.Session.GoalStatus == "" {
		return nil
	}
	data, err := os.ReadFile(state.Path)
	if err != nil {
		return nil
	}
	m := stateGoalLine.FindSubmatch([]byte(stateHeader(string(data))))
	if m == nil {
		return nil
	}
	says, goal := string(m[1]), c.Session.GoalStatus
	switch {
	case says == "active" && goal != "active":
		return []string{fmt.Sprintf("%s says the goal is active; the goal is %s", StateFile, goal)}
	case (says == "paused" || says == "ended") && goal == "active":
		return []string{fmt.Sprintf("%s says the window is %s; the goal is still active", StateFile, says)}
	}
	return nil
}
