package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	actions "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/actions"
	actionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/actions/actions_v1connect"
	checksproto "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	checksconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks/checks_v1connect"
	healing "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/healing"
	healingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/healing/healing_v1connect"
	incidentsproto "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/incidents"
	incidentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/incidents/incidents_v1connect"
	measures "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/measures"
	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/measures/measures_v1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	internalincidents "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reconcile"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RegisterTypedServices mounts the read-only, generated reliability surfaces.
// Legacy REST handlers remain available while clients migrate to these contracts.
func RegisterTypedServices(router *mux.Router, h *Handlers) {
	if router == nil || h == nil {
		return
	}
	checksPath, checksHandler := checksconnect.NewChecksServiceHandler(&typedChecks{h: h})
	actionsPath, actionsHandler := actionsconnect.NewActionsServiceHandler(&typedActions{h: h})
	incidentsPath, incidentsHandler := incidentsconnect.NewIncidentsServiceHandler(&typedIncidents{h: h})
	healingPath, healingHandler := healingconnect.NewHealingServiceHandler(&typedHealing{h: h})
	measuresPath, measuresHandler := measuresconnect.NewMeasuresServiceHandler(&typedMeasures{h: h})
	connectx.RegisterServices(router,
		connectx.ServiceMount{Path: checksPath, Handler: checksHandler},
		connectx.ServiceMount{Path: actionsPath, Handler: actionsHandler},
		connectx.ServiceMount{Path: incidentsPath, Handler: incidentsHandler},
		connectx.ServiceMount{Path: healingPath, Handler: healingHandler},
		connectx.ServiceMount{Path: measuresPath, Handler: measuresHandler},
	)
}

type typedChecks struct{ h *Handlers }

func (s *typedChecks) ListChecks(_ context.Context, _ *connect.Request[checksproto.ListChecksRequest]) (*connect.Response[checksproto.ListChecksResponse], error) {
	items := s.h.registry.ListChecks()
	checks := make([]*checksproto.CheckInfo, 0, len(items))
	for _, item := range items {
		platforms := make([]string, 0, len(item.Platforms))
		for _, declared := range item.Platforms {
			platforms = append(platforms, string(declared))
		}
		checks = append(checks, &checksproto.CheckInfo{
			Id: item.ID, Title: item.Title, Description: item.Description,
			Importance: item.Importance, Category: string(item.Category),
			IntervalSeconds: int32(item.IntervalSeconds),
			Platforms:       platforms,
		})
	}
	return connect.NewResponse(&checksproto.ListChecksResponse{Checks: checks}), nil
}

func (s *typedChecks) GetCheck(_ context.Context, req *connect.Request[checksproto.GetCheckRequest]) (*connect.Response[checksproto.GetCheckResponse], error) {
	checkID := strings.TrimSpace(req.Msg.GetCheckId())
	if checkID == "" {
		return nil, invalidArgument("check_id is required")
	}
	result, ok := s.h.registry.GetResult(checkID)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("check result %q was not found", checkID))
	}
	return connect.NewResponse(&checksproto.GetCheckResponse{Result: checkResult(result)}), nil
}

func (s *typedChecks) GetHistory(ctx context.Context, req *connect.Request[checksproto.GetHistoryRequest]) (*connect.Response[checksproto.GetHistoryResponse], error) {
	checkID := strings.TrimSpace(req.Msg.GetCheckId())
	if checkID == "" {
		return nil, invalidArgument("check_id is required")
	}
	results, err := s.h.store.GetRecentResults(ctx, checkID, boundedLimit(int(req.Msg.GetLimit()), 20, 200))
	if err != nil {
		return nil, internalError("load check history", err)
	}
	response := &checksproto.GetHistoryResponse{Results: make([]*checksproto.CheckResult, 0, len(results))}
	for _, result := range results {
		response.Results = append(response.Results, checkResult(result))
	}
	return connect.NewResponse(response), nil
}

