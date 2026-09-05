package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Test helpers for handler testing

// marshalBodyOrString marshals body to JSON bytes, or converts string directly
func marshalBodyOrString(t *testing.T, body interface{}) []byte {
	t.Helper()
	if str, ok := body.(string); ok {
		return []byte(str)
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}
	return bodyBytes
}

// testHandlerRequest executes a handler with a test request and returns the response recorder
func testHandlerRequest(t *testing.T, method, url string, body interface{}, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	bodyBytes := marshalBodyOrString(t, body)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// assertHandlerStatus checks if handler returned expected status code
func assertHandlerStatus(t *testing.T, rr *httptest.ResponseRecorder, expected int, handlerName string) {
	t.Helper()
	if rr.Code != expected {
		t.Errorf("%s status = %v, want %v", handlerName, rr.Code, expected)
	}
}

// assertHandlerError decodes error response and checks message
func assertHandlerError(t *testing.T, rr *httptest.ResponseRecorder, wantError, handlerName string) {
	t.Helper()
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if response["error"] != wantError {
		t.Errorf("%s error = %v, want %v", handlerName, response["error"], wantError)
	}
}

// assertParseResponseFields validates parse endpoint response has required fields
func assertParseResponseFields(t *testing.T, rr *httptest.ResponseRecorder, handlerName string) {
	t.Helper()
	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode success response: %v", err)
	}
	if _, ok := response["issues"]; !ok {
		t.Errorf("%s response missing 'issues' field", handlerName)
	}
	if _, ok := response["count"]; !ok {
		t.Errorf("%s response missing 'count' field", handlerName)
	}
}

// Test respondJSON helper function
func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
	}{
		{
			name:       "success response",
			status:     http.StatusOK,
			data:       map[string]string{"message": "success"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "created response",
			status:     http.StatusCreated,
			data:       map[string]int{"id": 123},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			respondJSON(rr, tt.status, tt.data)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("respondJSON() status = %v, want %v", status, tt.wantStatus)
			}

			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("respondJSON() Content-Type = %v, want application/json", ct)
			}
		})
	}
}

// Test respondError helper function
func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
	}{
		{
			name:       "bad request error",
			status:     http.StatusBadRequest,
			message:    "invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal server error",
			status:     http.StatusInternalServerError,
			message:    "something went wrong",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			respondError(rr, tt.status, tt.message)

			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("respondError() status = %v, want %v", status, tt.wantStatus)
			}

			var response map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("respondError() message = %v, want %v", response["error"], tt.message)
			}
		})
	}
}

func TestBuildTidinessScanReportsMaintainabilityOnly(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}

	var content strings.Builder
	content.WriteString("package main\n\n")
	content.WriteString("import \"fmt\"\n\n")
	content.WriteString("func main() {\n")
	for i := 0; i < 12; i++ {
		content.WriteString("\t// TODO: planned cleanup\n")
	}
	content.WriteString("\tfmt.Println(\"ok\")\n")
	content.WriteString("}\n")
	for i := 0; i < 520; i++ {
		content.WriteString("// filler line\n")
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(content.String()), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	result, err := buildTidinessScan(context.Background(), "demo", tmpDir, 0)
	if err != nil {
		t.Fatalf("buildTidinessScan failed: %v", err)
	}

	if result.Summary.LongFiles == 0 {
		t.Fatalf("expected long-file finding, got summary %+v", result.Summary)
	}
	if result.Summary.TechDebt == 0 {
		t.Fatalf("expected tech-debt finding, got summary %+v", result.Summary)
	}
	if err := assessment.ValidateAssessment(result.Assessment); err != nil {
		t.Fatalf("assessment invalid: %v", err)
	}
	if result.Assessment.GetProvider() != "tidiness-manager" || result.Assessment.GetPhase() != "tidiness" {
		t.Fatalf("assessment identity = %s/%s, want tidiness-manager/tidiness", result.Assessment.GetProvider(), result.Assessment.GetPhase())
	}
	for _, finding := range result.Findings {
		if finding.Category == "type_safety" || strings.Contains(finding.RuleID, "TS_CONFIG") {
			t.Fatalf("tidiness scan must not emit static-quality finding: %+v", finding)
		}
		if finding.WhyItMatters == "" || finding.RecommendedRemediation == "" {
			t.Fatalf("finding missing guidance fields: %+v", finding)
		}
	}
}

