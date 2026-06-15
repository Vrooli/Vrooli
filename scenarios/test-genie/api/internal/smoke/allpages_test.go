package smoke

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"test-genie/internal/browsercapture"
	"test-genie/internal/pagediscovery"
	sharedartifacts "test-genie/internal/shared/artifacts"

	capturepb "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

// writeSolidScreenshot writes a real solid-color PNG (a clearly broken render)
// and returns a capture response pointing at it.
func writeSolidScreenshot(t *testing.T, name string, c color.Color) *capturepb.CaptureResponse {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return screenshotResponse(path)
}

// lighthouseScenario writes a scenario dir with a UI and a multi-page
// lighthouse.json so all-pages discovery yields more than the fallback page.
func lighthouseScenario(t *testing.T, pages string) string {
	t.Helper()
	dir := scenarioDirWithUI(t)
	vrooliDir := filepath.Join(dir, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"enabled": true, "pages": ` + pages + `}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "lighthouse.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// writeServerScreenshot creates a fake BAS server-side screenshot file and
// returns a capture response pointing at it (the single-host shared-filesystem
// path the smoke writer reads directly).
func writeServerScreenshot(t *testing.T, name string) *capturepb.CaptureResponse {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("PNGDATA-"+name), 0o644); err != nil {
		t.Fatal(err)
	}
	return screenshotResponse(path)
}

func screenshotResponse(path string) *capturepb.CaptureResponse {
	return &capturepb.CaptureResponse{
		Artifacts: []*capturepb.CaptureArtifact{
			{Type: capturepb.CaptureType_CAPTURE_TYPE_SCREENSHOT, Path: path},
		},
	}
}

// TestCostGuard_DefaultProfileSinglePageOnly is the REQUIRED cost guard: with no
// capture profile, the smoke runner must issue exactly ONE capture (the
// single-page workflow handshake smoke) and ZERO all-pages CaptureService calls,
// so routine comprehensive cost is unchanged.
func TestCostGuard_DefaultProfileSinglePageOnly(t *testing.T) {
	dir := lighthouseScenario(t, `[{"path":"/"},{"path":"/a"},{"path":"/b"}]`)

	// Track the workflow (single-page) capturer calls via the fake.
	workflow := &browsercapture.FakeWorkflowClient{Timeline: handshakeTimeline(true, "ref"), Asset: []byte("PNG")}
	capture := &browsercapture.FakeCaptureClient{Response: screenshotResponse("/srv/shot.png")}

	// Default profile: do NOT wire all-pages.
	runner := NewRunner(browsercapture.New(workflow), WithUIURL("http://localhost:3000"))

	result, err := runner.Run(context.Background(), "demo", dir, "run-default")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusPassed {
		t.Fatalf("status = %v, want passed", result.Status)
	}
	if len(result.PageCaptures) != 0 {
		t.Fatalf("default profile must not produce page captures, got %d", len(result.PageCaptures))
	}
	if capture.CallCount() != 0 {
		t.Fatalf("default profile must issue ZERO CaptureService calls, got %d", capture.CallCount())
	}
}

// TestBaselineProfile_AllPagesPlusVideo verifies the baseline profile issues one
// CaptureService.Capture per discovered page, requests VIDEO, and records the
// captures + artifacts.
func TestBaselineProfile_AllPagesPlusVideo(t *testing.T) {
	dir := lighthouseScenario(t, `[{"path":"/","label":"Home"},{"path":"/backlog","label":"Backlog"}]`)

	workflow := &browsercapture.FakeWorkflowClient{Timeline: handshakeTimeline(true, "ref"), Asset: []byte("PNG")}
	capture := &browsercapture.FakeCaptureClient{
		Responses: map[string]*capturepb.CaptureResponse{
			"scenario=demo,path=/":        writeServerScreenshot(t, "root.png"),
			"scenario=demo,path=/backlog": writeServerScreenshot(t, "backlog.png"),
		},
	}
	mc := browsercapture.NewMultiCapturer(capture)

	runner := NewRunner(browsercapture.New(workflow),
		WithUIURL("http://localhost:3000"),
		WithAllPagesCapture(mc, true),
	)

	result, err := runner.Run(context.Background(), "demo", dir, "run-baseline")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusPassed {
		t.Fatalf("status = %v, want passed (%s)", result.Status, result.Message)
	}
	if len(result.PageCaptures) != 2 {
		t.Fatalf("expected 2 page captures, got %d", len(result.PageCaptures))
	}
	if capture.CallCount() != 2 {
		t.Fatalf("expected 2 CaptureService calls (one per page), got %d", capture.CallCount())
	}
	// Video must be requested.
	for _, req := range capture.Requests {
		hasVideo := false
		for _, c := range req.GetCaptures() {
			if c == capturepb.CaptureType_CAPTURE_TYPE_VIDEO {
				hasVideo = true
			}
		}
		if !hasVideo {
			t.Fatalf("baseline profile must request VIDEO for %s", req.GetUrl())
		}
	}

	// Visual artifacts must be enumerable via the run-artifact seam.
	visuals, err := sharedartifacts.ListRunVisuals(dir, "run-baseline")
	if err != nil {
		t.Fatalf("ListRunVisuals error: %v", err)
	}
	if len(visuals) != 2 {
		t.Fatalf("expected 2 enumerated visuals, got %d", len(visuals))
	}
	for _, v := range visuals {
		if v.ScreenshotRelPath == "" {
			t.Fatalf("visual %q missing screenshot rel path", v.Page)
		}
	}
}

