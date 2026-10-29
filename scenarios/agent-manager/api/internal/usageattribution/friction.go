package usageattribution

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Problem statuses. A problem the operator has not acted on is open.
const (
	StatusOpen      = "open"
	StatusReported  = "reported"
	StatusDismissed = "dismissed"
)

// Owner kinds: who can fix a problem.
const (
	OwnerCommand   = "command"
	OwnerScenario  = "scenario"
	OwnerAgent     = "agent"
	agentBehaviour = "Agent behaviour"
)

// Problem is one recurring friction problem: every episode with the same
// fingerprint, priced in the current window.
type Problem struct {
	Fingerprint string
	Pattern     string
	// Description is one plain sentence derived from the episodes' evidence.
	Description string
	// Owner names who can fix it: a command, a scenario, or "Agent behaviour".
	Owner         string
	OwnerKind     string
	OwnerScenario string
	OwnerCommand  string
	Episodes      int
	Runs          int
	EpisodeTurns  int
	// WallClockMS is the time the episodes wasted in total and TypicalMS their
	// median, each episode counted by its pattern's own definition (see
	// runsignal.wastedMS). TimeLost is false where extra work is not waste by
	// itself (handoff-continuation): then the count, EpisodeTurns and Tokens
	// are reported instead and both times are 0.
	WallClockMS, TypicalMS int64
	TimeLost               bool // true only for wasted work (TimeKind wasted)
	// Tokens is the usage the episodes' turns spanned.
	Tokens int64
	// Points is the quota the problem's turns used in the window.
	Points      float64
	Provider    string
	Pool        string
	Window      string
	FirstAt     time.Time
	LastAt      time.Time
	RunIDs      []string
	Evidence    []Ref
	Status      string
	BugID       string
	StatusSetAt time.Time

	// EstSaved is the saving of fixing half the problem over the window's
	// whole horizon at this rate; EstRecoverable is the part still to be had
	// before the window resets.
	EstSaved, EstRecoverable float64
	// TimeKind says what WallClockMS and TypicalMS measure: wasted (a
	// redundant or failed call's own duration), idle (a stall's gap), elapsed
	// (how long a loop or a slow start ran) or none (handoff-continuation:
	// extra work is not waste by itself).
	TimeKind string

	result *Result
	use    projection
}

// ProblemStatus is what the operator did with a problem.
type ProblemStatus struct {
	Fingerprint string
	Status      string
	BugID       string
	At          time.Time
}

// Time kinds of a friction problem.
const (
	TimeWasted  = "wasted"
	TimeIdle    = "idle"
	TimeElapsed = "elapsed"
	TimeNone    = "none"
)

// timeKind names what a pattern's episode cost measures (runsignal.wastedMS).
func timeKind(pattern string) string {
	switch pattern {
	case "stall":
		return TimeIdle
	case "handoff-continuation", "blocked-then-abandoned":
		return TimeNone
	case "time-to-first-success", "poll-loop", "wait-misuse", "oscillation", "edit-revert":
		return TimeElapsed
	}
	return TimeWasted
}

