package codexrollout

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// TurnErrorSchema is the schema id of turn_errors.json.
const TurnErrorSchema = "codex-rollout.turn-errors/v1"

// Recovery actions a class may name.
const (
	// ActionResume starts the goal again in the same session after a backoff.
	ActionResume = "resume"
	// ActionEscalate hands the halt to whoever manages the session.
	ActionEscalate = "escalate"
)

//go:embed turn_errors.json
var defaultTurnErrors []byte

// TurnErrorTable is turn_errors.json: rules that match a turn-ending error to
// a class, and the recovery each class takes. It is data so that a new error
// shape is one rule and a policy change is one field, for every consumer.
type TurnErrorTable struct {
	Schema   string           `json:"schema"`
	About    string           `json:"about"`
	Classes  []TurnErrorClass `json:"classes"`
	Rules    []TurnErrorRule  `json:"rules"`
	Fallback string           `json:"fallback"`

	byID map[string]*TurnErrorClass
}

// TurnErrorClass is one class of turn-ending error and its recovery.
type TurnErrorClass struct {
	ID       string   `json:"id"`
	Meaning  string   `json:"meaning"`
	Recovery Recovery `json:"recovery"`
}

// Recovery is what a consumer does about a halt of one class.
type Recovery struct {
	Action string `json:"action"`
	// MaxAttempts caps automatic resumes counted within Window.
	MaxAttempts int `json:"max_attempts,omitempty"`
	// Backoff is the wait after the error before attempt n (1-based); the
	// last entry repeats.
	Backoff []Duration `json:"backoff,omitempty"`
	// Window is how far back attempts count toward MaxAttempts: a resumed
	// turn that ran longer than this before failing again starts afresh.
	Window Duration `json:"window,omitempty"`
	// StaleAfter is how old an error may be before it is no longer resumed
	// automatically: nobody acted while it was fresh, so its context is
	// unknown and the halt is escalated instead.
	StaleAfter Duration `json:"stale_after,omitempty"`
}

// TurnErrorRule matches one error shape. Every set field must match.
type TurnErrorRule struct {
	ID    string `json:"id"`
	Class string `json:"class"`
	// CodexErrorInfo is the normalised codex_error_info (TurnError.Info).
	CodexErrorInfo  string `json:"codex_error_info,omitempty"`
	MessageContains string `json:"message_contains,omitempty"`
	HTTPStatusMin   int    `json:"http_status_min,omitempty"`
	HTTPStatusMax   int    `json:"http_status_max,omitempty"`
	// Seen records where the shape was observed; it is not matched.
	Seen string `json:"seen,omitempty"`
}

// Duration is a time.Duration written as a Go duration string ("2m").
type Duration time.Duration

func (d *Duration) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("duration must be a string such as \"5m\": %w", err)
	}
	value, err := time.ParseDuration(text)
	if err != nil {
		return err
	}
	*d = Duration(value)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) { return json.Marshal(time.Duration(d).String()) }

// LoadTurnErrorTable parses and validates a turn-error table.
func LoadTurnErrorTable(data []byte) (*TurnErrorTable, error) {
	var t TurnErrorTable
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("turn-error table: %w", err)
	}
	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("turn-error table: %w", err)
	}
	return &t, nil
}

var defaultTable = sync.OnceValues(func() (*TurnErrorTable, error) { return LoadTurnErrorTable(defaultTurnErrors) })

// DefaultTurnErrorTable is the table embedded from turn_errors.json.
func DefaultTurnErrorTable() (*TurnErrorTable, error) { return defaultTable() }

