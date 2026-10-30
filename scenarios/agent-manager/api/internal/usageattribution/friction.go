package usageattribution

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/runsignal"
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
	// WallClockMS is the redundant call time (or, for a stall, the idle gap) of
	// the episodes in total and TypicalMS their median; see TimeKind.
	WallClockMS, TypicalMS int64
	// Tokens is the argument and result tokens of the episodes' redundant calls.
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
	// TimeKind says what WallClockMS and TypicalMS measure: redundant (the
	// redundant calls' own duration) or idle (a stall's gap).
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

// frictionProblems groups the episodes of runs that consumed quota by
// fingerprint, keeps those that recurred, and prices each as its turns' share
// of its run's turns times the run's attributed points: a run's friction
// never costs more than the run consumed. Largest saving first.
func frictionProblems(in Inputs, primary []*Result) []Problem {
	type runUsage struct {
		result  *Result
		points  float64
		reports int
		first   time.Time
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
			if usage.first.IsZero() || turn.At.Before(usage.first) {
				usage.first = turn.At
			}
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
		// A pattern that is no longer detected, is informational or fails the
		// precision gate (quarantined) is not a problem to fix.
		if !runsignal.Shown(episode.Pattern) {
			continue
		}
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
			runUse := project(usage.result, share*usage.points, usage.first, usage.last)
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
		spec, _ := runsignal.PatternByID(p.Pattern)
		p.TimeKind = spec.TimeKind
		p.Owner, p.OwnerKind = ownerOf(p.OwnerScenario, p.OwnerCommand)
		p.Description = Describe(p.Pattern, dominantElement(g.episodes), p.OwnerScenario, p.OwnerCommand, p.typicalCycles(g.episodes))
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

// dominantElement is the repeated element most episodes of a group share
// (ties alphabetically), or "" when it covers under half of them: a sentence
// naming one episode's element would misdescribe the rest.
func dominantElement(episodes []Episode) string {
	counts := map[string]int{}
	for _, e := range episodes {
		counts[e.RepeatedElement]++
	}
	best, bestCount := "", 0
	for element, n := range counts {
		if element != "" && (n > bestCount || n == bestCount && element < best) {
			best, bestCount = element, n
		}
	}
	if bestCount*2 < len(episodes) {
		return ""
	}
	return best
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
	var sentence string
	if spec, ok := runsignal.PatternByID(pattern); ok {
		sentence = spec.Render(cmd, element, cycles)
	} else {
		sentence = capitalize(strings.ReplaceAll(pattern, "-", " ") + ".")
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
