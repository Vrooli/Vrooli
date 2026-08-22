package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/apihttptest"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"

	"github.com/gorilla/mux"
)

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Tick with Auto-Heal Tests
// =============================================================================

func TestTick_WithAutoHeal(t *testing.T) {
	store := &mockStore{}
	h, criticalCheck := setupTestHandlersWithAutoHeal(store)

	req := httptest.NewRequest("POST", "/api/v1/tick?force=true", nil)
	w := httptest.NewRecorder()

	h.Tick(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Tick() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have auto-heal results
	autoHeal, ok := resp["autoHeal"].([]interface{})
	if !ok {
		t.Fatal("autoHeal field missing or invalid")
	}

	if len(autoHeal) == 0 {
		t.Error("autoHeal should have at least one result for critical check")
	}

	// Verify the safe action was executed (start, not restart)
	if criticalCheck.executeCalled {
		if criticalCheck.executeResult.ActionID != "start" {
			t.Errorf("Auto-heal should execute 'start' (non-dangerous), got %q", criticalCheck.executeResult.ActionID)
		}
	}
}

func TestTick_AutoHealSkippedWhenDisabled(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	registry := checks.NewRegistry(caps)
	_ = registry.SetAutoHealPolicy(checks.AutoHealPolicy{
		BaseCooldown:       5 * time.Minute,
		MaxRestartAttempts: 3,
	})

	// Create a critical check
	criticalCheck := &mockHealableCheckCritical{
		mockCheck: mockCheck{
			id:      "critical-check",
			status:  checks.StatusCritical,
			message: "Service down",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "start", Name: "Start", Available: true, Dangerous: false},
		},
		executeResult: checks.ActionResult{
			CheckID: "critical-check",
			Success: true,
		},
	}
	registry.Register(criticalCheck)

	// Disable auto-heal
	configProvider := &mockConfigProvider{
		enabledChecks:  map[string]bool{"critical-check": true},
		autoHealChecks: map[string]bool{"critical-check": false},
	}
	registry.SetConfigProvider(configProvider)

	store := &mockStore{}
	h := NewWithInterface(registry, store, caps)

	req := httptest.NewRequest("POST", "/api/v1/tick?force=true", nil)
	w := httptest.NewRecorder()

	h.Tick(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Tick() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	autoHeal, ok := resp["autoHeal"].([]interface{})
	if !ok {
		t.Fatal("autoHeal field missing")
	}

	// Should have auto-heal result with "not enabled" reason
	if len(autoHeal) > 0 {
		first := autoHeal[0].(map[string]interface{})
		if first["attempted"] == true {
			t.Error("Auto-heal should NOT be attempted when disabled")
		}
		if reason, ok := first["reason"].(string); ok {
			t.Logf("Auto-heal skipped: %s", reason)
		}
	}

	// Verify no action was executed
	if criticalCheck.executeCalled {
		t.Error("No action should be executed when auto-heal is disabled")
	}
}

func TestTick_AutoHealSkipsNonCritical(t *testing.T) {
	caps := &platform.Capabilities{
		Platform: platform.Linux,
	}

	registry := checks.NewRegistry(caps)
	_ = registry.SetAutoHealPolicy(checks.AutoHealPolicy{
		BaseCooldown:       5 * time.Minute,
		MaxRestartAttempts: 3,
	})

	// Create a warning (non-critical) check
	warningCheck := &mockHealableCheckCritical{
		mockCheck: mockCheck{
			id:      "warning-check",
			status:  checks.StatusWarning, // Not critical
			message: "Minor issue",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "fix", Name: "Fix", Available: true, Dangerous: false},
		},
	}
	registry.Register(warningCheck)

	configProvider := &mockConfigProvider{
		enabledChecks:  map[string]bool{"warning-check": true},
		autoHealChecks: map[string]bool{"warning-check": true},
	}
	registry.SetConfigProvider(configProvider)

	store := &mockStore{}
	h := NewWithInterface(registry, store, caps)

	req := httptest.NewRequest("POST", "/api/v1/tick?force=true", nil)
	w := httptest.NewRecorder()

	h.Tick(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Tick() status = %d, want %d", w.Code, http.StatusOK)
	}

	// Warning check should NOT trigger auto-heal
	if warningCheck.executeCalled {
		t.Error("Auto-heal should NOT execute for non-critical checks")
	}
}

