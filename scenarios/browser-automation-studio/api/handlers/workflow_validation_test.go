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

func TestValidateWorkflow_MalformedJSON(t *testing.T) {
	handler := &Handler{workflowValidator: newTestValidator(t)}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", bytes.NewReader([]byte(`{invalid json`)))
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for malformed JSON, got %d", rr.Code)
	}
}

func TestValidateWorkflow_ValidationErrors(t *testing.T) {
	handler := &Handler{workflowValidator: newTestValidator(t)}

	payload := map[string]any{
		"workflow": map[string]any{
			"nodes": []any{
				map[string]any{
					"id":   "click-missing-selector",
					"type": "click",
					"data": map[string]any{},
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

	if result.Valid {
		t.Fatal("expected workflow to be invalid due to missing selector")
	}

	foundFieldRequired := false
	for _, issue := range result.Errors {
		if issue.Code == "WF_NODE_FIELD_REQUIRED" {
			foundFieldRequired = true
			break
		}
	}
	if !foundFieldRequired {
		t.Fatalf("expected WF_NODE_FIELD_REQUIRED error, got %+v", result.Errors)
	}
}

func TestValidateWorkflow_EmptyWorkflow(t *testing.T) {
	handler := &Handler{workflowValidator: newTestValidator(t)}

	payload := map[string]any{
		"workflow": map[string]any{
			"nodes": []any{},
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

	// Empty workflow should be invalid
	if result.Valid {
		t.Fatal("expected empty workflow to be invalid")
	}
}

func TestValidateWorkflow_StatsInResponse(t *testing.T) {
	handler := &Handler{workflowValidator: newTestValidator(t)}

	payload := map[string]any{
		"workflow": map[string]any{
			"metadata": map[string]any{
				"description": "test workflow with multiple nodes",
			},
			"nodes": []any{
				map[string]any{
					"id":       "click1",
					"type":     "click",
					"position": map[string]any{"x": 0, "y": 0},
					"data":     map[string]any{"selector": "#button"},
				},
				map[string]any{
					"id":       "wait1",
					"type":     "wait",
					"position": map[string]any{"x": 100, "y": 0},
					"data": map[string]any{
						"waitType":   "element",
						"selector":   "#result",
					},
				},
			},
			"edges": []any{
				map[string]any{
					"id":     "e1",
					"source": "click1",
					"target": "wait1",
				},
			},
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

	// Check that stats are populated
	if result.Stats.NodeCount != 2 {
		t.Errorf("expected NodeCount=2, got %d", result.Stats.NodeCount)
	}
	if result.Stats.EdgeCount != 1 {
		t.Errorf("expected EdgeCount=1, got %d", result.Stats.EdgeCount)
	}
	if !result.Stats.HasMetadata {
		t.Error("expected HasMetadata=true")
	}
}

func TestValidateWorkflow_NilValidator(t *testing.T) {
	handler := &Handler{workflowValidator: nil}

	payload := map[string]any{
		"workflow": map[string]any{
			"nodes": []any{},
			"edges": []any{},
		},
		"strict": false,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	// Should return error status when validator is nil
	if rr.Code == http.StatusOK {
		t.Fatal("expected non-OK status when validator is nil")
	}
}
