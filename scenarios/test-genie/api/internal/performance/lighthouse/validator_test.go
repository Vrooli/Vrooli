package lighthouse

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

// mockClient is a test double for the Client interface.
type mockClient struct {
	healthErr    error
	auditResp    *AuditResponse
	auditErr     error
	auditCalled  bool
	auditRequest AuditRequest
}

func (m *mockClient) Health(ctx context.Context) error {
	return m.healthErr
}

func (m *mockClient) Audit(ctx context.Context, req AuditRequest) (*AuditResponse, error) {
	m.auditCalled = true
	m.auditRequest = req
	return m.auditResp, m.auditErr
}

func TestValidator_Audit_Disabled(t *testing.T) {
	v := New(ValidatorConfig{
		Config: &Config{Enabled: false},
	})

	result := v.Audit(context.Background())

	if !result.Skipped {
		t.Error("expected audit to be skipped when disabled")
	}
	if !result.Success {
		t.Error("expected success when skipped")
	}
}

func TestValidator_Audit_NoPages(t *testing.T) {
	v := New(ValidatorConfig{
		Config: &Config{Enabled: true, Pages: nil},
	})

	result := v.Audit(context.Background())

	if !result.Skipped {
		t.Error("expected audit to be skipped when no pages")
	}
}

func TestValidator_Audit_NoUIURL(t *testing.T) {
	// This tests the critical behavior where Lighthouse is enabled with pages configured,
	// but no UI URL is provided. Previously this would silently fallback to localhost:3000.
	// Now it should explicitly skip with a helpful message.
	v := New(ValidatorConfig{
		BaseURL:        "", // Empty - no UI URL configured
		Config: &Config{
			Enabled: true,
			Pages: []PageConfig{
				{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.75, Warn: 0.90}}},
			},
		},
	})

	result := v.Audit(context.Background())

	if !result.Skipped {
		t.Error("expected audit to be skipped when no UI URL configured")
	}
	if !result.Success {
		t.Error("expected success when skipped")
	}

	// Verify we get helpful observations
	if len(result.Observations) < 2 {
		t.Errorf("expected at least 2 observations (skip reason + info), got %d", len(result.Observations))
	}

	// Check for the skip observation
	foundSkip := false
	foundInfo := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationSkip && obs.Message == "no UI URL configured" {
			foundSkip = true
		}
		if obs.Type == ObservationInfo {
			foundInfo = true
		}
	}
	if !foundSkip {
		t.Error("expected skip observation with 'no UI URL configured' message")
	}
	if !foundInfo {
		t.Error("expected info observation with guidance on how to enable Lighthouse")
	}
}

func TestValidator_Audit_InvalidConfig(t *testing.T) {
	v := New(ValidatorConfig{
		BaseURL:        "http://localhost:3000", // Must provide BaseURL to reach config validation
		Config: &Config{
			Enabled: true,
			Pages: []PageConfig{
				{ID: "", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}}},
			},
		},
	})

	result := v.Audit(context.Background())

	if result.Success {
		t.Error("expected failure for invalid config")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration failure, got %v", result.FailureClass)
	}
}

func TestValidator_Audit_LighthouseUnavailable(t *testing.T) {
	mock := &mockClient{
		healthErr: errors.New("connection refused"),
	}

	v := New(ValidatorConfig{
		BaseURL:        "http://localhost:3000",
		Config: &Config{
			Enabled: true,
			Pages: []PageConfig{
				{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.5, Warn: 0.7}}},
			},
		},
	}, WithClient(mock))

	result := v.Audit(context.Background())

	if result.Success {
		t.Error("expected failure when Lighthouse unavailable")
	}
	if result.FailureClass != FailureClassUnavailable {
		t.Errorf("expected unavailable failure, got %v", result.FailureClass)
	}
}