func TestTidinessMaturitySpecCoversEmittedRules(t *testing.T) {
	spec, err := loadTidinessMaturitySpec()
	if err != nil {
		t.Fatalf("loadTidinessMaturitySpec() error = %v", err)
	}
	if spec.Version != "2.0.0" {
		t.Fatalf("version = %q, want 2.0.0", spec.Version)
	}
	if len(spec.Capabilities) != 5 {
		t.Fatalf("capabilities = %d, want 5", len(spec.Capabilities))
	}
	for _, ruleID := range emittedTidinessRuleIDs() {
		mapping, ok := spec.Findings[ruleID]
		if !ok {
			t.Fatalf("maturity spec missing emitted rule %q", ruleID)
		}
		if mapping.CapabilityID == "" {
			t.Fatalf("maturity spec finding %q must declare capability_id", ruleID)
		}
	}
	if spec.Fallback.CapabilityID != "scan_contract" {
		t.Fatalf("fallback capability = %q, want scan_contract", spec.Fallback.CapabilityID)
	}

	got, err := buildTidinessMaturityAssessment("demo", []TidinessFinding{
		newTidinessFinding("demo", "long-file", "length", "medium", "api/server.go", "", 1, "large file", "too large", nil, "why", "split it", "file-size"),
	}, spec)
	if err != nil {
		t.Fatalf("buildTidinessMaturityAssessment() error = %v", err)
	}
	if len(got.GetCapabilities()) != 5 {
		t.Fatalf("capabilities = %d, want 5", len(got.GetCapabilities()))
	}
	if got.GetFindings()[0].GetMaturity().GetCapabilityId() != "local_debt_control" {
		t.Fatalf("long-file capability = %q, want local_debt_control", got.GetFindings()[0].GetMaturity().GetCapabilityId())
	}
	if got.GetHighestPriorityCapability().GetCapabilityId() != "local_debt_control" {
		t.Fatalf("warning-only focus = %#v, want local_debt_control", got.GetHighestPriorityCapability())
	}
	duplication := maturityCapability(got, "duplication_control")
	if duplication == nil || duplication.GetCurrentLevel() != "L4" || !duplication.GetClean() {
		t.Fatalf("structural-only duplication capability = %#v, want clean L4", duplication)
	}

	blocking, err := buildTidinessMaturityAssessment("demo", []TidinessFinding{
		newTidinessFinding("demo", "tidiness-budget-exceeded", "budget", "high", "", "", 0, "budget exceeded", "duplicate debt exceeded budget", nil, "why", "reduce debt", "tidiness-budget"),
	}, spec)
	if err != nil {
		t.Fatalf("buildTidinessMaturityAssessment(blocking) error = %v", err)
	}
	if blocking.GetFindings()[0].GetMaturity().GetCapabilityId() != "duplication_control" {
		t.Fatalf("budget capability = %q, want duplication_control", blocking.GetFindings()[0].GetMaturity().GetCapabilityId())
	}
	if blocking.GetHighestPriorityCapability().GetCapabilityId() != "duplication_control" {
		t.Fatalf("blocking focus = %#v, want duplication_control", blocking.GetHighestPriorityCapability())
	}
	if blocking.GetLocal().GetCurrentLevel() != "L2" {
		t.Fatalf("blocking assessment current level = %q, want L2", blocking.GetLocal().GetCurrentLevel())
	}
	blockingDuplication := maturityCapability(blocking, "duplication_control")
	if blockingDuplication == nil || blockingDuplication.GetCurrentLevel() != "L2" || blockingDuplication.GetClean() {
		t.Fatalf("budget-breached duplication capability = %#v, want non-clean L2", blockingDuplication)
	}

	opportunity, err := buildTidinessMaturityAssessment("demo", []TidinessFinding{
		newTidinessFinding("demo", "duplicated-code", "duplication", "medium", "api/copy.go", "", 1, "duplicate", "open refactor opportunity", nil, "why", "extract", "duplication"),
	}, spec)
	if err != nil {
		t.Fatalf("buildTidinessMaturityAssessment(opportunity) error = %v", err)
	}
	opportunityDuplication := maturityCapability(opportunity, "duplication_control")
	if opportunityDuplication == nil || opportunityDuplication.GetCurrentLevel() != "L3" || opportunityDuplication.GetClean() {
		t.Fatalf("open-opportunity duplication capability = %#v, want non-clean L3", opportunityDuplication)
	}
}

func maturityCapability(assessment *commonv1.MaturityAssessment, id string) *commonv1.CapabilityMaturityAssessment {
	for _, capability := range assessment.GetCapabilities() {
		if capability.GetId() == id {
			return capability
		}
	}
	return nil
}

