package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	internalroutes "tunnel-manager/internal/routes"
)

// RoutesReader is the narrow seam the audit service depends on to read the
// exposure manifest. The production wiring satisfies it with
// internalroutes.Service; service unit tests wire a fake. Locally declared
// (not imported from routes) so the audit domain depends only on the one
// method it actually uses.
type RoutesReader interface {
	// List returns manifest routes, optionally filtered by tier. The audit
	// service passes the empty Tier to read every route.
	List(ctx context.Context, tier internalroutes.Tier) ([]internalroutes.Route, error)
}

// Service is the application-layer surface the audit handlers depend on. It
// owns the compliance classification; the handler is intentionally thin around
// it: decode → call service → translate errors.
type Service interface {
	// RunAudit computes a port-compliance finding for every ENABLED manifest
	// route by comparing the route's expected local_port against the UI port
	// declared in the scenario's service.json. Returns one PortAuditResult per
	// enabled route. An error is returned only when the manifest itself cannot
	// be read; per-route problems (missing scenario, missing port) are
	// reported as findings, not errors.
	RunAudit(ctx context.Context) ([]PortAuditResult, error)
}

type service struct {
	routes        RoutesReader
	scenariosRoot string
}

// NewService constructs the production Service. scenariosRoot is the path to
// the directory holding scenario folders (each with a .vrooli/service.json);
// it is injectable so tests can point it at a temp tree.
func NewService(routes RoutesReader, scenariosRoot string) Service {
	return &service{routes: routes, scenariosRoot: scenariosRoot}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) RunAudit(ctx context.Context) ([]PortAuditResult, error) {
	routes, err := s.routes.List(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("port audit: list routes: %w", err)
	}

	results := make([]PortAuditResult, 0, len(routes))
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		results = append(results, s.auditRoute(route))
	}
	return results, nil
}

// newResult builds a base PortAuditResult from a route with the common fields
// pre-filled.
func newResult(route internalroutes.Route, status AuditStatus, detail string) PortAuditResult {
	return PortAuditResult{
		Subdomain:    route.Subdomain,
		Scenario:     route.Scenario,
		ExpectedPort: route.LocalPort,
		Status:       status,
		Detail:       detail,
	}
}

// auditRoute classifies one route against its scenario's service.json. The
// real service.json shape is `{"ports": {"ui": {"port": <int>, "env_var":
// "...", "range": "..."}}}`; a fixed UI port lives at ports.ui.port and is 0
// (absent) when the scenario uses a ranged/dynamic port instead.
func (s *service) auditRoute(route internalroutes.Route) PortAuditResult {
	svcPath := filepath.Join(s.scenariosRoot, route.Scenario, ".vrooli", "service.json")
	data, err := os.ReadFile(svcPath)
	if err != nil {
		return newResult(route, StatusMissingScenario, fmt.Sprintf("service.json not found: %s", svcPath))
	}

	var svc struct {
		Ports map[string]struct {
			Port   int    `json:"port"`
			EnvVar string `json:"env_var"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(data, &svc); err != nil {
		return newResult(route, StatusMissingPort, fmt.Sprintf("failed to parse service.json: %v", err))
	}

	// Tunnel routes point at the UI surface, so the fixed UI port is the
	// compliance anchor.
	uiPort, hasUI := svc.Ports["ui"]
	if !hasUI || uiPort.Port == 0 {
		return newResult(route, StatusMissingPort, "no fixed UI port defined in service.json")
	}

	if uiPort.Port != route.LocalPort {
		r := newResult(route, StatusMismatch, fmt.Sprintf("manifest expects port %d but service.json has %d", route.LocalPort, uiPort.Port))
		r.ActualPort = uiPort.Port
		return r
	}

	r := newResult(route, StatusCompliant, "")
	r.ActualPort = uiPort.Port
	return r
}

// ViolationCount returns the number of results whose status is not compliant.
// Provided as a domain helper so both the handler and CLI compute the count
// the same way.
func ViolationCount(results []PortAuditResult) int {
	n := 0
	for _, r := range results {
		if r.Status != StatusCompliant {
			n++
		}
	}
	return n
}