func (s *typedChecks) GetStatus(_ context.Context, _ *connect.Request[checksproto.GetStatusRequest]) (*connect.Response[checksproto.GetStatusResponse], error) {
	summary := s.h.registry.GetSummary()
	results := make([]*checksproto.CheckResult, 0, len(summary.Checks))
	for _, result := range summary.Checks {
		results = append(results, checkResult(result))
	}
	return connect.NewResponse(&checksproto.GetStatusResponse{
		Status: checkStatus(summary.Status), TotalCount: int32(summary.TotalCount),
		OkCount: int32(summary.OkCount), WarningCount: int32(summary.WarnCount),
		CriticalCount: int32(summary.CritCount), NotApplicableCount: int32(summary.NotApplicableCount),
		Checks: results, ComputedAt: timestamppb.New(summary.Timestamp),
	}), nil
}

func (s *typedChecks) GetTransitions(ctx context.Context, req *connect.Request[checksproto.GetTransitionsRequest]) (*connect.Response[checksproto.GetTransitionsResponse], error) {
	transitions, err := s.h.store.GetTransitions(ctx, boundedLimit(int(req.Msg.GetWindowHours()), 24, 168), boundedLimit(int(req.Msg.GetLimit()), 50, 200))
	if err != nil {
		return nil, internalError("load check transitions", err)
	}
	response := &checksproto.GetTransitionsResponse{Transitions: make([]*checksproto.Transition, 0, len(transitions.Transitions))}
	for _, transition := range transitions.Transitions {
		observedAt, parseErr := parseTimestamp(transition.Timestamp)
		if parseErr != nil {
			return nil, internalError("parse transition timestamp", parseErr)
		}
		response.Transitions = append(response.Transitions, &checksproto.Transition{
			CheckId: transition.CheckID, FromStatus: checkStatus(checks.Status(transition.FromStatus)),
			ToStatus: checkStatus(checks.Status(transition.ToStatus)), Message: transition.Message,
			ObservedAt: timestamppb.New(observedAt),
		})
	}
	return connect.NewResponse(response), nil
}

func (s *typedChecks) GetReconcile(ctx context.Context, _ *connect.Request[checksproto.GetReconcileRequest]) (*connect.Response[checksproto.GetReconcileResponse], error) {
	response := connect.NewResponse(&checksproto.GetReconcileResponse{Reconcile: &checksproto.Reconcile{
		Available: false, ComputedAt: timestamppb.New(time.Now().UTC()),
	}})
	if s.h.reconcileProvider == nil {
		response.Msg.Reconcile.UnavailableReason = "reconcile provider is not configured"
		return response, nil
	}
	expected, err := s.h.reconcileProvider.Expected(ctx)
	if err != nil {
		response.Msg.Reconcile.UnavailableReason = err.Error()
		return response, nil
	}
	registered := make([]string, 0, len(s.h.registry.ListChecks()))
	for _, item := range s.h.registry.ListChecks() {
		registered = append(registered, item.ID)
	}
	// A nil installed set means "unknown", not "empty". Compare refuses to
	// call anything a ghost in that case rather than treating every check as
	// targeting a vanished element.
	var installed []string
	installedReason := "installed provider is not configured"
	if s.h.installedProvider != nil {
		installed, err = s.h.installedProvider.Installed(ctx)
		if err != nil {
			installed, installedReason = nil, err.Error()
		}
	}
	diff := reconcile.Compare(registered, installed, expected)
	if !diff.GhostDetectionAvailable && diff.GhostUnavailableReason != "" {
		diff.GhostUnavailableReason = installedReason
	}
	response.Msg.Reconcile.Available = true
	response.Msg.Reconcile.GhostCheckIds = diff.GhostChecks
	response.Msg.Reconcile.OutOfScopeCheckIds = diff.OutOfScopeChecks
	response.Msg.Reconcile.UnsupervisedPlant = diff.UnsupervisedPlant
	response.Msg.Reconcile.GhostDetectionAvailable = diff.GhostDetectionAvailable
	response.Msg.Reconcile.GhostUnavailableReason = diff.GhostUnavailableReason
	return response, nil
}

