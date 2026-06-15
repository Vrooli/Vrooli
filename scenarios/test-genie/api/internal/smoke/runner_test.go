package smoke

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/browsercapture"
	"test-genie/internal/playbooks/execution"

	tl "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// scenarioDirWithUI creates a scenario dir with a ui/ directory and an
// iframe-bridge dependency so the runner's preflight reaches the capture step.
func scenarioDirWithUI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(filepath.Join(uiDir, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A built bundle so the freshness preflight passes.
	if err := os.WriteFile(filepath.Join(uiDir, "dist", "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	pkg := `{"dependencies":{"@vrooli/iframe-bridge":"^1.0.0"}}`
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// fakeCapturer builds a Capturer over a fake workflow client returning the given
// timeline.
func fakeCapturer(timeline *execution.ParsedTimeline) *browsercapture.Capturer {
	return browsercapture.New(&browsercapture.FakeWorkflowClient{Timeline: timeline, Asset: []byte("PNG")})
}

func handshakeTimeline(passed bool, screenshotRef string) *execution.ParsedTimeline {
	frames := []execution.ParsedFrame{
		{
			NodeID:    "smoke-assert-handshake",
			StepType:  "assert",
			Success:   passed,
			Assertion: &execution.ParsedAssertion{Passed: passed, Error: errorFor(passed)},
		},
		{
			NodeID:     "smoke-screenshot-frame",
			StepType:   "screenshot",
			Success:    true,
			Screenshot: &execution.FrameScreenshot{URL: screenshotRef},
		},
	}
	return &execution.ParsedTimeline{Frames: frames, Proto: &tl.ExecutionTimeline{}}
}

func errorFor(passed bool) string {
	if passed {
		return ""
	}
	return "handshake timed out"
}

func TestRunner_Pass(t *testing.T) {
	dir := scenarioDirWithUI(t)
	runner := NewRunner(fakeCapturer(handshakeTimeline(true, "http://bas/shot.png")),
		WithUIURL("http://localhost:3000"))

	result, err := runner.Run(context.Background(), "demo", dir, "run-1")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusPassed {
		t.Fatalf("status = %v, want passed (%s)", result.Status, result.Message)
	}
	if !result.Handshake.Signaled {
		t.Fatalf("expected handshake signaled")
	}
}

func TestRunner_HandshakeFailFails(t *testing.T) {
	dir := scenarioDirWithUI(t)
	runner := NewRunner(fakeCapturer(handshakeTimeline(false, "")),
		WithUIURL("http://localhost:3000"))

	result, err := runner.Run(context.Background(), "demo", dir, "run-1")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusFailed {
		t.Fatalf("status = %v, want failed", result.Status)
	}
}

func TestRunner_NoUIDirectorySkips(t *testing.T) {
	dir := t.TempDir() // no ui/ subdir
	runner := NewRunner(fakeCapturer(handshakeTimeline(true, "")), WithUIURL("http://localhost:3000"))

	result, err := runner.Run(context.Background(), "demo", dir, "run-1")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusSkipped {
		t.Fatalf("status = %v, want skipped", result.Status)
	}
}

func TestRunner_MissingIframeBridgeFails(t *testing.T) {
	dir := t.TempDir()
	uiDir := filepath.Join(dir, "ui")
	if err := os.MkdirAll(filepath.Join(uiDir, "dist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "dist", "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	// package.json without the iframe-bridge dependency.
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"dependencies":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(fakeCapturer(handshakeTimeline(true, "")), WithUIURL("http://localhost:3000"))

	result, err := runner.Run(context.Background(), "demo", dir, "run-1")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusFailed {
		t.Fatalf("status = %v, want failed (missing iframe-bridge)", result.Status)
	}
}

func TestRunner_ArtifactsWritten(t *testing.T) {
	dir := scenarioDirWithUI(t)
	runner := NewRunner(fakeCapturer(handshakeTimeline(true, "ref")), WithUIURL("http://localhost:3000"))

	result, err := runner.Run(context.Background(), "demo", dir, "run-art")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Artifacts.Network == "" {
		t.Fatalf("expected network artifact path to be set")
	}
	if result.Artifacts.Screenshot == "" {
		t.Fatalf("expected screenshot artifact path to be set")
	}
	if result.Artifacts.Readme == "" {
		t.Fatalf("expected README path to be set")
	}
	if _, err := os.Stat(result.Artifacts.Readme); err != nil {
		t.Fatalf("README not written: %v", err)
	}
}
