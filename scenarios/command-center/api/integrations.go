package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"command-center/upstream"
	"github.com/gorilla/mux"
	capreg "github.com/vrooli/vrooli/packages/capability-registry-go"
)

// integrationChecker is deliberately a cheap health probe. Feature payloads are
// qualified separately by the outcome adapters; a healthy process is not proof
// that a feature exists.
type integrationChecker struct {
	client                 upstream.Client
	features               []string
	slug                   string
	label                  string
	failureActionKind      capreg.ActionKind
	failureActionLabel     string
	failureOperatorCommand string
}

func (c integrationChecker) Check(ctx context.Context) (capreg.Status, string) {
	result := c.CheckResult(ctx)
	return result.Status, result.Message
}

func (c integrationChecker) CheckResult(ctx context.Context) capreg.CheckResult {
	_, err := c.client.Fetch(ctx, "/health")
	if err != nil {
		actionKind := c.failureActionKind
		if actionKind == capreg.ActionKindNone {
			actionKind = capreg.ActionKindOwnerGuidance
		}
		actionLabel := c.failureActionLabel
		operatorCommand := c.failureOperatorCommand
		if actionKind == capreg.ActionKindScenarioStart || actionKind == capreg.ActionKindScenarioRestart {
			actionLabel = first(actionLabel, "Start "+c.label)
			operatorCommand = first(operatorCommand, "vrooli scenario start "+c.slug+" --json")
		} else {
			actionLabel = first(actionLabel, "Review "+c.label)
			operatorCommand = first(operatorCommand, "vrooli status --json")
		}
		return capreg.CheckResult{Status: capreg.StatusUnavailable, Message: err.Error(), ReasonCode: "upstream_unavailable", ActionKind: actionKind, ActionLabel: actionLabel, OperatorCommand: operatorCommand, FeatureStatus: featureStatuses(c.features, "unknown")}
	}
	statuses := featureStatuses(c.features, "unknown")
	reasons := map[string]string{}
	if probe, ok := c.client.(upstream.FeatureProbe); ok {
		probed, probedReasons := probe.ProbeFeatures(ctx)
		for feature, status := range probed {
			if _, declared := statuses[feature]; declared {
				statuses[feature] = status
			}
		}
		for feature, reason := range probedReasons {
			reasons[feature] = reason
		}
	}
	return capreg.CheckResult{Status: capreg.StatusAvailable, Message: "health endpoint reachable", ReasonCode: "health_reachable", FeatureStatus: statuses, FeatureReason: reasons}
}

type integrationSnapshot struct {
	GeneratedAt time.Time      `json:"generatedAt"`
	States      []capreg.State `json:"states"`
}

