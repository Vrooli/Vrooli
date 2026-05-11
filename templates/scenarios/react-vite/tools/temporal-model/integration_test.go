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
	testPath := filepath.Join(root, "ui/src/features/notes/AttachmentUploadWorkflow.test.ts")
	original, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	t.Cleanup(func() { _ = os.WriteFile(testPath, original, 0o644) })

	corrupted := []byte(`// negative-path: helper imported but never called
import { runFormalReplay } from "./generated/attachmentupload/replay.helper";
import { transitionAttachmentUpload } from "./AttachmentUploadWorkflow";
import { attachmentUploadFormalFixtures } from "./AttachmentUploadWorkflow.fixtures";
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
	if !strings.Contains(err.Error(), "AttachmentUploadWorkflow.test.ts") {
		t.Fatalf("error message missing offending file path: %v", err)
	}
}
