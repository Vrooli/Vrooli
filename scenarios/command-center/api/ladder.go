package main

import (
	"sort"
	"strings"
)

// LadderReading is Offer Desk's release schedule projected for the room: each
// ranked marketed deliverable with what it opens and what it waits on. The UI
// renders it as the Next Rung hero, the Reach Map and a strip summary; it never
// re-derives schedule facts beyond what this projection states.
type LadderReading struct {
	Rungs       []LadderRung   `json:"rungs"`
	NextRank    int            `json:"nextRank"`
	Enabling    []LadderWork   `json:"enabling"`
	Unscheduled []LadderNode   `json:"unscheduled"`
	Reach       []LadderUnlock `json:"reach"`
	Unavailable []LadderGap    `json:"unavailable,omitempty"`
}

type LadderRung struct {
	Rank      int             `json:"rank"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Status    string          `json:"status"`
	FinishBar string          `json:"finishBar,omitempty"`
	Ramps     []string        `json:"ramps"`
	Streams   []string        `json:"streams"`
	Audiences []string        `json:"audiences"`
	Blockers  []LadderWork    `json:"blockers"`
	Goals     []LadderGoal    `json:"goals"`
	Readiness LadderReadiness `json:"readiness"`
}

// LadderWork is enabling (unranked) work. Direct marks an explicit enables edge
// into the rung; otherwise it is due because its derived urgency is at or
// before the rung's rank.
type LadderWork struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	FinishBar string `json:"finishBar,omitempty"`
	Urgency   int    `json:"urgency"`
	Direct    bool   `json:"direct,omitempty"`
}

type LadderNode struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	FinishBar string `json:"finishBar,omitempty"`
}

type LadderGoal struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Priority int    `json:"priority"`
}

// LadderReadiness is Reported only when the producer stated a readiness goal;
// proto3 JSON omits false, so absence is "not reported", never "not ready".
type LadderReadiness struct {
	Reported       bool   `json:"reported"`
	GoalClosed     bool   `json:"goalClosed,omitempty"`
	ApprovedCommit string `json:"approvedCommit,omitempty"`
}

// LadderUnlock names the first rank at which a ramp, stream or audience opens.
// OpensAt 0 means no scheduled rung opens it.
type LadderUnlock struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	OpensAt int    `json:"opensAt"`
}

type LadderGap struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

type ladderSelector func(payload any) (*LadderReading, bool)

var ladderSelectors = map[string]ladderSelector{
	"release_ladder": releaseLadder,
}

const maxLadderRungs = 500

func releaseLadder(payload any) (*LadderReading, bool) {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil, false
	}
	entries, ok := root["entries"].([]any)
	if !ok || len(entries) == 0 {
		return nil, false
	}
	ladder := &LadderReading{Rungs: []LadderRung{}, Enabling: []LadderWork{}, Unscheduled: []LadderNode{}, Reach: []LadderUnlock{}}
	for _, raw := range objects(root["enabling"]) {
		node := object(raw["node"])
		ladder.Enabling = append(ladder.Enabling, LadderWork{Name: str(node, "name"), Status: str(node, "status"), FinishBar: str(node, "finishBar", "finish_bar"), Urgency: num(raw, "derivedUrgency", "derived_urgency")})
	}
	for _, node := range objects(root["unscheduled"]) {
		ladder.Unscheduled = append(ladder.Unscheduled, LadderNode{Name: str(node, "name"), Status: str(node, "status"), FinishBar: str(node, "finishBar", "finish_bar")})
	}
	for _, gap := range objects(root["availability"]) {
		ladder.Unavailable = append(ladder.Unavailable, LadderGap{Source: str(gap, "source"), Reason: str(gap, "reason")})
	}
	for _, item := range entries {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		deliverable := object(entry["deliverable"])
		rung := LadderRung{
			Rank:      num(deliverable, "releaseRank", "release_rank"),
			ID:        str(deliverable, "id"),
			Name:      str(deliverable, "name"),
			Status:    str(deliverable, "status"),
			FinishBar: str(deliverable, "finishBar", "finish_bar"),
			Ramps:     names(entry, "unlockedRamps", "unlocked_ramps"),
			Streams:   names(entry, "unlockedStreams", "unlocked_streams"),
			Audiences: names(entry, "audiences"),
			Blockers:  []LadderWork{},
			Goals:     []LadderGoal{},
		}
		for _, goal := range objects(first2(entry, "goalImpacts", "goal_impacts")) {
			rung.Goals = append(rung.Goals, LadderGoal{Name: str(goal, "goalName", "goal_name"), Title: str(goal, "goalTitle", "goal_title"), Priority: num(goal, "projectedPriority", "projected_priority")})
		}
		exists, _ := first2(entry, "readinessGoalExists", "readiness_goal_exists").(bool)
		closed, _ := first2(entry, "readinessGoalClosed", "readiness_goal_closed").(bool)
		rung.Readiness = LadderReadiness{Reported: exists, GoalClosed: closed, ApprovedCommit: str(entry, "readinessApprovedCommit", "readiness_approved_commit")}
		direct := map[string]bool{}
		for _, raw := range objects(entry["enablers"]) {
			node := object(raw["node"])
			direct[str(node, "name")] = true
			if open(str(node, "status")) {
				rung.Blockers = append(rung.Blockers, LadderWork{Name: str(node, "name"), Status: str(node, "status"), FinishBar: str(node, "finishBar", "finish_bar"), Urgency: num(raw, "derivedUrgency", "derived_urgency"), Direct: true})
			}
		}
		for _, work := range ladder.Enabling {
			if !direct[work.Name] && open(work.Status) && work.Urgency > 0 && work.Urgency <= rung.Rank {
				rung.Blockers = append(rung.Blockers, work)
			}
		}
		ladder.Rungs = append(ladder.Rungs, rung)
	}
	sort.SliceStable(ladder.Rungs, func(i, j int) bool { return ladder.Rungs[i].Rank < ladder.Rungs[j].Rank })
	if !plausibleLadder(ladder.Rungs) {
		return nil, false
	}
	for _, rung := range ladder.Rungs {
		if rung.Status != "SHIPPED" {
			ladder.NextRank = rung.Rank
			break
		}
	}
	ladder.Reach = append(ladder.Reach, reach(ladder.Rungs, "ramp", root["ramps"], func(r LadderRung) []string { return r.Ramps })...)
	ladder.Reach = append(ladder.Reach, reach(ladder.Rungs, "stream", root["streams"], func(r LadderRung) []string { return r.Streams })...)
	ladder.Reach = append(ladder.Reach, reach(ladder.Rungs, "audience", root["audiences"], func(r LadderRung) []string { return r.Audiences })...)
	return ladder, true
}

// plausibleLadder refuses a schedule a producer could not legally emit: ranks
// are positive and unique, and every rung names its deliverable.
func plausibleLadder(rungs []LadderRung) bool {
	if len(rungs) == 0 || len(rungs) > maxLadderRungs {
		return false
	}
	for i, rung := range rungs {
		if rung.Rank <= 0 || strings.TrimSpace(rung.Name) == "" || strings.TrimSpace(rung.ID) == "" {
			return false
		}
		if i > 0 && rungs[i-1].Rank == rung.Rank {
			return false
		}
	}
	return true
}

// ladderRows flattens the schedule into panel rows for the strip and the scene:
// value is the rank, share is how far the rung has moved through its lifecycle.
func ladderRows(ladder *LadderReading) []PanelRow {
	rows := make([]PanelRow, 0, len(ladder.Rungs))
	for _, rung := range ladder.Rungs {
		rows = append(rows, PanelRow{Key: rung.ID, Label: rung.Name, Value: float64(rung.Rank), Share: lifecycleStage(rung.Status), Detail: rung.Status})
	}
	return rows
}

func lifecycleStage(status string) float64 {
	switch status {
	case "SHIPPED":
		return 1
	case "ACTIVE":
		return 0.8
	case "TRIGGER_MET", "PROPOSED":
		return 0.6
	case "CANDIDATE":
		return 0.4
	default:
		return 0.2
	}
}

func open(status string) bool { return status != "SHIPPED" && status != "RETIRED" }

func reach(rungs []LadderRung, kind string, declared any, opened func(LadderRung) []string) []LadderUnlock {
	var out []LadderUnlock
	seen := map[string]bool{}
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		unlock := LadderUnlock{Kind: kind, Name: name}
		for _, rung := range rungs {
			if contains(opened(rung), name) {
				unlock.OpensAt = rung.Rank
				break
			}
		}
		out = append(out, unlock)
	}
	for _, node := range objects(declared) {
		add(str(node, "name"))
	}
	for _, rung := range rungs {
		for _, name := range opened(rung) {
			add(name)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].OpensAt == 0) != (out[j].OpensAt == 0) {
			return out[j].OpensAt == 0
		}
		return out[i].OpensAt < out[j].OpensAt
	})
	return out
}

func objects(value any) []map[string]any {
	items, _ := value.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func object(value any) map[string]any {
	m, _ := value.(map[string]any)
	return m
}

func first2(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value
		}
	}
	return nil
}

func str(m map[string]any, keys ...string) string {
	value, _ := first2(m, keys...).(string)
	return value
}

func num(m map[string]any, keys ...string) int {
	value, _ := first2(m, keys...).(float64)
	return int(value)
}

func names(entry map[string]any, keys ...string) []string {
	out := []string{}
	for _, node := range objects(first2(entry, keys...)) {
		if name := str(node, "name"); name != "" {
			out = append(out, name)
		}
	}
	return out
}