func TestTidinessBudgetFinding_RatchetRejectsDebtRegressionAndBudgetLoosening(t *testing.T) {
	scenarioPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatalf("create testing config directory: %v", err)
	}
	writeConfig := func(t *testing.T, budget, baseline int) {
		t.Helper()
		config := fmt.Sprintf(`{"phases":{"tidiness":{"budgets":{"duplication_line_debt":%d,"baseline_duplication_line_debt":%d,"ratchet":true}}}}`, budget, baseline)
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "testing.json"), []byte(config), 0o600); err != nil {
			t.Fatalf("write testing config: %v", err)
		}
	}

	writeConfig(t, 120, 100)
	loosened := tidinessBudgetFinding("demo", scenarioPath, TidinessScanSummary{DuplicationLineDebt: 80})
	if loosened == nil || loosened.Evidence["violation"] != "ratchet_loosened_budget" {
		t.Fatalf("ratchet budget loosening = %#v, want named violation", loosened)
	}

	writeConfig(t, 100, 100)
	worsened := tidinessBudgetFinding("demo", scenarioPath, TidinessScanSummary{DuplicationLineDebt: 101})
	if worsened == nil || worsened.Evidence["violation"] != "ratchet_worsened_debt" || worsened.Evidence["delta"] != 1 {
		t.Fatalf("ratchet debt regression = %#v, want named delta", worsened)
	}
}

