package cliapptest

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestNewCapturedRunContext_CapturesStdout(t *testing.T) {
	schema := cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "title"}},
	}
	ctx, buf := NewCapturedRunContext(nil, schema, TestRunContextOptions{
		Flags: map[string]string{"title": "hello"},
	})

	if ctx.Flag("title") != "hello" {
		t.Fatalf("Flag(title) = %q, want hello", ctx.Flag("title"))
	}

	if err := ctx.RenderList(cliapp.ListReport{Summary: []string{"a"}}); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	if !strings.Contains(buf.String(), "a") {
		t.Fatalf("captured stdout = %q, want to contain %q", buf.String(), "a")
	}
}

// TestNewCapturedRunContext_OverwritesSchemaCoreStdout pins the contract
// that helper-supplied fields take precedence: a caller passing Schema /
// Core / Stdout in opts has them clobbered. This is intentional —
// guarantees the returned buffer is always the fresh one.
func TestNewCapturedRunContext_OverwritesSchemaCoreStdout(t *testing.T) {
	bogus := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "bogus"}}}
	want := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "title"}}}

	ctx, buf := NewCapturedRunContext(nil, want, TestRunContextOptions{
		Schema: bogus,
		Flags:  map[string]string{"title": "hi"},
	})
	if ctx.Flag("title") != "hi" {
		t.Fatalf("expected helper's schema (with 'title') to win over opts.Schema")
	}
	if buf == nil {
		t.Fatal("returned buffer must not be nil")
	}
}

// TestNewCapturedRunContext_JSONRoutes pins JSON pass-through.
func TestNewCapturedRunContext_JSONRoutes(t *testing.T) {
	ctx, buf := NewCapturedRunContext(nil, cliapp.ArgSchema{}, TestRunContextOptions{JSON: true})
	if !ctx.JSON() {
		t.Fatal("JSON flag did not propagate")
	}
	if err := ctx.RenderList(cliapp.ListReport{Summary: []string{"x"}}); err != nil {
		t.Fatalf("RenderList: %v", err)
	}
	// JSON output is JSON, not the human renderer's table.
	if !strings.HasPrefix(strings.TrimSpace(buf.String()), "{") {
		t.Fatalf("captured = %q, want JSON object", buf.String())
	}
}