func TestIncidentRemediationsListsCandidates(t *testing.T) {
	store := &mockStore{incident: testRemediationIncident()}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/incidents/inc_test/remediations", nil)
	w := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/incidents/{incidentId}/remediations", h.IncidentRemediations)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("IncidentRemediations() status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	resp := apihttptest.MustDecodeJSON[map[string]interface{}](t, w.Body.Bytes())
	if total, ok := resp["total"].(float64); !ok || total != 1 {
		t.Fatalf("total = %v, want 1", resp["total"])
	}
}

func TestGenerateIncidentRemediationPersistsArtifactReference(t *testing.T) {
	t.Setenv("VROOLI_STATE_ROOT", t.TempDir())
	store := &mockStore{incident: testRemediationIncident()}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("POST", "/api/v1/incidents/inc_test/remediations/ubuntu-nvidia-kernel-module-mismatch/generate", nil)
	w := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/incidents/{incidentId}/remediations/{remediationId}/generate", h.GenerateIncidentRemediation)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GenerateIncidentRemediation() status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if store.recordedArtifact == nil {
		t.Fatal("expected generated artifact to be recorded on the incident")
	}
	if store.recordedArtifact.RemediationID != "ubuntu-nvidia-kernel-module-mismatch" {
		t.Fatalf("recorded remediation id = %q", store.recordedArtifact.RemediationID)
	}
}

func TestRecordIncidentRemediationOutcome(t *testing.T) {
	store := &mockStore{incident: testRemediationIncident()}
	h := setupTestHandlers(store)

	body := bytes.NewBufferString(`{"status":"verified","note":"nvidia-smi is healthy"}`)
	req := httptest.NewRequest("POST", "/api/v1/incidents/inc_test/remediations/ubuntu-nvidia-kernel-module-mismatch/outcome", body)
	w := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/incidents/{incidentId}/remediations/{remediationId}/outcome", h.RecordIncidentRemediationOutcome)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RecordIncidentRemediationOutcome() status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if store.recordedOutcome == nil {
		t.Fatal("expected outcome to be recorded")
	}
	if store.recordedOutcome.Status != "verified" || !strings.Contains(store.recordedOutcome.Note, "nvidia-smi") {
		t.Fatalf("recorded outcome = %#v", store.recordedOutcome)
	}
}

func testRemediationIncident() *incidents.Incident {
	return &incidents.Incident{
		ID:          "inc_test",
		Fingerprint: "incfp_test",
		EvidenceItems: []incidents.EvidenceItem{{
			Kind: "missing_nvidia_module_package",
			Data: map[string]any{
				"expectedPackage": "linux-modules-nvidia-580-open-6.17.0-23-generic",
				"runningKernel":   "6.17.0-23-generic",
			},
		}},
		RemediationCandidates: []incidents.RemediationCandidate{{
			ID:                "ubuntu-nvidia-kernel-module-mismatch",
			Title:             "Install matching NVIDIA kernel module package",
			Applicability:     "applicable",
			RequiresOperator:  true,
			RequiresPrivilege: true,
			RiskLevel:         "moderate",
			TemplateID:        "ubuntu-nvidia-kernel-module-mismatch",
			PostChecks:        []string{"nvidia-smi"},
		}},
	}
}

// =============================================================================
// Docs Handler Tests (if DocsManifest/DocsContent exist)
// =============================================================================

func TestDocsManifest(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/docs/manifest", nil)
	w := httptest.NewRecorder()

	h.DocsManifest(w, req)

	// Should return OK or NotFound (depending on whether docs exist)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("DocsManifest() status = %d, want 200 or 404", w.Code)
	}
}

