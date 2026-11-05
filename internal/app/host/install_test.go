package hostapp

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// TestWriteHostInstallJSONContract pins the `host install --json` wire shape and
// the mapping from runtime ItemStatus → vrooli.cli.v1.CliHostInstallStatus.
func TestWriteHostInstallJSONContract(t *testing.T) {
	status := hostreqkit.ItemStatus{
		Name:           "realesrgan-ncnn-vulkan",
		Command:        "realesrgan-ncnn-vulkan",
		Installed:      true,
		SupportClass:   hostreqkit.SupportSupported,
		ExecutionState: hostreqkit.ExecutionInstalled,
		Version:        "1.0",
		Notes:          []string{"install fetches https://example/x.zip into ~/.vrooli/bin"},
	}

	var buf bytes.Buffer
	if err := cliout.WriteProtoJSON(&buf, hostInstallStatusResponse(status)); err != nil {
		t.Fatalf("writeHostInstallJSON: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, buf.String())
	}
	if got["name"] != "realesrgan-ncnn-vulkan" {
		t.Errorf("name: %v", got["name"])
	}
	if got["execution_state"] != "installed" {
		t.Errorf("execution_state: %v", got["execution_state"])
	}
	if got["support_class"] != "supported" {
		t.Errorf("support_class: %v", got["support_class"])
	}
	if got["installed"] != true {
		t.Errorf("installed: %v", got["installed"])
	}
	if got["ok"] != true {
		t.Errorf("ok: %v", got["ok"])
	}
	notes, ok := got["notes"].([]any)
	if !ok || len(notes) != 1 {
		t.Fatalf("notes: %v", got["notes"])
	}
}

// TestHostInstallOKClassification documents which terminal states count as a
// successful (exit 0) host-install outcome.
func TestHostInstallOKClassification(t *testing.T) {
	ok := []hostreqkit.ExecutionState{
		hostreqkit.ExecutionInstalled,
		hostreqkit.ExecutionAlreadyPresent,
		hostreqkit.ExecutionWouldInstall,
		hostreqkit.ExecutionNotApplicable,
	}
	for _, state := range ok {
		if !hostInstallOK(hostreqkit.ItemStatus{ExecutionState: state}) {
			t.Errorf("state %q should be ok", state)
		}
	}
	notOK := []hostreqkit.ExecutionState{
		hostreqkit.ExecutionFailed,
		hostreqkit.ExecutionUnsupported,
		hostreqkit.ExecutionManualActionRequired,
		hostreqkit.ExecutionPending,
	}
	for _, state := range notOK {
		if hostInstallOK(hostreqkit.ItemStatus{ExecutionState: state}) {
			t.Errorf("state %q should NOT be ok", state)
		}
	}
}