func TestFrozenBudgetFailsTheGate(t *testing.T) {
	scenarioPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"phases":{"tidiness":{"budgets":{"duplication_line_debt":100,"baseline_duplication_line_debt":100,"ratchet":true}}}}`
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "testing.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	finding := tidinessBudgetFinding("demo", scenarioPath, TidinessScanSummary{DuplicationLineDebt: 100})
	if finding == nil {
		t.Fatal("frozen budget passed the gate")
	}
	if finding.Evidence["violation"] != "frozen_budget" {
		t.Fatalf("violation = %#v, want frozen_budget", finding.Evidence["violation"])
	}
	message := finding.Description
	if !strings.Contains(message, filepath.Join(scenarioPath, ".vrooli", "testing.json")) || !strings.Contains(message, "duplication_line_debt") || !strings.Contains(message, "100") {
		t.Fatalf("message = %q, want file, key, and observation", message)
	}

	config = `{"phases":{"tidiness":{"budgets":{"duplication_line_debt":99,"baseline_duplication_line_debt":100,"ratchet":true}}}}`
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "testing.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if finding := tidinessBudgetFinding("demo", scenarioPath, TidinessScanSummary{DuplicationLineDebt: 99}); finding != nil {
		t.Fatalf("tightened budget failed = %#v", finding)
	}
}

func TestTidinessBudgetReserveRequiresReasonAndCoversMeasuredSeed(t *testing.T) {
	scenarioPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(scenarioPath, ".vrooli", "testing.json")
	write := func(config string) {
		t.Helper()
		if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write(`{"phases":{"tidiness":{"budgets":{"duplication_line_debt":99,"baseline_duplication_line_debt":100,"reserve":1,"reserve_reason":"Measured seed keeps the current observation visible until cleanup earns a tighter budget.","ratchet":true}}}}`)
	if finding := tidinessBudgetFinding("demo", scenarioPath, TidinessScanSummary{DuplicationLineDebt: 100}); finding != nil {
		t.Fatalf("measured reserve failed = %#v", finding)
	}
	report := newBudgetAuditReport("demo", scenarioPath, nil, nil, TidinessScanSummary{DuplicationLineDebt: 100})
	if report.Blocking || report.Metrics[0].Reserve != 1 || report.Metrics[0].ReserveReason == "" {
		t.Fatalf("reserve audit = %#v", report.Metrics[0])
	}

	write(`{"phases":{"tidiness":{"budgets":{"duplication_line_debt":99,"baseline_duplication_line_debt":100,"reserve":1,"reserve_reason":"short","ratchet":true}}}}`)
	report = newBudgetAuditReport("demo", scenarioPath, nil, nil, TidinessScanSummary{DuplicationLineDebt: 100})
	if !report.Blocking || !slices.Contains(report.Metrics[0].Verdicts, "reserve_reason_missing") {
		t.Fatalf("short reserve reason passed = %#v", report.Metrics[0])
	}
}

func TestTidinessBudgetRatchetAppliesToEveryMetric(t *testing.T) {
	metrics := []struct {
		name         string
		baselineName string
		summary      func(int) TidinessScanSummary
	}{
		{name: "duplication_line_debt", baselineName: "baseline_duplication_line_debt", summary: func(value int) TidinessScanSummary { return TidinessScanSummary{DuplicationLineDebt: value} }},
		{name: "long_files", baselineName: "baseline_long_files", summary: func(value int) TidinessScanSummary { return TidinessScanSummary{LongFiles: value} }},
		{name: "complexity_over_threshold", baselineName: "baseline_complexity_over_threshold", summary: func(value int) TidinessScanSummary { return TidinessScanSummary{Complexity: value} }},
		{name: "coupling_over_threshold", baselineName: "baseline_coupling_over_threshold", summary: func(value int) TidinessScanSummary { return TidinessScanSummary{Coupling: value} }},
		{name: "debt_markers", baselineName: "baseline_debt_markers", summary: func(value int) TidinessScanSummary { return TidinessScanSummary{TechDebt: value} }},
	}
	for _, metric := range metrics {
		t.Run(metric.name, func(t *testing.T) {
			scenarioPath := t.TempDir()
			if err := os.Mkdir(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
				t.Fatal(err)
			}
			writeBudget := func(budget int, baseline *int) {
				t.Helper()
				fields := fmt.Sprintf(`"%s":%d`, metric.name, budget)
				if baseline != nil {
					fields += fmt.Sprintf(`,"%s":%d`, metric.baselineName, *baseline)
				}
				config := fmt.Sprintf(`{"phases":{"tidiness":{"budgets":{%s,"ratchet":true}}}}`, fields)
				if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "testing.json"), []byte(config), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			baseline := 5
			writeBudget(5, &baseline)
			if finding := tidinessBudgetFinding("demo", scenarioPath, metric.summary(5)); finding == nil || finding.Evidence["violation"] != "frozen_budget" || finding.Evidence["metric"] != metric.name {
				t.Fatalf("frozen finding = %#v", finding)
			}
			writeBudget(6, &baseline)
			if finding := tidinessBudgetFinding("demo", scenarioPath, metric.summary(4)); finding == nil || finding.Evidence["violation"] != "ratchet_loosened_budget" {
				t.Fatalf("loosened finding = %#v", finding)
			}
			writeBudget(4, &baseline)
			if finding := tidinessBudgetFinding("demo", scenarioPath, metric.summary(6)); finding == nil || finding.Evidence["violation"] != "ratchet_worsened_debt" {
				t.Fatalf("worsened finding = %#v", finding)
			}
			writeBudget(5, nil)
			if finding := tidinessBudgetFinding("demo", scenarioPath, metric.summary(5)); finding != nil {
				t.Fatalf("baseline opt-in finding = %#v", finding)
			}
		})
	}
}

func TestTidinessBudgetFinding_EnforcesDeclaredZero(t *testing.T) {
	scenarioPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"phases":{"tidiness":{"budgets":{"duplication_line_debt":0,"baseline_duplication_line_debt":0,"debt_markers":0,"ratchet":true}}}}`
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "testing.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if finding := tidinessBudgetFinding("demo", scenarioPath, TidinessScanSummary{TechDebt: 1}); finding == nil || finding.Evidence["metric"] != "debt_markers" {
		t.Fatalf("declared zero finding = %#v", finding)
	}
	if finding := tidinessBudgetFinding("demo", scenarioPath, TidinessScanSummary{}); finding != nil {
		t.Fatalf("clean declared zero = %#v", finding)
	}
}

