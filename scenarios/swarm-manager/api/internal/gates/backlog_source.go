package gates

import (
	"context"
	"sort"
	"strings"

	"swarm-manager/internal/backlog"
)

// ItemStore is the narrow slice of backlog.Store the backlog-backed gate
// sources need.
type ItemStore interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
	ItemDir(kind backlog.BacklogKind, name string) string
}

// lockedStatuses are mid-execution statuses: the item is in flight, so its
// gates are not presented (mirrors the Command Post's locked skip).
var lockedStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusQueued:     true,
	backlog.StatusInProgress: true,
}

// terminalStatuses are finished-execution statuses. Terminal items surface
// as Done outcomes, not gates.
var terminalStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusCompleted: true,
	backlog.StatusFailed:    true,
}

func itemKey(item backlog.BacklogItem) string {
	return string(item.Kind) + "/" + item.Name
}

// directDependents builds a reverse depends_on map: item key -> keys of
// items that directly depend on it. Archived dependents are excluded —
// they no longer represent blocked work.
func directDependents(items []backlog.BacklogItem) map[string][]string {
	out := make(map[string][]string)
	for _, item := range items {
		if backlog.IsArchived(item) {
			continue
		}
		key := itemKey(item)
		for _, dep := range item.DependsOn {
			out[dep] = append(out[dep], key)
		}
	}
	for _, deps := range out {
		sort.Strings(deps)
	}
	return out
}

// gateEligible reports whether an item can carry a gate at all.
func gateEligible(item backlog.BacklogItem) bool {
	if backlog.IsArchived(item) {
		return false
	}
	if lockedStatuses[item.Status] || terminalStatuses[item.Status] {
		return false
	}
	return true
}

// DecideSource enumerates KindDecide gates: backlog items whose latest
// workshop round has unanswered decisions.
type DecideSource struct {
	Store ItemStore
}

// Name identifies the source in degradation logs.
func (s DecideSource) Name() string { return "decide" }

// Enumerate implements Source.
func (s DecideSource) Enumerate(_ context.Context) ([]Gate, error) {
	items, err := s.Store.LoadAll(nil)
	if err != nil {
		return nil, err
	}
	dependents := directDependents(items)

	var out []Gate
	for _, item := range items {
		if !gateEligible(item) {
			continue
		}
		latest, _, err := backlog.LoadLatestRound(s.Store.ItemDir(item.Kind, item.Name))
		if err != nil || latest == nil {
			continue
		}
		pending := backlog.CountPendingDecisions(latest)
		if pending == 0 {
			continue
		}
		key := itemKey(item)
		out = append(out, Gate{
			ID:             GateID(KindDecide, "backlog", key),
			Kind:           KindDecide,
			OwnerType:      "backlog",
			OwnerKind:      string(item.Kind),
			OwnerName:      item.Name,
			OwnerTitle:     itemTitle(item),
			Count:          pending,
			Blocks:         dependents[key],
			DecidableSince: latest.GeneratedAt,
		})
	}
	return out, nil
}

// WorkshopSource enumerates KindWorkshop gates: queueable items whose plan
// maturity is not execution-ready (or whose answered round still needs a
// synthesis pass). These are agent-actionable, not human gates — the board
// renders them as workshop/finalize item cards, not gate cards.
//
// Mirrors the Command Post CTA funnel: pending decisions take precedence
// (no workshop gate while questions are open); readiness comes from the
// effective workshop scores.
type WorkshopSource struct {
	Store ItemStore
}

// Name identifies the source in degradation logs.
func (s WorkshopSource) Name() string { return "workshop" }

// queueableStatuses mirror the UI's QUEUEABLE_BACKLOG_STATUSES.
var queueableStatuses = map[backlog.BacklogStatus]bool{
	backlog.StatusBacklog:     true,
	backlog.StatusResearching: true,
	backlog.StatusReady:       true,
}

// Enumerate implements Source.
func (s WorkshopSource) Enumerate(_ context.Context) ([]Gate, error) {
	items, err := s.Store.LoadAll(nil)
	if err != nil {
		return nil, err
	}
	dependents := directDependents(items)

	var out []Gate
	for _, item := range items {
		if !gateEligible(item) || !queueableStatuses[item.Status] {
			continue
		}
		itemDir := s.Store.ItemDir(item.Kind, item.Name)
		latest, roundCount, err := backlog.LoadLatestRound(itemDir)
		if err != nil {
			continue
		}
		if backlog.CountPendingDecisions(latest) > 0 {
			continue // decide gate takes precedence
		}

		raw := make(map[string]int, len(backlog.ReadinessDimensions))
		if latest != nil {
			for _, dim := range backlog.ReadinessDimensions {
				raw[dim] = latest.Readiness[dim]
			}
		}
		ready := backlog.IsReady(backlog.ComputeEffectiveScores(raw, roundCount, item.Kind))
		pendingSynthesis := backlog.NeedsSynthesis(latest)

		var suggested string
		switch {
		case pendingSynthesis && ready:
			suggested = "finalize"
		case pendingSynthesis || !ready:
			suggested = "workshop"
		default:
			continue // ready to run — no workshop gate
		}

		key := itemKey(item)
		gate := Gate{
			ID:         GateID(KindWorkshop, "backlog", key),
			Kind:       KindWorkshop,
			OwnerType:  "backlog",
			OwnerKind:  string(item.Kind),
			OwnerName:  item.Name,
			OwnerTitle: itemTitle(item),
			Count:      1,
			Blocks:     dependents[key],
			Suggested:  suggested,
		}
		if latest != nil {
			gate.DecidableSince = latest.GeneratedAt
		}
		out = append(out, gate)
	}
	return out, nil
}

func itemTitle(item backlog.BacklogItem) string {
	if t := strings.TrimSpace(item.Title); t != "" {
		return t
	}
	return item.Name
}