// ListSaturation tallies transitions for every registered check from a single
// window read. GetSaturation re-reads the same window per check, so a caller
// covering the whole registry paid one round trip and one full transition scan
// per check inside its own deadline.
func (s *typedChecks) ListSaturation(ctx context.Context, req *connect.Request[checksproto.ListSaturationRequest]) (*connect.Response[checksproto.ListSaturationResponse], error) {
	window := boundedLimit(int(req.Msg.GetWindowHours()), 24, 168)
	registered := s.h.registry.ListChecks()
	// The tally must be able to see every transition in the window, or a check
	// whose transitions fell outside the cap would read as "never transitioned"
	// — which is the saturated verdict. Ask for headroom over the registry and
	// report truncation rather than guessing.
	limit := len(registered) * saturationTransitionsPerCheck
	if limit < saturationMinTransitions {
		limit = saturationMinTransitions
	}
	transitions, err := s.h.store.GetTransitions(ctx, window, limit)
	if err != nil {
		return nil, internalError("load saturation transitions", err)
	}
	counts := make(map[string]int32, len(registered))
	for _, transition := range transitions.Transitions {
		counts[transition.CheckID]++
	}
	response := &checksproto.ListSaturationResponse{
		Saturations: make([]*checksproto.Saturation, 0, len(registered)),
		WindowHours: int32(window),
		ComputedAt:  timestamppb.New(time.Now().UTC()),
		Truncated:   len(transitions.Transitions) >= limit,
	}
	current := make(map[string]checks.Status, len(registered))
	for _, result := range s.h.registry.GetSummary().Checks {
		current[result.CheckID] = result.Status
	}
	for _, item := range registered {
		count := counts[item.ID]
		status := current[item.ID]
		abnormal := status == checks.StatusWarning || status == checks.StatusCritical
		response.Saturations = append(response.Saturations, &checksproto.Saturation{
			CheckId: item.ID, Transitioned: count > 0, TransitionCount: count,
			CurrentStatus: checkStatus(status),
			Saturated:     count == 0 && abnormal,
		})
	}
	return connect.NewResponse(response), nil
}

const (
	// saturationTransitionsPerCheck is the per-check transition headroom the
	// batch tally requests so a busy check cannot crowd out a quiet one.
	saturationTransitionsPerCheck = 20
	saturationMinTransitions      = 200
)

func (s *typedChecks) ListShelves(ctx context.Context, req *connect.Request[checksproto.ListShelvesRequest]) (*connect.Response[checksproto.ListShelvesResponse], error) {
	shelfStore, ok := s.h.store.(interface {
		ListCheckShelves(context.Context, bool) ([]persistence.CheckShelf, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("check shelf storage is unavailable"))
	}
	shelves, err := shelfStore.ListCheckShelves(ctx, req.Msg.GetIncludeExpired())
	if err != nil {
		return nil, internalError("load check shelves", err)
	}
	response := connect.NewResponse(&checksproto.ListShelvesResponse{Shelves: make([]*checksproto.Shelf, 0, len(shelves))})
	for _, shelf := range shelves {
		response.Msg.Shelves = append(response.Msg.Shelves, &checksproto.Shelf{
			CheckId: shelf.CheckID, Reason: shelf.Reason, ExpiresAt: timestamppb.New(shelf.ExpiresAt),
			SetBy: shelf.SetBy, CreatedAt: timestamppb.New(shelf.CreatedAt),
		})
	}
	return response, nil
}

func (s *typedChecks) GetSaturation(ctx context.Context, req *connect.Request[checksproto.GetSaturationRequest]) (*connect.Response[checksproto.GetSaturationResponse], error) {
	checkID := strings.TrimSpace(req.Msg.GetCheckId())
	if checkID == "" {
		return nil, invalidArgument("check_id is required")
	}
	window := boundedLimit(int(req.Msg.GetWindowHours()), 24, 168)
	transitions, err := s.h.store.GetTransitions(ctx, window, 200)
	if err != nil {
		return nil, internalError("load saturation transitions", err)
	}
	count := 0
	for _, transition := range transitions.Transitions {
		if transition.CheckID == checkID {
			count++
		}
	}
	return connect.NewResponse(&checksproto.GetSaturationResponse{
		CheckId: checkID, Transitioned: count > 0, TransitionCount: int32(count),
		WindowHours: int32(window), ComputedAt: timestamppb.New(time.Now().UTC()),
	}), nil
}

type typedActions struct{ h *Handlers }

type typedMeasures struct{ h *Handlers }

func (s *typedMeasures) GetUptimeByCheck(ctx context.Context, req *connect.Request[measures.GetUptimeByCheckRequest]) (*connect.Response[measures.GetUptimeByCheckResponse], error) {
	checkID := strings.TrimSpace(req.Msg.GetCheckId())
	if checkID == "" {
		return nil, invalidArgument("check_id is required")
	}
	window := boundedLimit(int(req.Msg.GetWindowHours()), 24, 168)
	trends, err := s.h.store.GetCheckTrends(ctx, window)
	if err != nil {
		return nil, internalError("load uptime by check", err)
	}
	for _, trend := range trends.Trends {
		if trend.CheckID == checkID {
			return connect.NewResponse(&measures.GetUptimeByCheckResponse{Uptime: &measures.UptimeByCheck{
				CheckId: trend.CheckID, UptimePercent: trend.UptimePercent, Total: int32(trend.Total), Ok: int32(trend.Ok), Warning: int32(trend.Warning), Critical: int32(trend.Critical), ComputedAt: timestamppb.Now(),
			}}), nil
		}
	}
	return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("check %q has no uptime evidence", checkID))
}