func TestValidator_Audit_Success(t *testing.T) {
	score := 0.85
	mock := &mockClient{
		auditResp: &AuditResponse{
			Categories: map[string]CategoryResult{
				"performance": {ID: "performance", Score: &score},
			},
		},
	}

	v := New(ValidatorConfig{
		BaseURL:        "http://localhost:3000",
		Config: &Config{
			Enabled: true,
			Pages: []PageConfig{
				{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.75, Warn: 0.90}}},
			},
		},
	}, WithClient(mock))

	result := v.Audit(context.Background())

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}
	if len(result.PageResults) != 1 {
		t.Errorf("expected 1 page result, got %d", len(result.PageResults))
	}
	if !result.PageResults[0].Success {
		t.Error("expected page to pass")
	}
	if result.PageResults[0].Scores["performance"] != 0.85 {
		t.Errorf("expected performance score 0.85, got %f", result.PageResults[0].Scores["performance"])
	}
}

func TestValidator_Audit_ThresholdViolation(t *testing.T) {
	score := 0.65 // Below error threshold of 0.75
	mock := &mockClient{
		auditResp: &AuditResponse{
			Categories: map[string]CategoryResult{
				"performance": {ID: "performance", Score: &score},
			},
		},
	}

	v := New(ValidatorConfig{
		BaseURL:        "http://localhost:3000",
		Config: &Config{
			Enabled: true,
			Pages: []PageConfig{
				{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.75, Warn: 0.90}}},
			},
		},
	}, WithClient(mock))

	result := v.Audit(context.Background())

	if result.Success {
		t.Error("expected failure due to threshold violation")
	}
	if len(result.PageResults) != 1 {
		t.Errorf("expected 1 page result, got %d", len(result.PageResults))
	}
	if result.PageResults[0].Success {
		t.Error("expected page to fail")
	}
	if len(result.PageResults[0].Violations) != 1 {
		t.Errorf("expected 1 violation, got %d", len(result.PageResults[0].Violations))
	}
	violation := result.PageResults[0].Violations[0]
	if violation.Category != "performance" {
		t.Errorf("expected performance violation, got %s", violation.Category)
	}
	if violation.Level != "error" {
		t.Errorf("expected error level, got %s", violation.Level)
	}
}

func TestValidator_Audit_Warning(t *testing.T) {
	score := 0.80 // Between error (0.75) and warn (0.90)
	mock := &mockClient{
		auditResp: &AuditResponse{
			Categories: map[string]CategoryResult{
				"performance": {ID: "performance", Score: &score},
			},
		},
	}

	v := New(ValidatorConfig{
		BaseURL:        "http://localhost:3000",
		Config: &Config{
			Enabled: true,
			Pages: []PageConfig{
				{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.75, Warn: 0.90}}},
			},
		},
	}, WithClient(mock))

	result := v.Audit(context.Background())

	if !result.Success {
		t.Error("expected success (warning shouldn't cause failure)")
	}
	if len(result.PageResults[0].Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.PageResults[0].Warnings))
	}
}

func TestValidator_Audit_AuditError(t *testing.T) {
	mock := &mockClient{
		auditErr: errors.New("audit failed"),
	}

	v := New(ValidatorConfig{
		BaseURL:        "http://localhost:3000",
		Config: &Config{
			Enabled: true,
			Pages: []PageConfig{
				{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.75, Warn: 0.90}}},
			},
		},
	}, WithClient(mock))

	result := v.Audit(context.Background())

	if result.Success {
		t.Error("expected failure when audit errors")
	}
	if result.PageResults[0].Error == nil {
		t.Error("expected page result to have error")
	}
}

