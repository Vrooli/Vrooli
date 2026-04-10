package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"tunnel-manager/domain"
)

// PortAuditor checks that published routes match scenario service.json port fields.
type PortAuditor struct {
	routeLister   RouteLister
	scenariosRoot string // path to the scenarios directory
}

func NewPortAuditor(routeLister RouteLister, scenariosRoot string) *PortAuditor {
	return &PortAuditor{
		routeLister:   routeLister,
		scenariosRoot: scenariosRoot,
	}
}

// Audit checks all enabled routes for port compliance with their scenario service.json.
func (pa *PortAuditor) Audit() ([]domain.PortAuditResult, error) {
	routes, err := pa.routeLister.List()
	if err != nil {
		return nil, fmt.Errorf("port audit: %w", err)
	}

	var results []domain.PortAuditResult
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		result := pa.auditRoute(route)
		results = append(results, result)
	}
	return results, nil
}

// newAuditResult creates a base PortAuditResult from a route with common fields pre-filled.
func newAuditResult(route domain.Route, status, detail string) domain.PortAuditResult {
	return domain.PortAuditResult{
		Subdomain:    route.Subdomain,
		ScenarioName: route.ScenarioName,
		ExpectedPort: route.LocalPort,
		Status:       status,
		Detail:       detail,
	}
}

func (pa *PortAuditor) auditRoute(route domain.Route) domain.PortAuditResult {
	svcPath := filepath.Join(pa.scenariosRoot, route.ScenarioName, ".vrooli", "service.json")
	data, err := os.ReadFile(svcPath)
	if err != nil {
		return newAuditResult(route, "missing_scenario", fmt.Sprintf("service.json not found: %s", svcPath))
	}

	var svc struct {
		Ports map[string]struct {
			Port   int    `json:"port"`
			EnvVar string `json:"env_var"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(data, &svc); err != nil {
		return newAuditResult(route, "missing_port", fmt.Sprintf("failed to parse service.json: %v", err))
	}

	// Look for UI port (tunnel routes typically point to UI)
	uiPort, hasUI := svc.Ports["ui"]
	if !hasUI || uiPort.Port == 0 {
		return newAuditResult(route, "missing_port", "no fixed UI port defined in service.json")
	}

	if uiPort.Port != route.LocalPort {
		r := newAuditResult(route, "mismatch", fmt.Sprintf("manifest expects port %d but service.json has %d", route.LocalPort, uiPort.Port))
		r.ActualPort = uiPort.Port
		return r
	}

	r := newAuditResult(route, "compliant", "")
	r.ActualPort = uiPort.Port
	return r
}
