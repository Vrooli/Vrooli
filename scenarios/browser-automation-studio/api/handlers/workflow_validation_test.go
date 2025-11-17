package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

func newTestValidator(t *testing.T) *workflowvalidator.Validator {
	t.Helper()
	v, err := workflowvalidator.NewValidator()
	if err != nil {
		t.Fatalf("failed to init validator: %v", err)
	}
	return v
}

func TestValidateWorkflow_Success(t *testing.T) {
	handler := &Handler{workflowValidator: newTestValidator(t)}

	payload := map[string]any{
		"workflow": map[string]any{
			"metadata": map[string]any{
				"description": "lint happy path",
				"requirement": "BAS-TEST-OK",
				"version":     1,
			},
			"nodes": []any{
				map[string]any{
					"id":       "navigate",
					"type":     "navigate",
					"position": map[string]any{"x": 0, "y": 0},
					"data": map[string]any{
						"destinationType": "url",
						"url":             "https://example.com",
					},
				},
			},
			"edges": []any{},
		},
		"strict": false,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var result workflowvalidator.Result
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !result.Valid {
		t.Fatalf("expected workflow to be valid, got %+v", result.Errors)
	}
}

func TestValidateWorkflow_StrictPromotesWarnings(t *testing.T) {
	handler := &Handler{workflowValidator: newTestValidator(t)}

	payload := map[string]any{
		"workflow": map[string]any{
			"nodes": []any{
				map[string]any{
					"id":       "wait",
					"type":     "wait",
					"position": map[string]any{"x": 0, "y": 0},
					"data": map[string]any{
						"waitType":   "duration",
						"durationMs": 1000,
					},
				},
			},
			"edges": []any{},
		},
		"strict": true,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var result workflowvalidator.Result
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Valid {
		t.Fatal("expected validation to fail in strict mode")
	}

	foundStrict := false
	for _, issue := range result.Errors {
		if issue.Code == "WF_STRICT_WARNING" {
			foundStrict = true
			break
		}
	}
	if !foundStrict {
		t.Fatalf("expected WF_STRICT_WARNING error, got %+v", result.Errors)
	}
}

func TestValidateWorkflow_InvalidPayload(t *testing.T) {
	handler := &Handler{workflowValidator: newTestValidator(t)}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", bytes.NewReader([]byte(`{"strict":false}`)))
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
