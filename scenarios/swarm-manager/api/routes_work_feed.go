package main

import (
	"errors"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/planworkshop"

	"github.com/gorilla/mux"
)

// workFeedEntry is the deliberately small, typed projection used by the
// backlog Activity surface. Detail remains at the source-specific route.
type workFeedEntry struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	Title        string            `json:"title"`
	Outcome      string            `json:"outcome,omitempty"`
	Actor        string            `json:"actor,omitempty"`
	StartedAt    string            `json:"started_at"`
	EndedAt      string            `json:"ended_at,omitempty"`
	CostEstimate float64           `json:"cost_estimate,omitempty"`
	Correlation  map[string]string `json:"correlation,omitempty"`
	DetailRef    string            `json:"detail_ref"`
	DetailAPIRef string            `json:"detail_api_ref,omitempty"`
}

func (s *Server) registerWorkFeedRoutes() {
	s.router.HandleFunc("/api/v1/backlog/{kind}/{name}/work-feed", s.getBacklogWorkFeed).Methods(http.MethodGet)
}

func (s *Server) getBacklogWorkFeed(w http.ResponseWriter, r *http.Request) {
	if s.executionSvc == nil || s.agentActivitySvc == nil || s.reviewSvc == nil || s.eventRepo == nil {
		apierr.MapError(w, "[work-feed] get", apierr.Unavailable("work feed is not initialized"))
		return
	}
	vars := mux.Vars(r)
	kind, name := strings.TrimSpace(vars["kind"]), strings.TrimSpace(vars["name"])
	if kind == "" || name == "" {
		apierr.MapError(w, "[work-feed] get", apierr.BadRequest("backlog kind and name are required"))
		return
	}
	entries := make([]workFeedEntry, 0)
	strategies, err := s.executionSvc.ExecutionStrategies()
	if err != nil {
		apierr.MapError(w, "[work-feed] strategies", err)
		return
	}
	costByStrategy := make(map[string]float64, len(strategies))
	for _, strategy := range strategies {
		costByStrategy[strategy.ID] = strategy.CostEstimate
	}
	executions, err := s.executionSvc.ListSnapshot(r.Context(), execution.ListFilters{BacklogKind: kind, BacklogName: name})
	if err != nil {
		apierr.MapError(w, "[work-feed] executions", err)
		return
	}
	for _, item := range executions {
		strategyID := item.ExecutionStrategy
		if strategyID == "" {
			strategyID = "phased-plan-drain"
		}
		entries = append(entries, workFeedEntry{ID: "execution/" + item.ExecutionID, Kind: "execution", Title: "Executing plan", Outcome: string(item.Status), Actor: item.StartedBy, StartedAt: firstTime(item.StartedAt, item.QueuedAt, item.CreatedAt), EndedAt: item.FinishedAt, CostEstimate: costByStrategy[strategyID], Correlation: compactCorrelation(map[string]string{"execution_id": item.ExecutionID, "workflow_execution_id": item.AgentWorkflowExecutionID, "plan_execution_id": item.PlanManagerExecutionID}), DetailRef: "/executions/" + item.ExecutionID, DetailAPIRef: "/api/v1/execution/" + item.ExecutionID})
	}
	activities, err := s.agentActivitySvc.ListSnapshot(r.Context(), agentactivity.ListFilters{OwnerType: "backlog", OwnerKind: kind, OwnerName: name})
	if err != nil {
		apierr.MapError(w, "[work-feed] activities", err)
		return
	}
	for _, item := range activities {
		entries = append(entries, workFeedEntry{ID: "activity/" + item.ActivityID, Kind: "workflow", Title: workTitle(item.Metadata["workflow_key"], string(item.Purpose)), Outcome: string(item.Status), Actor: item.RequestedBy, StartedAt: firstTime(item.StartedAt, item.RequestedAt), EndedAt: item.FinishedAt, Correlation: compactCorrelation(map[string]string{"activity_id": item.ActivityID, "run_id": item.RunID, "workflow_execution_id": item.Metadata["workflow_execution_id"]}), DetailRef: "/api/v1/agent-activities/" + item.ActivityID, DetailAPIRef: "/api/v1/agent-activities/" + item.ActivityID})
	}
	rounds, err := s.reviewSvc.ListRounds(kind, name)
	if err != nil {
		apierr.MapError(w, "[work-feed] reviews", err)
		return
	}
	for _, round := range rounds {
		entries = append(entries, workFeedEntry{ID: "review/" + strconv.Itoa(round.RoundNum), Kind: "review", Title: "Reviewing result", Outcome: round.Classification, StartedAt: round.GeneratedAt, Correlation: compactCorrelation(map[string]string{"execution_id": round.ExecutionID, "workflow_execution_id": round.AgentWorkflowExecutionID}), DetailRef: "/api/v1/backlog/" + kind + "/" + name + "/review/" + strconv.Itoa(round.RoundNum), DetailAPIRef: "/api/v1/backlog/" + kind + "/" + name + "/review/" + strconv.Itoa(round.RoundNum)})
	}
	// A workshop is a durable, operator-facing work episode even when no agent
	// run is active. Its embedded review correlation adds the corresponding
	// workflow episode without creating a second source of truth.
	workshopID := planworkshop.WorkshopID(planworkshop.Subject{Kind: planworkshop.SubjectBacklog, Ref: kind + "/" + name})
	if workshop, workshopErr := planworkshop.NewStore(s.dataRoot).Load(workshopID); workshopErr == nil {
		outcome := "open"
		if workshop.Review != nil {
			outcome = string(workshop.Review.State)
		}
		entries = append(entries, workFeedEntry{ID: "workshop/" + workshop.ID, Kind: "workshop", Title: "Plan workshop", Outcome: outcome, StartedAt: firstTime(workshop.UpdatedAt, workshop.CreatedAt), DetailRef: "/api/v1/plan-workshops/" + workshop.ID, DetailAPIRef: "/api/v1/plan-workshops/" + workshop.ID})
		if workshop.Review != nil && strings.TrimSpace(workshop.Review.Workflow.ExecutionID) != "" {
			entries = append(entries, workFeedEntry{ID: "workshop-run/" + workshop.Review.Workflow.ExecutionID, Kind: "workflow", Title: "Reviewing plan workshop", Outcome: string(workshop.Review.State), StartedAt: firstTime(workshop.Review.Workflow.StartedAt, workshop.UpdatedAt), DetailRef: "/api/v1/plan-workshops/" + workshop.ID + "/review", DetailAPIRef: "/api/v1/plan-workshops/" + workshop.ID + "/review"})
		}
	} else if !errors.Is(workshopErr, os.ErrNotExist) {
		apierr.MapError(w, "[work-feed] workshop", apierr.Internal("load plan workshop: %s", workshopErr))
		return
	}
	events, err := s.eventRepo.QueryByEntity(r.Context(), eventlog.EntityBacklogItem, kind+"/"+name, 0, 250)
	if err != nil {
		apierr.MapError(w, "[work-feed] events", err)
		return
	}
	for _, event := range events {
		entries = append(entries, workFeedEntry{ID: "event/" + strconv.FormatInt(event.ID, 10), Kind: "event", Title: string(event.EventType), Actor: event.ActorID, StartedAt: event.Timestamp.UTC().Format(time.RFC3339Nano), DetailRef: "/api/v1/events?entity=backlog/" + kind + "/" + name, DetailAPIRef: "/api/v1/events?entity=backlog/" + kind + "/" + name})
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].StartedAt > entries[j].StartedAt })
	if err := httputil.JSON(w, map[string]any{"items": entries}); err != nil {
		apierr.MapError(w, "[work-feed] encode", apierr.Internal("encode work feed"))
	}
}

func compactCorrelation(values map[string]string) map[string]string {
	for key, value := range values {
		if strings.TrimSpace(value) == "" {
			delete(values, key)
		}
	}
	return values
}

func firstTime(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func workTitle(key, fallback string) string {
	if key == "swarm-manager/phased-plan-drain" {
		return "Executing plan"
	}
	if fallback == "review" {
		return "Reviewing result"
	}
	return "Agent work"
}