func (s *typedMeasures) GetRestartCount(ctx context.Context, req *connect.Request[measures.GetRestartCountRequest]) (*connect.Response[measures.GetRestartCountResponse], error) {
	window := boundedLimit(int(req.Msg.GetWindowHours()), 24, 168)
	logs, err := s.h.store.GetActionLogs(ctx, 200)
	if err != nil {
		return nil, internalError("load restart evidence", err)
	}
	cutoff := time.Now().UTC().Add(-time.Duration(window) * time.Hour)
	count := 0
	for _, entry := range logs.Logs {
		observedAt, parseErr := parseTimestamp(entry.Timestamp)
		if parseErr != nil || observedAt.Before(cutoff) {
			continue
		}
		if strings.Contains(strings.ToLower(entry.ActionID), "restart") {
			count++
		}
	}
	return connect.NewResponse(&measures.GetRestartCountResponse{Restarts: &measures.RestartCount{Count: int32(count), WindowHours: int32(window), ComputedAt: timestamppb.Now()}}), nil
}

func (s *typedMeasures) GetHealOutcomes(ctx context.Context, req *connect.Request[measures.GetHealOutcomesRequest]) (*connect.Response[measures.GetHealOutcomesResponse], error) {
	window := boundedLimit(int(req.Msg.GetWindowHours()), 24, 168)
	logs, err := s.h.store.GetActionLogs(ctx, 200)
	if err != nil {
		return nil, internalError("load healing outcomes", err)
	}
	cutoff := time.Now().UTC().Add(-time.Duration(window) * time.Hour)
	counts := map[string]int32{"SUCCEEDED": 0, "FAILED": 0, "TIMED_OUT": 0}
	for _, entry := range logs.Logs {
		observedAt, parseErr := parseTimestamp(entry.Timestamp)
		if parseErr != nil || observedAt.Before(cutoff) {
			continue
		}
		outcome := "FAILED"
		if entry.Success {
			outcome = "SUCCEEDED"
		} else if entry.TimedOut {
			outcome = "TIMED_OUT"
		}
		counts[outcome]++
	}
	outcomes := make([]*measures.HealOutcomeCount, 0, len(counts))
	for _, outcome := range []string{"SUCCEEDED", "FAILED", "TIMED_OUT"} {
		outcomes = append(outcomes, &measures.HealOutcomeCount{Outcome: outcome, Count: counts[outcome]})
	}
	return connect.NewResponse(&measures.GetHealOutcomesResponse{Outcomes: outcomes, WindowHours: int32(window), ComputedAt: timestamppb.Now()}), nil
}

func (s *typedMeasures) GetCriticalCount(ctx context.Context, req *connect.Request[measures.GetCriticalCountRequest]) (*connect.Response[measures.GetCriticalCountResponse], error) {
	window := boundedLimit(int(req.Msg.GetWindowHours()), 24, 168)
	stats, err := s.h.store.GetUptimeStats(ctx, window)
	if err != nil {
		return nil, internalError("load critical count", err)
	}
	return connect.NewResponse(&measures.GetCriticalCountResponse{Critical: &measures.CriticalCount{Count: int32(stats.CriticalEvents), WindowHours: int32(window), ComputedAt: timestamppb.Now()}}), nil
}

