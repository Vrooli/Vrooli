package platform

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestIdentityUnsupportedIsTypedOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		if err := RunAsInvokingUserInSession(context.Background(), "whoami", nil, IdentityCommandOptions{}); !errors.Is(err, ErrSessionExecutionUnsupported) {
			t.Fatalf("error = %v, want ErrSessionExecutionUnsupported", err)
		}
	}
}

func TestIdentityCommandRejectsEmptyName(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "windows has a typed unsupported result before command validation")
	}
	if err := RunAsInvokingUserInSession(context.Background(), "", nil, IdentityCommandOptions{}); err == nil {
		t.Fatal("empty command unexpectedly succeeded")
	}
}

func TestIdentityOutputKeepsCommandOutputSeparate(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "portable command fixture is Unix-specific")
	}
	output, err := RunAsInvokingUserInSessionOutput(context.Background(), "printf", []string{"%s", "identity-output"}, IdentityCommandOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "identity-output" {
		t.Fatalf("output = %q", output)
	}
}

func TestIdentityInputUsesStdinInsteadOfArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "portable command fixture is Unix-specific")
	}
	var output bytes.Buffer
	if err := RunAsInvokingUserInSessionWithInput(context.Background(), "cat", nil, []byte("secret-through-stdin"), IdentityCommandOptions{Stdout: &output}); err != nil {
		t.Fatal(err)
	}
	if output.String() != "secret-through-stdin" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestIdentityCommandPreservesWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "portable command fixture is Unix-specific")
	}
	dir := t.TempDir()
	output, err := RunAsInvokingUserInSessionOutput(context.Background(), "pwd", nil, IdentityCommandOptions{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(output)); got != dir {
		t.Fatalf("working directory = %q, want %q", got, dir)
	}
}
