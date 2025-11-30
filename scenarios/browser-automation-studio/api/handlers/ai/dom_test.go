package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

// =============================================================================
// Mock Automation Runner
// =============================================================================

type mockAutomationRunner struct {
	runFunc func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error)
}

func (m *mockAutomationRunner) Run(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, width, height, instructions)
	}
	return nil, nil, nil
}

// =============================================================================
// NewDOMHandler Tests
// =============================================================================

func TestNewDOMHandler(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] creates handler with logger", func(t *testing.T) {
		log := logrus.New()
		log.SetOutput(os.Stderr)

		handler := NewDOMHandler(log)

		require.NotNil(t, handler, "handler should not be nil")
		assert.Equal(t, log, handler.log, "handler should store the logger")
	})

	t.Run("creates handler with nil logger", func(t *testing.T) {
		handler := NewDOMHandler(nil)

		require.NotNil(t, handler, "handler should not be nil even with nil logger")
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] accepts custom runner option", func(t *testing.T) {
		log := logrus.New()
		log.SetOutput(os.Stderr)

		mockRunner := &mockAutomationRunner{}
		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))

		require.NotNil(t, handler)
		assert.Equal(t, mockRunner, handler.runner, "handler should use provided runner")
	})
}

// =============================================================================
// ExtractDOMTree URL Normalization Tests
// =============================================================================

func TestExtractDOMTree_URLNormalization(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] adds https:// to bare domain", func(t *testing.T) {
		var capturedInstructions []autocontracts.CompiledInstruction

		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				capturedInstructions = instructions
				return []autocontracts.StepOutcome{
					{
						NodeID:        "dom.extract",
						Success:       true,
						ExtractedData: map[string]any{"value": map[string]any{"tagName": "BODY"}},
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "example.com")

		require.NoError(t, err)
		require.NotEmpty(t, capturedInstructions)

		// Verify URL was normalized
		navigateParams := capturedInstructions[0].Params
		assert.Equal(t, "https://example.com", navigateParams["url"])
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] preserves http:// URLs", func(t *testing.T) {
		var capturedInstructions []autocontracts.CompiledInstruction

		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				capturedInstructions = instructions
				return []autocontracts.StepOutcome{
					{
						NodeID:        "dom.extract",
						Success:       true,
						ExtractedData: map[string]any{"value": map[string]any{"tagName": "BODY"}},
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "http://example.com")

		require.NoError(t, err)
		require.NotEmpty(t, capturedInstructions)

		navigateParams := capturedInstructions[0].Params
		assert.Equal(t, "http://example.com", navigateParams["url"])
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] preserves https:// URLs", func(t *testing.T) {
		var capturedInstructions []autocontracts.CompiledInstruction

		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				capturedInstructions = instructions
				return []autocontracts.StepOutcome{
					{
						NodeID:        "dom.extract",
						Success:       true,
						ExtractedData: map[string]any{"value": map[string]any{"tagName": "BODY"}},
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		require.NoError(t, err)
		require.NotEmpty(t, capturedInstructions)

		navigateParams := capturedInstructions[0].Params
		assert.Equal(t, "https://example.com", navigateParams["url"])
	})
}

// =============================================================================
// ExtractDOMTree Error Handling Tests
// =============================================================================