func (s *typedActions) GetEventWeightedUptime(ctx context.Context, req *connect.Request[actions.GetEventWeightedUptimeRequest]) (*connect.Response[actions.GetEventWeightedUptimeResponse], error) {
	stats, err := s.h.store.GetUptimeStats(ctx, boundedLimit(int(req.Msg.GetWindowHours()), 24, 168))
	if err != nil {
		return nil, internalError("load event-weighted uptime", err)
	}
	return connect.NewResponse(&actions.GetEventWeightedUptimeResponse{Uptime: &actions.EventWeightedUptime{
		TotalEvents: int32(stats.TotalEvents), OkEvents: int32(stats.OkEvents), WarningEvents: int32(stats.WarningEvents),
		CriticalEvents: int32(stats.CriticalEvents), UptimePercentage: stats.UptimePercentage,
		WindowHours: int32(stats.WindowHours), ComputedAt: timestamppb.New(time.Now().UTC()),
	}}), nil
}

func (s *typedActions) GetPerCheckTrends(ctx context.Context, req *connect.Request[actions.GetPerCheckTrendsRequest]) (*connect.Response[actions.GetPerCheckTrendsResponse], error) {
	trends, err := s.h.store.GetCheckTrends(ctx, boundedLimit(int(req.Msg.GetWindowHours()), 24, 168))
	if err != nil {
		return nil, internalError("load per-check trends", err)
	}
	response := &actions.GetPerCheckTrendsResponse{WindowHours: int32(trends.WindowHours), Trends: make([]*actions.PerCheckTrend, 0, len(trends.Trends))}
	for _, trend := range trends.Trends {
		lastChecked, parseErr := parseTimestamp(trend.LastChecked)
		if parseErr != nil {
			return nil, internalError("parse trend timestamp", parseErr)
		}
		statuses := make([]checksproto.CheckStatus, 0, len(trend.RecentStatuses))
		for _, status := range trend.RecentStatuses {
			statuses = append(statuses, checkStatus(checks.Status(status)))
		}
		response.Trends = append(response.Trends, &actions.PerCheckTrend{
			CheckId: trend.CheckID, Total: int32(trend.Total), Ok: int32(trend.Ok), Warning: int32(trend.Warning),
			Critical: int32(trend.Critical), UptimePercent: trend.UptimePercent,
			CurrentStatus: checkStatus(checks.Status(trend.CurrentStatus)), RecentStatuses: statuses,
			LastChecked: timestamppb.New(lastChecked),
		})
	}
	return connect.NewResponse(response), nil
}

func (s *typedActions) GetHistory(ctx context.Context, req *connect.Request[actions.GetHistoryRequest]) (*connect.Response[actions.GetHistoryResponse], error) {
	var logs *persistence.ActionLogsResponse
	var err error
	limit := boundedLimit(int(req.Msg.GetLimit()), 20, 200)
	if checkID := strings.TrimSpace(req.Msg.GetCheckId()); checkID != "" {
		logs, err = s.h.store.GetActionLogsForCheck(ctx, checkID, limit)
	} else {
		logs, err = s.h.store.GetActionLogs(ctx, limit)
	}
	if err != nil {
		return nil, internalError("load action history", err)
	}
	response := &actions.GetHistoryResponse{Entries: make([]*actions.ActionHistoryEntry, 0, len(logs.Logs))}
	for _, logEntry := range logs.Logs {
		observedAt, parseErr := parseTimestamp(logEntry.Timestamp)
		if parseErr != nil {
			return nil, internalError("parse action timestamp", parseErr)
		}
		response.Entries = append(response.Entries, &actions.ActionHistoryEntry{
			Id: logEntry.ID, CheckId: logEntry.CheckID, ActionId: logEntry.ActionID, Success: logEntry.Success,
			TimedOut: logEntry.TimedOut, Message: logEntry.Message, Error: logEntry.Error,
			DurationMs: logEntry.DurationMs, ObservedAt: timestamppb.New(observedAt),
		})
	}
	return connect.NewResponse(response), nil
}

