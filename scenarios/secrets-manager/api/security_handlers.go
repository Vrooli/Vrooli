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

func (s *APIServer) securityScanHandler(w http.ResponseWriter, r *http.Request) {
	s.handlers.security.SecurityScan(w, r)
}

func (s *APIServer) complianceHandler(w http.ResponseWriter, r *http.Request) {
	s.handlers.security.Compliance(w, r)
}

func (s *APIServer) vulnerabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	s.handlers.security.Vulnerabilities(w, r)
}

func (s *APIServer) vulnerabilityStatusHandler(w http.ResponseWriter, r *http.Request) {
	s.handlers.security.VulnerabilityStatus(w, r)
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

type vulnerabilitySeverityCounts struct {
	critical int
	high     int
	medium   int
	low      int
}

func calculateComplianceMetrics(vaultStatus *VaultSecretsStatus, securityResults *SecurityScanResult) ComplianceMetrics {
	vault := VaultSecretsStatus{}
	if vaultStatus != nil {
		vault = *vaultStatus
	}

	summary := ComponentScanSummary{}
	vulnerabilities := []SecurityVulnerability{}
	riskScore := 0

	if securityResults != nil {
		summary = securityResults.ComponentsSummary
		vulnerabilities = securityResults.Vulnerabilities
		riskScore = securityResults.RiskScore
	}

	vaultHealth := calculatePercentage(vault.ConfiguredResources, vault.TotalResources)
	securityScore := 100 - riskScore
	if securityScore < 0 {
		securityScore = 0
	}
	overallCompliance := (vaultHealth + securityScore) / 2

	severityCounts := tallyVulnerabilitySeverities(vulnerabilities)
	configuredComponents := vault.ConfiguredResources
	if summary.TotalComponents > 0 {
		configuredComponents += summary.ConfiguredCount
	}

	return ComplianceMetrics{
		VaultSecretsHealth:   vaultHealth,
		SecurityScore:        securityScore,
		OverallCompliance:    overallCompliance,
		ConfiguredComponents: configuredComponents,
		CriticalIssues:       severityCounts.critical,
		HighIssues:           severityCounts.high,
		MediumIssues:         severityCounts.medium,
		LowIssues:            severityCounts.low,
	}
}

func calculatePercentage(part, total int) int {
	if total == 0 {
		return 0
	}
	return (part * 100) / total
}

func tallyVulnerabilitySeverities(vulnerabilities []SecurityVulnerability) vulnerabilitySeverityCounts {
	counts := vulnerabilitySeverityCounts{}
	for _, vuln := range vulnerabilities {
		switch vuln.Severity {
		case "critical":
			counts.critical++
		case "high":
			counts.high++
		case "medium":
			counts.medium++
		case "low":
			counts.low++
		}
	}
	return counts
}

func buildComplianceResponse(metrics ComplianceMetrics, vaultStatus *VaultSecretsStatus, securityResults *SecurityScanResult) map[string]interface{} {
	vault := VaultSecretsStatus{}
	if vaultStatus != nil {
		vault = *vaultStatus
	}

	componentsSummary := ComponentScanSummary{}
	totalVulnerabilities := 0
	if securityResults != nil {
		componentsSummary = securityResults.ComponentsSummary
		totalVulnerabilities = len(securityResults.Vulnerabilities)
	}

	return map[string]interface{}{
		"overall_score":        metrics.OverallCompliance,
		"vault_secrets_health": metrics.VaultSecretsHealth,
		"vulnerability_summary": map[string]int{
			"critical": metrics.CriticalIssues,
			"high":     metrics.HighIssues,
			"medium":   metrics.MediumIssues,
			"low":      metrics.LowIssues,
		},
		"remediation_progress":  metrics,
		"total_resources":       vault.TotalResources,
		"configured_resources":  vault.ConfiguredResources,
		"configured_components": metrics.ConfiguredComponents,
		"total_components":      componentsSummary.TotalComponents,
		"components_summary":    componentsSummary,
		"total_vulnerabilities": totalVulnerabilities,
		"last_updated":          time.Now(),
	}
}

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
