package apierr

import (
	"errors"
	"net/http"
	"testing"
)

func TestDomainErrorImplementsError(t *testing.T) {
	err := NotFound("item %q not found", "test")
	if err.Error() != `item "test" not found` {
		t.Errorf("unexpected message: %s", err.Error())
	}
}

func TestDomainErrorUnwrap(t *testing.T) {
	err := NotFound("gone")
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected errors.Is(err, ErrNotFound) to be true")
	}
}

func TestErrorsAs(t *testing.T) {
	err := BadRequest("invalid input")
	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		t.Fatal("expected errors.As to succeed")
	}
	if domainErr.Status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", domainErr.Status)
	}
}

func TestConvenienceConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *DomainError
		sentinel error
		status   int
	}{
		{"NotFound", NotFound("x"), ErrNotFound, 404},
		{"AlreadyExists", AlreadyExists("x"), ErrAlreadyExists, 409},
		{"Conflict", Conflict("x"), ErrConflict, 409},
		{"BadRequest", BadRequest("x"), ErrBadRequest, 400},
		{"Unavailable", Unavailable("x"), ErrServiceUnavailable, 503},
		{"BadGateway", BadGateway("x"), ErrBadGateway, 502},
		{"Internal", Internal("x"), nil, 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, tt.err.Status)
			}
			if tt.sentinel != nil && !errors.Is(tt.err, tt.sentinel) {
				t.Errorf("expected errors.Is to match sentinel")
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	err := Wrapf(ErrCircuitBroken, http.StatusConflict, "breaker tripped for %s", "item-1")
	if err.Message != "breaker tripped for item-1" {
		t.Errorf("unexpected message: %s", err.Message)
	}
	if !errors.Is(err, ErrCircuitBroken) {
		t.Error("expected errors.Is to match ErrCircuitBroken")
	}
}