func (s *typedActions) GetTransitions(ctx context.Context, req *connect.Request[actions.GetTransitionsRequest]) (*connect.Response[actions.GetTransitionsResponse], error) {
	transitions, err := s.h.store.GetTransitions(ctx, boundedLimit(int(req.Msg.GetWindowHours()), 24, 168), boundedLimit(int(req.Msg.GetLimit()), 50, 200))
	if err != nil {
		return nil, internalError("load action transitions", err)
	}
	response := &actions.GetTransitionsResponse{Transitions: make([]*actions.Transition, 0, len(transitions.Transitions))}
	for _, transition := range transitions.Transitions {
		observedAt, parseErr := parseTimestamp(transition.Timestamp)
		if parseErr != nil {
			return nil, internalError("parse action transition timestamp", parseErr)
		}
		response.Transitions = append(response.Transitions, &actions.Transition{
			CheckId: transition.CheckID, FromStatus: checkStatus(checks.Status(transition.FromStatus)),
			ToStatus: checkStatus(checks.Status(transition.ToStatus)), Message: transition.Message,
			ObservedAt: timestamppb.New(observedAt),
		})
	}
	return connect.NewResponse(response), nil
}

type typedIncidents struct{ h *Handlers }

func (s *typedIncidents) ListIncidents(ctx context.Context, req *connect.Request[incidentsproto.ListIncidentsRequest]) (*connect.Response[incidentsproto.ListIncidentsResponse], error) {
	list, err := s.h.store.ListIncidents(ctx, internalincidents.ListFilters{
		Status: internalincidents.Status(req.Msg.GetStatus()), Severity: internalincidents.Severity(req.Msg.GetSeverity()),
		Limit: boundedLimit(int(req.Msg.GetLimit()), 50, 200),
	})
	if err != nil {
		return nil, internalError("load incidents", err)
	}
	response := &incidentsproto.ListIncidentsResponse{Incidents: make([]*incidentsproto.Incident, 0, len(list.Incidents))}
	for _, incident := range list.Incidents {
		response.Incidents = append(response.Incidents, incidentMessage(incident))
	}
	return connect.NewResponse(response), nil
}

func (s *typedIncidents) GetIncident(ctx context.Context, req *connect.Request[incidentsproto.GetIncidentRequest]) (*connect.Response[incidentsproto.GetIncidentResponse], error) {
	id := strings.TrimSpace(req.Msg.GetIncidentId())
	if id == "" {
		return nil, invalidArgument("incident_id is required")
	}
	incident, err := s.h.store.GetIncident(ctx, id)
	if err != nil {
		return nil, internalError("load incident", err)
	}
	if incident == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("incident %q was not found", id))
	}
	return connect.NewResponse(&incidentsproto.GetIncidentResponse{Incident: incidentMessage(*incident)}), nil
}

func (s *typedIncidents) ListObservations(ctx context.Context, req *connect.Request[incidentsproto.ListObservationsRequest]) (*connect.Response[incidentsproto.ListObservationsResponse], error) {
	id := strings.TrimSpace(req.Msg.GetIncidentId())
	if id == "" {
		return nil, invalidArgument("incident_id is required")
	}
	items, err := s.h.store.ListIncidentObservations(ctx, id, boundedLimit(int(req.Msg.GetLimit()), 50, 200))
	if err != nil {
		return nil, internalError("load incident observations", err)
	}
	response := &incidentsproto.ListObservationsResponse{Observations: make([]*incidentsproto.Observation, 0, len(items))}
	for _, item := range items {
		payload, marshalErr := json.Marshal(item.Evidence)
		if marshalErr != nil {
			return nil, internalError("encode observation evidence", marshalErr)
		}
		response.Observations = append(response.Observations, &incidentsproto.Observation{
			Id: fmt.Sprint(item.ID), IncidentId: item.IncidentID, Kind: item.SourceCheckID,
			PayloadJson: string(payload), ObservedAt: timestamppb.New(item.ObservedAt),
		})
	}
	return connect.NewResponse(response), nil
}

type typedHealing struct{ h *Handlers }

func (s *typedHealing) ListOutcomes(ctx context.Context, req *connect.Request[healing.ListOutcomesRequest]) (*connect.Response[healing.ListOutcomesResponse], error) {
	logs, err := s.h.store.GetActionLogs(ctx, boundedLimit(int(req.Msg.GetWindowHours()), 24, 168))
	if err != nil {
		return nil, internalError("load healing outcomes", err)
	}
	response := &healing.ListOutcomesResponse{Outcomes: make([]*healing.HealOutcome, 0, len(logs.Logs))}
	for _, logEntry := range logs.Logs {
		observedAt, parseErr := parseTimestamp(logEntry.Timestamp)
		if parseErr != nil {
			return nil, internalError("parse healing timestamp", parseErr)
		}
		response.Outcomes = append(response.Outcomes, healOutcome(logEntry, observedAt))
	}
	return connect.NewResponse(response), nil
}

