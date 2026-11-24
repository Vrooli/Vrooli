package domains

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"architecture-cartographer/internal/zones"
)

const (
	classifyZonesPath       = "/api/v1/domains/classify-zones"
	maxZoneRequestBytes     = 16 << 10
	maxZonePaths            = 128
	maxZonePathBytes        = 512
	maxZoneScenarioSlugSize = 80
)

var scenarioSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type classifyZonesRequest struct {
	Scenario string   `json:"scenario"`
	Paths    []string `json:"paths"`
}

type classifyZonesResponse struct {
	Scenario     string               `json:"scenario"`
	Results      []zoneClassification `json:"results"`
	Degradations []zoneDegradation    `json:"degradations"`
}

type zoneClassification struct {
	Path           string `json:"path"`
	Zone           string `json:"zone,omitempty"`
	Domain         string `json:"domain,omitempty"`
	Archetype      string `json:"archetype,omitempty"`
	DomainDeclared bool   `json:"domain_declared"`
	Basis          string `json:"basis"`
	Status         string `json:"status"`
}

type zoneDegradation struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

// ClassifyZones classifies a bounded caller-selected set of existing paths
// using the scenario's derived domain map and template zone rules. It never
// scans or returns the scenario's file inventory.
func (h *Handler) ClassifyZones(w http.ResponseWriter, r *http.Request) {
	if h.locator == nil {
		writeZoneError(w, http.StatusServiceUnavailable, "scenario_locator_unavailable")
		return
	}
	var req classifyZonesRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxZoneRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			writeZoneError(w, http.StatusRequestEntityTooLarge, "request_too_large")
		} else {
			writeZoneError(w, http.StatusBadRequest, "invalid_request")
		}
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeZoneError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	req.Scenario = strings.TrimSpace(req.Scenario)
	if len(req.Scenario) > maxZoneScenarioSlugSize || (req.Scenario != "control-plane" && !scenarioSlugPattern.MatchString(req.Scenario)) {
		writeZoneError(w, http.StatusBadRequest, "invalid_scenario")
		return
	}
	if len(req.Paths) == 0 || len(req.Paths) > maxZonePaths {
		writeZoneError(w, http.StatusBadRequest, "invalid_path_count")
		return
	}

	scenarioDir, err := h.locator.Locate(req.Scenario)
	if err != nil {
		writeZoneError(w, http.StatusNotFound, "scenario_not_found")
		return
	}
	resolvedRoot, err := filepath.EvalSymlinks(scenarioDir)
	if err != nil {
		writeZoneError(w, http.StatusNotFound, "scenario_not_found")
		return
	}
	domainMap, err := h.svc.GetDomainMap(r.Context(), req.Scenario)
	if err != nil {
		writeZoneError(w, http.StatusInternalServerError, "domain_map_unavailable")
		return
	}
	zoneConfig := zones.LoadForScenario(resolvedRoot)
	response := classifyZonesResponse{Scenario: req.Scenario, Results: make([]zoneClassification, 0, len(req.Paths)), Degradations: []zoneDegradation{}}
	unknownCount := 0
	for _, requestedPath := range req.Paths {
		relativePath, ok := existingScenarioPath(resolvedRoot, requestedPath)
		if !ok {
			writeZoneError(w, http.StatusBadRequest, "path_outside_or_missing")
			return
		}
		info := zoneConfig.Classify(relativePath, domainMap)
		status := "KNOWN"
		if info.Zone == zones.Unknown {
			status = "UNKNOWN"
			unknownCount++
		}
		response.Results = append(response.Results, zoneClassification{
			Path: relativePath, Zone: info.Zone, Domain: info.Domain, Archetype: info.Archetype,
			DomainDeclared: info.Domain != "" && info.Declared, Basis: info.Basis, Status: status,
		})
	}
	if unknownCount > 0 {
		response.Degradations = append(response.Degradations, zoneDegradation{Code: "unclassified_paths", Count: unknownCount})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func existingScenarioPath(scenarioRoot, requested string) (string, bool) {
	requested = strings.TrimSpace(requested)
	if requested == "" || len(requested) > maxZonePathBytes || strings.ContainsRune(requested, '\x00') || strings.Contains(requested, "\\") {
		return "", false
	}
	clean := path.Clean(requested)
	if clean == "." || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") || (requested != clean && requested != clean+"/") {
		return "", false
	}
	absolute := filepath.Join(scenarioRoot, filepath.FromSlash(clean))
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil || filepath.Clean(resolved) != filepath.Clean(absolute) {
		return "", false
	}
	relative, err := filepath.Rel(scenarioRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", false
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", false
	}
	return filepath.ToSlash(relative), true
}

func writeZoneError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code}})
}
