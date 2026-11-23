package home

import (
	"fmt"
	"strings"
	"sync"
	"time"

	codexrollout "github.com/vrooli/vrooli/packages/codex-rollout"
)

// Halt is why a halted commission's goal stopped: Codex marks a goal blocked
// when a goal turn ends on an error, and the turn's error is read from the
// thread's rollout and classified by packages/codex-rollout/turn_errors.json
// (SYSTEM.md §4, §6). A goal blocked for any other reason is not a halt.
type Halt struct {
	TurnID string
	// At is when the turn ended on the error.
	At     time.Time
	Source string
	Error  codexrollout.TurnError
	codexrollout.Classification
	// Attempts counts the desk's automatic resumes recorded in LAUNCH.json
	// within the class's window.
	Attempts int
	// Plan is what the recovery pass does now: resume when due, or
	// escalate, with why.
	Plan codexrollout.Plan
	// Escalated says LAUNCH.json records the escalation of this halt.
	Escalated bool
	// Last is the newest recovery entry for this halt, nil when none.
	Last *RecoveryEntry
}

// rolloutTail bounds how much of a rollout is read back to find its newest
// turn boundary. A halted goal's error is within a few records of the end.
const rolloutTail = 16 << 20

// RecoveryOverdue is how long past its due time an automatic resume may
// wait before the window is flagged: the pass runs every minute, so a
// resume this late means the pass is not running.
const RecoveryOverdue = 10 * time.Minute

// turnCache caches each rollout's newest turn boundary by size and
// modification time.
type turnCache struct {
	mu    sync.Mutex
	files map[string]cachedTurn
}

type cachedTurn struct {
	size    int64
	mod     time.Time
	outcome codexrollout.TurnOutcome
	err     error
}

func (c *turnCache) last(path string) (codexrollout.TurnOutcome, error) {
	size, mod, ok := fileKey(path)
	if !ok {
		return codexrollout.TurnOutcome{}, fmt.Errorf("%s cannot be read", path)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if hit, ok := c.files[path]; ok && hit.size == size && hit.mod.Equal(mod) {
		return hit.outcome, hit.err
	}
	outcome, err := codexrollout.ReadLastTurn(path, rolloutTail)
	if c.files == nil {
		c.files = map[string]cachedTurn{}
	}
	c.files[path] = cachedTurn{size: size, mod: mod, outcome: outcome, err: err}
	return outcome, err
}

// halt reads why the exact thread's blocked goal stopped. It returns nil and
// the reason when the goal is not blocked by a turn error, or that cannot be
// told.
func (c *Commission) halt(env deriveEnv) (*Halt, string) {
	s := c.Session
	if s == nil || s.GoalStatus != "blocked" || !s.WindowOpen {
		return nil, ""
	}
	if env.turnErrors == nil {
		return nil, "the turn-error table is not loaded: " + env.turnErrorsWhy
	}
	rollout, ok := rolloutFor(env.codexDir, s.SessionID, s.ThreadID)
	if !ok {
		return nil, "no rollout for thread " + s.ThreadID + ", so why it is blocked is unknown"
	}
	outcome, err := env.lastTurn(rollout.Path)
	switch {
	case err != nil:
		return nil, "why it is blocked is unknown: " + err.Error()
	case !outcome.EndedOnError():
		return nil, fmt.Sprintf("its newest turn boundary is %s at %s, not an error", outcome.Event, outcome.At.UTC().Format("2006-01-02 15:04Z"))
	}
	return HaltOf(c, outcome, rollout.Path, env.turnErrors, env.now), ""
}

// HaltOf is the halt of commission c whose thread's newest turn ended on an
// error (outcome.EndedOnError, read from source): its class in table, the
// automatic resumes LAUNCH.json records, and the plan now. A resume is
// planned only with allowance left.
func HaltOf(c *Commission, outcome codexrollout.TurnOutcome, source string, table *codexrollout.TurnErrorTable, now time.Time) *Halt {
	h := &Halt{TurnID: outcome.TurnID, At: outcome.At, Source: source, Error: *outcome.Error, Classification: table.Classify(*outcome.Error)}
	h.account(c.Launch, now)
	h.Plan = h.Classification.Plan(h.At, now, h.Attempts)
	if h.Plan.Action == codexrollout.ActionResume {
		switch {
		case !c.AllowanceEnd.Known:
			h.Plan = codexrollout.Plan{Action: codexrollout.ActionEscalate, Attempt: h.Attempts, Escalation: "its allowance is unknown (" + c.AllowanceEnd.Reason + "), so it is not resumed automatically"}
		case !now.Before(c.AllowanceEnd.Value):
			h.Plan = codexrollout.Plan{Action: codexrollout.ActionEscalate, Attempt: h.Attempts, Escalation: "its allowance ended at " + c.AllowanceEnd.Value.UTC().Format("2006-01-02 15:04Z")}
		}
	}
	return h
}

// account reads the halt's recovery history from LAUNCH.json: automatic
// resumes within the class's window count toward its cap, whichever turn
// they answered; an escalation counts only for this turn.
func (h *Halt) account(record *LaunchRecord, now time.Time) {
	if record == nil {
		return
	}
	since := now.Add(-time.Duration(h.Recovery.Window))
	for i := range record.Recoveries {
		r := &record.Recoveries[i]
		if r.Action == codexrollout.ActionResume && r.At.After(since) {
			h.Attempts++
		}
		if r.TurnID == h.TurnID {
			h.Last = r
			if r.Action == codexrollout.ActionEscalate {
				h.Escalated = true
			}
		}
	}
}

// Said is the halt in one line: class, time and Codex's message.
func (h *Halt) Said() string {
	return fmt.Sprintf("its last turn ended on a %s error at %s (%s)", h.Class, h.At.UTC().Format("2006-01-02 15:04Z"), clip(h.Error.Message, 160))
}

// flags are the halt's lines for `now` and the steward: an escalation, or an
// automatic resume that is overdue. A halt the desk is about to resume is
// not flagged; nobody needs to act on it.
func (h *Halt) flags(now time.Time) []string {
	switch {
	case h.Plan.Action == codexrollout.ActionEscalate:
		return []string{fmt.Sprintf("halted: %s; %s; not resumed automatically: the steward decides whether to resume, relaunch or stop it", h.Said(), h.Plan.Escalation)}
	case now.After(h.Plan.Due.Add(RecoveryOverdue)):
		return []string{fmt.Sprintf("halted: %s; automatic resume %d was due at %s and has not happened, so the desk's recovery pass may not be running", h.Said(), h.Plan.Attempt, h.Plan.Due.UTC().Format("15:04Z"))}
	}
	return nil
}

func clip(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
