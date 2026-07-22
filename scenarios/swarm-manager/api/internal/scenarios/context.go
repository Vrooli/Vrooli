package scenarios

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"

	"github.com/gorilla/mux"
)

// GoalsLister enumerates goals with their authoritative dependency-derived
// closures. Satisfied by *goals.Service.
type GoalsLister interface {
	List() ([]goals.GoalWithScope, error)
}

// ScenarioOrphanItem is a backlog item that targets a scenario but is not
// assigned to any initiative. Orphans are the primary signal that a
// readiness-style initiative is needed.
type ScenarioOrphanItem struct {
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Priority   int     `json:"priority"`
	ArchivedAt *string `json:"archived_at,omitempty"`
}

// ScenarioFix is a backlog fix item targeting a scenario, in or out of an
// initiative. Surfaces the fields needed to decide "is this a recurrence?"
// without an extra fetch per item.
type ScenarioFix struct {
	Name       string  `json:"name"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Priority   int     `json:"priority"`
	Goal       string  `json:"goal,omitempty"`
	Updated    string  `json:"updated,omitempty"`
	ArchivedAt *string `json:"archived_at,omitempty"`
	Path       string  `json:"path"`
}

// ScenarioFixHistory partitions fixes into active vs archived for the UI's
// Active/Archived/All toggle.
type ScenarioFixHistory struct {
	Active   []ScenarioFix `json:"active"`
	Archived []ScenarioFix `json:"archived"`
}

// ScenarioContextRollup aggregates completion stats across every item
// (whether inside an initiative or orphan) targeting the scenario.
type ScenarioContextRollup struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Failed     int `json:"failed"`
	Pending    int `json:"pending"`
	Archived   int `json:"archived"`
}

// ScenarioGoal is one goal whose derived scope contains scenario-targeting
// work. Its rollup covers only that scenario's items in the goal closure.
type ScenarioGoal struct {
	Name     string                `json:"name"`
	Title    string                `json:"title"`
	Status   string                `json:"status"`
	Priority int                   `json:"priority"`
	Scope    ScenarioContextRollup `json:"scope"`
}

// ScenarioContext is the full coverage view for a scenario: every active
// goal whose derived scope targets the scenario, every orphan item
// targeting the scenario, and a combined completion rollup.
type ScenarioContext struct {
	ScenarioName string                `json:"scenario_name"`
	Goals        []ScenarioGoal        `json:"goals"`
	OrphanItems  []ScenarioOrphanItem  `json:"orphan_items"`
	Rollup       ScenarioContextRollup `json:"rollup"`
	Fixes        ScenarioFixHistory    `json:"fixes"`
}

// SetGoalsLister injects a goal lister for scenario context computation.
func (h *Handler) SetGoalsLister(gl GoalsLister) {
	h.goalsLister = gl
}

// GetContext returns the scenario-scoped coverage view: initiatives whose
// items target this scenario, orphan items targeting this scenario, and a
// combined completion rollup. An unknown scenario returns an empty-but-well-
// formed payload rather than 404 — "no coverage yet" is a valid state.
func (h *Handler) GetContext(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(mux.Vars(r)["name"])
	if name == "" {
		apierr.MapError(w, "[scenarios] context", apierr.BadRequest("scenario name is required"))
		return
	}

	ctx := ScenarioContext{
		ScenarioName: name,
		Goals:        []ScenarioGoal{},
		OrphanItems:  []ScenarioOrphanItem{},
		Fixes:        ScenarioFixHistory{Active: []ScenarioFix{}, Archived: []ScenarioFix{}},
	}

	// Items drive scenario targeting, goal rollups, and orphan detection.
	if h.backlogLister != nil {
		items, err := h.backlogLister.LoadAll(nil)
		if err != nil {
			slog.Error("failed to load backlog items for scenario context", "scenario", name, "error", err)
			apierr.MapError(w, "[scenarios] context", apierr.Internal("failed to load backlog items"))
			return
		}
		byRef := make(map[string]backlog.BacklogItem, len(items))
		for _, item := range items {
			byRef[string(item.Kind)+"/"+item.Name] = item
		}
		assigned := map[string]string{}
		if h.goalsLister != nil {
			all, err := h.goalsLister.List()
			if err != nil {
				apierr.MapError(w, "[scenarios] context", apierr.Internal("failed to list goals"))
				return
			}
			for _, goal := range all {
				if goal.Goal.ArchivedAt != nil {
					continue
				}
				rollup := ScenarioContextRollup{}
				for _, ref := range goal.Scope.Closure {
					item, ok := byRef[ref]
					if !ok {
						continue
					}
					assigned[ref] = goal.Goal.Name
					if !itemTargetsScenario(item, name) {
						continue
					}
					addItemRollup(&rollup, item)
				}
				if rollup.Total > 0 {
					ctx.Goals = append(ctx.Goals, ScenarioGoal{Name: goal.Goal.Name, Title: goal.Goal.Title, Status: goal.Goal.Status, Priority: goal.Goal.Priority, Scope: rollup})
					addRollup(&ctx.Rollup, rollup)
				}
			}
		}
		for _, item := range items {
			if !itemTargetsScenario(item, name) {
				continue
			}
			if item.Kind == backlog.KindFix {
				fix := ScenarioFix{
					Name:       item.Name,
					Title:      item.Title,
					Status:     string(item.Status),
					Priority:   item.Priority,
					Goal:       assigned[string(item.Kind)+"/"+item.Name],
					Updated:    item.Updated,
					ArchivedAt: item.ArchivedAt,
					Path:       "fix/" + item.Name,
				}
				if item.ArchivedAt != nil {
					ctx.Fixes.Archived = append(ctx.Fixes.Archived, fix)
				} else {
					ctx.Fixes.Active = append(ctx.Fixes.Active, fix)
				}
			}
			if assigned[string(item.Kind)+"/"+item.Name] != "" {
				continue
			}
			orphan := ScenarioOrphanItem{
				Kind:       string(item.Kind),
				Name:       item.Name,
				Title:      item.Title,
				Status:     string(item.Status),
				Priority:   item.Priority,
				ArchivedAt: item.ArchivedAt,
			}
			ctx.OrphanItems = append(ctx.OrphanItems, orphan)
			addItemRollup(&ctx.Rollup, item)
		}
		sortFixes(ctx.Fixes.Active)
		sortFixes(ctx.Fixes.Archived)
	}

	if err := httputil.JSON(w, ctx); err != nil {
		apierr.MapError(w, "[scenarios] context", apierr.Internal("failed to encode response"))
	}
}

func addItemRollup(rollup *ScenarioContextRollup, item backlog.BacklogItem) {
	rollup.Total++
	switch item.Status {
	case backlog.StatusCompleted:
		rollup.Completed++
	case backlog.StatusFailed:
		rollup.Failed++
	case backlog.StatusInProgress, backlog.StatusQueued, backlog.StatusResearching:
		rollup.InProgress++
	default:
		rollup.Pending++
	}
	if item.ArchivedAt != nil {
		rollup.Archived++
	}
}
func addRollup(dst *ScenarioContextRollup, src ScenarioContextRollup) {
	dst.Total += src.Total
	dst.Completed += src.Completed
	dst.InProgress += src.InProgress
	dst.Failed += src.Failed
	dst.Pending += src.Pending
	dst.Archived += src.Archived
}

func itemTargetsScenario(item backlog.BacklogItem, scenario string) bool {
	for _, s := range pathutil.ScenariosFromGlobs(item.AcceptanceAllow) {
		if s == scenario {
			return true
		}
	}
	return false
}

// sortFixes orders by priority desc, then updated desc (newest first), then
// name for deterministic output when ties remain.
func sortFixes(fixes []ScenarioFix) {
	sort.SliceStable(fixes, func(i, j int) bool {
		if fixes[i].Priority != fixes[j].Priority {
			return fixes[i].Priority > fixes[j].Priority
		}
		if fixes[i].Updated != fixes[j].Updated {
			return fixes[i].Updated > fixes[j].Updated
		}
		return fixes[i].Name < fixes[j].Name
	})
}

func stringsContains(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}
