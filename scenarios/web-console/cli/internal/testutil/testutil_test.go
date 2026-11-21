package testutil

import (
	"errors"
	"testing"
)

func TestErrorMessageAcceptsNil(t *testing.T) {
	if got := ErrorMessage(nil, "nil error"); got != "" {
		t.Fatalf("ErrorMessage(nil) = %q", got)
	}
}

func TestErrorMessageIncludesContext(t *testing.T) {
	if got := ErrorMessage(errors.New("sentinel"), "expected failure"); got != "expected failure: sentinel" {
		t.Fatalf("ErrorMessage(error) = %q", got)
	}
}