func (s *typedHealing) GetEpisodes(_ context.Context, _ *connect.Request[healing.GetEpisodesRequest]) (*connect.Response[healing.GetEpisodesResponse], error) {
	// Episodes are not yet persisted by the legacy auto-heal engine. Returning an
	// empty typed collection is intentional: action logs are outcomes, not episodes.
	return connect.NewResponse(&healing.GetEpisodesResponse{Episodes: []*healing.HealEpisode{}}), nil
}

func (s *typedHealing) GetHistory(ctx context.Context, req *connect.Request[healing.GetHistoryRequest]) (*connect.Response[healing.GetHistoryResponse], error) {
	checkID := strings.TrimSpace(req.Msg.GetCheckId())
	var logs *persistence.ActionLogsResponse
	var err error
	limit := boundedLimit(int(req.Msg.GetLimit()), 20, 200)
	if checkID == "" {
		logs, err = s.h.store.GetActionLogs(ctx, limit)
	} else {
		logs, err = s.h.store.GetActionLogsForCheck(ctx, checkID, limit)
	}
	if err != nil {
		return nil, internalError("load healing history", err)
	}
	response := &healing.GetHistoryResponse{Outcomes: make([]*healing.HealOutcome, 0, len(logs.Logs))}
	for _, logEntry := range logs.Logs {
		observedAt, parseErr := parseTimestamp(logEntry.Timestamp)
		if parseErr != nil {
			return nil, internalError("parse healing history timestamp", parseErr)
		}
		response.Outcomes = append(response.Outcomes, healOutcome(logEntry, observedAt))
	}
	return connect.NewResponse(response), nil
}

func checkResult(result checks.Result) *checksproto.CheckResult {
	details, _ := json.Marshal(result.Details)
	return &checksproto.CheckResult{
		CheckId: result.CheckID, Status: checkStatus(result.Status), Message: result.Message,
		ObservedAt: timestamppb.New(result.Timestamp), DurationMs: result.Duration.Milliseconds(), DetailsJson: string(details),
	}
}

func checkStatus(status checks.Status) checksproto.CheckStatus {
	switch status {
	case checks.StatusOK:
		return checksproto.CheckStatus_CHECK_STATUS_OK
	case checks.StatusWarning:
		return checksproto.CheckStatus_CHECK_STATUS_WARNING
	case checks.StatusCritical:
		return checksproto.CheckStatus_CHECK_STATUS_CRITICAL
	case checks.StatusNotApplicable:
		return checksproto.CheckStatus_CHECK_STATUS_NOT_APPLICABLE
	default:
		return checksproto.CheckStatus_CHECK_STATUS_UNSPECIFIED
	}
}

func incidentMessage(incident internalincidents.Incident) *incidentsproto.Incident {
	return &incidentsproto.Incident{
		Id: incident.ID, Fingerprint: incident.Fingerprint, Title: incident.Title,
		Severity: string(incident.Severity), Status: string(incident.Status), Summary: incident.Summary,
		FirstSeenAt: timestamppb.New(incident.DetectedAt), LastSeenAt: timestamppb.New(incident.LastSeenAt),
	}
}

func healOutcome(logEntry persistence.ActionLog, observedAt time.Time) *healing.HealOutcome {
	outcome := healing.Outcome_OUTCOME_FAILED
	if logEntry.Success {
		outcome = healing.Outcome_OUTCOME_SUCCEEDED
	} else if logEntry.TimedOut {
		outcome = healing.Outcome_OUTCOME_TIMED_OUT
	}
	return &healing.HealOutcome{
		CheckId: logEntry.CheckID, ActionId: logEntry.ActionID, Outcome: outcome,
		Message: firstNonEmpty(logEntry.Error, logEntry.Message), ObservedAt: timestamppb.New(observedAt), DurationMs: logEntry.DurationMs,
	}
}

func parseTimestamp(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, errors.New("empty timestamp")
	}
	return time.Parse(time.RFC3339Nano, raw)
}

func boundedLimit(value, fallback, maximum int) int {
	if value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func invalidArgument(message string) error {
	return connect.NewError(connect.CodeInvalidArgument, errors.New(message))
}

func internalError(operation string, err error) error {
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s: %w", operation, err))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