func TestDocsContent(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/docs/content?path=README.md", nil)
	w := httptest.NewRecorder()

	h.DocsContent(w, req)

	// Should return OK or NotFound (depending on whether docs exist)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("DocsContent() status = %d, want 200 or 404", w.Code)
	}
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

func TestGetCheckActions_NotHealable(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store) // Uses non-healable mockCheck

	req := httptest.NewRequest("GET", "/api/v1/checks/test-check/actions", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/actions", h.GetCheckActions)
	router.ServeHTTP(w, req)

	// Non-healable check should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("GetCheckActions() for non-healable check status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestExecuteCheckAction_Failed(t *testing.T) {
	caps := &platform.Capabilities{
		Platform: platform.Linux,
	}

	registry := checks.NewRegistry(caps)

	// Create a healable check that fails on action
	failingCheck := &mockHealableCheck{
		mockCheck: mockCheck{
			id:      "failing-check",
			status:  checks.StatusCritical,
			message: "Failed",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "restart", Name: "Restart", Available: true},
		},
		executeResult: checks.ActionResult{
			CheckID: "failing-check",
			Success: false,
			Error:   "Service failed to start",
		},
	}
	registry.Register(failingCheck)

	store := &mockStore{}
	h := NewWithInterface(registry, store, caps)

	req := httptest.NewRequest("POST", "/api/v1/checks/failing-check/actions/restart", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/actions/{actionId}", h.ExecuteCheckAction)
	router.ServeHTTP(w, req)

	// Failed action should return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ExecuteCheckAction() with failure status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp checks.ActionResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success=false for failed action")
	}

	if resp.Error == "" {
		t.Error("Expected error message for failed action")
	}
}

// The distinction this test protects is the one whose absence caused the
// 2026-08-01 outage: a database that is reachable but slow is NOT a database
// that is down.
//
// While a retention cycle held the write lock, every 150ms health probe expired,
// /health reported the database disconnected, and the supervisor restarted an
// API that was serving correctly — which aborted the cycle and guaranteed the
// next one met the same state. Readiness must survive contention.
func TestHealth_SlowDatabaseIsBusyNotUnhealthy(t *testing.T) {
	// Longer than the fast probe's budget, well inside the confirming probe's.
	store := &mockStore{pingDelay: healthDependencyTimeout * 4}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.Health(w, req)

	resp := apihttptest.MustDecodeJSON[map[string]interface{}](t, w.Body.Bytes())

	if resp["status"] != "healthy" {
		t.Errorf("status = %v, want healthy: a slow database is a live one", resp["status"])
	}
	if resp["readiness"] != true {
		t.Error("readiness = false for a reachable database; this is the signal that gets the process restarted")
	}

	deps := resp["dependencies"].(map[string]interface{})
	db, ok := deps["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("dependencies.database should be a DependencyStatus object, got %T", deps["database"])
	}
	if db["connected"] != true {
		t.Errorf("dependencies.database.connected = %v, want true", db["connected"])
	}
	if db["busy"] != true {
		t.Error("a slow probe should be reported as busy, so the condition is visible rather than merely tolerated")
	}
	if db["error"] != nil {
		t.Errorf("dependencies.database.error = %v, want none for a reachable database", db["error"])
	}
}

// The escalation must not blunt real failure detection: a database that is
// genuinely gone still has to report unhealthy, promptly.
func TestHealth_UnreachableDatabaseStillReportsUnhealthy(t *testing.T) {
	store := &mockStore{pingErr: errors.New("database is closed")}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	h.Health(w, req)
	elapsed := time.Since(start)

	resp := apihttptest.MustDecodeJSON[map[string]interface{}](t, w.Body.Bytes())
	if resp["status"] != "unhealthy" {
		t.Errorf("status = %v, want unhealthy", resp["status"])
	}
	if resp["readiness"] != false {
		t.Error("readiness should be false when the database is genuinely unreachable")
	}
	// A driver error is an answer and needs no second probe; only an ambiguous
	// timeout does.
	if elapsed > healthDependencyTimeout {
		t.Errorf("a definite driver error took %s to report; it must not wait out the confirming probe", elapsed)
	}
}