func commandCenterIntegrationRegistry(s *Server) *capreg.Registry {
	path := os.Getenv("COMMAND_CENTER_SERVICE_MANIFEST")
	if path == "" {
		path = "../.vrooli/service.json"
	}
	overlays := map[string]capreg.Overlay{
		"swarm-manager":               {ID: "swarm-manager", Name: "Swarm Manager", Features: []string{"throughput_stats", "agent_stats", "swarm_throughput", "swarm_active_agents", "timing_stats", "scope_stats", "blocking_stats", "dashboard_stats", "review_stats", "composite_throughput"}, FeatureRequirements: featureRequirements([]string{"throughput_stats", "agent_stats", "swarm_throughput", "swarm_active_agents", "timing_stats", "scope_stats", "blocking_stats", "dashboard_stats", "review_stats", "composite_throughput"})},
		"landing-page-business-suite": {ID: "landing-page-business-suite", Name: "Landing Page Business Suite", Features: []string{"visitors", "conversions", "revenue_rollup", "revenue_mrr", "revenue_today", "subscriber_counts", "churn", "credit_balances", "credit_consumption", "usage_records", "cta_clicks", "scroll_depth", "variant_ab", "composite_revenue", "composite_reach"}, FeatureRequirements: featureRequirements([]string{"visitors", "conversions", "revenue_rollup", "revenue_mrr", "revenue_today", "subscriber_counts", "churn", "credit_balances", "credit_consumption", "usage_records", "cta_clicks", "scroll_depth", "variant_ab", "composite_revenue", "composite_reach"})},
		"prompt-manager":              {ID: "prompt-manager", Name: "Prompt Manager", Features: []string{"team_instrument"}, FeatureRequirements: featureRequirements([]string{"team_instrument"}), ActionKind: capreg.ActionKindOwnerGuidance, ActionLabel: "Review Prompt Manager", OperatorCommand: "vrooli scenario status prompt-manager --json"},
	}
	defs, err := capreg.ProjectManifest(path, overlays)
	if err != nil {
		// A missing or malformed manifest is a startup contract failure. Hiding
		// it would make every declared producer disappear from the registry and
		// turn binding validation into a false success.
		panic("command-center integration manifest invalid: " + err.Error())
	}
	coreFeatures := []string{"scenario_health", "scenario_inventory", "active_scenarios", "total_scenarios", "scenario_completeness", "scenario_health_detail", "scenario_ports", "system_uptime", "composite_system_health", "composite_portfolio"}
	defs = append(defs, capreg.Def{ID: "vrooli-core", Origin: "control_plane", Name: "Vrooli Core", Description: "Control plane used for scenario discovery and lifecycle observation.", DependencyKind: capreg.DependencyControlPlane, DependencySlug: "vrooli", Features: coreFeatures, FeatureRequirements: featureRequirements(coreFeatures), ActionKind: capreg.ActionKindOwnerGuidance, ActionLabel: "Review control plane", OperatorCommand: "vrooli status --json", Enabled: true})
	checkers := map[string]capreg.Checker{
		"vrooli-core":                 integrationChecker{client: s.vrooli, features: []string{"scenario_health", "scenario_inventory", "active_scenarios", "total_scenarios", "scenario_completeness", "scenario_health_detail", "scenario_ports", "system_uptime", "composite_system_health", "composite_portfolio"}, slug: "vrooli", label: "Vrooli Core", failureActionKind: capreg.ActionKindOwnerGuidance, failureActionLabel: "Review control plane", failureOperatorCommand: "vrooli status --json"},
		"swarm-manager":               integrationChecker{client: s.swarm, features: []string{"throughput_stats", "agent_stats", "swarm_throughput", "swarm_active_agents", "timing_stats", "scope_stats", "blocking_stats", "dashboard_stats", "review_stats", "composite_throughput"}, slug: "swarm-manager", label: "Swarm Manager", failureActionKind: capreg.ActionKindScenarioStart},
		"landing-page-business-suite": integrationChecker{client: s.lpbs, features: []string{"visitors", "conversions", "revenue_rollup", "revenue_mrr", "revenue_today", "subscriber_counts", "churn", "credit_balances", "credit_consumption", "usage_records", "cta_clicks", "scroll_depth", "variant_ab", "composite_revenue", "composite_reach"}, slug: "landing-page-business-suite", label: "Landing Page Business Suite", failureActionKind: capreg.ActionKindScenarioStart},
	}
	// Prompt Manager is checked through the same typed transmitter used for
	// objective and team-instrument projections. A missing transmitter remains
	// an honest owner-guidance state in tests or minimal deployments.
	if s.promptManager != nil {
		checkers["prompt-manager"] = s.promptManager
	} else {
		checkers["prompt-manager"] = unavailableChecker{"no Prompt Manager transmitter configured"}
	}
	return capreg.New(defs, checkers, 5*time.Second)
}

func featureRequirements(ids []string) map[string]capreg.FeatureRequirement {
	out := make(map[string]capreg.FeatureRequirement, len(ids))
	for _, id := range ids {
		out[id] = capreg.FeatureRequirement{ContractVersion: "legacy.v1"}
	}
	return out
}

type unavailableChecker struct{ message string }

func (c unavailableChecker) Check(context.Context) (capreg.Status, string) {
	return capreg.StatusUnavailable, c.message
}

func (c unavailableChecker) CheckResult(context.Context) capreg.CheckResult {
	return capreg.CheckResult{Status: capreg.StatusUnavailable, Message: c.message, ReasonCode: "feature_transmitter_unavailable", ActionKind: capreg.ActionKindOwnerGuidance, ActionLabel: "Review Prompt Manager", OperatorCommand: "vrooli scenario status prompt-manager --json", FeatureStatus: featureStatuses([]string{"team_instrument"}, "unknown")}
}

func featureStatuses(features []string, status string) map[string]string {
	out := make(map[string]string, len(features))
	for _, feature := range features {
		out[feature] = status
	}
	return out
}

