package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/testkit"
)

func TestRunListAndExplainUseDiscoveredContracts(t *testing.T) {
	root := t.TempDir()
	writeFlow(t, root, "api/internal/visible/visible.flow.json", "example.visible.api")

	var stdout bytes.Buffer
	if err := Run(context.Background(), []string{"list", "--root", root}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(list) error = %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "example.visible.api" {
		t.Fatalf("list output = %q", got)
	}

	stdout.Reset()
	if err := Run(context.Background(), []string{"explain", "--root", root, "--flow", "example.visible.api"}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("Run(explain) error = %v", err)
	}
	for _, want := range []string{
		"flow: example.visible.api",
		"Hand-authored (edit these):",
		"Generated (regenerated; do not edit):",
		"generated/visible/",
		"Runtime:",
		"Topology:",
		"states: 1 (initial idle; terminal none)",
		"events: 1",
		"expanded transitions: 1",
		"Replay:",
		"Coverage requirements:",
		"named traces: 1",
		"Commands:",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("explain output missing %q in:\n%s", want, stdout.String())
		}
	}
}

func TestRunRejectsUnknownFlow(t *testing.T) {
	root := t.TempDir()
	writeFlow(t, root, "api/internal/visible/visible.flow.json", "example.visible.api")

	err := Run(context.Background(), []string{"list", "--root", root, "--flow", "missing.flow.api"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown flow id missing.flow.api") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestServiceGenerateUsesInjectedRunnerAndFileSystem(t *testing.T) {
	root := t.TempDir()
	testkit.WriteFile(t, root, "tools/temporal-model/go.mod", "module test\n")
	raw := testkit.ValidRawContract()
	raw.FlowID = "example.visible.api"
	testkit.WriteFlowJSON(t, root, "api/internal/visible/visible.flow.json", raw)

	fs := recordingFS{}
	var stdout bytes.Buffer
	service := Service{Runner: testkit.FakeRunner{}, FS: &fs, Stdout: &stdout}
	if err := service.Run(context.Background(), []string{"generate", "--root", root, "--flow", "example.visible.api"}); err != nil {
		t.Fatalf("Service.Run(generate) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "generated 1 temporal flow(s)") {
		t.Fatalf("generate output = %q", stdout.String())
	}
	for _, want := range []string{
		"generated/visible/model.qnt",
		"generated/visible/artifact.json",
		"generated/visible/runtime.go",
		"generated/visible/replay.go",
	} {
		if !fs.wroteSuffix(want) {
			t.Fatalf("expected generated write ending in %s; writes=%v", want, fs.writes)
		}
	}
}

func TestParseFlagsRequiresValues(t *testing.T) {
	if _, err := parseFlags([]string{"--root"}); err == nil {
		t.Fatal("expected --root without value to fail")
	}
	if _, err := parseFlags([]string{"--flow"}); err == nil {
		t.Fatal("expected --flow without value to fail")
	}
}

func writeFlow(t *testing.T, root string, rel string, flowID string) {
	t.Helper()
	raw := testkit.ValidRawContract()
	raw.FlowID = flowID
	raw.States = raw.States[:1]
	raw.Events = raw.Events[:1]
	raw.TransitionDefaults.Terminal = nil
	raw.Transitions = raw.Transitions[:1]
	raw.Transitions[0].To = model.SelfTarget
	wantError := true
	raw.Transitions[0].WantError = &wantError
	raw.Traces = []contract.Trace{{Name: "idle", Initial: "idle", Steps: []contract.TraceStep{}}}
	testkit.WriteFlowJSON(t, root, rel, raw)
}

type recordingFS struct {
	writes []string
}

func (fs *recordingFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *recordingFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	fs.writes = append(fs.writes, path)
	return os.WriteFile(path, data, perm)
}

func (fs *recordingFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs *recordingFS) wroteSuffix(suffix string) bool {
	for _, path := range fs.writes {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			return true
		}
	}
	return false
}
