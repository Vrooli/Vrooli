package projectcli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/project"
)

func TestRenderOrphansResponseHumanEmpty(t *testing.T) {
	var stdout bytes.Buffer
	if err := RenderOrphansResponse(&stdout, cliout.FormatHuman, OrphansResponse{}); err != nil {
		t.Fatalf("RenderOrphansResponse: %v", err)
	}
	if !strings.Contains(stdout.String(), "No orphaned Vrooli processes found.") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRenderLocksResponseHumanIncludesStaleStatus(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderLocksResponse(&stdout, cliout.FormatHuman, LocksResponse{
		List: []maintenance.LockInfo{{
			Port:     21234,
			Scenario: "alpha",
			PID:      999,
			Stale:    true,
		}},
	})
	if err != nil {
		t.Fatalf("RenderLocksResponse: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "21234") || !strings.Contains(output, "stale") {
		t.Fatalf("stdout = %q", output)
	}
}

func TestRenderDoctorReportJSONIncludesChecks(t *testing.T) {
	var stdout bytes.Buffer
	report := project.DoctorReport{
		Checks: []project.DoctorCheck{{Name: "jq", Status: "ok"}},
	}
	if err := RenderDoctorReport(&stdout, cliout.FormatJSON, report); err != nil {
		t.Fatalf("RenderDoctorReport: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"checks":`) {
		t.Fatalf("stdout = %s", output)
	}
}

func TestRenderStopReportHumanIncludesFailures(t *testing.T) {
	var stdout bytes.Buffer
	report := control.StopReport{
		Stopped: []control.ResultItem{control.Stopped("alpha", "Stopped successfully")},
		Failed:  []control.ResultItem{control.Failed("beta", errors.New("boom"))},
	}
	if err := RenderStopReport(&stdout, cliout.FormatHuman, report); err != nil {
		t.Fatalf("RenderStopReport: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Stopped alpha") || !strings.Contains(output, "Failed beta: boom") {
		t.Fatalf("stdout = %q", output)
	}
}
