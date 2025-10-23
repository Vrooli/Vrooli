package handlers

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONReturnsRequestTooLarge(t *testing.T) {
	t.Helper()

	bigPayload := strings.Repeat("a", 2048)
	body := "{\"data\":\"" + bigPayload + "\"}"
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))

	var dst struct{}
	if err := DecodeJSON(req, &dst, 512); !errors.Is(err, ErrRequestTooLarge) {
		t.Fatalf("expected ErrRequestTooLarge, got %v", err)
	}
}