// frictionProblems groups the episodes of runs that consumed quota by
// fingerprint, keeps those that recurred, and prices each as its turns' share
// of its run's turns times the run's attributed points: a run's friction
// never costs more than the run consumed. Largest saving first.
func frictionProblems(in Inputs, primary []*Result) []Problem {
	type runUsage struct {
		result  *Result
		points  float64
		reports int
		last    time.Time
	}
	runs := map[string]*runUsage{}
	for _, result := range primary {
		for i, turn := range result.Turns {
			usage := runs[turn.RunID]
			if usage == nil {
				usage = &runUsage{result: result}
				runs[turn.RunID] = usage
			}
			usage.points += result.Points[i]
			usage.reports += turn.Reports
			if turn.At.After(usage.last) {
				usage.last = turn.At
			}
		}
	}
	type group struct {
		first    Episode
		episodes []Episode
		runTurns map[string]int
	}
	groups := map[string]*group{}
	runFriction := map[string]int{}
	for _, episode := range in.Episodes {
		usage := runs[episode.RunID]
		if usage == nil || usage.points <= 0 || usage.reports <= 0 || episode.Turns <= 0 {
			continue
		}
		runFriction[episode.RunID] += episode.Turns
		key := episode.Fingerprint
		if key == "" {
			key = episode.Pattern + "|" + episode.RepeatedElement
		}
		g := groups[key]
		if g == nil {
			g = &group{first: episode, runTurns: map[string]int{}}
			groups[key] = g
		}
		g.episodes = append(g.episodes, episode)
		g.runTurns[episode.RunID] += episode.Turns
	}
	var problems []Problem
	for key, g := range groups {
		if len(g.episodes) < frictionMinEpisodes {
			continue
		}
		p := Problem{Fingerprint: key, Pattern: g.first.Pattern, OwnerScenario: g.first.OwnerScenario, OwnerCommand: g.first.OwnerCommand, Episodes: len(g.episodes), Runs: len(g.runTurns), Status: StatusOpen}
		var saved float64
		var largest *runUsage // stated in its largest run's window
		for id, episodeTurns := range g.runTurns {
			usage := runs[id]
			// Episodes overlap; a run's friction is at most all of its turns,
			// shared among its fingerprints by their turns.
			share := min(1, float64(runFriction[id])/float64(usage.reports)) * float64(episodeTurns) / float64(runFriction[id])
			p.Points += share * usage.points
			runUse := project(usage.result, share*usage.points, usage.last)
			saved += runUse.weeklySaving(share * usage.points * 0.5)
			p.use = p.use.merge(runUse)
			p.EpisodeTurns += episodeTurns
			if largest == nil || usage.points > largest.points {
				largest = usage
			}
		}
		p.result, p.EstSaved = largest.result, saved
		p.EstRecoverable = recoverable(saved, p.use.use(largest.result, in.Now))
		p.Provider, p.Pool, p.Window = largest.result.Window.Provider, largest.result.Window.Pool, largest.result.Window.Name
		p.fillEvidence(g.episodes, g.runTurns)
		p.TimeKind = timeKind(p.Pattern)
		p.TimeLost = p.TimeKind == TimeWasted
		p.Owner, p.OwnerKind = ownerOf(p.OwnerScenario, p.OwnerCommand)
		p.Description = Describe(p.Pattern, g.first.RepeatedElement, p.OwnerScenario, p.OwnerCommand, p.typicalCycles(g.episodes))
		problems = append(problems, p)
	}
	sort.Slice(problems, func(i, j int) bool {
		if problems[i].EstSaved != problems[j].EstSaved {
			return problems[i].EstSaved > problems[j].EstSaved
		}
		return problems[i].Fingerprint < problems[j].Fingerprint
	})
	return problems
}

