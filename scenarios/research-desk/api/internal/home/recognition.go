package home

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MatchKind says how strongly a session is tied to a commission (SYSTEM.md §3).
type MatchKind string

const (
	// MatchExact: the launch record, a goal objective or a /goal prompt names
	// the commission's charter path.
	MatchExact MatchKind = "exact"
	// MatchRelated: another prompt names the folder; shown, never used for state.
	MatchRelated MatchKind = "related"
)

// Match ties one Codex thread to a commission.
type Match struct {
	Kind       MatchKind
	Basis      string // launch-record, goal-objective, goal-objective-file, goal-history, history-mention
	SessionID  string
	ThreadID   string
	GoalStatus string
	// Ended is when the goal was marked complete (the goal row's update
	// time); zero otherwise.
	Ended time.Time
	// GoalUpdated is the goal row's update time: for a paused goal, no
	// earlier than the pause.
	GoalUpdated time.Time
	At          time.Time
	Source      string
	WindowOpen  bool
}

// recognize finds the sessions that name a commission. charterForms are the
// spellings of the charter path that count as exact (absolute and ~-form).
// Matches before windowStart are ignored.
func recognize(id string, charterForms []string, launch *LaunchRecord, launchPath string, records CodexRecords, windowStart time.Time) []Match {
	goalByThread := map[string]Goal{}
	for _, goal := range records.Goals {
		goalByThread[goal.SessionID+"/"+goal.ThreadID] = goal
	}
	seen := map[string]bool{}
	var matches []Match
	add := func(m Match) {
		key := m.SessionID + "/" + m.ThreadID + "/" + string(m.Kind)
		if seen[key] {
			return
		}
		seen[key] = true
		matches = append(matches, m)
	}
	namesCharter := func(text string) bool {
		for _, form := range charterForms {
			if strings.Contains(text, form) {
				return true
			}
		}
		return false
	}

	if launch != nil && launch.WebConsoleSessionID != "" {
		m := Match{Kind: MatchExact, Basis: "launch-record", SessionID: launch.WebConsoleSessionID, ThreadID: launch.ThreadID, Source: launchPath}
		if goal, ok := goalByThread[m.SessionID+"/"+m.ThreadID]; ok {
			m.setGoal(goal)
			m.At = goal.Created
		} else if len(launch.Steps) > 0 {
			m.At = launch.Steps[0].At
		}
		add(m)
	}
	for _, goal := range records.Goals {
		if goal.Created.Before(windowStart) {
			continue
		}
		basis := "goal-objective"
		if goal.ObjectiveFile != "" {
			basis = "goal-objective-file"
		}
		m := Match{Kind: MatchExact, Basis: basis, SessionID: goal.SessionID, ThreadID: goal.ThreadID, At: goal.Created, Source: goal.Source}
		m.setGoal(goal)
		if namesCharter(goal.Objective) {
			add(m)
		} else if namesFolder(goal.Objective, id) {
			m.Kind = MatchRelated
			add(m)
		}
	}
	for _, prompt := range records.Prompts {
		if prompt.At.Before(windowStart) || !namesFolder(prompt.Text, id) {
			continue
		}
		goal, hasGoal := goalByThread[prompt.SessionID+"/"+prompt.ThreadID]
		m := Match{Kind: MatchRelated, Basis: "history-mention", SessionID: prompt.SessionID, ThreadID: prompt.ThreadID, At: prompt.At, Source: prompt.Source}
		if prompt.IsGoal && namesCharter(prompt.Text) {
			m.Kind, m.Basis = MatchExact, "goal-history"
			if hasGoal {
				m.setGoal(goal)
			}
		}
		if seen[m.SessionID+"/"+m.ThreadID+"/"+string(MatchExact)] {
			continue // already exact through the goal row
		}
		add(m)
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Kind != matches[j].Kind {
			return matches[i].Kind == MatchExact
		}
		return matches[i].At.After(matches[j].At)
	})
	return matches
}

// setGoal copies the goal row's status and, when complete, its end.
func (m *Match) setGoal(g Goal) {
	m.GoalStatus, m.GoalUpdated = g.Status, g.Updated
	if g.Status == "complete" {
		m.Ended = g.Updated
	}
}

// namesFolder reports whether text names cohorts/<id> exactly (not a longer id).
func namesFolder(text, id string) bool {
	for _, sub := range commissionRef.FindAllStringSubmatch(text, -1) {
		if strings.TrimRight(sub[1], ".") == id {
			return true
		}
	}
	return false
}

// charterForms returns the spellings of a commission's charter path that a
// goal or /goal prompt may use.
func charterForms(researchDir, userHome, id string) []string {
	abs := filepath.Join(researchDir, "cohorts", id, CharterFile)
	forms := []string{abs}
	if userHome != "" && strings.HasPrefix(abs, userHome+string(filepath.Separator)) {
		forms = append(forms, "~"+strings.TrimPrefix(abs, userHome))
	}
	return forms
}

// windowStart is the start of the recognition window: the local midnight one
// day before the commission folder's earliest file time.
func windowStart(files []FileFact) time.Time {
	var earliest time.Time
	for _, f := range files {
		if earliest.IsZero() || f.Modified.Before(earliest) {
			earliest = f.Modified
		}
	}
	if earliest.IsZero() {
		return time.Time{}
	}
	local := earliest.Local()
	return time.Date(local.Year(), local.Month(), local.Day()-1, 0, 0, 0, 0, local.Location())
}
