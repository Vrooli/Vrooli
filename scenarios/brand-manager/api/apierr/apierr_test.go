package apierr

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidationError(t *testing.T) {
	e := Validation("name is required")
	if e.Code != CodeValidation {
		t.Errorf("Code = %q, want %q", e.Code, CodeValidation)
	}
	if e.StatusCode() != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", e.StatusCode(), http.StatusBadRequest)
	}
	if e.Recovery == "" {
		t.Error("expected non-empty Recovery hint")
	}
}

func TestNotFoundError(t *testing.T) {
	e := NotFound("brand")
	if e.Code != CodeNotFound {
		t.Errorf("Code = %q, want %q", e.Code, CodeNotFound)
	}
	if e.Message != "brand not found" {
		t.Errorf("Message = %q, want %q", e.Message, "brand not found")
	}
	if e.StatusCode() != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", e.StatusCode(), http.StatusNotFound)
	}
}

func TestInternalError(t *testing.T) {
	e := Internal("create brand", errors.New("disk full"))
	if e.Code != CodeInternal {
		t.Errorf("Code = %q, want %q", e.Code, CodeInternal)
	}
	if e.Message != "failed to create brand" {
		t.Errorf("Message = %q, want %q", e.Message, "failed to create brand")
	}
	if e.StatusCode() != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want %d", e.StatusCode(), http.StatusInternalServerError)
	}
}

func TestDependencyError(t *testing.T) {
	e := Dependency("database", errors.New("connection refused"))
	if e.Code != CodeDependency {
		t.Errorf("Code = %q, want %q", e.Code, CodeDependency)
	}
	if e.StatusCode() != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d, want %d", e.StatusCode(), http.StatusServiceUnavailable)
	}
}

func TestErrorInterface(t *testing.T) {
	e := Validation("bad input")
	var err error = e
	if err.Error() != "[validation] bad input" {
		t.Errorf("Error() = %q, want %q", err.Error(), "[validation] bad input")
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	e := NotFound("brand")
	Write(w, e)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got Error
	json.NewDecoder(w.Body).Decode(&got)
	if got.Code != CodeNotFound {
		t.Errorf("body code = %q, want %q", got.Code, CodeNotFound)
	}
	if got.Recovery == "" {
		t.Error("expected recovery hint in JSON body")
	}
}

// TestConflictError verifies the Conflict constructor and status mapping.
func TestConflictError(t *testing.T) {
	e := Conflict("brand has been modified")
	if e.Code != CodeConflict {
		t.Errorf("Code = %q, want %q", e.Code, CodeConflict)
	}
	if e.StatusCode() != http.StatusConflict {
		t.Errorf("StatusCode = %d, want %d", e.StatusCode(), http.StatusConflict)
	}
	if e.Recovery == "" {
		t.Error("expected non-empty Recovery hint")
	}
	if e.Message != "brand has been modified" {
		t.Errorf("Message = %q, want %q", e.Message, "brand has been modified")
	}
}
