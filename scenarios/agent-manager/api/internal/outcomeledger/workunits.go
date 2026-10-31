package outcomeledger

import (
	"sort"
	"strings"
	"time"
)

// Link kinds and their confidence.
const (
	LinkRoot              = "root"
	LinkChild             = "child"
	LinkSuccessor         = "successor"
	LinkResume            = "resume"
	LinkTask              = "task"
	LinkGoal              = "goal"
	LinkWorkloadRetry     = "workload_retry"
	LinkWorkloadSuccessor = "workload_successor"
	LinkWebConsole        = "web_console_session"
	// LinkOverCap starts a new unit: the run's links led into a unit already at
	// MaxUnitRuns, so that unit's outcome no longer speaks for it.
	LinkOverCap = "over_cap"

	workloadLinkWindow   = 48 * time.Hour
	webConsoleLinkWindow = 12 * time.Hour
	// A key that many runs share names a kind of work, not one task.
	maxWorkloadKeyRuns = 12
	maxGoalRuns        = 40
	// MaxUnitRuns bounds a unit whatever its links: one outcome cannot speak for
	// more. A long orchestrator's later children continue in another unit.
	MaxUnitRuns = 40
	// A link below lowConfidenceLink is circumstantial (same tab, same workload
	// key). It never joins different tasks or goals, its window runs from the unit's
	// first run rather than sliding along a chain, and a unit takes at most
	// maxLowConfidenceLinks of them.
	lowConfidenceLink     = 0.7
	maxLowConfidenceLinks = 3
)

var linkConfidence = map[string]float64{
	LinkRoot: 1, LinkChild: 1, LinkSuccessor: 0.95, LinkResume: 0.95, LinkTask: 0.9, LinkGoal: 0.85,
	LinkWorkloadRetry: 0.8, LinkWorkloadSuccessor: 0.6, LinkWebConsole: 0.5, LinkOverCap: 1,
}

// Assignment is one run's place in a work unit.
type Assignment struct {
	RunID, WorkUnitID, LinkKind string
	LinkConfidence              float64
}

