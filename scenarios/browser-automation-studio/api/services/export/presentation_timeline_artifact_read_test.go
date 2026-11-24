package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/browser-automation-studio/database"
	bastelemetry "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestPresentationArtifactSidecarsShareReadOwnerAcrossExportCallers(t *testing.T) {
	root := t.TempDir()

	telemetryPath := filepath.Join(root, "console.json")
	if err := os.WriteFile(telemetryPath, []byte(`[{"message":"console entry"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := telemetryArtifactToExport("console_log", &bastelemetry.TelemetryArtifact{Path: &telemetryPath}, nil)
	entries, ok := artifact.Payload["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("telemetry entries = %#v, want one decoded entry", artifact.Payload["entries"])
	}
	entry, ok := entries[0].(map[string]any)
	if !ok || entry["message"] != "console entry" {
		t.Fatalf("telemetry entry = %#v, want preserved JSON object", entries[0])
	}

	width, height := int32(1365), int32(768)
	executionJSON, err := protojson.Marshal(&basexecution.Execution{Parameters: &basexecution.ExecutionParameters{ViewportWidth: &width, ViewportHeight: &height}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "execution.proto.json"), executionJSON, 0o600); err != nil {
		t.Fatal(err)
	}
	view := &ExecutionTimeline{}
	loadExecutionCaptureGeometry(&database.ExecutionIndex{ResultPath: filepath.Join(root, "result.json")}, view)
	if view.ViewportWidth != int(width) || view.ViewportHeight != int(height) {
		t.Fatalf("execution geometry = %dx%d, want %dx%d", view.ViewportWidth, view.ViewportHeight, width, height)
	}
}