func TestValidator_Audit_MultiplePages(t *testing.T) {
	homeScore := 0.85
	aboutScore := 0.60 // Below threshold
	callCount := 0

	mock := &mockClient{}
	// Return different responses for different URLs
	origAudit := mock.Audit
	_ = origAudit // Avoid unused variable

	v := New(ValidatorConfig{
		BaseURL:        "http://localhost:3000",
		Config: &Config{
			Enabled: true,
			Pages: []PageConfig{
				{ID: "home", Path: "/", Thresholds: CategoryThresholds{"performance": {Error: 0.75, Warn: 0.90}}},
				{ID: "about", Path: "/about", Thresholds: CategoryThresholds{"performance": {Error: 0.75, Warn: 0.90}}},
			},
		},
	}, WithClient(&multiPageMock{
		responses: []*AuditResponse{
			{Categories: map[string]CategoryResult{"performance": {Score: &homeScore}}},
			{Categories: map[string]CategoryResult{"performance": {Score: &aboutScore}}},
		},
		callCount: &callCount,
	}))

	result := v.Audit(context.Background())

	if result.Success {
		t.Error("expected failure due to second page failing")
	}
	if len(result.PageResults) != 2 {
		t.Errorf("expected 2 page results, got %d", len(result.PageResults))
	}
	if !result.PageResults[0].Success {
		t.Error("expected first page to pass")
	}
	if result.PageResults[1].Success {
		t.Error("expected second page to fail")
	}
}

// multiPageMock returns different responses for consecutive calls.
type multiPageMock struct {
	responses []*AuditResponse
	callCount *int
}

func (m *multiPageMock) Health(ctx context.Context) error {
	return nil
}

func (m *multiPageMock) Audit(ctx context.Context, req AuditRequest) (*AuditResponse, error) {
	idx := *m.callCount
	*m.callCount++
	if idx < len(m.responses) {
		return m.responses[idx], nil
	}
	return nil, errors.New("no more responses")
}

func TestValidator_BuildPageURL(t *testing.T) {
	// NOTE: buildPageURL now assumes BaseURL is non-empty (checked in Audit before reaching here).
	// The empty BaseURL case is handled by skipping in Audit, not by fallback here.
	tests := []struct {
		baseURL  string
		path     string
		expected string
	}{
		{"http://localhost:3000", "/", "http://localhost:3000/"},
		{"http://localhost:3000", "/about", "http://localhost:3000/about"},
		{"http://localhost:3000/", "/about", "http://localhost:3000/about"},
		{"http://localhost:3000/app", "/about", "http://localhost:3000/about"},
		{"http://example.com:8080", "/dashboard", "http://example.com:8080/dashboard"},
	}

	for _, tt := range tests {
		t.Run(tt.baseURL+tt.path, func(t *testing.T) {
			v := &validator{
				config: ValidatorConfig{BaseURL: tt.baseURL},
			}
			result := v.buildPageURL(tt.path)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestValidator_WithLogger(t *testing.T) {
	var buf bytes.Buffer
	v := New(ValidatorConfig{
		Config: &Config{Enabled: false},
	}, WithLogger(&buf))

	v.Audit(context.Background())

	if buf.Len() == 0 {
		t.Error("expected log output")
	}
}

func TestAuditResult_Summary(t *testing.T) {
	tests := []struct {
		name     string
		result   AuditResult
		expected string
	}{
		{
			name:     "skipped",
			result:   AuditResult{Skipped: true},
			expected: "Lighthouse audits skipped",
		},
		{
			name:     "no pages",
			result:   AuditResult{PageResults: nil},
			expected: "No pages audited",
		},
		{
			name: "all passed",
			result: AuditResult{
				PageResults: []PageResult{
					{Success: true},
					{Success: true},
				},
			},
			expected: "Lighthouse: 2 page(s) passed",
		},
		{
			name: "mixed results",
			result: AuditResult{
				PageResults: []PageResult{
					{Success: true},
					{Success: false},
				},
			},
			expected: "Lighthouse: 1 passed, 1 failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.Summary(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestCategoryViolation_String(t *testing.T) {
	v := CategoryViolation{
		Category:  "performance",
		Score:     0.65,
		Threshold: 0.75,
	}

	expected := "performance: 65% (threshold: 75%)"
	if got := v.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
