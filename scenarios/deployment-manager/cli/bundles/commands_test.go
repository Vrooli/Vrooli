package bundles

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestNormalizeTier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"desktop", "tier-2-desktop"},
		{"2", "tier-2-desktop"},
		{"tier-2", "tier-2-desktop"},
		{"tier2", "tier-2-desktop"},
		{"mobile", "tier-3-mobile"},
		{"3", "tier-3-mobile"},
		{"tier-3", "tier-3-mobile"},
		{"tier3", "tier-3-mobile"},
		{"saas", "tier-4-saas"},
		{"cloud", "tier-4-saas"},
		{"4", "tier-4-saas"},
		{"tier-4", "tier-4-saas"},
		{"tier4", "tier-4-saas"},
		{"enterprise", "tier-5-enterprise"},
		{"5", "tier-5-enterprise"},
		{"tier-5", "tier-5-enterprise"},
		{"tier5", "tier-5-enterprise"},
		// Already formatted passes through
		{"tier-2-desktop", "tier-2-desktop"},
		{"tier-3-mobile", "tier-3-mobile"},
		{"tier-4-saas", "tier-4-saas"},
		{"tier-5-enterprise", "tier-5-enterprise"},
		// Case insensitivity
		{"DESKTOP", "tier-2-desktop"},
		{"Mobile", "tier-3-mobile"},
		// Unknown defaults to tier-2-desktop
		{"unknown", "tier-2-desktop"},
		{"", "tier-2-desktop"},
		// Whitespace trimming
		{"  desktop  ", "tier-2-desktop"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeTier(tt.input)
			if got != tt.want {
				t.Errorf("normalizeTier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

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
	if err != nil {
		t.Fatalf("expected no error for no args (prints help), got: %v", err)
	}
}

func TestRun_Help(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	for _, arg := range []string{"help", "-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			if err := cmd.Run([]string{arg}); err != nil {
				t.Errorf("expected no error for %q, got: %v", arg, err)
			}
		})
	}
}

func TestRun_UnknownSubcommand(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Run([]string{"bogus"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown bundle subcommand") {
		t.Errorf("expected 'unknown bundle subcommand' in error, got: %v", err)
	}
}

func TestAssemble_MissingScenario(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Assemble(nil)
	if err == nil || !strings.Contains(err.Error(), "scenario name is required") {
		t.Fatalf("expected scenario name required error, got: %v", err)
	}
}

func TestValidate_MissingFile(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Validate(nil)
	if err == nil || !strings.Contains(err.Error(), "manifest file path is required") {
		t.Fatalf("expected manifest file path required error, got: %v", err)
	}
}

func TestExport_MissingScenario(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Export(nil)
	if err == nil || !strings.Contains(err.Error(), "scenario name is required") {
		t.Fatalf("expected scenario name required error, got: %v", err)
	}
}