func TestExtractDOMTree_ErrorHandling(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles context cancellation", func(t *testing.T) {
		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				return nil, nil, ctx.Err()
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := handler.ExtractDOMTree(ctx, "https://example.com")

		assert.Error(t, err)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles runner error", func(t *testing.T) {
		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				return nil, nil, errors.New("runner failed")
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "automation run failed")
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles extraction failure", func(t *testing.T) {
		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				return []autocontracts.StepOutcome{
					{
						NodeID:  "dom.extract",
						Success: false,
						Failure: &autocontracts.StepFailure{Message: "extraction failed"},
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dom extraction failed")
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles nil extracted data", func(t *testing.T) {
		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				return []autocontracts.StepOutcome{
					{
						NodeID:        "dom.extract",
						Success:       true,
						ExtractedData: nil,
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "returned no data")
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles missing value payload", func(t *testing.T) {
		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				return []autocontracts.StepOutcome{
					{
						NodeID:        "dom.extract",
						Success:       true,
						ExtractedData: map[string]any{"other": "data"},
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing value payload")
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles no matching outcome", func(t *testing.T) {
		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				return []autocontracts.StepOutcome{
					{
						NodeID:  "other.node",
						Success: true,
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no dom extraction outcome")
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles nil runner", func(t *testing.T) {
		handler := &DOMHandler{log: log, runner: nil}
		_, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not configured")
	})
}

// =============================================================================
// ExtractDOMTree Success Tests
// =============================================================================

func TestExtractDOMTree_Success(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] returns JSON encoded DOM tree", func(t *testing.T) {
		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				return []autocontracts.StepOutcome{
					{
						NodeID:  "dom.extract",
						Success: true,
						ExtractedData: map[string]any{
							"value": map[string]any{
								"tagName":  "BODY",
								"id":       "main",
								"children": []any{},
							},
						},
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		result, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		require.NoError(t, err)
		assert.Contains(t, result, "tagName")
		assert.Contains(t, result, "BODY")
		assert.Contains(t, result, "main")

		// Verify it's valid JSON
		var parsed map[string]any
		err = json.Unmarshal([]byte(result), &parsed)
		require.NoError(t, err)
		assert.Equal(t, "BODY", parsed["tagName"])
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] passes correct viewport dimensions", func(t *testing.T) {
		var capturedWidth, capturedHeight int

		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				capturedWidth = width
				capturedHeight = height
				return []autocontracts.StepOutcome{
					{
						NodeID:        "dom.extract",
						Success:       true,
						ExtractedData: map[string]any{"value": map[string]any{}},
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		require.NoError(t, err)
		// Should use preview default viewport
		assert.Equal(t, previewDefaultViewportWidth, capturedWidth)
		assert.Equal(t, previewDefaultViewportHeight, capturedHeight)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] generates correct instruction sequence", func(t *testing.T) {
		var capturedInstructions []autocontracts.CompiledInstruction

		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				capturedInstructions = instructions
				return []autocontracts.StepOutcome{
					{
						NodeID:        "dom.extract",
						Success:       true,
						ExtractedData: map[string]any{"value": map[string]any{}},
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))
		_, err := handler.ExtractDOMTree(context.Background(), "https://example.com")

		require.NoError(t, err)
		require.Len(t, capturedInstructions, 3)

		// Verify instruction sequence
		assert.Equal(t, "navigate", capturedInstructions[0].Type)
		assert.Equal(t, "dom.navigate", capturedInstructions[0].NodeID)

		assert.Equal(t, "wait", capturedInstructions[1].Type)
		assert.Equal(t, "dom.wait", capturedInstructions[1].NodeID)

		assert.Equal(t, "evaluate", capturedInstructions[2].Type)
		assert.Equal(t, "dom.extract", capturedInstructions[2].NodeID)
	})
}

// =============================================================================
// GetDOMTree HTTP Handler Tests
// =============================================================================

func TestGetDOMTree_HTTPHandler(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects invalid JSON", func(t *testing.T) {
		handler := NewDOMHandler(log)

		req := httptest.NewRequest("POST", "/api/v1/dom-tree", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		handler.GetDOMTree(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "INVALID_REQUEST", response.Code)
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects missing URL", func(t *testing.T) {
		handler := NewDOMHandler(log)

		reqBody := map[string]string{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/dom-tree", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.GetDOMTree(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response APIError
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "MISSING_REQUIRED_FIELD", response.Code)

		detailsMap, ok := response.Details.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "url", detailsMap["field"])
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects empty URL", func(t *testing.T) {
		handler := NewDOMHandler(log)

		reqBody := map[string]string{"url": ""}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/dom-tree", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.GetDOMTree(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] returns DOM tree on success", func(t *testing.T) {
		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				return []autocontracts.StepOutcome{
					{
						NodeID:  "dom.extract",
						Success: true,
						ExtractedData: map[string]any{
							"value": map[string]any{
								"tagName": "BODY",
								"id":      "root",
							},
						},
					},
				}, nil, nil
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))

		reqBody := map[string]string{"url": "https://example.com"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/dom-tree", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.GetDOMTree(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var result map[string]any
		require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
		assert.Equal(t, "BODY", result["tagName"])
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] returns error on extraction failure", func(t *testing.T) {
		mockRunner := &mockAutomationRunner{
			runFunc: func(ctx context.Context, width, height int, instructions []autocontracts.CompiledInstruction) ([]autocontracts.StepOutcome, []autocontracts.EventEnvelope, error) {
				return nil, nil, errors.New("driver not available")
			},
		}

		handler := NewDOMHandler(log, WithDOMRunner(mockRunner))

		reqBody := map[string]string{"url": "https://example.com"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/dom-tree", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		handler.GetDOMTree(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response APIError
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, "INTERNAL_SERVER_ERROR", response.Code)
	})
}

