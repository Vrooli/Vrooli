package resourcecli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestWriteStatusHumanIncludesLegacyAdapterMetadata(t *testing.T) {
	healthy := false
	status := resources.Status{
		Resource: resources.Resource{
			Name:            "fixture",
			Enabled:         true,
			ControlMode:     "legacy-adapter",
			Driver:          "legacy-adapter",
			PortabilityTier: "partial",
			LegacyAdapter: resources.ResourceLegacyAdapter{
				Owner:            "CLI tests",
				DecisionDeadline: "2026-12-31",
				FinalDisposition: "migrate",
				LegacyCLIPath:    "resources/fixture/cli.sh",
				Notes:            "Adapter note",
			},
		},
		Installed:  true,
		Running:    false,
		Healthy:    &healthy,
		Message:    "legacy adapter",
		ProbeError: "probe failed",
	}

	var stdout bytes.Buffer
	if err := WriteStatus(&stdout, cliout.FormatHuman, status); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"fixture", "Adapter Owner", "CLI tests", "Adapter note", "Probe Error"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}
}

func TestWriteControlReportJSONIncludesEnvelope(t *testing.T) {
	var stdout bytes.Buffer
	report := control.StopReport{
		Stopped: []control.ResultItem{control.Stopped("fixture", "Stopped successfully")},
	}
	if err := WriteControlReport(&stdout, cliout.FormatJSON, "report", "Stopped", report, report.Stopped, nil); err != nil {
		t.Fatalf("WriteControlReport: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"report":`) {
		t.Fatalf("output = %s", output)
	}
}

func TestWriteBlueprintSearchHumanIncludesRows(t *testing.T) {
	var stdout bytes.Buffer
	err := WriteBlueprintSearch(&stdout, cliout.FormatHuman, "cache", []resources.Blueprint{{
		Name:     "redis",
		Category: "storage",
		Status:   "ready",
		Summary:  "cache",
	}})
	if err != nil {
		t.Fatalf("WriteBlueprintSearch: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "redis") || !strings.Contains(output, "storage") {
		t.Fatalf("output = %s", output)
	}
}
