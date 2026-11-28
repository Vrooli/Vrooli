package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/credentialleak"
)

func TestScannerCommandReportsSyntheticArtifactLeakAsMetadataOnly(t *testing.T) {
	manifest := filepath.Join("..", "..", "..", ".vrooli", "service.json")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal("LPBS manifest unavailable")
	}
	families, err := credentialleak.ManifestFamilies(data)
	if err != nil {
		t.Fatal(err)
	}
	canary := credentialleak.SyntheticCanaries(families)[0]
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "phase.result.json"), []byte("injected="+canary.Value), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "findings.report.json"), []byte(`{"status":"failed"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if code := scan(manifest, root, &output); code != 1 {
		t.Fatalf("scanner exit code = %d, want leak failure", code)
	}
	if strings.Contains(output.String(), canary.Value) || strings.Contains(output.String(), "phase.result") {
		t.Fatal("scanner CLI emitted a canary or leaking artifact path")
	}
	var report credentialleak.ArtifactScanResult
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal("scanner output was not a JSON receipt")
	}
	if report.Status != "leak_detected" || len(report.Findings) == 0 || report.Findings[0].Family == "" {
		t.Fatalf("scanner receipt is incomplete: status=%q finding_count=%d", report.Status, len(report.Findings))
	}
}

func TestScannerCommandReportsCleanArtifactSet(t *testing.T) {
	manifest := filepath.Join("..", "..", "..", ".vrooli", "service.json")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "phase.result.json"), []byte(`{"status":"failed","sentinel_count":27}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "findings.report.json"), []byte(`{"findings":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if code := scan(manifest, root, &output); code != 0 {
		t.Fatalf("scanner exit code = %d, want clean", code)
	}
	var report credentialleak.ArtifactScanResult
	if err := json.Unmarshal(output.Bytes(), &report); err != nil || report.Status != "clean" || report.FilesScanned != 2 {
		t.Fatalf("clean scanner receipt = %+v, %v", report, err)
	}
}
