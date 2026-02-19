package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// PortAuditResult describes the compliance status of a single route.
type PortAuditResult struct {
	Subdomain    string `json:"subdomain"`
	ScenarioName string `json:"scenario_name"`
	ExpectedPort int    `json:"expected_port"`
	ActualPort   int    `json:"actual_port,omitempty"`
	Status       string `json:"status"` // "compliant", "mismatch", "missing_scenario", "missing_port"
	Detail       string `json:"detail,omitempty"`
}

// PortAuditor checks that published routes match scenario service.json port fields.
type PortAuditor struct {
	routeSvc      *RouteService
	scenariosRoot string // path to the scenarios directory
}

func NewPortAuditor(routeSvc *RouteService, scenariosRoot string) *PortAuditor {
	return &PortAuditor{
		routeSvc:      routeSvc,
		scenariosRoot: scenariosRoot,
	}
}

// Audit checks all enabled routes for port compliance with their scenario service.json.
func (pa *PortAuditor) Audit() ([]PortAuditResult, error) {
	routes, err := pa.routeSvc.List()
	if err != nil {
		return nil, fmt.Errorf("port audit: %w", err)
	}

	var results []PortAuditResult
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		result := pa.auditRoute(route)
		results = append(results, result)
	}
	return results, nil
}

func (pa *PortAuditor) auditRoute(route Route) PortAuditResult {
	svcPath := filepath.Join(pa.scenariosRoot, route.ScenarioName, ".vrooli", "service.json")
	data, err := os.ReadFile(svcPath)
	if err != nil {
		return PortAuditResult{
			Subdomain:    route.Subdomain,
			ScenarioName: route.ScenarioName,
			ExpectedPort: route.LocalPort,
			Status:       "missing_scenario",
			Detail:       fmt.Sprintf("service.json not found: %s", svcPath),
		}
	}

	var svc struct {
		Ports map[string]struct {
			Port   int    `json:"port"`
			EnvVar string `json:"env_var"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(data, &svc); err != nil {
		return PortAuditResult{
			Subdomain:    route.Subdomain,
			ScenarioName: route.ScenarioName,
			ExpectedPort: route.LocalPort,
			Status:       "missing_port",
			Detail:       fmt.Sprintf("failed to parse service.json: %v", err),
		}
	}

	// Look for UI port (tunnel routes typically point to UI)
	uiPort, hasUI := svc.Ports["ui"]
	if !hasUI || uiPort.Port == 0 {
		return PortAuditResult{
			Subdomain:    route.Subdomain,
			ScenarioName: route.ScenarioName,
			ExpectedPort: route.LocalPort,
			Status:       "missing_port",
			Detail:       "no fixed UI port defined in service.json",
		}
	}

	if uiPort.Port != route.LocalPort {
		return PortAuditResult{
			Subdomain:    route.Subdomain,
			ScenarioName: route.ScenarioName,
			ExpectedPort: route.LocalPort,
			ActualPort:   uiPort.Port,
			Status:       "mismatch",
			Detail:       fmt.Sprintf("manifest expects port %d but service.json has %d", route.LocalPort, uiPort.Port),
		}
	}

	return PortAuditResult{
		Subdomain:    route.Subdomain,
		ScenarioName: route.ScenarioName,
		ExpectedPort: route.LocalPort,
		ActualPort:   uiPort.Port,
		Status:       "compliant",
	}
}

// --- HTTP Handler ---

func handlePortAudit(auditor *PortAuditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results, err := auditor.Audit()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []PortAuditResult{}
		}

		// Compute summary
		violations := 0
		for _, r := range results {
			if r.Status != "compliant" {
				violations++
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"results":    results,
			"total":      len(results),
			"violations": violations,
			"compliant":  len(results) - violations,
		})
	}
}
