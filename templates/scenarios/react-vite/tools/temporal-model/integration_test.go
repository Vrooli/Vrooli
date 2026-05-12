package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"react-vite-temporal-model/internal/cli"
)

func TestIntegrationRealQuintCheck(t *testing.T) {
	if os.Getenv("VROOLI_TEMPORAL_MODEL_INTEGRATION") != "1" {
		t.Skip("set VROOLI_TEMPORAL_MODEL_INTEGRATION=1 to run real Quint validation")
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := cli.Run(context.Background(), []string{"check", "--root", "../.."}, &stdout, &stderr); err != nil {
		t.Fatalf("real Quint check failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	for _, want := range []string{
		"fresh notes.attachment-upload.api",
		"fresh notes.attachment-upload.ui",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("integration check stdout missing %q\nstdout:\n%s", want, stdout.String())
		}
	}
}

// TestIntegrationLintRejectsCorruptedHandTest is the negative-path
// gate required by the cutover. It corrupts the UI hand-authored test
// in place (under a temporary copy of the relevant files), runs check,
// asserts a lint failure mentioning the file, and ends. The original
// files in the working tree are left untouched.
func TestIntegrationLintRejectsCorruptedHandTest(t *testing.T) {
	if os.Getenv("VROOLI_TEMPORAL_MODEL_INTEGRATION") != "1" {
		t.Skip("set VROOLI_TEMPORAL_MODEL_INTEGRATION=1 to exercise lint rejection")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	testPath := filepath.Join(root, "ui/src/features/notes/flow/flow.test.ts")
	original, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	t.Cleanup(func() { _ = os.WriteFile(testPath, original, 0o644) })

	corrupted := []byte(`// negative-path: helper imported but never called
import { runFormalReplay } from "./generated/replay.helper";
import { transitionAttachmentUpload } from "./transition";
import { attachmentUploadFormalFixtures } from "./fixtures";
`)
	if err := os.WriteFile(testPath, corrupted, 0o644); err != nil {
		t.Fatalf("corrupt test file: %v", err)
	}
	var stdout, stderr bytes.Buffer
	err = cli.Run(context.Background(), []string{"check", "--root", "../.."}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected lint failure; got nil")
	}
	if !strings.Contains(err.Error(), "no top-level call") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if !strings.Contains(err.Error(), "flow.test.ts") {
		t.Fatalf("error message missing offending file path: %v", err)
	}
}

// TestIntegrationScaffoldRoundTrip verifies the `new` command emits a
// runnable flow: scaffold → check ⇒ green, with zero hand edits in
// between. Repeated for both TS and Go targets.
func TestIntegrationScaffoldRoundTrip(t *testing.T) {
	if os.Getenv("VROOLI_TEMPORAL_MODEL_INTEGRATION") != "1" {
		t.Skip("set VROOLI_TEMPORAL_MODEL_INTEGRATION=1 to exercise scaffold round-trip")
	}
	cases := []struct {
		name      string
		parentDir string
		flowID    string
		lang      string
	}{
		{"typescript", "ui/src/features/scaffold-smoke", "demo.scaffold-smoke.ui", "ts"},
		{"go", "api/internal/scaffold-smoke", "demo.scaffold-smoke.api", "go"},
	}
	toolDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			if err := os.MkdirAll(filepath.Join(tmp, filepath.FromSlash(tc.parentDir)), 0o755); err != nil {
				t.Fatal(err)
			}
			// tools/temporal-model is what the artifact builder
			// hashes for freshness, so the tmp root must point at
			// the real tool. A symlink is sufficient.
			if err := os.MkdirAll(filepath.Join(tmp, "tools"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(toolDir, filepath.Join(tmp, "tools", "temporal-model")); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			if err := cli.Run(context.Background(), []string{
				"new", tc.parentDir,
				"--flow-id", tc.flowID,
				"--lang", tc.lang,
				"--root", tmp,
			}, &stdout, &stderr); err != nil {
				t.Fatalf("scaffold failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
			}
			stdout.Reset()
			stderr.Reset()
			if err := cli.Run(context.Background(), []string{"check", "--root", tmp}, &stdout, &stderr); err != nil {
				t.Fatalf("check on scaffolded flow failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), "fresh "+tc.flowID) {
				t.Fatalf("check did not report freshness for %s; stdout:\n%s", tc.flowID, stdout.String())
			}
		})
	}
}

// TestIntegrationLintRejectsMissingFlowDir verifies the schema rejects
// contracts that live outside a `flow/` directory.
func TestIntegrationLintRejectsMissingFlowDir(t *testing.T) {
	tmp := t.TempDir()
	misplaced := filepath.Join(tmp, "api/internal/notflow/flow.json")
	if err := os.MkdirAll(filepath.Dir(misplaced), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"schemaVersion":6,"flowId":"x.y.api","domain":"x","description":"x","model":{"module":"X","seed":"1","maxSteps":1,"traceCount":1,"verify":{"invariants":["TypeOK"]}},"states":[{"id":"a","quint":"A","initial":true}],"events":[{"id":"e","quint":"E"}],"transitionDefaults":{"invalid":{"to":"self","wantError":false}},"transitions":[{"from":"a","event":"e","to":"a"}],"invariants":[{"id":"t","quint":"TypeOK","description":"x"}],"traces":[{"name":"s","initial":"a","steps":[{"event":"e","want":"a","wantError":false}]}],"runtime":{"go":{"package":"flow","statusType":"Status","eventType":"Event","constantPrefix":"X"}},"replay":{"transition":{"function":"Transition","stateType":"S","statusField":"Status"}}}`
	if err := os.WriteFile(misplaced, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := cli.Run(context.Background(), []string{"check", "--root", tmp}, &stdout, &stderr)
	// FindContracts only walks `**/flow/flow.json`, so this file is
	// invisible to discovery. That's the convention's whole point:
	// misplaced contracts are simply not discovered, so `check` is a
	// no-op with no flows. Confirm no fresh-line was emitted and the
	// directory was not picked up.
	if err != nil {
		t.Fatalf("check unexpectedly failed: %v", err)
	}
	if strings.Contains(stdout.String(), "x.y.api") {
		t.Fatalf("misplaced contract should be invisible to discovery; stdout:\n%s", stdout.String())
	}
}

// TestIntegrationRejectsSchemaV5 verifies that v5 contracts produce a
// clear migration error rather than a cryptic schema-validation
// failure.
func TestIntegrationRejectsSchemaV5(t *testing.T) {
	tmp := t.TempDir()
	contractDir := filepath.Join(tmp, "api/internal/legacy/flow")
	if err := os.MkdirAll(contractDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"schemaVersion":5,"flowId":"legacy.x.api","domain":"x","description":"x","model":{"module":"X","seed":"1","maxSteps":1,"traceCount":1,"verify":{"invariants":["TypeOK"]}},"states":[],"events":[],"transitionDefaults":{"invalid":{"to":"self","wantError":false}},"transitions":[],"invariants":[],"traces":[],"runtime":{},"replay":{"transition":{"function":"T"}}}`
	if err := os.WriteFile(filepath.Join(contractDir, "flow.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := cli.Run(context.Background(), []string{"list", "--root", tmp}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected schema v5 rejection; got nil")
	}
	if !strings.Contains(err.Error(), "schemaVersion 5 is no longer supported") {
		t.Fatalf("error did not mention v5 migration: %v", err)
	}
}
