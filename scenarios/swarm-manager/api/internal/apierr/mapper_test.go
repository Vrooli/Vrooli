package apierr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMapError_DomainError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "not found",
			err:        NotFound("item gone"),
			wantStatus: 404,
			wantBody:   "item gone",
		},
		{
			name:       "bad request",
			err:        BadRequest("invalid field"),
			wantStatus: 400,
			wantBody:   "invalid field",
		},
		{
			name:       "conflict",
			err:        Conflict("already queued"),
			wantStatus: 409,
			wantBody:   "already queued",
		},
		{
			name:       "unavailable",
			err:        Unavailable("agent down"),
			wantStatus: 503,
			wantBody:   "agent down",
		},
		{
			name:       "bad gateway",
			err:        BadGateway("upstream failed"),
			wantStatus: 502,
			wantBody:   "upstream failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			MapError(w, "[test]", tt.err)
			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", w.Code, tt.wantStatus)
			}
			body := strings.TrimSpace(w.Body.String())
			if body != tt.wantBody {
				t.Errorf("body: got %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestMapError_UntypedError(t *testing.T) {
	w := httptest.NewRecorder()
	MapError(w, "[test]", errors.New("something broke"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	if body != "something broke" {
		t.Errorf("body: got %q, want %q", body, "something broke")
	}
}

func TestMapError_WrappedDomainError(t *testing.T) {
	inner := NotFound("original")
	wrapped := errors.Join(errors.New("context"), inner)
	// errors.As should still find the DomainError through Join
	w := httptest.NewRecorder()
	MapError(w, "", wrapped)
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestMapError_NilLogPrefix(t *testing.T) {
	w := httptest.NewRecorder()
	MapError(w, "", NotFound("gone"))
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}

func TestTruncateMessage(t *testing.T) {
	long := strings.Repeat("x", 300)
	result := truncateMessage(errors.New(long), 240)
	if len(result) != 243 { // 240 + "..."
		t.Errorf("expected truncated length 243, got %d", len(result))
	}

	if truncateMessage(nil, 100) != "unknown error" {
		t.Error("expected 'unknown error' for nil")
	}
}
