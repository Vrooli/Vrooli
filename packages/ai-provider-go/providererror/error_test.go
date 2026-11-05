package providererror

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestMarkerRoundTrip(t *testing.T) {
	original := &Error{Code: CodeRateLimited, HTTPStatus: http.StatusTooManyRequests, RetryAfter: "7", Message: "slow down"}
	line, err := original.MarkerLine()
	if err != nil {
		t.Fatal(err)
	}
	parsed, ok := ParseMarker("noise\n" + string(line) + "\n")
	if !ok {
		t.Fatal("marker not parsed")
	}
	if parsed.Code != CodeRateLimited || parsed.HTTPStatus != http.StatusTooManyRequests || parsed.RetryAfter != "7" {
		t.Fatalf("parsed = %+v", parsed)
	}
}

func TestClassifyHTTPStatus(t *testing.T) {
	cases := map[int]string{
		http.StatusUnauthorized:       CodeUnconfigured,
		http.StatusForbidden:          CodeUnconfigured,
		http.StatusPaymentRequired:    CodeInsufficientCredits,
		http.StatusTooManyRequests:    CodeRateLimited,
		http.StatusServiceUnavailable: CodeProviderOverloaded,
		http.StatusBadRequest:         CodeProviderFailed,
	}
	for status, want := range cases {
		if got := ClassifyHTTPStatus(status); got != want {
			t.Errorf("status %d = %q, want %q", status, got, want)
		}
	}
}

func TestAsWrapped(t *testing.T) {
	wrapped := errors.New("outer")
	if _, ok := As(wrapped); ok {
		t.Fatal("plain error must not be typed")
	}
	typed := &Error{Code: CodeUnreachable}
	if _, ok := As(typed); !ok {
		t.Fatal("typed error not found")
	}
}

func TestMarkerIgnoresUnknownCode(t *testing.T) {
	if _, ok := ParseMarker(Marker + `{"code":"not_a_code"}`); ok {
		t.Fatal("unknown code must be ignored")
	}
	if !strings.Contains((&Error{Code: CodeProviderFailed, Err: errors.New("x")}).Error(), CodeProviderFailed) {
		t.Fatal("error text missing code")
	}
}
