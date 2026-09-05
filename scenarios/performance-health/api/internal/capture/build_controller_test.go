package capture

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:PH-CAPTURE-005] StartProfile returns the UI URL only after the served
// bundle is verified to carry a react-dom/profiling marker (profile build took).
func TestStartProfileVerifiesInstrumentedBundle(t *testing.T) {
	c := &CLIBuildController{
		Restart:      func(context.Context, string, bool) error { return nil },
		ResolveUIURL: func(context.Context, string) (string, error) { return "http://ui", nil },
		VerifyBundle: func(context.Context, string) (bool, error) { return true, nil },
	}
	url, err := c.StartProfile(context.Background(), "demo")
	if err != nil {
		t.Fatalf("StartProfile: %v", err)
	}
	if url != "http://ui" {
		t.Fatalf("expected ui url, got %q", url)
	}
}

// [REQ:PH-CAPTURE-005] When the served bundle is NOT instrumented, StartProfile
// FAILS (so the orchestrator does not mislabel a non-profile capture as Tier 1).
func TestStartProfileFailsOnUninstrumentedBundle(t *testing.T) {
	c := &CLIBuildController{
		Restart:      func(context.Context, string, bool) error { return nil },
		ResolveUIURL: func(context.Context, string) (string, error) { return "http://ui", nil },
		VerifyBundle: func(context.Context, string) (bool, error) { return false, nil },
	}
	if _, err := c.StartProfile(context.Background(), "demo"); err == nil {
		t.Fatal("expected error for uninstrumented bundle")
	}
}

// [REQ:PH-CAPTURE-003] A scenario serving no UI URL yields ("",nil) — a clean
// skip upstream, not an error.
func TestStartProfileNoUIIsCleanSkip(t *testing.T) {
	c := &CLIBuildController{
		Restart:      func(context.Context, string, bool) error { return nil },
		ResolveUIURL: func(context.Context, string) (string, error) { return "", errors.New("no ui port") },
	}
	url, err := c.StartProfile(context.Background(), "demo")
	if err != nil || url != "" {
		t.Fatalf("expected clean skip, got url=%q err=%v", url, err)
	}
}

// [REQ:PH-CAPTURE-005] A failed restart surfaces an error (not a silent skip).
func TestStartProfileRestartError(t *testing.T) {
	c := &CLIBuildController{
		Restart: func(context.Context, string, bool) error { return errors.New("restart boom") },
	}
	if _, err := c.StartProfile(context.Background(), "demo"); err == nil {
		t.Fatal("expected restart error")
	}
}

// [REQ:PH-CAPTURE-005] defaultVerifyBundle finds the marker in a linked JS bundle.
func TestDefaultVerifyBundleScansLinkedScript(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><script src="/assets/app.js"></script></body></html>`))
	})
	mux.HandleFunc("/assets/app.js", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`function injectProfilingHooks(){}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ok, err := defaultVerifyBundle(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("expected marker found in linked bundle")
	}
}

// [REQ:PH-CAPTURE-005] defaultVerifyBundle reports false when no bundle carries
// the marker (default prod build).
func TestDefaultVerifyBundleAbsentMarker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><div id="root"></div></body></html>`))
	}))
	defer srv.Close()
	ok, err := defaultVerifyBundle(context.Background(), srv.Client(), srv.URL)
	if err != nil || ok {
		t.Fatalf("expected absent marker, got ok=%v err=%v", ok, err)
	}
}

// TestBundleMarkerRejectsAppLevelProfilerName: a plain production bundle that
// merely mentions the app's own onProfilerRender export must NOT verify as
// instrumented. When a scenario reaches its profiler util through a dynamic
// import, that name becomes a chunk export and survives minification in the
// default build too — keying on it reported a plain prod bundle as a perf
// build, and the capture then produced a trace with no ⚛ marks that read as
// "no component activity" rather than "not a perf build".
func TestBundleMarkerRejectsAppLevelProfilerName(t *testing.T) {
	defaultBundle := `const {onProfilerRender:n}=await import("./profiler-D7qAmvGi.js");`
	if containsBundleMarker(defaultBundle) {
		t.Errorf("a default production bundle verified as instrumented: %s", defaultBundle)
	}

	profileBundle := `function injectProfilingHooks(){};function markComponentRenderStarted(){}`
	if !containsBundleMarker(profileBundle) {
		t.Errorf("a react-dom/profiling bundle failed to verify: %s", profileBundle)
	}

	// Each marker independently proves the profiling build, so one React
	// rename degrades the check rather than breaking it.
	for _, marker := range bundleMarkers {
		if !containsBundleMarker("prefix " + marker + " suffix") {
			t.Errorf("marker %q did not match on its own", marker)
		}
	}
}