// LinkWorkUnits assigns every run to a work unit. Each run takes the strongest
// link to an earlier run. Attached rows hold no work of their own and are left
// to the caller; pass the runs that carry usage. The result is deterministic.
func LinkWorkUnits(runs []RunFacts) map[string]Assignment {
	ordered := append([]RunFacts(nil), runs...)
	sort.Slice(ordered, func(i, j int) bool {
		a, b := runTime(ordered[i]), runTime(ordered[j])
		if !a.Equal(b) {
			return a.Before(b)
		}
		return ordered[i].RunID < ordered[j].RunID
	})
	workloadCount, goalCount := map[string]int{}, map[string]int{}
	byID := make(map[string]RunFacts, len(ordered))
	for _, run := range ordered {
		byID[run.RunID] = run
		if run.WorkloadKey != "" {
			workloadCount[run.WorkloadKey]++
		}
		if run.GoalID != "" {
			goalCount[run.GoalID]++
		}
	}
	parent := map[string]string{}
	units := map[string]unitFacts{}
	var find func(string) string
	find = func(id string) string {
		if parent[id] == "" || parent[id] == id {
			return id
		}
		parent[id] = find(parent[id])
		return parent[id]
	}
	size := map[string]int{}
	unitOf := func(id string) unitFacts { return units[find(id)] }
	union := func(later, earlier string) {
		a, b := find(later), find(earlier)
		if a == b {
			return
		}
		parent[a] = b
		size[b] = max(size[b], 1) + max(size[a], 1)
		merged := units[b]
		merged.absorb(units[a])
		units[b] = merged
	}
	type lastSeen struct {
		id string
		at time.Time
	}
	bySession, byTask, byGoal, byWorkload, byConsole := map[string]string{}, map[string]string{}, map[string]string{}, map[string]lastSeen{}, map[string]lastSeen{}
	// spill names the unit that takes over when a unit is full.
	spill := map[string]string{}
	out := make(map[string]Assignment, len(ordered))
	for _, run := range ordered {
		at := runTime(run)
		// A child can sort before its parent, so this run may already lead a unit.
		size[run.RunID] = max(size[run.RunID], 1)
		facts := newUnitFacts(run, at)
		facts.absorb(units[run.RunID])
		units[run.RunID] = facts
		kind, target := "", ""
		consider := func(k, id string) {
			if id == "" || id == run.RunID {
				return
			}
			if kind == "" || linkConfidence[k] > linkConfidence[kind] {
				kind, target = k, id
			}
		}
		// A circumstantial link is bounded by the unit it would join, not the
		// run it happens to follow, so it cannot grow a chain.
		considerLow := func(k, id string, window time.Duration) {
			if id == "" || id == run.RunID {
				return
			}
			facts := unitOf(id)
			if at.Sub(facts.start) > window || facts.lowLinks >= maxLowConfidenceLinks || facts.conflicts(run) {
				return
			}
			consider(k, id)
		}
		if _, ok := byID[run.ParentRunID]; ok {
			consider(LinkChild, run.ParentRunID)
		}
		for _, source := range run.SourceRunIDs {
			if _, ok := byID[source]; ok {
				consider(LinkSuccessor, source)
			}
		}
		consider(LinkResume, bySession[run.SessionKey])
		consider(LinkTask, byTask[run.TaskID])
		if goalCount[run.GoalID] <= maxGoalRuns {
			consider(LinkGoal, byGoal[run.GoalID])
		}
		if key := run.WorkloadKey; key != "" && workloadCount[key] <= maxWorkloadKeyRuns {
			if previous := byWorkload[key]; previous.id != "" && at.Sub(previous.at) <= workloadLinkWindow {
				if attempt := byID[previous.id]; attempt.Status == "failed" || attempt.Status == "cancelled" {
					consider(LinkWorkloadRetry, previous.id)
				} else {
					considerLow(LinkWorkloadSuccessor, previous.id, workloadLinkWindow)
				}
			}
		}
		if console := run.WebConsoleSession; console != "" {
			if previous := byConsole[console]; previous.id != "" {
				considerLow(LinkWebConsole, previous.id, webConsoleLinkWindow)
			}
		}
		if kind != "" {
			// A full unit hands over to the unit that took its overflow.
			for size[find(target)]+size[find(run.RunID)] > MaxUnitRuns {
				next, ok := spill[find(target)]
				if !ok {
					spill[find(target)] = run.RunID
					kind, target = LinkOverCap, ""
					break
				}
				target = next
			}
		}
		switch kind {
		case "":
			out[run.RunID] = Assignment{RunID: run.RunID, LinkKind: LinkRoot, LinkConfidence: 1}
		case LinkOverCap:
			out[run.RunID] = Assignment{RunID: run.RunID, LinkKind: LinkOverCap, LinkConfidence: 1}
		default:
			union(run.RunID, target)
			if linkConfidence[kind] < lowConfidenceLink {
				joined := units[find(run.RunID)]
				joined.lowLinks++
				units[find(run.RunID)] = joined
			}
			out[run.RunID] = Assignment{RunID: run.RunID, LinkKind: kind, LinkConfidence: linkConfidence[kind]}
		}
		if run.SessionKey != "" {
			if _, ok := bySession[run.SessionKey]; !ok {
				bySession[run.SessionKey] = run.RunID
			}
		}
		if run.TaskID != "" {
			byTask[run.TaskID] = run.RunID
		}
		if run.GoalID != "" {
			byGoal[run.GoalID] = run.RunID
		}
		if run.WorkloadKey != "" {
			byWorkload[run.WorkloadKey] = lastSeen{run.RunID, at}
		}
		if run.WebConsoleSession != "" && !run.Child() {
			byConsole[run.WebConsoleSession] = lastSeen{run.RunID, at}
		}
	}
	// A unit is named by its earliest run, which union keeps as the root.
	for id, assignment := range out {
		assignment.WorkUnitID = find(id)
		out[id] = assignment
	}
	return out
}

// unitFacts is what a low-confidence link checks about the unit it would join.
type unitFacts struct {
	start       time.Time
	tasks, goal map[string]bool
	lowLinks    int
}

func newUnitFacts(run RunFacts, at time.Time) unitFacts {
	facts := unitFacts{start: at, tasks: map[string]bool{}, goal: map[string]bool{}}
	if run.TaskID != "" {
		facts.tasks[run.TaskID] = true
	}
	if run.GoalID != "" {
		facts.goal[run.GoalID] = true
	}
	return facts
}

