package config

import (
	"errors"
	"testing"
)

func TestNewMissingPreservesConfigurationContext(t *testing.T) {
	err := NewMissing("AGENT_MANAGER_PORT", "must be set", nil)
	if !err.Missing {
		t.Fatal("missing configuration error must be marked missing")
	}
	if got, want := err.Error(), "config error for AGENT_MANAGER_PORT: must be set"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestNewInvalidWrapsCause(t *testing.T) {
	cause := errors.New("not a number")
	err := NewInvalid("AGENT_MANAGER_PORT", "must be numeric", cause)
	if err.Missing {
		t.Fatal("invalid configuration error must not be marked missing")
	}
	if !errors.Is(err, cause) {
		t.Fatal("invalid configuration error must preserve its cause")
	}
}
