package signing

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
	if !strings.Contains(err.Error(), "unknown signing subcommand") {
		t.Errorf("expected 'unknown signing subcommand' in error, got: %v", err)
	}
}

func TestSet_MissingProfile(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Set(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected usage error for missing profile, got: %v", err)
	}
}

func TestSet_MissingPlatform(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Set([]string{"my-profile"})
	if err == nil || !strings.Contains(err.Error(), "--platform is required") {
		t.Fatalf("expected --platform required error, got: %v", err)
	}
}

func TestSet_InvalidPlatform(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Set([]string{"my-profile", "--platform", "freebsd"})
	if err == nil || !strings.Contains(err.Error(), "invalid platform") {
		t.Fatalf("expected invalid platform error, got: %v", err)
	}
}

func TestSet_MacOS_MissingIdentity(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Set([]string{"my-profile", "--platform", "macos"})
	if err == nil || !strings.Contains(err.Error(), "--identity and --team-id") {
		t.Fatalf("expected identity/team-id error, got: %v", err)
	}
}

func TestSet_MacOS_MissingTeamID(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Set([]string{"my-profile", "--platform", "macos", "--identity", "Dev ID"})
	if err == nil || !strings.Contains(err.Error(), "--identity and --team-id") {
		t.Fatalf("expected identity/team-id error, got: %v", err)
	}
}

func TestDiscover_MissingPlatform(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Discover(nil)
	if err == nil || !strings.Contains(err.Error(), "--platform is required") {
		t.Fatalf("expected --platform required error, got: %v", err)
	}
}

func TestDiscover_InvalidPlatform(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Discover([]string{"--platform", "solaris"})
	if err == nil || !strings.Contains(err.Error(), "invalid platform") {
		t.Fatalf("expected invalid platform error, got: %v", err)
	}
}

func TestShow_MissingProfile(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Show(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected usage error for missing profile, got: %v", err)
	}
}

func TestRemove_MissingProfile(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Remove(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected usage error for missing profile, got: %v", err)
	}
}

func TestValidate_MissingProfile(t *testing.T) {
	cmd := New(testAPIClient("http://localhost:0"))
	err := cmd.Validate(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected usage error for missing profile, got: %v", err)
	}
}
