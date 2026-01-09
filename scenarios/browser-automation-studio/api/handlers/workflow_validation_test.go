package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	workflowvalidator "github.com/vrooli/browser-automation-studio/workflow/validator"
)

// createTestHandlerWithValidator creates a handler with a real workflow validator for testing.
func createTestHandlerWithValidator() *Handler {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	validator, err := workflowvalidator.NewValidator()
	if err != nil {
		panic("failed to create validator: " + err.Error())
	}

	return &Handler{
		workflowValidator: validator,
		log:               log,
	}
}

// ============================================================================
// ValidateWorkflow Tests
// ============================================================================

func TestValidateWorkflow_Success(t *testing.T) {
	handler := createTestHandlerWithValidator()

	body := `{
		"workflow": {
			"nodes": [
				{
					"id": "node-1",
					"type": "navigate",
					"position": {"x": 0, "y": 0},
					"data": {
						"url": "https://example.com"
					}
				}
			],
			"edges": []
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestValidateWorkflow_InvalidJSON(t *testing.T) {
	handler := createTestHandlerWithValidator()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestValidateWorkflow_NilWorkflow(t *testing.T) {
	handler := createTestHandlerWithValidator()

	body := `{"workflow": null}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for nil workflow, got %d", rr.Code)
	}
}

func TestValidateWorkflow_EmptyWorkflow(t *testing.T) {
	handler := createTestHandlerWithValidator()

	body := `{"workflow": {}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	// Empty workflow should still validate (may have warnings/errors but won't crash)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 for empty workflow, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestValidateWorkflow_NilHandler(t *testing.T) {
	var handler *Handler = nil

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// This should not panic
	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 for nil handler, got %d", rr.Code)
	}
}

func TestValidateWorkflow_NilValidator(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := &Handler{
		workflowValidator: nil,
		log:               log,
	}

	body := `{"workflow": {}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 for nil validator, got %d", rr.Code)
	}
}

func TestValidateWorkflow_V2Format(t *testing.T) {
	handler := createTestHandlerWithValidator()

	// V2 format has 'action' field in nodes - note: proto unmarshal is strict,
	// so only include proto-compatible fields (no 'type' field at node level)
	body := `{
		"workflow": {
			"nodes": [
				{
					"id": "node-1",
					"position": {"x": 0, "y": 0},
					"action": {
						"navigate": {
							"url": "https://example.com"
						}
					}
				}
			],
			"edges": []
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 for V2 format, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestValidateWorkflow_StrictMode(t *testing.T) {
	handler := createTestHandlerWithValidator()

	body := `{
		"workflow": {
			"nodes": [],
			"edges": []
		},
		"strict": true
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateWorkflow(rr, req)

	// Should still return 200 even in strict mode (strict affects validation result, not response code)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 with strict mode, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// ValidateResolvedWorkflow Tests
// ============================================================================

func TestValidateResolvedWorkflow_Success(t *testing.T) {
	handler := createTestHandlerWithValidator()

	body := `{
		"workflow": {
			"nodes": [
				{
					"id": "node-1",
					"type": "navigate",
					"position": {"x": 0, "y": 0},
					"data": {
						"url": "https://example.com"
					}
				}
			],
			"edges": []
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate-resolved", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateResolvedWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestValidateResolvedWorkflow_InvalidJSON(t *testing.T) {
	handler := createTestHandlerWithValidator()

	body := `{invalid json`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate-resolved", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateResolvedWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestValidateResolvedWorkflow_NilWorkflow(t *testing.T) {
	handler := createTestHandlerWithValidator()

	body := `{"workflow": null}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate-resolved", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateResolvedWorkflow(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for nil workflow, got %d", rr.Code)
	}
}

func TestValidateResolvedWorkflow_NilHandler(t *testing.T) {
	var handler *Handler = nil

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate-resolved", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateResolvedWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 for nil handler, got %d", rr.Code)
	}
}

func TestValidateResolvedWorkflow_NilValidator(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := &Handler{
		workflowValidator: nil,
		log:               log,
	}

	body := `{"workflow": {}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate-resolved", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateResolvedWorkflow(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 for nil validator, got %d", rr.Code)
	}
}

func TestValidateResolvedWorkflow_V2Format(t *testing.T) {
	handler := createTestHandlerWithValidator()

	// V2 format with proto-compatible fields only (no 'type' field at node level)
	body := `{
		"workflow": {
			"nodes": [
				{
					"id": "node-1",
					"position": {"x": 0, "y": 0},
					"action": {
						"navigate": {
							"url": "https://example.com"
						}
					}
				}
			],
			"edges": []
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/validate-resolved", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ValidateResolvedWorkflow(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 for V2 format, got %d: %s", rr.Code, rr.Body.String())
	}
}

// ============================================================================
// looksLikeWorkflowDefinitionV2 Tests
// ============================================================================

func TestLooksLikeWorkflowDefinitionV2_WithAction(t *testing.T) {
	doc := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "node-1",
				"type": "navigate",
				"action": map[string]any{
					"navigate": map[string]any{"url": "https://example.com"},
				},
			},
		},
	}

	if !looksLikeWorkflowDefinitionV2(doc) {
		t.Error("expected looksLikeWorkflowDefinitionV2 to return true for V2 format")
	}
}

func TestLooksLikeWorkflowDefinitionV2_WithoutAction(t *testing.T) {
	doc := map[string]any{
		"nodes": []any{
			map[string]any{
				"id":   "node-1",
				"type": "navigate",
				"data": map[string]any{"url": "https://example.com"},
			},
		},
	}

	if looksLikeWorkflowDefinitionV2(doc) {
		t.Error("expected looksLikeWorkflowDefinitionV2 to return false for V1 format")
	}
}

func TestLooksLikeWorkflowDefinitionV2_EmptyNodes(t *testing.T) {
	doc := map[string]any{
		"nodes": []any{},
	}

	if looksLikeWorkflowDefinitionV2(doc) {
		t.Error("expected looksLikeWorkflowDefinitionV2 to return false for empty nodes")
	}
}

func TestLooksLikeWorkflowDefinitionV2_NoNodes(t *testing.T) {
	doc := map[string]any{}

	if looksLikeWorkflowDefinitionV2(doc) {
		t.Error("expected looksLikeWorkflowDefinitionV2 to return false for missing nodes")
	}
}

func TestLooksLikeWorkflowDefinitionV2_InvalidNodeType(t *testing.T) {
	doc := map[string]any{
		"nodes": []any{
			"not-a-map",
		},
	}

	if looksLikeWorkflowDefinitionV2(doc) {
		t.Error("expected looksLikeWorkflowDefinitionV2 to return false for invalid node type")
	}
}
