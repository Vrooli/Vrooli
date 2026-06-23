package smoke

import (
	"context"
	"fmt"
	"strings"
	"testing"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// --- Phase 0 unit tests: delegation to the canonical freshness engine ---

func TestUIBundleStaleReason(t *testing.T) {
	cases := []struct {
		name, target, cause, file, want string
	}{
		{
			"cause and file", "ui/dist/index.html", "content changed", "packages/iframe-bridge/dist/index.js",
			"ui/dist/index.html stale (content changed): packages/iframe-bridge/dist/index.js",
		},
		{
			"cause only", "ui/dist/index.html", "missing artifact", "",
			"ui/dist/index.html stale (missing artifact)",
		},
		{"neither", "ui/dist/index.html", "", "", "ui/dist/index.html stale"},
		{"empty target defaults", "", "source newer", "", "UI bundle stale (source newer)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := uiBundleStaleReason(tc.target, tc.cause, tc.file); got != tc.want {
				t.Errorf("uiBundleStaleReason(%q,%q,%q) = %q, want %q", tc.target, tc.cause, tc.file, got, tc.want)
			}
		})
	}
}

// fakeVrooliRunner is a vroolicli.Runner returning a fixed JSON body, used to
// drive cliFreshnessChecker without shelling the real CLI.
type fakeVrooliRunner struct {
	out     []byte
	err     error
	gotArgs []string
}

func (f *fakeVrooliRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	f.gotArgs = args
	return f.out, f.err
}

func (f *fakeVrooliRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return f.Run(ctx, name, args...)
}

func checkerWithResponse(body string, err error) (cliFreshnessChecker, *fakeVrooliRunner) {
	runner := &fakeVrooliRunner{out: []byte(body), err: err}
	return cliFreshnessChecker{client: vroolicli.New(vroolicli.WithRunner(runner))}, runner
}

