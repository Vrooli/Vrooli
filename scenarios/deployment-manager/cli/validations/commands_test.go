package validations

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func testAPIClient(base string) *cliutil.APIClient {
	return cliutil.NewAPIClient(
		cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{DefaultBase: base}}),
		func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: base} },
		func() string { return "" },
	)
}

func TestRun_NoArgs(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run(nil)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if !strings.Contains(err.Error(), "validation subcommand is required") {
		t.Errorf("expected subcommand required error, got: %v", err)
	}
}

func TestRun_UnknownSubcommand(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown validation subcommand") {
		t.Errorf("expected 'unknown validation subcommand' in error, got: %v", err)
	}
}

func TestRunValidation_MissingProfile(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"run"})
	if err == nil || !strings.Contains(err.Error(), "profile ID is required") {
		t.Fatalf("expected profile ID required error, got: %v", err)
	}
}

func TestRunValidation_MissingCommit(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"run", "my-profile"})
	if err == nil || !strings.Contains(err.Error(), "--commit is required") {
		t.Fatalf("expected --commit required error, got: %v", err)
	}
}

func TestReview_MissingValidationID(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"review"})
	if err == nil || !strings.Contains(err.Error(), "validation ID is required") {
		t.Fatalf("expected validation ID required error, got: %v", err)
	}
}

func TestReview_InvalidDecision(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"review", "val-123", "--decision", "maybe"})
	if err == nil || !strings.Contains(err.Error(), "--decision must be 'approve' or 'reject'") {
		t.Fatalf("expected invalid decision error, got: %v", err)
	}
}

func TestReview_MissingDecision(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"review", "val-123"})
	if err == nil || !strings.Contains(err.Error(), "--decision must be 'approve' or 'reject'") {
		t.Fatalf("expected decision required error, got: %v", err)
	}
}

func TestStatus_MissingValidationID(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"status"})
	if err == nil || !strings.Contains(err.Error(), "validation ID is required") {
		t.Fatalf("expected validation ID required error, got: %v", err)
	}
}

func TestVideo_MissingValidationID(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"video"})
	if err == nil || !strings.Contains(err.Error(), "validation ID is required") {
		t.Fatalf("expected validation ID required error, got: %v", err)
	}
}

func TestList_MissingProfile(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"list"})
	if err == nil || !strings.Contains(err.Error(), "--profile is required") {
		t.Fatalf("expected --profile required error, got: %v", err)
	}
}
