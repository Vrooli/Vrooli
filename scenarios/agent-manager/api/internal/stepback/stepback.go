// Package stepback derives the transcript half of an effort's step-back
// triggers: patterns in a run's stored events that show a long effort losing
// its objective or its scale. goalhome.EvaluateEpoch owns the goal-home half
// (flatline, overrun, growth, spend, wall-clock), and `effort epoch-check`
// records either kind as a step_back effort signal.
//
// The trigger IDs and thresholds are the live tripwires of the
// effort-diagnosis skill (`effort_anatomy.py drift`), computed from what Agent
// Manager already ingests instead of from the rollout file. Triggers are
// observations: Agent Manager never stops, parks or re-tasks a run on one.
// Durations are wall-clock spans between stored events, so idle gaps inside a
// turn count, unlike the script's active time.
package stepback

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// Trigger IDs. They match `effort_anatomy.py drift` so a supervisor reads one
// vocabulary on the board, in `run tree` and from the script.
const (
	ObjectiveLost     = "objective-lost"
	StanceNotRetained = "stance-not-retained"
	UnitPerTurn       = "unit-per-turn"
	ShortWorkers      = "short-workers"
	InheritedContext  = "inherited-context"
)

// Thresholds, shared with `effort_anatomy.py drift`.
const (
	// objectiveLostAfter is how long a turn may keep working after a compaction
	// omitted its objective before the omission counts as lost.
	objectiveLostAfter = 30 * time.Minute
	// longTurn and longTurnCompactions mark a turn that ran past several
	// compactions without ending, so the objective was never re-presented.
	longTurn            = 3 * time.Hour
	longTurnCompactions = 2
	// shortWorker is the lifetime under which a worker cannot own an outcome
	// (LONG_EFFORT_DESIGN §1: none of 446 such workers changed 1,000 lines).
	shortWorker = 6 * time.Minute
	minWorkers  = 4
	minSpawns   = 4
)

// Trigger is one observed step-back pattern.
type Trigger struct {
	ID string
	// Mechanism names the effort-diagnosis catalog entry (M1–M16).
	Mechanism string
	Detail    string
	// Remedy is the re-orientation the pattern calls for, never a stop.
	Remedy string
}

// Turn is one harness turn, from its start to its terminal record.
type Turn struct {
	ID    string
	Start time.Time
	// End is the turn's terminal record, or its latest event while Ended is
	// false (the turn is still open or ended without a terminal record).
	End   time.Time
	Ended bool
}

// Compaction is one context compaction and what it kept of the native goal.
type Compaction struct {
	At time.Time
	// GoalObjective is domain.CompactionGoalRetained, domain.CompactionGoalOmitted
	// or empty when unknown.
	GoalObjective string
	// OperatorMessagesKept is how many operator-typed messages the compaction
	// kept verbatim; nil when unknown or the run is a native sub-agent.
	OperatorMessagesKept *int
}

// Transcript is what one run's stored events say about its turns,
// compactions and sub-agent spawns.
type Transcript struct {
	Turns       []Turn
	Compactions []Compaction
	// Spawns counts harness-native sub-agent spawns; SpawnsWithHistory those
	// that inherited the parent's history (Codex fork_turns other than none).
	Spawns            int
	SpawnsWithHistory int
}

// Descendant is one run below the assessed run in its tree.
type Descendant struct {
	Live     bool
	Duration time.Duration
}

// Input is everything one assessment reads.
type Input struct {
	Transcript  Transcript
	Descendants []Descendant
}

// Assess returns the step-back triggers the input shows, in a stable order.
func Assess(in Input) []Trigger {
	var triggers []Trigger
	for _, check := range []func(Input) *Trigger{objectiveLost, stanceNotRetained, unitPerTurn, shortWorkers, inheritedContext} {
		if trigger := check(in); trigger != nil {
			triggers = append(triggers, *trigger)
		}
	}
	return triggers
}

// IDs lists the trigger IDs, for compact one-line displays.
func IDs(triggers []Trigger) string {
	ids := make([]string, 0, len(triggers))
	for _, trigger := range triggers {
		ids = append(ids, trigger.ID)
	}
	return strings.Join(ids, ",")
}

