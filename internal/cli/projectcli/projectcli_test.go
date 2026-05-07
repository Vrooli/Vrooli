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
	projectsetup "github.com/vrooli/vrooli/internal/setup"
)

func TestParseStatusRequestRejectsConflictingFilters(t *testing.T) {
	if _, err := ParseStatusRequest([]string{"--resources", "--scenarios"}); err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestParseSetupOptionsAcceptsFlags(t *testing.T) {
	opts, err := ParseSetupOptions([]string{"--environment", "minimal", "--resources", "none", "--yes", "yes", "--sudo-mode", "skip", "--dry-run"})
	if err != nil {
		t.Fatalf("ParseSetupOptions: %v", err)
	}
	want := projectsetup.Options{
		Environment: "minimal",
		Resources:   "none",
		Yes:         "yes",
		SudoMode:    "skip",
		DryRun:      true,
	}
	if opts != want {
		t.Fatalf("opts = %+v, want %+v", opts, want)
	}
}

func TestParseSetupOptionsStatusSubcommand(t *testing.T) {
	opts, err := ParseSetupOptions([]string{"status", "--environment", "minimal"})
	if err != nil {
		t.Fatalf("ParseSetupOptions(status): %v", err)
	}
	if opts.Subcommand != "status" {
		t.Fatalf("Subcommand = %q, want status", opts.Subcommand)
	}
	if opts.Environment != "minimal" {
		t.Fatalf("Environment = %q", opts.Environment)
	}
}

func TestParseSetupOptionsExplainRequiresName(t *testing.T) {
	if _, err := ParseSetupOptions([]string{"explain"}); err == nil {
		t.Fatal("expected explain to require a name")
	}
}

func TestParseSetupOptionsExplainAcceptsName(t *testing.T) {
	opts, err := ParseSetupOptions([]string{"explain", "mcelog"})
	if err != nil {
		t.Fatalf("ParseSetupOptions(explain mcelog): %v", err)
	}
	if opts.Subcommand != "explain" {
		t.Fatalf("Subcommand = %q", opts.Subcommand)
	}
	if opts.ExplainName != "mcelog" {
		t.Fatalf("ExplainName = %q", opts.ExplainName)
	}
}

func TestParseCleanupRequestRejectsUnknownTarget(t *testing.T) {
	if _, err := ParseCleanupRequest([]string{"bogus"}); err == nil {
		t.Fatal("expected cleanup target error")
	}
}

func TestParseCleanupRequestAcceptsTemplateValidationTarget(t *testing.T) {
	req, err := ParseCleanupRequest([]string{"template-validation", "--dry-run", "--older-than", "24h", "--include-retained", "--run", "run-1"})
	if err != nil {
		t.Fatalf("ParseCleanupRequest: %v", err)
	}
	if req.Target != "template-validation" {
		t.Fatalf("target = %q", req.Target)
	}
	args := strings.Join(req.Args, " ")
	for _, want := range []string{"--dry-run", "--older-than 24h", "--include-retained", "--run run-1"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args = %q, want %q", args, want)
		}
	}
}

func TestParseDiagnosePortRequestRejectsInvalidPort(t *testing.T) {
	if _, err := ParseDiagnosePortRequest([]string{"bogus"}); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestParseDiagnosePortRequestRejectsOutOfRange(t *testing.T) {
	for _, v := range []string{"0", "-1", "65536", "99999"} {
		if _, err := ParseDiagnosePortRequest([]string{v}); err == nil {
			t.Fatalf("expected out-of-range port error for %q", v)
		}
	}
}

func TestParseDiagnosePortRequestAcceptsBoundaries(t *testing.T) {
	for _, v := range []string{"1", "65535"} {
		if _, err := ParseDiagnosePortRequest([]string{v}); err != nil {
			t.Fatalf("unexpected error for port %q: %v", v, err)
		}
	}
}

func TestParseOrphansRequestDryRunRequiresKill(t *testing.T) {
	if _, err := ParseOrphansRequest([]string{"--dry-run"}); err == nil {
		t.Fatal("expected error: --dry-run alone should be rejected")
	}
}

func TestParseOrphansRequestKillWithDryRun(t *testing.T) {
	req, err := ParseOrphansRequest([]string{"kill", "--dry-run"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !req.Kill || !req.DryRun {
		t.Fatalf("req = %+v, want Kill=true DryRun=true", req)
	}
}

func TestRenderOrphansResponseDryRunHumanListsPIDs(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderOrphansResponse(&stdout, cliout.FormatHuman, OrphansResponse{
		List:   []maintenance.SystemProcess{{PID: 999, PPID: 1, Command: "scenario-api"}},
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("RenderOrphansResponse: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "[dry-run]") {
		t.Fatalf("stdout = %q", output)
	}
	if !strings.Contains(output, "999") {
		t.Fatalf("stdout missing pid: %q", output)
	}
	if !strings.Contains(output, "Re-run without --dry-run") {
		t.Fatalf("stdout missing re-run hint: %q", output)
	}
}

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

func TestRenderOrphansResponseHumanHandlesKillReport(t *testing.T) {
	var stdout bytes.Buffer
	err := RenderOrphansResponse(&stdout, cliout.FormatHuman, OrphansResponse{
		KillReport: &control.StopReport{
			Stopped: []control.ResultItem{control.Stopped("123", "sleep 30")},
		},
	})
	if err != nil {
		t.Fatalf("RenderOrphansResponse: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Stopped orphan PID 123") {
		t.Fatalf("stdout = %q", got)
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