func (p *Problem) fillEvidence(episodes []Episode, runTurns map[string]int) {
	durations := make([]int64, 0, len(episodes))
	for _, e := range episodes {
		p.Tokens += e.Tokens
		durations = append(durations, e.WallClockMS)
		if p.FirstAt.IsZero() || e.At.Before(p.FirstAt) {
			p.FirstAt = e.At
		}
		if e.At.After(p.LastAt) {
			p.LastAt = e.At
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p.TypicalMS = durations[len(durations)/2]
	for _, d := range durations {
		p.WallClockMS += d
	}
	p.RunIDs = topRunsByTurns(runTurns, maxEvidence/2)
	for _, id := range p.RunIDs {
		p.Evidence = append(p.Evidence, Ref{Kind: "run", ID: id})
	}
	sort.Slice(episodes, func(i, j int) bool { return episodes[i].Turns > episodes[j].Turns })
	for _, e := range episodes {
		if len(p.Evidence) >= maxEvidence {
			break
		}
		p.Evidence = append(p.Evidence, Ref{Kind: "episode", ID: e.EpisodeID, Label: "run " + e.RunID})
	}
}

// typicalCycles is the median repeat count of the episodes that count one.
func (p *Problem) typicalCycles(episodes []Episode) int {
	var cycles []int
	for _, e := range episodes {
		if e.CycleCount > 0 {
			cycles = append(cycles, e.CycleCount)
		}
	}
	if len(cycles) == 0 {
		return 0
	}
	sort.Ints(cycles)
	return cycles[len(cycles)/2]
}

func ownerOf(scenario, command string) (string, string) {
	switch {
	case command != "":
		return command, OwnerCommand
	case scenario != "":
		return scenario, OwnerScenario
	}
	return agentBehaviour, OwnerAgent
}

// Describe states a friction pattern in one plain sentence from the facts the
// episodes carry: the command, the preferred command and how often a step
// repeats. Durations are separate fields, never part of the sentence. It never invents a cause the evidence does
// not hold.
func Describe(pattern, element, scenario, command string, cycles int) string {
	cmd := command
	if cmd == "" {
		cmd = scenario
	}
	quoted := "a command"
	if cmd != "" {
		quoted = "`" + cmd + "`"
	}
	var sentence string
	switch pattern {
	case "stall":
		sentence = "A run sits idle between tool calls for a long stretch."
	case "read-then-reread":
		sentence = "An agent reads a file, then reads the same file again with no edit in between."
	case "wrong-primitive":
		sentence = fmt.Sprintf("Agents run %s repeatedly instead of the preferred `%s`.", quoted, element)
	case "repeated-work":
		sentence = fmt.Sprintf("%s runs again, back to back, with the same input.", capitalize(quoted))
	case "command-failure":
		sentence = fmt.Sprintf("%s fails twice in a row.", capitalize(quoted))
	case "help-recovery":
		sentence = fmt.Sprintf("%s fails and the agent opens help to recover.", capitalize(quoted))
	case "poll-loop":
		sentence = fmt.Sprintf("An agent polls %s over and over instead of waiting once.", quoted)
	case "oscillation":
		sentence = fmt.Sprintf("An agent alternates between two calls of %s without making progress.", quoted)
	case "edit-revert":
		sentence = "A file edit is undone soon after it is made."
	case "wait-misuse":
		sentence = "An agent repeats a short wait instead of one blocking wait."
		if cycles > 1 {
			sentence = fmt.Sprintf("An agent repeats a short wait %d times instead of one blocking wait.", cycles)
		}
	case "blocked-then-abandoned":
		sentence = "A run hits a blocker and stops without resolving it."
	case "guidance-repair":
		sentence = "The operator corrects the agent, and it switches approach."
	case "handoff-continuation":
		sentence = "The agent declares it is done, then keeps working with no new instruction."
	case "fallback-after-capability":
		sentence = fmt.Sprintf("%s works, then the agent falls back to a tool outside Vrooli for the same job.", capitalize(quoted))
	case "capability-abandoned":
		sentence = fmt.Sprintf("%s fails and the agent gives up on it instead of retrying.", capitalize(quoted))
	case "retry-after-failure":
		sentence = fmt.Sprintf("%s fails and the agent retries it straight away.", capitalize(quoted))
	case "flag-hunting":
		sentence = fmt.Sprintf("After %s fails, the agent tries several different flag shapes for it.", quoted)
	case "abandoned-command-family":
		sentence = fmt.Sprintf("%s fails and never succeeds before the agent moves on.", capitalize(quoted))
	case "time-to-first-success":
		sentence = "A run takes a long time to make its first successful tool call."
	default:
		sentence = strings.ReplaceAll(pattern, "-", " ") + "."
		sentence = capitalize(sentence)
	}
	// Problems of one pattern that differ only by command read differently.
	if cmd != "" && !strings.Contains(sentence, "`"+cmd+"`") {
		sentence += " Involves `" + cmd + "`."
	}
	return sentence
}

func capitalize(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

// FormatDuration renders milliseconds the way an operator reads them.
func FormatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	switch {
	case d < time.Second:
		return fmt.Sprintf("%d ms", ms)
	case d < time.Minute:
		return fmt.Sprintf("%.0f s", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %02dm", int(d.Hours()), int(d.Minutes())%60)
}
