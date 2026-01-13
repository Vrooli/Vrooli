package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainerrors "scenario-to-desktop-api/shared/errors"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		payload    interface{}
		expectCode int
	}{
		{
			name:       "success with payload",
			status:     http.StatusOK,
			payload:    map[string]string{"key": "value"},
			expectCode: http.StatusOK,
		},
		{
			name:       "created status",
			status:     http.StatusCreated,
			payload:    map[string]int{"id": 123},
			expectCode: http.StatusCreated,
		},
		{
			name:       "nil payload",
			status:     http.StatusOK,
			payload:    nil,
			expectCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			WriteJSON(rr, tc.status, tc.payload)

			if rr.Code != tc.expectCode {
				t.Errorf("expected status %d, got %d", tc.expectCode, rr.Code)
			}

			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got %q", contentType)
			}

			if tc.payload != nil {
				var result map[string]interface{}
				if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
			}
		})
	}
}

func TestWriteJSONOK(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSONOK(rr, map[string]string{"status": "ok"})

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestWriteJSONCreated(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSONCreated(rr, map[string]string{"id": "new-resource"})

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}
}

func TestWriteJSONAccepted(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSONAccepted(rr, map[string]string{"job_id": "123"})

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", rr.Code)
	}
}

func TestWriteJSONNoContent(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteJSONNoContent(rr)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", rr.Code)
	}

	if rr.Body.Len() != 0 {
		t.Errorf("expected empty body, got %d bytes", rr.Body.Len())
	}
}

func TestWriteSuccess(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteSuccess(rr, "operation completed")

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp SuccessResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.Message != "operation completed" {
		t.Errorf("expected message 'operation completed', got %q", resp.Message)
	}
}

func TestWriteError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		rr := httptest.NewRecorder()
		WriteError(rr, nil)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}

		var resp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Error != "unknown error" {
			t.Errorf("expected error 'unknown error', got %q", resp.Error)
		}
	})

	t.Run("domain error not found", func(t *testing.T) {
		rr := httptest.NewRecorder()
		WriteError(rr, domainerrors.ErrNotFound("resource"))

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("domain error bad request", func(t *testing.T) {
		rr := httptest.NewRecorder()
		WriteError(rr, domainerrors.ErrBadRequest("invalid input"))

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("domain error validation", func(t *testing.T) {
		rr := httptest.NewRecorder()
		details := map[string]interface{}{"field": "name is required"}
		WriteError(rr, domainerrors.ErrValidation("validation failed", details))

		if rr.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status 422, got %d", rr.Code)
		}

		var resp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Details == nil {
			t.Error("expected details to be set")
		}
	})

	t.Run("non-domain error", func(t *testing.T) {
		rr := httptest.NewRecorder()
		WriteError(rr, errors.New("something went wrong"))

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}

		var resp ErrorResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Error != "something went wrong" {
			t.Errorf("expected error message, got %q", resp.Error)
		}
	})
}

func TestWriteErrorWithStatus(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		rr := httptest.NewRecorder()
		WriteErrorWithStatus(rr, http.StatusTeapot, nil)

		if rr.Code != http.StatusTeapot {
			t.Errorf("expected status 418, got %d", rr.Code)
		}
	})

	t.Run("domain error with custom status", func(t *testing.T) {
		rr := httptest.NewRecorder()
		WriteErrorWithStatus(rr, http.StatusConflict, domainerrors.ErrBadRequest("conflict"))

		if rr.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", rr.Code)
		}
	})

	t.Run("regular error with custom status", func(t *testing.T) {
		rr := httptest.NewRecorder()
		WriteErrorWithStatus(rr, http.StatusServiceUnavailable, errors.New("service down"))

		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", rr.Code)
		}
	})
}

func TestWriteNotFound(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteNotFound(rr, "build")

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestWriteBadRequest(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteBadRequest(rr, "invalid parameter")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestWriteValidationError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteValidationError(rr, "validation failed", map[string]interface{}{"field": "required"})

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", rr.Code)
	}
}

func TestWriteInternalError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteInternalError(rr, "database connection failed")

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}

func TestWriteUnavailable(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteUnavailable(rr, "external-api")

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rr.Code)
	}
}

func TestDecodeJSON(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		body := strings.NewReader(`{"name": "test", "value": 42}`)
		req := httptest.NewRequest(http.MethodPost, "/", body)
		rr := httptest.NewRecorder()

		var target struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		ok := DecodeJSON(rr, req, &target)
		if !ok {
			t.Error("expected DecodeJSON to return true")
		}
		if target.Name != "test" {
			t.Errorf("expected name 'test', got %q", target.Name)
		}
		if target.Value != 42 {
			t.Errorf("expected value 42, got %d", target.Value)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		body := strings.NewReader(`{invalid}`)
		req := httptest.NewRequest(http.MethodPost, "/", body)
		rr := httptest.NewRecorder()

		var target map[string]interface{}
		ok := DecodeJSON(rr, req, &target)
		if ok {
			t.Error("expected DecodeJSON to return false")
		}
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestDecodeJSONStrict(t *testing.T) {
	t.Run("valid json without unknown fields", func(t *testing.T) {
		body := strings.NewReader(`{"name": "test"}`)
		req := httptest.NewRequest(http.MethodPost, "/", body)
		rr := httptest.NewRecorder()

		var target struct {
			Name string `json:"name"`
		}

		ok := DecodeJSONStrict(rr, req, &target)
		if !ok {
			t.Error("expected DecodeJSONStrict to return true")
		}
	})

	t.Run("json with unknown fields", func(t *testing.T) {
		body := strings.NewReader(`{"name": "test", "unknown": "field"}`)
		req := httptest.NewRequest(http.MethodPost, "/", body)
		rr := httptest.NewRecorder()

		var target struct {
			Name string `json:"name"`
		}

		ok := DecodeJSONStrict(rr, req, &target)
		if ok {
			t.Error("expected DecodeJSONStrict to return false for unknown fields")
		}
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestRequireParam(t *testing.T) {
	t.Run("present param", func(t *testing.T) {
		rr := httptest.NewRecorder()
		ok := RequireParam(rr, "build_id", "123")
		if !ok {
			t.Error("expected RequireParam to return true")
		}
	})

	t.Run("missing param", func(t *testing.T) {
		rr := httptest.NewRecorder()
		ok := RequireParam(rr, "build_id", "")
		if ok {
			t.Error("expected RequireParam to return false")
		}
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

func TestRequireParams(t *testing.T) {
	t.Run("all present", func(t *testing.T) {
		rr := httptest.NewRecorder()
		params := map[string]string{
			"build_id": "123",
			"platform": "linux",
		}
		ok := RequireParams(rr, params)
		if !ok {
			t.Error("expected RequireParams to return true")
		}
	})

	t.Run("one missing", func(t *testing.T) {
		rr := httptest.NewRecorder()
		params := map[string]string{
			"build_id": "123",
			"platform": "",
		}
		ok := RequireParams(rr, params)
		if ok {
			t.Error("expected RequireParams to return false")
		}
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}
