package scenariohandlers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/scenariostale"
)

func TestExtractCompletenessScenarioHandlesScoreSubcommands(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"score", "get", "algorithm-library"}, "algorithm-library"},
		{[]string{"score", "calculate", "foo"}, "foo"},
		{[]string{"score", "validation", "bar"}, "bar"},
		{[]string{"score", "history", "baz"}, "baz"},
		{[]string{"score", "trends", "qux"}, "qux"},
		{[]string{"score", "recommend", "zap"}, "zap"},
		{[]string{"--json", "score", "get", "algorithm-library"}, "algorithm-library"},
		{[]string{"--api-base", "http://x", "score", "get", "algorithm-library"}, "algorithm-library"},
		{[]string{"--auto-start", "score", "get", "algorithm-library"}, "algorithm-library"},
		{[]string{"score", "list"}, ""},
		{[]string{"score", "refresh-all"}, ""},
		{[]string{"config", "show"}, ""},
		{[]string{"monitoring", "status"}, ""},
		{[]string{}, ""},
		{[]string{"score", "get"}, ""},
	}
	for _, c := range cases {
		got := extractCompletenessScenario(c.args)
		if got != c.want {
			t.Errorf("extractCompletenessScenario(%v) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestExtractUISmokeScenarioConsumesValuedFlags(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"test-genie"}, "test-genie"},
		{[]string{"--json", "test-genie"}, "test-genie"},
		{[]string{"--url", "http://x", "test-genie"}, "test-genie"},
		{[]string{"--browserless", "http://b", "test-genie", "--json"}, "test-genie"},
		{[]string{"--timeout", "90000", "test-genie"}, "test-genie"},
		{[]string{"--url=http://x", "test-genie"}, "test-genie"},
		{[]string{"--no-recovery", "--shared-mode", "test-genie"}, "test-genie"},
		{[]string{"--url", "http://x"}, ""},
		{[]string{}, ""},
	}
	for _, c := range cases {
		if got := extractUISmokeScenario(c.args); got != c.want {
			t.Errorf("extractUISmokeScenario(%v) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestFirstPositionalArgSkipsFlags(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"algorithm-library"}, "algorithm-library"},
		{[]string{"--json", "algorithm-library"}, "algorithm-library"},
		{[]string{"-h"}, ""},
		{[]string{}, ""},
		{[]string{"--flag", "--json", "foo", "bar"}, "foo"},
	}
	for _, c := range cases {
		if got := firstPositionalArg(c.args); got != c.want {
			t.Errorf("firstPositionalArg(%v) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestEmitScenarioStaleWarningRespectsGlobalFlag(t *testing.T) {
	root, name := scaffoldScenarioTree(t)
	writeStaleSidecar(t, filepath.Join(root, "scenarios", name))

	var buf bytes.Buffer
	emitScenarioStaleWarning(&buf, root, name, rootcli.GlobalOptions{NoStaleCheck: true})
	if buf.Len() != 0 {
		t.Fatalf("--no-stale-check should silence warnings; got %q", buf.String())
	}

	emitScenarioStaleWarning(&buf, root, name, rootcli.GlobalOptions{})
	if !strings.Contains(buf.String(), "WARNING: scenario") {
		t.Fatalf("expected warning, got %q", buf.String())
	}
}

func TestEmitScenarioStaleWarningRespectsEnvVar(t *testing.T) {
	root, name := scaffoldScenarioTree(t)
	writeStaleSidecar(t, filepath.Join(root, "scenarios", name))

	t.Setenv(staleCheckEnvVar, "1")
	var buf bytes.Buffer
	emitScenarioStaleWarning(&buf, root, name, rootcli.GlobalOptions{})
	if buf.Len() != 0 {
		t.Fatalf("env-var override should silence warnings; got %q", buf.String())
	}
}

func TestEmitScenarioStaleWarningSilentForUnknownScenario(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	emitScenarioStaleWarning(&buf, root, "does-not-exist", rootcli.GlobalOptions{})
	if buf.Len() != 0 {
		t.Fatalf("missing scenario must not produce output; got %q", buf.String())
	}
	emitScenarioStaleWarning(&buf, root, "", rootcli.GlobalOptions{})
	if buf.Len() != 0 {
		t.Fatalf("empty scenario name must not produce output")
	}
	emitScenarioStaleWarning(&buf, root, "../etc/passwd", rootcli.GlobalOptions{})
	if buf.Len() != 0 {
		t.Fatalf("scenario name with separator must not produce output")
	}
}

func TestEmitScenarioStaleWarningSilentOnFreshSidecar(t *testing.T) {
	root, name := scaffoldScenarioTree(t)
	// First call initializes the sidecar — status is InitialBaseline, no warning.
	var buf bytes.Buffer
	emitScenarioStaleWarning(&buf, root, name, rootcli.GlobalOptions{})
	if buf.Len() != 0 {
		t.Fatalf("initial-baseline should be silent, got %q", buf.String())
	}
	// Second call without edits must be silent too (Fresh).
	buf.Reset()
	emitScenarioStaleWarning(&buf, root, name, rootcli.GlobalOptions{})
	if buf.Len() != 0 {
		t.Fatalf("fresh scenario should be silent, got %q", buf.String())
	}
}

func scaffoldScenarioTree(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	name := "foo"
	scenarioDir := filepath.Join(root, "scenarios", name)
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "foo-api"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}
	return root, name
}

func writeStaleSidecar(t *testing.T, scenarioDir string) {
	t.Helper()
	// Seed a sidecar so the first call classifies the dir as stale.
	if _, err := scenariostale.Check(scenarioDir, filepath.Base(scenarioDir), scenariostale.Options{}); err != nil {
		t.Fatalf("seed sidecar: %v", err)
	}
	// Now edit the source without touching the binary mtime — forces StatusStale on next check.
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "main.go"), []byte("package main\nfunc main(){ println(\"edit\") }\n"), 0o644); err != nil {
		t.Fatalf("edit source: %v", err)
	}
}
