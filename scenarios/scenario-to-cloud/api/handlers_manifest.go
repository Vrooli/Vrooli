package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
)

// ManifestInitRequest describes API input for creating a starter cloud manifest.
type ManifestInitRequest struct {
	ScenarioID string `json:"scenario_id,omitempty"`
	Host       string `json:"host,omitempty"`
	Domain     string `json:"domain,omitempty"`
	User       string `json:"user,omitempty"`
	Port       int    `json:"port,omitempty"`
	KeyPath    string `json:"key_path,omitempty"`
	Workdir    string `json:"workdir,omitempty"`
	CaddyEmail string `json:"caddy_email,omitempty"`
}

// ManifestEnvelopeRequest wraps a manifest payload for doctor/fix operations.
type ManifestEnvelopeRequest struct {
	Manifest map[string]interface{} `json:"manifest"`
}

func (s *Server) handleManifestSchema(w http.ResponseWriter, r *http.Request) {
	var schema map[string]interface{}
	if err := json.Unmarshal(manifest.SchemaJSON(), &schema); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "schema_parse_failed",
			Message: "Failed to parse manifest schema",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"schema":    schema,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleManifestTemplate(w http.ResponseWriter, r *http.Request) {
	variant := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("variant")))
	if variant == "" {
		variant = "minimal"
	}

	if variant != "minimal" && variant != "full" {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_variant",
			Message: "variant must be 'minimal' or 'full'",
		})
		return
	}

	manifestTemplate := defaultTemplateManifest()
	if variant == "full" {
		manifestTemplate = fullTemplateManifest()
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"variant":   variant,
		"manifest":  manifestTemplate,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleManifestInit(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[ManifestInitRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	built, source, err := s.buildInitManifest(r.Context(), req)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "manifest_init_failed",
			Message: "Failed to initialize manifest",
			Hint:    err.Error(),
		})
		return
	}

	raw, err := toRawMap(built)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "manifest_encode_failed",
			Message: "Failed to encode initialized manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, issues, err := manifest.ValidateRaw(raw)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "manifest_validate_failed",
			Message: "Failed to validate initialized manifest",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"manifest":  normalized,
		"issues":    issues,
		"source":    source,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleManifestDoctor(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[ManifestEnvelopeRequest](r.Body, 2<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if req.Manifest == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_manifest",
			Message: "manifest field is required",
		})
		return
	}

	normalized, issues, err := manifest.ValidateRaw(req.Manifest)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "manifest_doctor_failed",
			Message: "Failed to analyze manifest",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"valid":       !manifest.HasBlockingIssues(issues),
		"issues":      issues,
		"manifest":    normalized,
		"can_fix":     len(issues) > 0,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"schema_hint": "Use /api/v1/manifest/schema for the canonical structural contract.",
	})
}

func (s *Server) handleManifestFix(w http.ResponseWriter, r *http.Request) {
	req, err := httputil.DecodeJSON[ManifestEnvelopeRequest](r.Body, 2<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	if req.Manifest == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "missing_manifest",
			Message: "manifest field is required",
		})
		return
	}

	normalized, issues, err := manifest.ValidateRaw(req.Manifest)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "manifest_fix_failed",
			Message: "Failed to fix manifest",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"valid":       !manifest.HasBlockingIssues(issues),
		"issues":      issues,
		"manifest":    normalized,
		"applied":     true,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"schema_hint": "Use /api/v1/manifest/schema for the canonical structural contract.",
	})
}

func (s *Server) buildInitManifest(ctx context.Context, req ManifestInitRequest) (domain.CloudManifest, string, error) {
	m := defaultTemplateManifest()

	if strings.TrimSpace(req.ScenarioID) != "" {
		m.Scenario.ID = strings.TrimSpace(req.ScenarioID)
		m.Dependencies.Scenarios = []string{m.Scenario.ID}
		m.Bundle.Scenarios = []string{m.Scenario.ID, "vrooli-autoheal"}
	}
	if strings.TrimSpace(req.Host) != "" {
		m.Target.VPS.Host = strings.TrimSpace(req.Host)
	}
	if strings.TrimSpace(req.Domain) != "" {
		m.Edge.Domain = strings.TrimSpace(req.Domain)
	}
	if strings.TrimSpace(req.User) != "" {
		m.Target.VPS.User = strings.TrimSpace(req.User)
	}
	if req.Port > 0 {
		m.Target.VPS.Port = req.Port
	}
	if strings.TrimSpace(req.KeyPath) != "" {
		m.Target.VPS.KeyPath = strings.TrimSpace(req.KeyPath)
	}
	if strings.TrimSpace(req.Workdir) != "" {
		m.Target.VPS.Workdir = strings.TrimSpace(req.Workdir)
	}
	if strings.TrimSpace(req.CaddyEmail) != "" {
		m.Edge.Caddy.Email = strings.TrimSpace(req.CaddyEmail)
	}

	source := "template"
	if strings.TrimSpace(m.Scenario.ID) == "" {
		return m, source, nil
	}

	// Pull dependencies from analyzer/service.json fallback.
	if deps, err := s.fetchDependenciesWithFallback(ctx, m.Scenario.ID); err == nil {
		source = deps.Source
		m.Dependencies.Resources = deps.Resources
		m.Dependencies.Scenarios = deps.Scenarios
		m.Bundle.Resources = append([]string(nil), deps.Resources...)
		m.Bundle.Scenarios = append([]string(nil), deps.Scenarios...)
		if !manifest.Contains(m.Bundle.Scenarios, "vrooli-autoheal") {
			m.Bundle.Scenarios = append(m.Bundle.Scenarios, "vrooli-autoheal")
		}
	} else {
		s.log("manifest init dependency fetch failed", map[string]interface{}{
			"scenario_id": m.Scenario.ID,
			"error":       err.Error(),
		})
	}

	if ports, err := loadScenarioPortMap(m.Scenario.ID); err == nil && len(ports) > 0 {
		m.Ports = ports
	} else if err != nil {
		s.log("manifest init port fetch failed", map[string]interface{}{
			"scenario_id": m.Scenario.ID,
			"error":       err.Error(),
		})
	}

	return m, source, nil
}