// objectiveLost fires when a turn kept working at least objectiveLostAfter
// after a compaction omitted its native goal objective. Codex re-presents the
// objective only at the next goal-turn start, so the rest of that turn ran
// without it.
func objectiveLost(in Input) *Trigger {
	var lost int
	var worst time.Duration
	var worstAt time.Time
	compactions := append([]Compaction(nil), in.Transcript.Compactions...)
	sort.Slice(compactions, func(i, j int) bool { return compactions[i].At.Before(compactions[j].At) })
	for _, turn := range in.Transcript.Turns {
		for _, compaction := range compactions {
			if compaction.GoalObjective != domain.CompactionGoalOmitted || !turn.contains(compaction.At) {
				continue
			}
			// The first omission in a turn starts the span; later compactions in
			// the same turn keep it omitted.
			if after := turn.End.Sub(compaction.At); after >= objectiveLostAfter {
				lost++
				if after > worst {
					worst, worstAt = after, compaction.At
				}
			}
			break
		}
	}
	if lost == 0 {
		return nil
	}
	return &Trigger{
		ID: ObjectiveLost, Mechanism: "M1",
		Detail: fmt.Sprintf("%d turn(s) kept working after a compaction omitted the native goal objective; worst %s from %s",
			lost, minutes(worst), worstAt.UTC().Format("2006-01-02 15:04")),
		Remedy: "end the turn after each unit and keep the native goal to a kernel of 600 characters or fewer",
	}
}

// stanceNotRetained fires when a compaction kept no operator-typed message:
// the task prompt or stance message, with the operator's words, is gone from
// the context. Goal events and Agent Manager wakes count toward Codex's eight
// retained user messages but carry no stance.
func stanceNotRetained(in Input) *Trigger {
	lost := 0
	for _, compaction := range in.Transcript.Compactions {
		if compaction.OperatorMessagesKept != nil && *compaction.OperatorMessagesKept == 0 {
			lost++
		}
	}
	if lost == 0 {
		return nil
	}
	return &Trigger{
		ID: StanceNotRetained, Mechanism: "M1",
		Detail: fmt.Sprintf("%d compaction(s) kept no operator message, so the task prompt or stance message is gone from context", lost),
		Remedy: "keep the stance in the charter and re-read it after each compaction; a scheduler that messages the agent resends the stance",
	}
}

// unitPerTurn fires on a turn longer than longTurn that spanned at least
// longTurnCompactions compactions.
func unitPerTurn(in Input) *Trigger {
	var count int
	var longest time.Duration
	for _, turn := range in.Transcript.Turns {
		span := turn.End.Sub(turn.Start)
		if span <= longTurn {
			continue
		}
		compactions := 0
		for _, compaction := range in.Transcript.Compactions {
			if turn.contains(compaction.At) {
				compactions++
			}
		}
		if compactions < longTurnCompactions {
			continue
		}
		count++
		longest = max(longest, span)
	}
	if count == 0 {
		return nil
	}
	return &Trigger{
		ID: UnitPerTurn, Mechanism: "M1",
		Detail: fmt.Sprintf("%d turn(s) longer than %s with %d+ compactions; longest %.1f h", count, hours(longTurn), longTurnCompactions, longest.Hours()),
		Remedy: "end the turn after each round, segment return or owner return",
	}
}

// shortWorkers fires when at least half of the settled descendants (and at
// least minWorkers) ended within shortWorker. Agent Manager does not know a
// descendant's role, so reviewers and lookups count too.
func shortWorkers(in Input) *Trigger {
	var settled, short int
	for _, descendant := range in.Descendants {
		if descendant.Live {
			continue
		}
		settled++
		if descendant.Duration < shortWorker {
			short++
		}
	}
	if settled < minWorkers || short*2 < settled {
		return nil
	}
	return &Trigger{
		ID: ShortWorkers, Mechanism: "M8",
		Detail: fmt.Sprintf("%d of %d settled descendants ran under %s (reviewers and lookups included)", short, settled, minutes(shortWorker)),
		Remedy: "delegate by outcome: one long-lived owner per coherent outcome",
	}
}

// inheritedContext fires when at least half of minSpawns or more spawns
// inherited the parent's full history.
func inheritedContext(in Input) *Trigger {
	spawns, inherited := in.Transcript.Spawns, in.Transcript.SpawnsWithHistory
	if spawns < minSpawns || inherited*2 < spawns {
		return nil
	}
	return &Trigger{
		ID: InheritedContext, Mechanism: "M9",
		Detail: fmt.Sprintf("%d of %d spawns inherited the full parent history (fork_turns not none)", inherited, spawns),
		Remedy: "brief children fresh from the charter with fork_turns none",
	}
}

func (t Turn) contains(at time.Time) bool {
	return !at.Before(t.Start) && !at.After(t.End)
}

// TurnStart is when a turn began: the time a UUIDv7 turn ID encodes (Codex
// turn IDs are UUIDv7, minted when the turn starts), else the turn's first
// stored event.
func TurnStart(id string, firstEvent time.Time) time.Time {
	parsed, err := uuid.Parse(id)
	if err != nil || parsed.Version() != 7 {
		return firstEvent
	}
	sec, nsec := parsed.Time().UnixTime()
	start := time.Unix(sec, nsec).UTC()
	if start.After(firstEvent) {
		return firstEvent
	}
	return start
}

func minutes(d time.Duration) string { return fmt.Sprintf("%.0f min", d.Minutes()) }

func hours(d time.Duration) string { return fmt.Sprintf("%g h", d.Hours()) }