func (u *unitFacts) absorb(other unitFacts) {
	if u.start.IsZero() || (!other.start.IsZero() && other.start.Before(u.start)) {
		u.start = other.start
	}
	if u.tasks == nil {
		u.tasks, u.goal = map[string]bool{}, map[string]bool{}
	}
	for task := range other.tasks {
		u.tasks[task] = true
	}
	for goal := range other.goal {
		u.goal[goal] = true
	}
	u.lowLinks += other.lowLinks
}

// conflicts reports a run on a task or goal other than the unit's.
func (u unitFacts) conflicts(run RunFacts) bool {
	return (run.TaskID != "" && len(u.tasks) > 0 && !u.tasks[run.TaskID]) || (run.GoalID != "" && len(u.goal) > 0 && !u.goal[run.GoalID])
}

func runTime(run RunFacts) time.Time {
	switch {
	case !run.SessionStart.IsZero() && (run.FirstActivity.IsZero() || run.SessionStart.Before(run.FirstActivity)):
		return run.SessionStart
	case !run.FirstActivity.IsZero():
		return run.FirstActivity
	}
	return run.CreatedAt
}

// SessionBase strips the "#file" suffix a resumed rollout's provenance carries.
func SessionBase(importSessionID string) string {
	base, _, _ := strings.Cut(importSessionID, "#")
	return base
}

// ClassifyAttached decides what an attached row is. A row in the same Web
// Console session as an imported transcript, or starting within a minute
// before an unclaimed one of the same harness, duplicates it. Otherwise a row
// that lived under attachedSessionMin was a helper, not a session.
func ClassifyAttached(attached, imported []RunFacts) map[string]RunFacts {
	byConsole := map[string]string{}
	byHarness := map[string][]RunFacts{}
	for _, run := range imported {
		if run.Child() {
			continue
		}
		if run.WebConsoleSession != "" {
			if _, ok := byConsole[run.WebConsoleSession]; !ok {
				byConsole[run.WebConsoleSession] = run.RunID
			}
		}
		byHarness[run.Harness] = append(byHarness[run.Harness], run)
	}
	for harness := range byHarness {
		list := byHarness[harness]
		sort.Slice(list, func(i, j int) bool { return runTime(list[i]).Before(runTime(list[j])) })
	}
	ordered := append([]RunFacts(nil), attached...)
	sort.Slice(ordered, func(i, j int) bool { return runTime(ordered[i]).Before(runTime(ordered[j])) })
	used := map[string]bool{}
	out := make(map[string]RunFacts, len(ordered))
	for _, run := range ordered {
		run.AttachedClass, run.LinkedRunID, run.LinkConfidence = "", "", 0
		if id, ok := byConsole[run.WebConsoleSession]; ok && run.WebConsoleSession != "" {
			run.AttachedClass, run.LinkedRunID, run.LinkConfidence = AttachedLinked, id, 0.95
			used[id] = true
			out[run.RunID] = run
			continue
		}
		start := runTime(run)
		for _, candidate := range byHarness[run.Harness] {
			gap := runTime(candidate).Sub(start)
			if gap < -5*time.Second {
				continue
			}
			if gap > time.Minute {
				break
			}
			if !used[candidate.RunID] {
				used[candidate.RunID] = true
				run.AttachedClass, run.LinkedRunID, run.LinkConfidence = AttachedLinked, candidate.RunID, 0.6
				break
			}
		}
		if run.AttachedClass == "" && run.LastActivity.Sub(start) >= attachedSessionMin {
			// A long-lived shim can start a session long after it attached.
			for _, candidate := range byHarness[run.Harness] {
				at := runTime(candidate)
				if at.After(run.LastActivity) {
					break
				}
				if !at.Before(start) && !used[candidate.RunID] {
					used[candidate.RunID] = true
					run.AttachedClass, run.LinkedRunID, run.LinkConfidence = AttachedLinked, candidate.RunID, 0.4
					break
				}
			}
		}
		if run.AttachedClass == "" {
			switch {
			case run.LastActivity.Sub(start) < attachedSessionMin && !run.Live:
				run.AttachedClass = AttachedHelper
			case HasTranscriptReader(run.Harness):
				run.AttachedClass = AttachedUnlinked
			default:
				run.AttachedClass = AttachedNoReader
			}
		}
		out[run.RunID] = run
	}
	return out
}

// HasTranscriptReader reports a harness the transcript indexer can read.
func HasTranscriptReader(harness string) bool {
	return harness == "codex" || harness == "claude-code"
}