func (t *TurnErrorTable) validate() error {
	var problems []string
	if t.Schema != TurnErrorSchema {
		problems = append(problems, fmt.Sprintf("schema %q, want %q", t.Schema, TurnErrorSchema))
	}
	t.byID = map[string]*TurnErrorClass{}
	for i := range t.Classes {
		c := &t.Classes[i]
		if c.ID == "" || t.byID[c.ID] != nil {
			problems = append(problems, fmt.Sprintf("class %d: id %q is empty or repeated", i, c.ID))
			continue
		}
		t.byID[c.ID] = c
		r := c.Recovery
		switch r.Action {
		case ActionEscalate:
		case ActionResume:
			if r.MaxAttempts < 1 || len(r.Backoff) == 0 || r.Window <= 0 || r.StaleAfter <= 0 {
				problems = append(problems, fmt.Sprintf("class %s: resume needs max_attempts, backoff, window and stale_after", c.ID))
			}
			for _, b := range r.Backoff {
				if b <= 0 || b >= r.StaleAfter {
					problems = append(problems, fmt.Sprintf("class %s: every backoff must be positive and shorter than stale_after", c.ID))
					break
				}
			}
		default:
			problems = append(problems, fmt.Sprintf("class %s: action %q is neither %q nor %q", c.ID, r.Action, ActionResume, ActionEscalate))
		}
	}
	seen := map[string]bool{}
	for i, r := range t.Rules {
		switch {
		case r.ID == "" || seen[r.ID]:
			problems = append(problems, fmt.Sprintf("rule %d: id %q is empty or repeated", i, r.ID))
		case t.byID[r.Class] == nil:
			problems = append(problems, fmt.Sprintf("rule %s: unknown class %q", r.ID, r.Class))
		case r.CodexErrorInfo == "" && r.MessageContains == "" && r.HTTPStatusMin == 0 && r.HTTPStatusMax == 0:
			problems = append(problems, fmt.Sprintf("rule %s: matches nothing", r.ID))
		case r.HTTPStatusMax < r.HTTPStatusMin:
			problems = append(problems, fmt.Sprintf("rule %s: http_status_max is below http_status_min", r.ID))
		}
		seen[r.ID] = true
	}
	if t.byID[t.Fallback] == nil {
		problems = append(problems, fmt.Sprintf("fallback %q is not a class", t.Fallback))
	} else if t.byID[t.Fallback].Recovery.Action != ActionEscalate {
		problems = append(problems, "the fallback class must escalate: an unrecognised error is never retried blind")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

// Classification is the class a turn error falls in and its recovery.
type Classification struct {
	Class   string
	Meaning string
	// Rule is the matching rule's id; empty when the fallback applied.
	Rule     string
	Recovery Recovery
}

// Classify returns the class of the first rule that matches, else the
// fallback class.
func (t *TurnErrorTable) Classify(e TurnError) Classification {
	for _, r := range t.Rules {
		if r.matches(e) {
			c := t.byID[r.Class]
			return Classification{Class: c.ID, Meaning: c.Meaning, Rule: r.ID, Recovery: c.Recovery}
		}
	}
	c := t.byID[t.Fallback]
	return Classification{Class: c.ID, Meaning: c.Meaning, Recovery: c.Recovery}
}

func (r TurnErrorRule) matches(e TurnError) bool {
	if r.CodexErrorInfo != "" && r.CodexErrorInfo != e.Info {
		return false
	}
	if r.MessageContains != "" && !strings.Contains(e.Message, r.MessageContains) {
		return false
	}
	if r.HTTPStatusMin != 0 && (e.HTTPStatus < r.HTTPStatusMin || e.HTTPStatus > r.HTTPStatusMax) {
		return false
	}
	return true
}

// Plan is what to do about one halt now.
type Plan struct {
	// Action is ActionResume when an automatic resume is (or will be) due,
	// else ActionEscalate.
	Action string
	// Attempt is the number of the next automatic resume (1-based).
	Attempt int
	// Due is when that resume may be made; zero when escalating.
	Due time.Time
	// Escalation says why the halt is not resumed automatically; empty
	// while it will be.
	Escalation string
}

// Plan decides one halt: the error ended its turn at errorAt and attempts
// automatic resumes were already made within the class's window. A resume
// is due Backoff[attempts] after the error, unless the class escalates, the
// cap is reached, or the error went stale before its resume was due.
func (c Classification) Plan(errorAt, now time.Time, attempts int) Plan {
	r := c.Recovery
	switch {
	case r.Action != ActionResume:
		return Plan{Action: ActionEscalate, Escalation: fmt.Sprintf("a %s error is not resumed automatically: %s", c.Class, c.Meaning)}
	case attempts >= r.MaxAttempts:
		return Plan{Action: ActionEscalate, Attempt: attempts, Escalation: fmt.Sprintf("automatic resume tried %d of %d times within %s and the goal halted again", attempts, r.MaxAttempts, time.Duration(r.Window))}
	}
	backoff := r.Backoff[min(attempts, len(r.Backoff)-1)]
	due := errorAt.Add(time.Duration(backoff))
	if age := now.Sub(errorAt); age > time.Duration(r.StaleAfter) {
		return Plan{Action: ActionEscalate, Attempt: attempts, Escalation: fmt.Sprintf("the error is %s old, past the %s within which it is resumed automatically", age.Round(time.Minute), time.Duration(r.StaleAfter))}
	}
	return Plan{Action: ActionResume, Attempt: attempts + 1, Due: due}
}
