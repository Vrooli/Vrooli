package runtimehandlers

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

type testContext struct{ stdin, stdout, stderr *bytes.Buffer }

func TestRootHandlerRendersRuntimeHelp(t *testing.T) {
	ctx := &testContext{stdin: &bytes.Buffer{}, stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}}
	handler := RootHandler(rootcli.HandlerDeps[*testContext]{
		Root:        func(*testContext) string { return "." },
		Globals:     func(*testContext) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		Stdin:       func(c *testContext) io.Reader { return c.stdin },
		Stdout:      func(c *testContext) io.Writer { return c.stdout },
		Stderr:      func(c *testContext) io.Writer { return c.stderr },
		HomeDir:     func(*testContext) (string, error) { return ".", nil },
		ResolveRoot: func(*testContext) (string, error) { return ".", nil },
		Version:     func(*testContext) string { return "test" },
	})
	if err := handler(ctx, []string{"--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx.stdout.String(), "vrooli runtime") {
		t.Fatalf("help = %q", ctx.stdout.String())
	}
	ctx.stdout.Reset()
	if err := handler(ctx, []string{"update", "--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx.stdout.String(), "retry CANDIDATE") || !strings.Contains(ctx.stdout.String(), "Enrollment controls") {
		t.Fatalf("update help was replaced by generic runtime help: %q", ctx.stdout.String())
	}
	ctx.stdout.Reset()
	if err := handler(ctx, []string{"lighting", "permissions", "apply", "--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctx.stdout.String(), "--plan-id") || !strings.Contains(ctx.stdout.String(), "--sudo-mode") {
		t.Fatalf("lighting help lost fixed typed operation: %q", ctx.stdout.String())
	}
}

// A help flag after a root update verb must render usage and never reach the
// effect path, where apply/retry would read it as the candidate path.
func TestRootUpdateHelpNeverSubmitsAnUpdate(t *testing.T) {
	for _, args := range [][]string{
		{"runtime", "update", "apply", "--help"},
		{"runtime", "update", "retry", "-h"},
		{"runtime", "update", "enroll", "--help"},
		{"runtime", "update", "status", "--help"},
	} {
		var stdout, stderr bytes.Buffer
		handled, code := RunRootUpdateReadCommand(args, &stdout, &stderr)
		if !handled || code != 0 || !strings.Contains(stdout.String(), "apply CANDIDATE") {
			t.Fatalf("read path %v: handled=%v code=%d stdout=%q stderr=%q", args, handled, code, stdout.String(), stderr.String())
		}
		stdout.Reset()
		resolveCalled := false
		handled, code = RunRootUpdateEffectCommand(t.Context(), args, &stdout, &stderr, func() (string, error) {
			resolveCalled = true
			return t.TempDir(), nil
		})
		if !handled || code != 0 || resolveCalled || !strings.Contains(stdout.String(), "apply CANDIDATE") {
			t.Fatalf("effect path %v: handled=%v code=%d resolved=%v stdout=%q", args, handled, code, resolveCalled, stdout.String())
		}
	}
}
