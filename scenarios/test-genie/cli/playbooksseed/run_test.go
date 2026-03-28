package playbooksseed

import (
	"strings"
	"testing"
)

func TestRunRejectsNilClient(t *testing.T) {
	err := Run(nil, []string{"apply", "--scenario", "demo"})
	if err == nil {
		t.Fatal("expected nil client to fail")
	}
	if !strings.Contains(err.Error(), "client is required") {
		t.Fatalf("expected client validation error, got %v", err)
	}
}

func TestRunRejectsMissingArguments(t *testing.T) {
	err := Run(&Client{}, nil)
	if err == nil {
		t.Fatal("expected empty args to fail")
	}
	if !strings.Contains(err.Error(), "usage: playbooks-seed") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestRunRejectsMissingScenario(t *testing.T) {
	err := Run(&Client{}, []string{"apply"})
	if err == nil {
		t.Fatal("expected missing scenario to fail")
	}
	if !strings.Contains(err.Error(), "--scenario is required") {
		t.Fatalf("expected scenario validation error, got %v", err)
	}
}

func TestRunRejectsCleanupWithoutToken(t *testing.T) {
	err := Run(&Client{}, []string{"cleanup", "--scenario", "demo"})
	if err == nil {
		t.Fatal("expected missing cleanup token to fail")
	}
	if !strings.Contains(err.Error(), "--token is required for cleanup") {
		t.Fatalf("expected token validation error, got %v", err)
	}
}

func TestRunRejectsUnknownOptionAndAction(t *testing.T) {
	err := Run(&Client{}, []string{"apply", "--scenario", "demo", "--bogus"})
	if err == nil {
		t.Fatal("expected unknown option to fail")
	}
	if !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("expected unknown option error, got %v", err)
	}

	err = Run(&Client{}, []string{"reseed", "--scenario", "demo"})
	if err == nil {
		t.Fatal("expected unknown action to fail")
	}
	if !strings.Contains(err.Error(), "unknown action") {
		t.Fatalf("expected unknown action error, got %v", err)
	}
}
