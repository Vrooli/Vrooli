package contractcli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/repocontractcheck"
)

func TestRenderValidateHumanIncludesFailedCheck(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderValidate(&stdout, cliout.FormatHuman, contractapp.ValidationOutput{
		Success: false,
		Root:    "/tmp/repo",
		Schema: contractapp.ValidationCheck{
			Passed:  true,
			Message: "ok",
		},
		Report: repocontractcheck.Report{
			Checks: []repocontractcheck.CheckResult{{
				Name:    "docs_alignment",
				Passed:  false,
				Message: "drift",
			}},
		},
	})
	if err != nil {
		t.Fatalf("RenderValidate: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Repo contract validation failed") || !strings.Contains(output, "docs_alignment: FAIL (drift)") {
		t.Fatalf("stdout = %q", output)
	}
}

func TestRenderMatchGlobJSONIncludesMatchedField(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderMatchGlob(&stdout, cliout.FormatJSON, contractapp.MatchGlobOutput{
		Success: true,
		Pattern: "scenarios/*",
		Path:    "scenarios/test-genie",
		Matched: true,
	})
	if err != nil {
		t.Fatalf("RenderMatchGlob: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, stdout.String())
	}
	if got["success"] != true || got["matched"] != true {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRenderResolveScenarioHelpTextUsesGeneratedSchemaHelp(t *testing.T) {
	got := RenderResolveScenarioHelpText()
	for _, want := range []string{
		"Usage:\n  vrooli contract resolve scenario <name> [options]",
		"Known keys: service, orientation, docs, requirements, api, ui, cli, initialization",
		"--file <key>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in help:\n%s", want, got)
		}
	}
}