// =============================================================================
// failureMessage Helper Tests
// =============================================================================

func TestFailureMessage(t *testing.T) {
	t.Run("returns message when present", func(t *testing.T) {
		failure := &autocontracts.StepFailure{Message: "specific error message"}
		result := failureMessage(failure)
		assert.Equal(t, "specific error message", result)
	})

	t.Run("trims whitespace from message", func(t *testing.T) {
		failure := &autocontracts.StepFailure{Message: "  error with whitespace  "}
		result := failureMessage(failure)
		assert.Equal(t, "error with whitespace", result)
	})

	t.Run("returns kind when message is empty", func(t *testing.T) {
		failure := &autocontracts.StepFailure{Kind: autocontracts.FailureKindTimeout}
		result := failureMessage(failure)
		assert.Equal(t, string(autocontracts.FailureKindTimeout), result)
	})

	t.Run("returns kind when message is whitespace-only", func(t *testing.T) {
		failure := &autocontracts.StepFailure{Message: "   ", Kind: autocontracts.FailureKindEngine}
		result := failureMessage(failure)
		assert.Equal(t, string(autocontracts.FailureKindEngine), result)
	})

	t.Run("returns unknown for nil failure", func(t *testing.T) {
		result := failureMessage(nil)
		assert.Equal(t, "unknown failure", result)
	})

	t.Run("returns unknown when both message and kind are empty", func(t *testing.T) {
		failure := &autocontracts.StepFailure{}
		result := failureMessage(failure)
		assert.Equal(t, "unknown failure", result)
	})
}

// =============================================================================
// DOM Extraction Expression Validation Tests
// =============================================================================

func TestDOMExtractionExpression(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] expression contains MAX_DEPTH limit", func(t *testing.T) {
		assert.Contains(t, domExtractionExpression, "MAX_DEPTH = 6")
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] expression contains MAX_CHILDREN_PER_NODE limit", func(t *testing.T) {
		assert.Contains(t, domExtractionExpression, "MAX_CHILDREN_PER_NODE = 12")
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] expression contains MAX_TOTAL_NODES limit", func(t *testing.T) {
		assert.Contains(t, domExtractionExpression, "MAX_TOTAL_NODES = 800")
	})

	t.Run("[REQ:BAS-AI-GENERATION-SMOKE] expression contains TEXT_LIMIT", func(t *testing.T) {
		assert.Contains(t, domExtractionExpression, "TEXT_LIMIT = 120")
	})

	t.Run("expression skips unwanted tags", func(t *testing.T) {
		assert.Contains(t, domExtractionExpression, "'script'")
		assert.Contains(t, domExtractionExpression, "'style'")
		assert.Contains(t, domExtractionExpression, "'noscript'")
	})

	t.Run("expression is a valid IIFE", func(t *testing.T) {
		assert.True(t, len(domExtractionExpression) > 0)
		assert.Contains(t, domExtractionExpression, "(function()")
		assert.Contains(t, domExtractionExpression, ")()")
	})
}

// =============================================================================
// Constants Tests
// =============================================================================

func TestDOMHandlerConstants(t *testing.T) {
	t.Run("domExtractionNodeID is set", func(t *testing.T) {
		assert.Equal(t, "dom.extract", domExtractionNodeID)
	})

	t.Run("defaultDomExtractionWaitMs is reasonable", func(t *testing.T) {
		assert.Equal(t, 750, defaultDomExtractionWaitMs)
		assert.Greater(t, defaultDomExtractionWaitMs, 0)
		assert.Less(t, defaultDomExtractionWaitMs, 10000)
	})
}
