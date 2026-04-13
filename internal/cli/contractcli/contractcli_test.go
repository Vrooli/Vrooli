package contractcli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/repocontractcheck"
)

func TestRenderValidateHumanIncludesFailedCheck(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderValidate(&stdout, cliout.FormatHuman, ValidationOutput{
		Success: false,
		Root:    "/tmp/repo",
		Schema: ValidationCheck{
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
	err := RenderMatchGlob(&stdout, cliout.FormatJSON, MatchGlobOutput{
		Success: true,
		Pattern: "scenarios/*",
		Path:    "scenarios/test-genie",
		Matched: true,
	})
	if err != nil {
		t.Fatalf("RenderMatchGlob: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"matched": true`) {
		t.Fatalf("stdout = %s", output)
	}
}