// The canonical content-hash engine reports each §1.5 false-negative as a stale
// ui-bundle check; this asserts smoke surfaces every one of them as stale (the
// mtime walk would have reported them fresh). Detection itself is the engine's
// concern and covered by its own tests; here we pin the delegation contract.
func TestCLIFreshnessChecker_StaleUIBundleCases(t *testing.T) {
	cases := []struct {
		name, cause, file, wantReason string
	}{
		// §1.5(2): file: workspace dep (iframe-bridge) changed, ui/src untouched.
		{
			"file dep changed", "content changed", "packages/iframe-bridge/dist/index.js",
			"ui/dist/index.html stale (content changed): packages/iframe-bridge/dist/index.js",
		},
		// §1.5(3): NODE_ENV flip — keyed build input differs.
		{
			"node_env keyed input", "key input changed", "node_env",
			"ui/dist/index.html stale (key input changed): node_env",
		},
		// §1.5(1): bundle missing / no recorded check — reported as missing artifact.
		{
			"missing artifact", "missing artifact", "ui/dist/index.html",
			"ui/dist/index.html stale (missing artifact): ui/dist/index.html",
		},
		// §1.5(4): per-file content differs even when the dist dir mtime is newer.
		{
			"per-file content", "content changed", "ui/src/App.tsx",
			"ui/dist/index.html stale (content changed): ui/src/App.tsx",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"success":true,"scenario":"demo","stale":true,"checks":[` +
				`{"check_type":"ui-bundle","target":"ui/dist/index.html","stale":true,"cause":"` + tc.cause + `","file":"` + tc.file + `"}]}`
			checker, runner := checkerWithResponse(body, nil)
			stale, reason, err := checker.UIBundleStale(context.Background(), "demo", "/scn/demo")
			if err != nil {
				t.Fatalf("UIBundleStale error: %v", err)
			}
			if !stale {
				t.Fatalf("expected stale for %s", tc.name)
			}
			if reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", reason, tc.wantReason)
			}
			// The path is threaded so the engine resolves the scenario.
			joined := strings.Join(runner.gotArgs, " ")
			if !strings.Contains(joined, "scenario freshness demo --json") || !strings.Contains(joined, "--path /scn/demo") {
				t.Errorf("unexpected CLI args: %v", runner.gotArgs)
			}
		})
	}
}

// A stale *binary* must not make smoke (a UI concern) report the bundle stale.
func TestCLIFreshnessChecker_IgnoresBinariesCheck(t *testing.T) {
	body := `{"success":true,"scenario":"demo","stale":true,"checks":[` +
		`{"check_type":"binaries","target":"api/demo-api","stale":true,"cause":"content changed","file":"api/main.go"},` +
		`{"check_type":"ui-bundle","target":"ui/dist/index.html","stale":false}]}`
	checker, _ := checkerWithResponse(body, nil)
	stale, _, err := checker.UIBundleStale(context.Background(), "demo", "/scn/demo")
	if err != nil {
		t.Fatalf("UIBundleStale error: %v", err)
	}
	if stale {
		t.Fatal("ui-bundle is fresh; a stale binary must not block smoke")
	}
}

func TestCLIFreshnessChecker_FreshBundle(t *testing.T) {
	body := `{"success":true,"scenario":"demo","stale":false,"checks":[{"check_type":"ui-bundle","target":"ui/dist/index.html","stale":false}]}`
	checker, _ := checkerWithResponse(body, nil)
	stale, reason, err := checker.UIBundleStale(context.Background(), "demo", "/scn/demo")
	if err != nil || stale || reason != "" {
		t.Fatalf("expected fresh, got stale=%v reason=%q err=%v", stale, reason, err)
	}
}

func TestCLIFreshnessChecker_PropagatesError(t *testing.T) {
	checker, _ := checkerWithResponse("", errFreshnessUnavailable)
	if _, _, err := checker.UIBundleStale(context.Background(), "demo", "/scn/demo"); err == nil {
		t.Fatal("expected error to propagate so the runner can degrade gracefully")
	}
}

// fakeFreshness is a freshnessChecker stub for runner-level wiring tests.
type fakeFreshness struct {
	stale  bool
	reason string
	err    error
}

func (f fakeFreshness) UIBundleStale(context.Context, string, string) (bool, string, error) {
	return f.stale, f.reason, f.err
}

// A stale verdict blocks the run with the bundle-stale reason and restart hint.
func TestRunner_StaleBundleBlocks(t *testing.T) {
	dir := scenarioDirWithUI(t)
	runner := NewRunner(fakeCapturer(handshakeTimeline(true, "ref")),
		WithUIURL("http://localhost:3000"),
		WithFreshnessChecker(fakeFreshness{stale: true, reason: "ui/dist/index.html stale (content changed): packages/iframe-bridge/dist/index.js"}))

	result, err := runner.Run(context.Background(), "demo", dir, "run-stale")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusBlocked {
		t.Fatalf("status = %v, want blocked", result.Status)
	}
	if result.BlockedReason != BlockedReasonBundleStale {
		t.Errorf("blocked reason = %v, want %v", result.BlockedReason, BlockedReasonBundleStale)
	}
	if result.Bundle == nil || result.Bundle.Fresh {
		t.Errorf("expected Bundle.Fresh=false, got %+v", result.Bundle)
	}
	if !strings.Contains(result.Message, "vrooli scenario restart demo") {
		t.Errorf("message missing restart remediation: %q", result.Message)
	}
}

// A freshness resolution error degrades gracefully: the run proceeds rather than
// blocking on an infra hiccup.
func TestRunner_FreshnessErrorDegradesGracefully(t *testing.T) {
	dir := scenarioDirWithUI(t)
	runner := NewRunner(fakeCapturer(handshakeTimeline(true, "ref")),
		WithUIURL("http://localhost:3000"),
		WithFreshnessChecker(fakeFreshness{err: errFreshnessUnavailable}))

	result, err := runner.Run(context.Background(), "demo", dir, "run-degrade")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusPassed {
		t.Fatalf("status = %v, want passed (freshness error must not block)", result.Status)
	}
}

var errFreshnessUnavailable = fmt.Errorf("freshness unavailable")
