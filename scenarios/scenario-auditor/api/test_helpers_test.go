//go:build legacy_auditor_tests
// +build legacy_auditor_tests

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type TestFixture struct {
	Server       *httptest.Server
	Client       *http.Client
	TempDir      string
	Manager      *AgentManager
	RuleStore    *RuleStateStore
	VulnStore    *TestVulnerabilityStore
	StdStore     *TestStandardsStore
	cleanupFuncs []func()
}

func NewTestFixture(t *testing.T) *TestFixture {
	tempDir := t.TempDir()

	fixture := &TestFixture{
		TempDir:      tempDir,
		Client:       &http.Client{Timeout: 10 * time.Second},
		cleanupFuncs: make([]func(), 0),
	}

	fixture.Manager = &AgentManager{
		config: AgentConfig{
			Provider: "test",
			Model:    "test-model",
			LogDir:   tempDir,
		},
		agents: make(map[string]*Agent),
		client: &mockHTTPClient{},
	}

	fixture.RuleStore = &RuleStateStore{
		states: make(map[string]bool),
		mu:     sync.RWMutex{},
	}

	fixture.VulnStore = &TestVulnerabilityStore{
		vulnerabilities: make(map[string][]Vulnerability),
		mu:              sync.RWMutex{},
	}

	fixture.StdStore = &TestStandardsStore{
		violations: make(map[string][]StandardViolation),
		mu:         sync.RWMutex{},
	}

	return fixture
}

func (f *TestFixture) Cleanup() {
	if f.Server != nil {
		f.Server.Close()
	}

	for _, cleanup := range f.cleanupFuncs {
		cleanup()
	}
}

func (f *TestFixture) AddCleanup(fn func()) {
	f.cleanupFuncs = append(f.cleanupFuncs, fn)
}

func (f *TestFixture) SetupMockServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", HandleHealth)
	mux.HandleFunc("/api/v1/agents", f.handleAgents)
	mux.HandleFunc("/api/v1/scan", f.handleScan)
	mux.HandleFunc("/api/v1/standards/check", f.handleStandardsCheck)

	f.Server = httptest.NewServer(mux)
}

func (f *TestFixture) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		agents := f.Manager.ListAgents()
		json.NewEncoder(w).Encode(agents)
	case "POST":
		var req struct {
			Type string `json:"type"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		agentType := AgentScanner
		switch req.Type {
		case "standards":
			agentType = AgentStandards
		case "analysis":
			agentType = AgentAnalysis
		case "fix":
			agentType = AgentFix
		}

		agent, err := f.Manager.CreateAgent(agentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(agent)
	}
}

func (f *TestFixture) handleScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scenario string   `json:"scenario"`
		Targets  []string `json:"targets"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result := map[string]interface{}{
		"scan_id": "test-scan-123",
		"status":  "completed",
		"results": []interface{}{},
	}

	json.NewEncoder(w).Encode(result)
}