func TestBudgetAuditReportIncludesEverySeamAndMetric(t *testing.T) {
	scenarioPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"phases":{"tidiness":{"budgets":{"long_files":2,"baseline_long_files":3,"ratchet":true}}}}`
	if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "testing.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	seams := []Seam{
		{ID: "exact", Budget: 1, ReserveReason: "tracked debt"},
		{ID: "loose", Budget: 3, Reserve: 1, ReserveReason: "one reviewed exception"},
	}
	hits := []SeamHit{{SeamID: "exact"}, {SeamID: "loose"}}
	report := newBudgetAuditReport("demo", scenarioPath, seams, hits, TidinessScanSummary{LongFiles: 1})
	if len(report.Seams) != 2 || len(report.Metrics) != 5 {
		t.Fatalf("audit shape = %#v", report)
	}
	if report.Seams[0].Observed != 1 || report.Seams[0].DeclaredBaseline != nil || report.Seams[0].Verdicts[0] != "ok" {
		t.Fatalf("exact seam audit = %#v", report.Seams[0])
	}
	if report.Seams[1].DeclaredBudget != 3 || report.Seams[1].Observed != 1 || report.Seams[1].Reserve != 1 || report.Seams[1].Verdicts[0] != "SEAM_BUDGET_SLACK" {
		t.Fatalf("loose seam audit = %#v", report.Seams[1])
	}
	if report.Metrics[1].Name != "long_files" || report.Metrics[1].DeclaredBudget == nil || *report.Metrics[1].DeclaredBudget != 2 || report.Metrics[1].DeclaredBaseline == nil || *report.Metrics[1].DeclaredBaseline != 3 || report.Metrics[1].Observed != 1 {
		t.Fatalf("long-file metric audit = %#v", report.Metrics[1])
	}
	if !report.Blocking {
		t.Fatal("slack seam did not block the audit")
	}
}

func TestHandleBudgetAuditRequiresAndReusesCompletedValidation(t *testing.T) {
	scenarioPath := t.TempDir()
	srv := &Server{}
	body := BudgetAuditRequest{Scenario: "demo", ScenarioPath: scenarioPath}
	missing := testHandlerRequest(t, "POST", "/api/v1/budget-audit", body, srv.handleBudgetAudit)
	assertHandlerStatus(t, missing, http.StatusConflict, "handleBudgetAudit() without validation")

	want := &BudgetAuditReport{Scenario: "demo", Seams: []SeamBudgetAudit{}, Metrics: []MetricBudgetAudit{}}
	srv.storeBudgetAudit(scenarioPath, want)
	cached := testHandlerRequest(t, "POST", "/api/v1/budget-audit", body, srv.handleBudgetAudit)
	assertHandlerStatus(t, cached, http.StatusOK, "handleBudgetAudit() with validation")
	var got BudgetAuditReport
	if err := json.Unmarshal(cached.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Scenario != want.Scenario {
		t.Fatalf("cached audit = %#v", got)
	}
}

func emittedTidinessRuleIDs() []string {
	return []string{
		"LONG_FILE",
		"TECH_DEBT_MARKERS",
		"HIGH_COUPLING",
		"HIGH_COMPLEXITY",
		"DUPLICATED_CODE",
	}
}

// Test handleParseLint
func TestHandleParseLint(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name: "valid lint parse request",
			body: ParseRequest{
				Scenario: "test-scenario",
				Tool:     "eslint",
				Output:   "src/main.ts:10:5: error: Unexpected token",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing scenario",
			body:       ParseRequest{Tool: "eslint", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "missing tool",
			body:       ParseRequest{Scenario: "test", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/parse/lint", tt.body, srv.handleParseLint)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleParseLint()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleParseLint()")
			} else {
				assertParseResponseFields(t, rr, "handleParseLint()")
			}
		})
	}
}

// Test handleParseType
func TestHandleParseType(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name: "valid type parse request",
			body: ParseRequest{
				Scenario: "test-scenario",
				Tool:     "tsc",
				Output:   "src/main.ts(10,5): error TS2304: Cannot find name 'foo'",
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing scenario",
			body:       ParseRequest{Tool: "tsc", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "missing tool",
			body:       ParseRequest{Scenario: "test", Output: "some output"},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario and tool are required",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/parse/type", tt.body, srv.handleParseType)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleParseType()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleParseType()")
			} else {
				assertParseResponseFields(t, rr, "handleParseType()")
			}
		})
	}
}

// Test handleLightScan
func TestHandleLightScan(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
		db:     nil, // No DB needed for basic tests
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing scenario_path",
			body:       LightScanRequest{},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario_path is required",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/scan/light", tt.body, srv.handleLightScan)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleLightScan()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleLightScan()")
			}
		})
	}
}

// Test handleSmartScan
func TestHandleSmartScan(t *testing.T) {
	srv := &Server{
		router: mux.NewRouter(),
	}

	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing scenario",
			body:       SmartScanRequest{},
			wantStatus: http.StatusBadRequest,
			wantError:  "scenario is required",
		},
		{
			name:       "missing files",
			body:       SmartScanRequest{Scenario: "test-scenario"},
			wantStatus: http.StatusBadRequest,
			wantError:  "files list cannot be empty",
		},
		{
			name:       "invalid JSON",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := testHandlerRequest(t, "POST", "/api/v1/scan/smart", tt.body, srv.handleSmartScan)
			assertHandlerStatus(t, rr, tt.wantStatus, "handleSmartScan()")

			if tt.wantError != "" {
				assertHandlerError(t, rr, tt.wantError, "handleSmartScan()")
			}
		})
	}
}
