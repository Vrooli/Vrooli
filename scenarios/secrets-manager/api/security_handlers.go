package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type SecurityHandlers struct {
	db     *sql.DB
	logger *Logger
}

func NewSecurityHandlers(db *sql.DB, logger *Logger) *SecurityHandlers {
	return &SecurityHandlers{db: db, logger: logger}
}

// RegisterRoutes mounts the security endpoints under /security.
func (h *SecurityHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/scan", h.SecurityScan).Methods("GET")
	router.HandleFunc("/compliance", h.Compliance).Methods("GET")
	router.HandleFunc("/vulnerabilities", h.Vulnerabilities).Methods("GET")
	router.HandleFunc("/vulnerabilities/fix", h.FixVulnerabilities).Methods("POST")
	router.HandleFunc("/vulnerabilities/fix/progress", h.FixProgress).Methods("POST")
	router.HandleFunc("/vulnerabilities/{id}/status", h.VulnerabilityStatus).Methods("POST")
}

// RegisterLegacyRoutes covers legacy paths that live at the API root rather than under /security.
func (h *SecurityHandlers) RegisterLegacyRoutes(router *mux.Router) {
	router.HandleFunc("/vulnerabilities", h.Vulnerabilities).Methods("GET")
	router.HandleFunc("/vulnerabilities/fix", h.FixVulnerabilities).Methods("POST")
	router.HandleFunc("/vulnerabilities/fix/progress", h.FixProgress).Methods("POST")
	router.HandleFunc("/files/content", h.FileContent).Methods("GET")
}

func (h *SecurityHandlers) SecurityScan(w http.ResponseWriter, r *http.Request) {
	componentFilter := r.URL.Query().Get("component")
	componentTypeFilter := r.URL.Query().Get("component_type")
	severityFilter := r.URL.Query().Get("severity")

	// Support legacy scenario parameter
	if scenarioFilter := r.URL.Query().Get("scenario"); scenarioFilter != "" && componentFilter == "" {
		componentFilter = scenarioFilter
		componentTypeFilter = "scenario"
	}

	result, err := scanComponentsForVulnerabilities(componentFilter, componentTypeFilter, severityFilter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Security scan failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Compliance dashboard handler
func (h *SecurityHandlers) Compliance(w http.ResponseWriter, r *http.Request) {
	// Get vault secrets status
	vaultStatus, err := getVaultSecretsStatus("")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get vault status: %v", err), http.StatusInternalServerError)
		return
	}

	// Get security scan results for all components
	securityResults, err := scanComponentsForVulnerabilities("", "", "")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan for vulnerabilities: %v", err), http.StatusInternalServerError)
		return
	}

	compliance := calculateComplianceMetrics(vaultStatus, securityResults)
	response := buildComplianceResponse(compliance, vaultStatus, securityResults)

	writeJSON(w, http.StatusOK, response)
}

// NOTE: Compliance domain functions (calculateComplianceMetrics, calculatePercentage,
// tallyVulnerabilitySeverities, buildComplianceResponse) have been moved to
// compliance_domain.go to separate domain logic from HTTP handlers.

func (h *SecurityHandlers) Vulnerabilities(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	severity := r.URL.Query().Get("severity")
	component := r.URL.Query().Get("component")
	componentType := r.URL.Query().Get("component_type")
	quickMode := r.URL.Query().Get("quick") == "true"

	// Support legacy scenario parameter
	if scenario := r.URL.Query().Get("scenario"); scenario != "" && component == "" {
		component = scenario
		componentType = "scenario"
	}

	// In quick mode or test mode, return minimal results
	if quickMode || os.Getenv("SECRETS_MANAGER_TEST_MODE") == "true" {
		quickResults := buildQuickSecurityScanResult()
		response := buildVulnerabilityPayload(quickResults, component, severity, "quick")
		writeJSON(w, http.StatusOK, response)
		return
	}

	// Get security scan results from real filesystem scan
	securityResults, err := scanComponentsForVulnerabilities(component, componentType, severity)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan for vulnerabilities: %v", err), http.StatusInternalServerError)
		return
	}

	// [REQ:SEC-SCAN-002] Vulnerability listing endpoint
	response := buildVulnerabilityPayload(securityResults, component, severity, "")
	writeJSON(w, http.StatusOK, response)
}

func buildQuickSecurityScanResult() *SecurityScanResult {
	return &SecurityScanResult{
		ScanID:          uuid.New().String(),
		Vulnerabilities: []SecurityVulnerability{},
		RiskScore:       0,
		ScanDurationMs:  1,
		Recommendations: generateRemediationSuggestions([]SecurityVulnerability{}),
	}
}

func buildVulnerabilityPayload(result *SecurityScanResult, component, severity, mode string) map[string]interface{} {
	if result == nil {
		result = &SecurityScanResult{}
	}

	return map[string]interface{}{
		"vulnerabilities": result.Vulnerabilities,
		"total_count":     len(result.Vulnerabilities),
		"scan_metadata":   buildScanMetadata(result, component, severity, mode),
		"recommendations": result.Recommendations,
	}
}

func buildScanMetadata(result *SecurityScanResult, component, severity, mode string) map[string]interface{} {
	if result == nil {
		result = &SecurityScanResult{}
	}

	metadata := map[string]interface{}{
		"scan_id":       result.ScanID,
		"scan_duration": result.ScanDurationMs,
		"risk_score":    result.RiskScore,
		"component":     component,
		"severity":      severity,
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	if mode != "" {
		metadata["mode"] = mode
	}

	return metadata
}

type vulnerabilityStatusRequest struct {
	Status     string `json:"status"`
	AssignedTo string `json:"assigned_to"`
}

func (h *SecurityHandlers) VulnerabilityStatus(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	vars := mux.Vars(r)
	vulnID := vars["id"]
	var req vulnerabilityStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}
	if _, ok := allowedVulnerabilityStatuses[status]; !ok {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}
	assigned := strings.TrimSpace(req.AssignedTo)
	query := `
		UPDATE security_vulnerabilities
		SET status = $1,
		    assigned_to = CASE WHEN $2 = '' THEN assigned_to ELSE $2 END,
		    resolved_at = CASE WHEN $1 = 'resolved' THEN CURRENT_TIMESTAMP ELSE resolved_at END
		WHERE id = $3
		RETURNING id
	`
	var updatedID string
	if err := h.db.QueryRowContext(r.Context(), query, status, assigned, vulnID).Scan(&updatedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "vulnerability not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to update vulnerability: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"id": updatedID, "status": status})
}