type dependencySnapshot struct {
	Resources []string
	Scenarios []string
	Source    string
}

func (s *Server) fetchDependenciesWithFallback(ctx context.Context, scenarioID string) (dependencySnapshot, error) {
	if deps, err := s.fetchDependenciesFromAnalyzer(ctx, scenarioID); err == nil {
		resources := append([]string(nil), deps.Resources...)
		scenarios := append([]string(nil), deps.Scenarios...)
		if !manifest.Contains(scenarios, scenarioID) {
			scenarios = append(scenarios, scenarioID)
		}
		scenarios = manifest.StableUniqueStrings(scenarios)
		resources = manifest.StableUniqueStrings(resources)
		return dependencySnapshot{Resources: resources, Scenarios: scenarios, Source: deps.Source}, nil
	}

	deps, err := s.extractDependenciesFromServiceJSON(scenarioID)
	if err != nil {
		return dependencySnapshot{}, err
	}
	resources := manifest.StableUniqueStrings(deps.Resources)
	scenarios := manifest.StableUniqueStrings(append(deps.Scenarios, scenarioID))
	return dependencySnapshot{Resources: resources, Scenarios: scenarios, Source: deps.Source}, nil
}

func loadScenarioPortMap(scenarioID string) (map[string]int, error) {
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		return nil, err
	}
	path, err := bundle.ResolveScenarioFile(repoRoot, scenarioID, "service")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var svc ServiceJSON
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, err
	}

	ports := make(map[string]int, len(svc.Ports))
	for name, cfg := range svc.Ports {
		if cfg.Port <= 0 {
			continue
		}
		normalizedName := name
		if name == "websocket" {
			normalizedName = "ws"
		}
		ports[normalizedName] = cfg.Port
	}
	return ports, nil
}

func defaultTemplateManifest() domain.CloudManifest {
	return domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host: "203.0.113.10",
				User: "root",
				Port: 22,
			},
		},
		Scenario: domain.ManifestScenario{ID: ""},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{},
			Resources: []string{},
		},
		Bundle: domain.ManifestBundle{
			IncludePackages: true,
			IncludeAutoheal: true,
			Scenarios:       []string{"vrooli-autoheal"},
			Resources:       []string{},
		},
		Ports: domain.ManifestPorts{
			"ui":  3000,
			"api": 3001,
			"ws":  3002,
		},
		Edge: domain.ManifestEdge{
			Domain:    "example.com",
			DNSPolicy: domain.DNSPolicyRequired,
			Caddy: domain.ManifestCaddy{
				Enabled: true,
				Email:   "",
			},
		},
	}
}

func fullTemplateManifest() domain.CloudManifest {
	m := defaultTemplateManifest()
	m.Target.VPS.KeyPath = "~/.ssh/id_ed25519"
	m.Target.VPS.Workdir = domain.DefaultVPSWorkdir
	m.Target.VPS.PreservePaths = []string{}
	m.Dependencies.Analyzer.Tool = "scenario-dependency-analyzer"
	m.Dependencies.Analyzer.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	m.Scenario.Ref = "main"

	// Ensure deterministic order in output.
	m.Bundle.Scenarios = manifest.StableUniqueStrings(m.Bundle.Scenarios)
	m.Dependencies.Scenarios = manifest.StableUniqueStrings(m.Dependencies.Scenarios)
	m.Dependencies.Resources = manifest.StableUniqueStrings(m.Dependencies.Resources)

	return m
}

func toRawMap(m domain.CloudManifest) (map[string]interface{}, error) {
	blob, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(blob, &out); err != nil {
		return nil, err
	}
	return out, nil
}
