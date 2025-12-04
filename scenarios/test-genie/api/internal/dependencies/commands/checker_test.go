package commands

import (
	"fmt"
	"io"
	"testing"
)

func TestCheckerCheckSuccess(t *testing.T) {
	c := New(io.Discard, WithLookup(func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}))

	path, err := c.Check("bash")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if path != "/usr/bin/bash" {
		t.Fatalf("expected /usr/bin/bash, got %s", path)
	}
}

func TestCheckerCheckFailure(t *testing.T) {
	c := New(io.Discard, WithLookup(func(name string) (string, error) {
		return "", fmt.Errorf("not found")
	}))

	_, err := c.Check("nonexistent")
	if err == nil {
		t.Fatalf("expected error for missing command")
	}
}

func TestCheckerCheckAllSuccess(t *testing.T) {
	c := New(io.Discard, WithLookup(func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}))

	result := c.CheckAll([]CommandRequirement{
		{Name: "bash", Reason: "shell"},
		{Name: "curl", Reason: "http"},
	})

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Verified) != 2 {
		t.Fatalf("expected 2 verified commands, got %d", len(result.Verified))
	}
	if len(result.Missing) != 0 {
		t.Fatalf("expected no missing commands, got %d", len(result.Missing))
	}
}

func TestCheckerCheckAllPartialFailure(t *testing.T) {
	c := New(io.Discard, WithLookup(func(name string) (string, error) {
		if name == "bash" {
			return "/usr/bin/bash", nil
		}
		return "", fmt.Errorf("not found")
	}))

	result := c.CheckAll([]CommandRequirement{
		{Name: "bash", Reason: "shell"},
		{Name: "missing", Reason: "test"},
	})

	if result.Success {
		t.Fatalf("expected failure for missing command")
	}
	if len(result.Verified) != 1 {
		t.Fatalf("expected 1 verified command, got %d", len(result.Verified))
	}
	if len(result.Missing) != 1 {
		t.Fatalf("expected 1 missing command, got %d", len(result.Missing))
	}
	if result.Missing[0].Name != "missing" {
		t.Fatalf("expected missing command 'missing', got '%s'", result.Missing[0].Name)
	}
}

func TestCheckerCheckAllGeneratesObservations(t *testing.T) {
	c := New(io.Discard, WithLookup(func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}))

	result := c.CheckAll([]CommandRequirement{
		{Name: "bash", Reason: "shell"},
	})

	if len(result.Observations) == 0 {
		t.Fatalf("expected observations to be recorded")
	}
}

func TestBaselineRequirements(t *testing.T) {
	reqs := BaselineRequirements()
	if len(reqs) != 3 {
		t.Fatalf("expected 3 baseline requirements, got %d", len(reqs))
	}

	names := make(map[string]bool)
	for _, req := range reqs {
		names[req.Name] = true
	}

	for _, expected := range []string{"bash", "curl", "jq"} {
		if !names[expected] {
			t.Fatalf("expected baseline requirement '%s' not found", expected)
		}
	}
}
