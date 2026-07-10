package authhandlers

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	authapp "github.com/vrooli/vrooli/internal/app/auth"
	"github.com/vrooli/vrooli/internal/cliout"
)

type fakeCtx struct {
	out    *bytes.Buffer
	format cliout.Format
	probes []authapp.SignInProbe
}

func newFake(probes []authapp.SignInProbe) *fakeCtx {
	return &fakeCtx{out: &bytes.Buffer{}, format: cliout.FormatHuman, probes: probes}
}

func deps(format cliout.Format) HandlerDeps[*fakeCtx] {
	return HandlerDeps[*fakeCtx]{
		Stdout:       func(c *fakeCtx) io.Writer { return c.out },
		OutputFormat: func(c *fakeCtx) (cliout.Format, error) { return format, nil },
		Probes:       func(c *fakeCtx) []authapp.SignInProbe { return c.probes },
	}
}

type stubProbe struct {
	name    string
	result  authapp.ProbeResult
	expSeen *bool
}

func (s stubProbe) Name() string { return s.name }
func (s stubProbe) Probe(ctx context.Context, opts authapp.ProbeOptions) authapp.ProbeResult {
	if s.expSeen != nil {
		*s.expSeen = opts.CheckExpiry
	}
	return s.result
}

func TestStatusRendersTable(t *testing.T) {
	ctx := newFake([]authapp.SignInProbe{stubProbe{
		name:   "buf",
		result: authapp.ProbeResult{State: authapp.StateSignedOut, Detail: "no ~/.netrc", SignInCommand: []string{"buf", "registry", "login"}},
	}})
	if err := RootHandler(deps(cliout.FormatHuman))(ctx, []string{"status"}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	out := ctx.out.String()
	if !strings.Contains(out, "buf") || !strings.Contains(out, "signed_out") {
		t.Fatalf("table output missing buf/signed_out: %q", out)
	}
	if !strings.Contains(out, "need attention") {
		t.Fatalf("table output should warn about signed_out tools: %q", out)
	}
}

func TestStatusRendersJSON(t *testing.T) {
	ctx := newFake([]authapp.SignInProbe{stubProbe{
		name:   "buf",
		result: authapp.ProbeResult{State: authapp.StateSignedIn, Detail: "ok"},
	}})
	if err := RootHandler(deps(cliout.FormatJSON))(ctx, []string{"status", "--json"}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	out := ctx.out.String()
	if !strings.Contains(out, `"state": "signed_in"`) {
		t.Fatalf("JSON missing signed_in state: %q", out)
	}
}

func TestCheckExpiryFlagPropagates(t *testing.T) {
	seen := false
	ctx := newFake([]authapp.SignInProbe{stubProbe{
		name:    "buf",
		result:  authapp.ProbeResult{State: authapp.StateSignedIn},
		expSeen: &seen,
	}})
	if err := RootHandler(deps(cliout.FormatHuman))(ctx, []string{"status", "--check-expiry"}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if !seen {
		t.Fatal("ProbeOptions.CheckExpiry should be true when --check-expiry is passed")
	}
}

func TestCheckExpiryDefaultsOff(t *testing.T) {
	seen := true // start true to assert reset
	ctx := newFake([]authapp.SignInProbe{stubProbe{
		name:    "buf",
		result:  authapp.ProbeResult{State: authapp.StateSignedIn},
		expSeen: &seen,
	}})
	if err := RootHandler(deps(cliout.FormatHuman))(ctx, []string{"status"}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if seen {
		t.Fatal("ProbeOptions.CheckExpiry should default to false")
	}
}

func TestExpiredStateSurfacedToTable(t *testing.T) {
	ctx := newFake([]authapp.SignInProbe{stubProbe{
		name:   "buf",
		result: authapp.ProbeResult{State: authapp.StateExpired, Detail: "401 from buf.build"},
	}})
	if err := RootHandler(deps(cliout.FormatHuman))(ctx, []string{"status"}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	out := ctx.out.String()
	if !strings.Contains(out, "expired") {
		t.Fatalf("expected expired in output: %q", out)
	}
}

func TestRootHandlerWithNoArgsRendersHelp(t *testing.T) {
	ctx := newFake(nil)
	if err := RootHandler(deps(cliout.FormatHuman))(ctx, []string{}); err != nil {
		t.Fatalf("handler: %v", err)
	}
	out := ctx.out.String()
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "vrooli auth") {
		t.Fatalf("help missing: %q", out)
	}
}

// Sanity: probes returning errors via panic shouldn't be silently swallowed —
// the handler doesn't catch panics, so an authoritative test would just be
// that the canonical probe set is non-empty. (This check belongs in the
// probe package itself, but we sanity-check the wiring here.)
func TestDefaultProbeSetNotEmpty(t *testing.T) {
	if len(authapp.DefaultProbes()) == 0 {
		t.Fatal("DefaultProbes returned no probes")
	}
}

// Ensure the stubProbe satisfies the SignInProbe interface (compile-time).
var _ authapp.SignInProbe = stubProbe{}
