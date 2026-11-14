package preflight

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunRejectsUnknownSubcommand(t *testing.T) {
	err := Run(nil, []string{"unknown-subcommand"})
	if err == nil {
		t.Fatal("expected unknown subcommand error")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRequirementsRejectsUnknownFlag(t *testing.T) {
	err := runRequirements(nil, []string{"--nope"})
	if err == nil {
		t.Fatal("expected unknown flag error")
	}
	if !strings.Contains(err.Error(), "unknown flag: --nope") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatSizeAndJoinPortsHelpers(t *testing.T) {
	if got := formatSize(1024); got != "1.0K" {
		t.Fatalf("formatSize(1024)=%q", got)
	}
	if got := formatSize(1024 * 1024); got != "1.0M" {
		t.Fatalf("formatSize(1MiB)=%q", got)
	}
	if got := joinPorts([]int{22, 80, 443}); got != "22, 80, 443" {
		t.Fatalf("joinPorts=%q", got)
	}
	if got := joinPorts(nil); got != "-" {
		t.Fatalf("joinPorts(nil)=%q", got)
	}
}

// A failing check used to exit 0: the command printed raw JSON and ignored
// its own verdict, so a scripted deploy could not tell a clean target from a
// broken one.
func TestPreflightVerdictFailsOnFailedCheck(t *testing.T) {
	resp := Response{Checks: []Check{
		{ID: "disk_free", Status: CheckPass},
		{ID: "ssh_connect", Status: CheckFail, Details: "unreachable"},
	}}

	err := preflightVerdict(resp)

	if err == nil {
		t.Fatal("a failed check must be a non-zero exit")
	}
	if !strings.Contains(err.Error(), "ssh_connect") {
		t.Errorf("error = %v, want the failing check named", err)
	}
}

// Warnings are the operator-awareness channel: they must be visible and must
// never block a deployment.
func TestPreflightVerdictIgnoresWarnings(t *testing.T) {
	resp := Response{Checks: []Check{
		{ID: "postgres_credentials", Status: CheckWarn, Details: "not configured"},
		{ID: "outbound_network", Status: CheckWarn},
	}}

	if err := preflightVerdict(resp); err != nil {
		t.Fatalf("warnings must not fail preflight: %v", err)
	}
	failed, warned, passed := groupChecks(resp.Checks)
	if len(failed) != 0 || len(warned) != 2 || len(passed) != 0 {
		t.Errorf("grouping = %d failed / %d warned / %d passed, want 0/2/0", len(failed), len(warned), len(passed))
	}
}

// The old CLI shape decoded `name`/`passed`, which no producer sends; every
// check arrived unnamed and indistinguishable.
func TestResponseDecodesTheProducerShape(t *testing.T) {
	body := []byte(`{"ok":false,"checks":[{"id":"disk_free","title":"Disk space","status":"warn",` +
		`"details":"6 GB free","hint":"free space"}],"issues":[{"path":"edge.domain","message":"missing","severity":"error"}]}`)

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Checks) != 1 {
		t.Fatalf("checks = %d, want 1", len(resp.Checks))
	}
	check := resp.Checks[0]
	if check.Title != "Disk space" || check.Status != CheckWarn || check.Details == "" || check.Hint == "" {
		t.Errorf("check decoded as %+v, want the producer's title, status, details and hint", check)
	}
	if len(resp.Issues) != 1 || resp.Issues[0].Severity != "error" {
		t.Errorf("issues = %+v, want one error-severity issue", resp.Issues)
	}
}
