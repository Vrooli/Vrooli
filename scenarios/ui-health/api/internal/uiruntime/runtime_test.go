package uiruntime

import (
	"context"
	"errors"
	"testing"

	"ui-health/internal/evidence"
	"ui-health/internal/services/manifestvalidation"
)

// fakeBAS is a basRunner double returning a canned result or an unavailability error.
type fakeBAS struct {
	res  *runResult
	err  error
	defs []map[string]any
}

func (f *fakeBAS) Run(_ context.Context, def map[string]any) (*runResult, error) {
	f.defs = append(f.defs, def)
	return f.res, f.err
}

func newRunner(uiURL string, uiErr error, bas basRunner) *Runner {
	return &Runner{
		bas:       bas,
		resolveUI: func(context.Context, string) (string, error) { return uiURL, uiErr },
	}
}

func codes(finds []manifestvalidation.Finding) []string {
	out := make([]string, 0, len(finds))
	for _, f := range finds {
		out = append(out, f.Code)
	}
	return out
}

func TestCheckSkipsWhenUIUnavailable(t *testing.T) {
	r := newRunner("", errors.New("port not allocated"), &fakeBAS{})
	finds := r.Check(context.Background(), Input{Scenario: "demo"})
	if len(finds) != 1 || finds[0].Code != "runtime_skipped_ui_unavailable" {
		t.Fatalf("want single runtime_skipped_ui_unavailable, got %v", codes(finds))
	}
	if finds[0].Severity != manifestvalidation.SeverityInfo {
		t.Fatalf("skip must be informational, got %s", finds[0].Severity)
	}
}

func TestCheckSkipsWhenBASUnavailable(t *testing.T) {
	r := newRunner("http://localhost:5173", nil, &fakeBAS{err: errBASUnavailable})
	finds := r.Check(context.Background(), Input{Scenario: "demo"})
	if len(finds) != 1 || finds[0].Code != "runtime_skipped_bas_unavailable" {
		t.Fatalf("want single runtime_skipped_bas_unavailable, got %v", codes(finds))
	}
	if finds[0].Severity != manifestvalidation.SeverityInfo {
		t.Fatalf("BAS-down skip must be informational (graceful degradation), got %s", finds[0].Severity)
	}
}

func TestCheckHandshakePasses(t *testing.T) {
	bas := &fakeBAS{res: &runResult{loaded: true, handshakeSignaled: true, screenshotRef: "captured"}}
	r := newRunner("http://localhost:5173", nil, bas)
	finds := r.Check(context.Background(), Input{Scenario: "demo"})
	if len(finds) != 1 || finds[0].Code != "runtime_render_ok" {
		t.Fatalf("want runtime_render_ok, got %v", codes(finds))
	}
	if finds[0].Severity != manifestvalidation.SeverityInfo {
		t.Fatalf("pass must be informational, got %s", finds[0].Severity)
	}
	if len(bas.defs) != 1 {
		t.Fatalf("expected one workflow run, got %d", len(bas.defs))
	}
}

func TestCheckHandshakeFails(t *testing.T) {
	bas := &fakeBAS{res: &runResult{loaded: true, handshakeSignaled: false, handshakeError: "timeout"}}
	r := newRunner("http://localhost:5173", nil, bas)
	finds := r.Check(context.Background(), Input{Scenario: "demo"})
	if len(finds) == 0 || finds[0].Code != "runtime_handshake_failed" {
		t.Fatalf("want runtime_handshake_failed first, got %v", codes(finds))
	}
	if finds[0].Severity != manifestvalidation.SeverityError {
		t.Fatalf("handshake failure must be an error, got %s", finds[0].Severity)
	}
	if finds[0].Suggestion == "" {
		t.Fatal("handshake failure must carry remediation")
	}
}

func TestCheckConsoleErrorsSurfaceAsWarningAlongsidePass(t *testing.T) {
	bas := &fakeBAS{res: &runResult{
		loaded:            true,
		handshakeSignaled: true,
		console:           []evidence.ConsoleEntry{{Level: "error", Message: "boom"}},
	}}
	r := newRunner("http://localhost:5173", nil, bas)
	got := codes(r.Check(context.Background(), Input{Scenario: "demo"}))
	if len(got) != 2 || got[0] != "runtime_render_ok" || got[1] != "runtime_console_errors" {
		t.Fatalf("want [runtime_render_ok runtime_console_errors], got %v", got)
	}
}

func TestCodeForFailurePrecedence(t *testing.T) {
	intp := func(n int) *int { return &n }
	cases := []struct {
		name string
		ev   evidence.Evidence
		want string
	}{
		{"not loaded", evidence.Evidence{Loaded: false}, "runtime_load_failed"},
		{"handshake", evidence.Evidence{Loaded: true, Handshake: evidence.Handshake{Signaled: false}}, "runtime_handshake_failed"},
		{"network", evidence.Evidence{Loaded: true, Handshake: evidence.Handshake{Signaled: true}, Network: []evidence.NetworkEntry{{URL: "x", Status: intp(500)}}}, "runtime_network_failure"},
		{"render", evidence.Evidence{Loaded: true, Handshake: evidence.Handshake{Signaled: true}, RenderBroken: true}, "runtime_render_broken"},
		{"page error", evidence.Evidence{Loaded: true, Handshake: evidence.Handshake{Signaled: true}, PageErrors: []evidence.PageError{{Message: "boom"}}}, "runtime_page_error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := codeForFailure(c.ev); got != c.want {
				t.Fatalf("codeForFailure = %q, want %q", got, c.want)
			}
		})
	}
}