// TestBaselineProfile_FailingPageDemotesResult verifies a page with network
// failures demotes the overall smoke result even when the single-page handshake
// passed.
func TestBaselineProfile_FailingPageDemotesResult(t *testing.T) {
	dir := lighthouseScenario(t, `[{"path":"/"},{"path":"/broken"}]`)

	workflow := &browsercapture.FakeWorkflowClient{Timeline: handshakeTimeline(true, "ref"), Asset: []byte("PNG")}
	brokenResp := &capturepb.CaptureResponse{
		Artifacts: []*capturepb.CaptureArtifact{
			{Type: capturepb.CaptureType_CAPTURE_TYPE_SCREENSHOT, Path: "/srv/broken.png"},
			{Type: capturepb.CaptureType_CAPTURE_TYPE_NETWORK, Metadata: map[string]string{"failure_count": "3"}},
		},
	}
	capture := &browsercapture.FakeCaptureClient{
		Responses: map[string]*capturepb.CaptureResponse{
			"scenario=demo,path=/":       screenshotResponse("/srv/root.png"),
			"scenario=demo,path=/broken": brokenResp,
		},
	}
	mc := browsercapture.NewMultiCapturer(capture)

	runner := NewRunner(browsercapture.New(workflow),
		WithUIURL("http://localhost:3000"),
		WithAllPagesCapture(mc, false),
	)

	result, err := runner.Run(context.Background(), "demo", dir, "run-broken")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusFailed {
		t.Fatalf("status = %v, want failed (a broken page must demote the result)", result.Status)
	}
}

// TestBaselineProfile_BlankRenderHardFails verifies a page whose screenshot is
// a solid color (a clearly-broken render) demotes the overall result via the
// pixel render-health check — even with a clean handshake and no network/page
// errors. The page-named blank message must surface.
func TestBaselineProfile_BlankRenderHardFails(t *testing.T) {
	dir := lighthouseScenario(t, `[{"path":"/"},{"path":"/blank"}]`)

	workflow := &browsercapture.FakeWorkflowClient{Timeline: handshakeTimeline(true, "ref"), Asset: []byte("PNG")}
	capture := &browsercapture.FakeCaptureClient{
		Responses: map[string]*capturepb.CaptureResponse{
			"scenario=demo,path=/":      writeServerScreenshot(t, "root.png"), // non-PNG bytes → render-health skipped
			"scenario=demo,path=/blank": writeSolidScreenshot(t, "blank.png", color.RGBA{255, 255, 255, 255}),
		},
	}
	mc := browsercapture.NewMultiCapturer(capture)

	runner := NewRunner(browsercapture.New(workflow),
		WithUIURL("http://localhost:3000"),
		WithAllPagesCapture(mc, false),
	)

	result, err := runner.Run(context.Background(), "demo", dir, "run-blank")
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if result.Status != StatusFailed {
		t.Fatalf("status = %v, want failed (a blank render must demote the result)", result.Status)
	}
	var blank *PageCapture
	for i := range result.PageCaptures {
		if result.PageCaptures[i].Page == "/blank" {
			blank = &result.PageCaptures[i]
		}
	}
	if blank == nil {
		t.Fatal("missing /blank page capture")
	}
	if blank.Status != StatusFailed {
		t.Fatalf("/blank status = %v, want failed", blank.Status)
	}
	if !bytes.Contains([]byte(blank.Message), []byte("blank/solid color")) {
		t.Fatalf("/blank message = %q, want it to name a blank/solid render", blank.Message)
	}
}

// TestAllPages_DiscovererOverride confirms the runner honors an injected page
// discoverer (filesystem seam) for all-pages enumeration.
func TestAllPages_DiscovererOverride(t *testing.T) {
	dir := scenarioDirWithUI(t) // no lighthouse.json on disk

	cfg := `{"enabled":true,"pages":[{"path":"/x"}]}`
	fakeFS := pagediscovery.FakeFileReader{Files: map[string][]byte{
		filepath.Join(dir, ".vrooli", "lighthouse.json"): []byte(cfg),
	}}

	capture := &browsercapture.FakeCaptureClient{Response: screenshotResponse("/srv/x.png")}
	mc := browsercapture.NewMultiCapturer(capture)
	workflow := &browsercapture.FakeWorkflowClient{Timeline: handshakeTimeline(true, "ref"), Asset: []byte("PNG")}

	runner := NewRunner(browsercapture.New(workflow),
		WithUIURL("http://localhost:3000"),
		WithAllPagesCapture(mc, false),
		WithPageDiscoverer(pagediscovery.New(fakeFS)),
	)

	if _, err := runner.Run(context.Background(), "demo", dir, "run-disc"); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if capture.CallCount() != 1 {
		t.Fatalf("expected 1 capture for the single discovered page, got %d", capture.CallCount())
	}
	if got := capture.Requests[0].GetUrl(); got != "scenario=demo,path=/x" {
		t.Fatalf("unexpected capture URL %q", got)
	}
}