func (f *TestFixture) handleStandardsCheck(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scenario string   `json:"scenario"`
		Targets  []string `json:"targets"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	result := map[string]interface{}{
		"violations": f.StdStore.GetViolations(req.Scenario),
		"summary": map[string]interface{}{
			"total":  0,
			"passed": 0,
			"failed": 0,
		},
	}

	json.NewEncoder(w).Encode(result)
}

type MockScanner struct {
	name            string
	vulnerabilities []Vulnerability
	shouldFail      bool
	delay           time.Duration
}

func (m *MockScanner) Name() string {
	return m.name
}

func (m *MockScanner) Scan(ctx context.Context, path string) (*ScanResult, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.shouldFail {
		return nil, fmt.Errorf("mock scan failed")
	}

	return &ScanResult{
		ScannerName:     m.name,
		TargetPath:      path,
		ScannedAt:       time.Now(),
		Vulnerabilities: m.vulnerabilities,
	}, nil
}

type MockRule struct {
	id         string
	name       string
	shouldPass bool
	message    string
	shouldErr  bool
}

func (m *MockRule) ToRule() Rule {
	return Rule{
		ID:   m.id,
		Name: m.name,
		Check: func(path string) (bool, string, error) {
			if m.shouldErr {
				return false, "", fmt.Errorf("mock rule error")
			}
			return m.shouldPass, m.message, nil
		},
	}
}

func CreateMockVulnerabilities(count int) []Vulnerability {
	vulns := make([]Vulnerability, count)
	severities := []string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}

	for i := 0; i < count; i++ {
		vulns[i] = Vulnerability{
			Type:        fmt.Sprintf("VULN-%d", i),
			Severity:    severities[i%len(severities)],
			File:        fmt.Sprintf("file%d.go", i),
			Line:        i + 1,
			Column:      10,
			Description: fmt.Sprintf("Test vulnerability %d", i),
			Suggestion:  fmt.Sprintf("Fix suggestion %d", i),
		}
	}

	return vulns
}

func CreateMockStandardViolations(count int) []StandardViolation {
	violations := make([]StandardViolation, count)

	for i := 0; i < count; i++ {
		violations[i] = StandardViolation{
			RuleID:   fmt.Sprintf("rule-%d", i),
			RuleName: fmt.Sprintf("Rule %d", i),
			File:     fmt.Sprintf("file%d.go", i),
			Line:     i + 1,
			Message:  fmt.Sprintf("Violation %d", i),
			Severity: "MEDIUM",
		}
	}

	return violations
}

func AssertJSONResponse(t *testing.T, resp *http.Response, expected interface{}) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var actual interface{}
	if err := json.Unmarshal(body, &actual); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
	actualJSON, _ := json.MarshalIndent(actual, "", "  ")

	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("JSON response mismatch\nExpected:\n%s\n\nActual:\n%s",
			expectedJSON, actualJSON)
	}
}

func AssertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()

	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status %d, got %d. Body: %s",
			expected, resp.StatusCode, body)
	}
}

func AssertContains(t *testing.T, actual, expected string) {
	t.Helper()

	if !strings.Contains(actual, expected) {
		t.Errorf("Expected to contain '%s', got '%s'", expected, actual)
	}
}

func AssertNotContains(t *testing.T, actual, substring string) {
	t.Helper()

	if strings.Contains(actual, substring) {
		t.Errorf("Expected NOT to contain '%s', but found it in '%s'", substring, actual)
	}
}

func AssertEqual(t *testing.T, actual, expected interface{}) {
	t.Helper()

	if actual != expected {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func AssertNotEqual(t *testing.T, actual, expected interface{}) {
	t.Helper()

	if actual == expected {
		t.Errorf("Expected NOT %v, but got %v", expected, actual)
	}
}

func AssertNil(t *testing.T, value interface{}) {
	t.Helper()

	if value != nil {
		t.Errorf("Expected nil, got %v", value)
	}
}

func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()

	if value == nil {
		t.Error("Expected non-nil value, got nil")
	}
}

func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func AssertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func AssertErrorContains(t *testing.T, err error, substring string) {
	t.Helper()

	if err == nil {
		t.Error("Expected error, got nil")
		return
	}

	if !strings.Contains(err.Error(), substring) {
		t.Errorf("Expected error to contain '%s', got '%s'", substring, err.Error())
	}
}

func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for condition: %s", message)
}

func RunParallel(t *testing.T, tests []func(t *testing.T)) {
	t.Helper()

	var wg sync.WaitGroup
	for _, test := range tests {
		wg.Add(1)
		go func(fn func(t *testing.T)) {
			defer wg.Done()
			fn(t)
		}(test)
	}
	wg.Wait()
}

type StandardViolation struct {
	RuleID   string
	RuleName string
	File     string
	Line     int
	Message  string
	Severity string
}

type TestStandardsStore struct {
	violations map[string][]StandardViolation
	mu         sync.RWMutex
}

func (s *TestStandardsStore) AddViolation(scenario string, violation StandardViolation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.violations == nil {
		s.violations = make(map[string][]StandardViolation)
	}

	s.violations[scenario] = append(s.violations[scenario], violation)
}

func (s *TestStandardsStore) GetViolations(scenario string) []StandardViolation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.violations[scenario]
}

func (s *TestStandardsStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.violations = make(map[string][]StandardViolation)
}

type TestVulnerabilityStore struct {
	vulnerabilities map[string][]Vulnerability
	mu              sync.RWMutex
}

func (v *TestVulnerabilityStore) AddVulnerability(target string, vuln Vulnerability) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.vulnerabilities == nil {
		v.vulnerabilities = make(map[string][]Vulnerability)
	}

	v.vulnerabilities[target] = append(v.vulnerabilities[target], vuln)
}

func (v *TestVulnerabilityStore) GetVulns(target string) []Vulnerability {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.vulnerabilities[target]
}

func (v *TestVulnerabilityStore) Clear() {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.vulnerabilities = make(map[string][]Vulnerability)
}

func (v *TestVulnerabilityStore) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()

	count := 0
	for _, vulns := range v.vulnerabilities {
		count += len(vulns)
	}
	return count
}
