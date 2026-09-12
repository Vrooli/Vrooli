package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

func TestWriteCaptureArtifactSummaryPersistsPrimaryArtifact(t *testing.T) {
	dir := t.TempDir()
	resultPath := filepath.Join(dir, "result.json")
	if err := os.WriteFile(resultPath, []byte(`{"status":"completed"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	artifacts := []*capturev1.CaptureArtifact{
		{Type: capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, Path: "/shots/first.png"},
		{Type: capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, Path: "/shots/final.png", Primary: true},
	}
	if err := writeCaptureArtifactSummary(dir, artifacts); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Primary   string `json:"primary_artifact_path"`
		Artifacts []struct {
			Primary bool `json:"primary"`
		} `json:"capture_artifacts"`
	}
	raw, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Primary != "/shots/final.png" || len(result.Artifacts) != 2 || !result.Artifacts[1].Primary {
		t.Fatalf("result = %+v", result)
	}
}
