package bats

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunner_Success(t *testing.T) {
	h := newBatsTestHarness(t)

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.PrimarySuite == "" {
		t.Error("expected primary suite to be set")
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

func TestRunner_BatsNotInstalled(t *testing.T) {
	h := newBatsTestHarness(t)

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithLookup(func(name string) (string, error) {
			return "", errors.New("bats not found")
		}),
	)

	result := r.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when bats not installed")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Errorf("expected missing_dependency, got: %s", result.FailureClass)
	}
	if !strings.Contains(result.Error.Error(), "bats") {
		t.Errorf("expected bats error, got: %v", result.Error)
	}
}

func TestRunner_NoPrimarySuiteFound(t *testing.T) {
	h := newBatsTestHarness(t)
	// Remove all .bats files
	os.Remove(filepath.Join(h.scenarioDir, "cli", h.scenarioName+".bats"))

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when no primary suite found")
	}
	if result.FailureClass != "misconfiguration" {
		t.Errorf("expected misconfiguration, got: %s", result.FailureClass)
	}
}

func TestRunner_PrimarySuiteFails(t *testing.T) {
	h := newBatsTestHarness(t)

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			return errors.New("test failed")
		}),
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when primary suite fails")
	}
	if result.FailureClass != "system" {
		t.Errorf("expected system, got: %s", result.FailureClass)
	}
	if result.PrimarySuite == "" {
		t.Error("expected primary suite path to be captured even on failure")
	}
}

func TestRunner_AdditionalSuitesFails(t *testing.T) {
	h := newBatsTestHarness(t)
	h.addAdditionalSuite(t, "extra.bats")

	callCount := 0
	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(func(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
			callCount++
			if callCount > 1 {
				return errors.New("additional suite failed")
			}
			return nil
		}),
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when additional suite fails")
	}
	if !strings.Contains(result.Error.Error(), "additional suite failed") {
		t.Errorf("expected additional suite error, got: %v", result.Error)
	}
}

func TestRunner_RunsAdditionalSuites(t *testing.T) {
	h := newBatsTestHarness(t)
	h.addAdditionalSuite(t, "extra1.bats")
	h.addAdditionalSuite(t, "extra2.bats")

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.AdditionalSuitesRun != 2 {
		t.Errorf("expected 2 additional suites, got: %d", result.AdditionalSuitesRun)
	}
}

func TestRunner_NoAdditionalSuitesDirectory(t *testing.T) {
	h := newBatsTestHarness(t)
	// Don't create cli/test directory

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success without test dir, got error: %v", result.Error)
	}
	if result.AdditionalSuitesRun != 0 {
		t.Errorf("expected 0 additional suites, got: %d", result.AdditionalSuitesRun)
	}
}

func TestRunner_ContextCancelled(t *testing.T) {
	h := newBatsTestHarness(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithLookup(h.successLookup),
	)

	result := r.Run(ctx)

	if result.Success {
		t.Fatal("expected failure when context cancelled")
	}
	if result.FailureClass != "system" {
		t.Errorf("expected system failure class, got: %s", result.FailureClass)
	}
}

func TestRunner_FindsScenarioNamedSuite(t *testing.T) {
	h := newBatsTestHarness(t)

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	expected := filepath.Join(h.scenarioDir, "cli", h.scenarioName+".bats")
	if result.PrimarySuite != expected {
		t.Errorf("expected primary suite %s, got: %s", expected, result.PrimarySuite)
	}
}

func TestRunner_FindsFallbackSuite(t *testing.T) {
	h := newBatsTestHarness(t)
	// Remove scenario-named suite, add different one
	os.Remove(filepath.Join(h.scenarioDir, "cli", h.scenarioName+".bats"))
	writeBatsFile(t, filepath.Join(h.scenarioDir, "cli", "other.bats"))

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if !strings.Contains(result.PrimarySuite, "other.bats") {
		t.Errorf("expected fallback suite, got: %s", result.PrimarySuite)
	}
}

func TestRunner_NoExecutorConfigured(t *testing.T) {
	h := newBatsTestHarness(t)

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		// No executor set
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure when executor not configured")
	}
	if !strings.Contains(result.Error.Error(), "executor not configured") {
		t.Errorf("expected executor error, got: %v", result.Error)
	}
}

func TestRunner_SkipsNonBatsFiles(t *testing.T) {
	h := newBatsTestHarness(t)
	testDir := filepath.Join(h.scenarioDir, "cli", "test")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	// Add non-bats files
	os.WriteFile(filepath.Join(testDir, "helper.sh"), []byte("#!/bin/bash"), 0o644)
	os.WriteFile(filepath.Join(testDir, "README.md"), []byte("# Tests"), 0o644)

	r := New(
		Config{
			ScenarioDir:  h.scenarioDir,
			ScenarioName: h.scenarioName,
		},
		WithLogger(io.Discard),
		WithExecutor(h.successExecutor),
		WithLookup(h.successLookup),
	)

	result := r.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.AdditionalSuitesRun != 0 {
		t.Errorf("expected 0 additional suites (non-bats files skipped), got: %d", result.AdditionalSuitesRun)
	}
}

// Test helpers

type batsTestHarness struct {
	scenarioDir  string
	scenarioName string
}

func newBatsTestHarness(t *testing.T) *batsTestHarness {
	t.Helper()
	tmpDir := t.TempDir()
	scenarioName := "demo"
	scenarioDir := filepath.Join(tmpDir, "scenarios", scenarioName)

	// Create cli directory
	cliDir := filepath.Join(scenarioDir, "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("failed to create cli dir: %v", err)
	}

	// Create primary bats suite
	writeBatsFile(t, filepath.Join(cliDir, scenarioName+".bats"))

	return &batsTestHarness{
		scenarioDir:  scenarioDir,
		scenarioName: scenarioName,
	}
}

func (h *batsTestHarness) addAdditionalSuite(t *testing.T, name string) {
	t.Helper()
	testDir := filepath.Join(h.scenarioDir, "cli", "test")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	writeBatsFile(t, filepath.Join(testDir, name))
}

func (h *batsTestHarness) successExecutor(ctx context.Context, dir string, w io.Writer, name string, args ...string) error {
	return nil
}

func (h *batsTestHarness) successLookup(name string) (string, error) {
	return "/usr/bin/" + name, nil
}

func writeBatsFile(t *testing.T, path string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	content := "#!/usr/bin/env bats\n\n@test \"sample\" {\n  true\n}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write bats file: %v", err)
	}
}

// Ensure runner implements interface at compile time
var _ Runner = (*runner)(nil)