func validateOutcomeBindings(defs []capreg.Def, metrics []MetricEntry) error {
	features := make(map[string]map[string]struct{}, len(defs))
	for _, def := range defs {
		set := make(map[string]struct{}, len(def.Features))
		for _, feature := range def.Features {
			set[feature] = struct{}{}
		}
		features[def.ID] = set
	}
	seen := make(map[string]struct{}, len(metrics))
	for _, metric := range metrics {
		if _, duplicate := seen[metric.ID]; duplicate {
			return fmt.Errorf("metric %q is declared more than once", metric.ID)
		}
		seen[metric.ID] = struct{}{}
		if metric.Source.IntegrationID == "none" {
			continue
		}
		set, ok := features[metric.Source.IntegrationID]
		if !ok {
			return fmt.Errorf("metric %q references undeclared integration %q", metric.ID, metric.Source.IntegrationID)
		}
		if _, ok := set[metric.Source.FeatureID]; !ok {
			return fmt.Errorf("metric %q references undeclared feature %q on integration %q", metric.ID, metric.Source.FeatureID, metric.Source.IntegrationID)
		}
	}
	return nil
}

func (s *Server) integrationSnapshot(ctx context.Context, force bool) integrationSnapshot {
	if s.integrationRegistry == nil {
		s.integrationRegistry = commandCenterIntegrationRegistry(s)
	}
	var states []capreg.State
	if force {
		states = s.integrationRegistry.ResolveForce(ctx)
	} else {
		states = s.integrationRegistry.Resolve(ctx)
	}
	return integrationSnapshot{GeneratedAt: time.Now().UTC(), States: states}
}

func (s *Server) registerIntegrationRoutes() {
	s.router.HandleFunc("/api/v1/integrations", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.integrationSnapshot(r.Context(), false))
	}).Methods("GET")
	s.router.HandleFunc("/api/v1/integrations/refresh", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.integrationSnapshot(r.Context(), true))
	}).Methods("POST")
	s.router.HandleFunc("/api/v1/integrations/{id}", s.handleIntegration).Methods("GET")
	s.router.HandleFunc("/api/v1/integrations/{id}/features/{feature}", s.handleIntegrationFeature).Methods("GET")
	s.router.HandleFunc("/api/v1/integrations/{id}/action", s.handleIntegrationAction).Methods("POST")
}

func (s *Server) handleIntegrationFeature(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, feature := strings.TrimSpace(vars["id"]), strings.TrimSpace(vars["feature"])
	for _, state := range s.integrationSnapshot(r.Context(), false).States {
		if state.ID != id {
			continue
		}
		declared := false
		for _, candidate := range state.Features {
			if candidate == feature {
				declared = true
				break
			}
		}
		status := "unsupported"
		reason := "feature is not declared by this integration"
		if declared {
			status = "unknown"
			reason = "feature is declared but has no producer compatibility evidence"
			if v := state.FeatureStatus[feature]; v != "" {
				status = v
				reason = state.FeatureReason[feature]
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"integrationId": id, "featureId": feature, "lifecycleStatus": state.Status, "status": status, "reason": reason, "checkedAt": state.CheckedAt})
		return
	}
	writeError(w, http.StatusNotFound, "integration_not_found", "Unknown integration: "+id, nil)
}

func (s *Server) handleIntegration(w http.ResponseWriter, r *http.Request) {
	id := muxVars(r, "id")
	snap := s.integrationSnapshot(r.Context(), false)
	for _, state := range snap.States {
		if state.ID == id {
			writeJSON(w, http.StatusOK, state)
			return
		}
	}
	writeError(w, http.StatusNotFound, "integration_not_found", "Unknown integration: "+id, nil)
}

func (s *Server) handleIntegrationAction(w http.ResponseWriter, r *http.Request) {
	id := muxVars(r, "id")
	var req struct {
		Action  string `json:"action"`
		Confirm bool   `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid_request", "Expected JSON action request", nil)
		return
	}
	for _, state := range s.integrationSnapshot(r.Context(), false).States {
		if state.ID == id {
			if !req.Confirm {
				writeError(w, 400, "confirmation_required", "Recovery actions require explicit confirmation", nil)
				return
			}
			if req.Action != string(state.ActionKind) {
				writeError(w, 400, "action_not_allowed", "Requested action is not eligible for this integration", nil)
				return
			}
			if state.ActionKind == capreg.ActionKindScenarioStart || state.ActionKind == capreg.ActionKindScenarioRestart {
				result, err := s.actionService.Run(r.Context(), capreg.LifecycleActionRequest{IntegrationID: state.ID, ActionKind: state.ActionKind})
				if err != nil {
					writeError(w, http.StatusBadRequest, "action_failed", err.Error(), nil)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"integrationId": id, "action": req.Action, "status": result.Status, "success": result.Success, "message": result.Message})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"integrationId": id, "action": req.Action, "status": "owner_guidance", "message": state.OperatorCommand})
			return
		}
	}
	writeError(w, 404, "integration_not_found", "Unknown integration: "+id, nil)
}

func muxVars(r *http.Request, key string) string { return strings.TrimSpace(mux.Vars(r)[key]) }
