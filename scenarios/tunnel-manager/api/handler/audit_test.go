package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tunnel-manager/domain"
)

// mockPortAuditor implements PortAuditor for testing.
type mockPortAuditor struct {
	auditFn func() ([]domain.PortAuditResult, error)
}

func (m *mockPortAuditor) Audit() ([]domain.PortAuditResult, error) { return m.auditFn() }

func TestPortAuditHandler(t *testing.T) {
	auditor := &mockPortAuditor{
		auditFn: func() ([]domain.PortAuditResult, error) {
			return []domain.PortAuditResult{}, nil
		},
	}

	h := HandlePortAudit(auditor)
	req := httptest.NewRequest("GET", "/api/v1/audit/ports", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Results    []domain.PortAuditResult `json:"results"`
		Total      int                      `json:"total"`
		Violations int                      `json:"violations"`
		Compliant  int                      `json:"compliant"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
}
