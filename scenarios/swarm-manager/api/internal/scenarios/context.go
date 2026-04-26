package scenarios

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/pathutil"

	"github.com/gorilla/mux"
)

// InitiativesLister enumerates every initiative with its rollup and the
// scenarios its member items target. Satisfied by *initiatives.Service.
type InitiativesLister interface {
	List() ([]initiatives.InitiativeWithRollup, error)
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
	Initiative string  `json:"initiative,omitempty"`
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

// ScenarioContext is the full coverage view for a scenario: every
// initiative whose member items target the scenario, every orphan item
// targeting the scenario, and a combined completion rollup.
type ScenarioContext struct {
	ScenarioName string                             `json:"scenario_name"`
	Initiatives  []initiatives.InitiativeWithRollup `json:"initiatives"`
	OrphanItems  []ScenarioOrphanItem               `json:"orphan_items"`
	Rollup       ScenarioContextRollup              `json:"rollup"`
	Fixes        ScenarioFixHistory                 `json:"fixes"`
}

// SetInitiativesLister injects an initiatives lister for scenario context
// computation. Without this, /scenarios/{name}/context returns an empty
// initiatives list and only populates orphan coverage.
func (h *Handler) SetInitiativesLister(il InitiativesLister) {
	h.initiativesLister = il
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
		Initiatives:  []initiatives.InitiativeWithRollup{},
		OrphanItems:  []ScenarioOrphanItem{},
		Fixes:        ScenarioFixHistory{Active: []ScenarioFix{}, Archived: []ScenarioFix{}},
	}

	// Enumerate initiatives targeting this scenario. The initiatives lister
	// populates TargetScenarios on every row so filtering is one pass.
	if h.initiativesLister != nil {
		all, err := h.initiativesLister.List()
		if err != nil {
			slog.Error("failed to list initiatives for scenario context", "scenario", name, "error", err)
			apierr.MapError(w, "[scenarios] context", apierr.Internal("failed to list initiatives"))
			return
		}
		for _, init := range all {
			if stringsContains(init.TargetScenarios, name) {
				ctx.Initiatives = append(ctx.Initiatives, init)
				ctx.Rollup.Total += init.Rollup.Total
				ctx.Rollup.Completed += init.Rollup.Completed
				ctx.Rollup.InProgress += init.Rollup.InProgress
				ctx.Rollup.Failed += init.Rollup.Failed
				ctx.Rollup.Pending += init.Rollup.Pending
				ctx.Rollup.Archived += init.Rollup.Archived
			}
		}
	}

	// Enumerate orphan items: items whose acceptance_allow targets this
	// scenario but whose Initiative field is empty. Requires the backlog
	// lister — without it orphans cannot be computed.
	if h.backlogLister != nil {
		items, err := h.backlogLister.LoadAll(nil)
		if err != nil {
			slog.Error("failed to load backlog items for scenario context", "scenario", name, "error", err)
			apierr.MapError(w, "[scenarios] context", apierr.Internal("failed to load backlog items"))
			return
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
					Initiative: item.Initiative,
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
			if strings.TrimSpace(item.Initiative) != "" {
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
			ctx.Rollup.Total++
			switch item.Status {
			case backlog.StatusCompleted:
				ctx.Rollup.Completed++
			case backlog.StatusFailed:
				ctx.Rollup.Failed++
			case backlog.StatusInProgress, backlog.StatusQueued, backlog.StatusResearching:
				ctx.Rollup.InProgress++
			default:
				ctx.Rollup.Pending++
			}
			if item.ArchivedAt != nil {
				ctx.Rollup.Archived++
			}
		}
		sortFixes(ctx.Fixes.Active)
		sortFixes(ctx.Fixes.Archived)
	}

	if err := httputil.JSON(w, ctx); err != nil {
		apierr.MapError(w, "[scenarios] context", apierr.Internal("failed to encode response"))
	}
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
