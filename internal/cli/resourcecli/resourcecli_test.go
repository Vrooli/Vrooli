package resourcecli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestWriteStatusHumanIncludesCoreMetadata(t *testing.T) {
	healthy := false
	status := resources.Status{
		Resource: resources.Resource{
			Name:            "fixture",
			Enabled:         true,
			ControlMode:     "manifest-native",
			Driver:          "external-cli",
			PortabilityTier: "partial",
		},
		Installed:  true,
		Running:    false,
		Healthy:    &healthy,
		Message:    "available",
		ProbeError: "probe failed",
	}

	var stdout bytes.Buffer
	if err := WriteStatus(&stdout, cliout.FormatHuman, status); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"fixture", "manifest-native", "external-cli", "Probe Error"} {
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

func TestRenderTemplateGenerateHelpTextUsesGeneratedSchema(t *testing.T) {
	text := RenderTemplateGenerateHelpText()
	for _, want := range []string{
		"vrooli resource template generate <template> [options]",
		"--from-blueprint <name>",
		"--var <key=value>",
		"--dry-run",
	} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(want)) {
			t.Fatalf("missing %q in help:\n%s", want, text)
		}
	}
}

func TestWriteSchemaValidationReportHumanIncludesMissingReferences(t *testing.T) {
	var stdout bytes.Buffer
	report := resources.ResourceSchemaValidationReport{
		Passed:        false,
		ResourceCount: 1,
		MissingReferences: []resources.ScenarioResourceReference{{
			Scenario:     "alpha",
			Resource:     "n8n",
			ManifestPath: "/repo/scenarios/alpha/.vrooli/service.json",
		}},
	}
	if err := WriteSchemaValidationReport(&stdout, cliout.FormatHuman, report); err != nil {
		t.Fatalf("WriteSchemaValidationReport: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{"Resource schema validation failed", "alpha", "n8n"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in output:\n%s", want, output)
		}
	}
}

func TestCommandHelpTextIncludesStandardLifecycleCommands(t *testing.T) {
	for _, command := range []CommandID{CommandInstall, CommandUninstall, CommandStart, CommandRestart, CommandStop, CommandLogs} {
		text := CommandHelpText(command)
		if !strings.Contains(strings.ToLower(text), "vrooli resource "+strings.ToLower(string(command))) {
			t.Fatalf("help for %s missing usage line:\n%s", command, text)
		}
		if !strings.Contains(strings.ToLower(text), "<name>") {
			t.Fatalf("help for %s missing positional help:\n%s", command, text)
		}
	}
}
