package visualhealth

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"slices"
	"testing"

	"ui-health/internal/visualhealth/pixel"

	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

func testPNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func testSolidPNG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 80, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 80; x++ {
			img.Set(x, y, color.White)
		}
	}
	return testPNG(t, img)
}

func testGradientPNG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 80, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 80; x++ {
			v := uint8(x * 255 / 80)
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return testPNG(t, img)
}

func testChromePNG(t *testing.T, top, bottom, left, right color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			v := uint8(60 + (x+y)%120)
			img.Set(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	for y := 0; y < 12; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, top)
		}
	}
	for y := 90; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, bottom)
		}
	}
	for y := 0; y < 100; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, left)
		}
		for x := 94; x < 100; x++ {
			img.Set(x, y, right)
		}
	}
	return testPNG(t, img)
}

func TestAnalyzeReportsPixelBlank(t *testing.T) {
	a := NewAnalyzer(pixel.DefaultThresholds())
	resp := a.Analyze(&visualpb.AnalyzeArtifactsRequest{Scenario: "demo", Steps: []*visualpb.VisualStepArtifact{{
		StepId:        "load",
		Url:           "http://demo.local/",
		ScreenshotPng: testSolidPNG(t),
	}}})
	if resp.GetVerdict() != "failed" {
		t.Fatalf("verdict = %q, want failed", resp.GetVerdict())
	}
	if len(resp.GetFindings()) != 1 || resp.GetFindings()[0].GetCode() != "visual_pixel_blank" {
		t.Fatalf("findings = %+v, want visual_pixel_blank", resp.GetFindings())
	}
}

func TestAnalyzeHealthyScreenshotPasses(t *testing.T) {
	a := NewAnalyzer(pixel.DefaultThresholds())
	resp := a.Analyze(&visualpb.AnalyzeArtifactsRequest{Scenario: "demo", Steps: []*visualpb.VisualStepArtifact{{
		StepId:        "load",
		ScreenshotPng: testGradientPNG(t),
		DomHtml:       "<main><h1>Ready</h1></main>",
	}}})
	if resp.GetVerdict() != "passed" {
		t.Fatalf("verdict = %q, want passed; findings=%+v", resp.GetVerdict(), resp.GetFindings())
	}
}

func TestAnalyzeReportsDOMBlank(t *testing.T) {
	a := NewAnalyzer(pixel.DefaultThresholds())
	resp := a.Analyze(&visualpb.AnalyzeArtifactsRequest{Scenario: "demo", Steps: []*visualpb.VisualStepArtifact{{
		StepId:  "step-1",
		DomHtml: "<main><script>hello()</script><style>.x{}</style><div> &nbsp; </div></main>",
	}}})
	if resp.GetVerdict() != "failed" {
		t.Fatalf("verdict = %q, want failed", resp.GetVerdict())
	}
	if len(resp.GetFindings()) != 1 || resp.GetFindings()[0].GetCode() != "visual_dom_blank" {
		t.Fatalf("findings = %+v, want visual_dom_blank", resp.GetFindings())
	}
}

func TestDOMAccessibleTextIsMeaningful(t *testing.T) {
	a := NewAnalyzer(pixel.DefaultThresholds())
	resp := a.Analyze(&visualpb.AnalyzeArtifactsRequest{Scenario: "demo", Steps: []*visualpb.VisualStepArtifact{{
		StepId:  "step-1",
		DomHtml: `<main><button aria-label="Save"></button><img alt="Logo"></main>`,
	}}})
	if resp.GetVerdict() != "passed" {
		t.Fatalf("verdict = %q, want passed; findings=%+v", resp.GetVerdict(), resp.GetFindings())
	}
}

func TestCompareArtifacts(t *testing.T) {
	a := NewAnalyzer(pixel.DefaultThresholds())
	resp := a.Compare(&visualpb.CompareArtifactsRequest{
		Base: []*visualpb.CompareArtifact{
			{Page: "/", Label: "Home", ScreenshotPng: testGradientPNG(t)},
			{Page: "/removed", ScreenshotPng: testGradientPNG(t)},
		},
		Current: []*visualpb.CompareArtifact{
			{Page: "/", Label: "Home", ScreenshotPng: testGradientPNG(t)},
			{Page: "/added", ScreenshotPng: testGradientPNG(t)},
		},
	})
	byPage := map[string]string{}
	for _, d := range resp.GetDeltas() {
		byPage[d.GetPage()] = d.GetStatus()
	}
	if byPage["/"] != "identical" || byPage["/added"] != "added" || byPage["/removed"] != "removed" {
		t.Fatalf("statuses = %#v", byPage)
	}
}

func TestAnalyzeReportsBrokenVisualAsset(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId: "assets",
		Network: []*visualpb.NetworkEntry{{
			Url:          "https://example.test/logo.png",
			ResourceType: "image",
			Status:       404,
		}},
	}}})
	assertFindingCodes(t, resp, "visual_broken_asset")
}

func TestAnalyzeReportsStuckLoading(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:  "loading",
		DomHtml: `<main aria-busy="true"><div class="spinner">Loading...</div></main>`,
	}}})
	assertFindingCodes(t, resp, "visual_stuck_loading")
}

func TestAnalyzeReportsViewportOverflow(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId: "layout",
		Viewport: &visualpb.Viewport{
			Width:  360,
			Height: 640,
		},
		LayoutJson: `{"document":{"scrollWidth":460,"scrollHeight":640},"elements":[]}`,
	}}})
	assertFindingCodes(t, resp, "visual_viewport_overflow")
}

func TestAnalyzeReportsOffscreenInteractive(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId: "layout",
		Viewport: &visualpb.Viewport{
			Width:  360,
			Height: 640,
		},
		LayoutJson: `{"document":{"scrollWidth":360,"scrollHeight":640},"elements":[{"selector":"#save","tag":"button","rect":{"x":390,"y":40,"width":80,"height":32}}]}`,
	}}})
	assertFindingCodes(t, resp, "visual_offscreen_interactive")
}

func TestAnalyzeAllowsInteractiveControlsInsideOwnedScrollContainers(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:     "layout",
		Viewport:   &visualpb.Viewport{Width: 360, Height: 640},
		LayoutJson: `{"document":{"scrollWidth":360,"scrollHeight":640},"elements":[{"selector":"#later","tag":"button","inScrollContainer":true,"rect":{"x":20,"y":800,"width":80,"height":32}}]}`,
	}}})
	for _, finding := range resp.GetFindings() {
		if finding.GetCode() == "visual_offscreen_interactive" {
			t.Fatalf("owned scroll content must not be reported as unreachable: %+v", resp.GetFindings())
		}
	}
}

func TestAnalyzeReportsClippedText(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:     "layout",
		LayoutJson: `{"elements":[{"selector":".title","text":"Important visible label","clientWidth":80,"scrollWidth":160,"clientHeight":24,"scrollHeight":24,"overflowX":"hidden","rect":{"x":0,"y":0,"width":80,"height":24}}]}`,
	}}})
	assertFindingCodes(t, resp, "visual_text_clipped")
}

func TestAnalyzeReportsFocusZoomRisk(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId: "layout",
		Viewport: &visualpb.Viewport{
			Width:  390,
			Height: 844,
		},
		LayoutJson: `{"elements":[{"selector":"input[name=q]","tag":"input","type":"search","fontSize":14,"rect":{"x":20,"y":30,"width":240,"height":36}}]}`,
	}}})
	assertFindingCodes(t, resp, "visual_focus_zoom_risk")
}

func TestAnalyzeReportsBlockingOverlay(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId: "layout",
		Viewport: &visualpb.Viewport{
			Width:  400,
			Height: 700,
		},
		LayoutJson: `{"elements":[{"selector":".blocker","position":"fixed","pointerEvents":"auto","rect":{"x":0,"y":0,"width":400,"height":650}}]}`,
	}}})
	assertFindingCodes(t, resp, "visual_blocking_overlay")
}

func TestAnalyzeChromeColorsPassWhenDeclaredMatchesRendered(t *testing.T) {
	chrome := color.RGBA{R: 2, G: 6, B: 23, A: 255}
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:        "chrome",
		ScreenshotPng: testChromePNG(t, chrome, chrome, chrome, chrome),
		LayoutJson: `{
			"chrome":{"themeColor":"#020617","statusBarColor":"#020617","safeAreaColor":"#020617"},
			"safeArea":{"top":12,"right":6,"bottom":10,"left":8},
			"document":{"scrollWidth":100,"scrollHeight":100},
			"elements":[]
		}`,
	}}})
	if resp.GetVerdict() != "passed" {
		t.Fatalf("verdict = %q, want passed; findings=%+v", resp.GetVerdict(), resp.GetFindings())
	}
}

func TestAnalyzeReportsChromeColorMismatches(t *testing.T) {
	declared := color.RGBA{R: 2, G: 6, B: 23, A: 255}
	wrong := color.RGBA{R: 239, G: 68, B: 68, A: 255}
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:        "chrome",
		ScreenshotPng: testChromePNG(t, wrong, declared, declared, declared),
		LayoutJson: `{
			"chrome":{"themeColor":"#020617","statusBarColor":"#020617","safeAreaColor":"#020617"},
			"safeArea":{"top":12,"right":6,"bottom":10,"left":8},
			"document":{"scrollWidth":100,"scrollHeight":100},
			"elements":[]
		}`,
	}}})
	assertFindingCodes(t, resp, "visual_status_bar_color_mismatch", "visual_safe_area_color_mismatch")
}

func TestAnalyzeReportsUnsafeEdgeTapZone(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId: "layout",
		Viewport: &visualpb.Viewport{
			Width:  390,
			Height: 844,
		},
		LayoutJson: `{
			"safeArea":{"top":44,"right":0,"bottom":34,"left":0},
			"document":{"scrollWidth":390,"scrollHeight":844},
			"elements":[{"selector":"#menu","tag":"button","rect":{"x":16,"y":8,"width":44,"height":36}}]
		}`,
	}}})
	assertFindingCodes(t, resp, "visual_unsafe_edge_tap_zone")
}

func TestIntentionalDialogDoesNotReportBlockingOverlay(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId: "layout",
		Viewport: &visualpb.Viewport{
			Width:  400,
			Height: 700,
		},
		LayoutJson: `{"elements":[{"selector":"[role=dialog]","role":"dialog","position":"fixed","pointerEvents":"auto","rect":{"x":0,"y":0,"width":400,"height":650}}]}`,
	}}})
	if resp.GetVerdict() != "passed" {
		t.Fatalf("verdict = %q, want passed; findings=%+v", resp.GetVerdict(), resp.GetFindings())
	}
}

func TestRulesIncludesPhaseThreeCodes(t *testing.T) {
	var ids []string
	for _, rule := range Rules() {
		ids = append(ids, rule.GetId())
	}
	for _, want := range []string{
		"visual_pixel_blank",
		"visual_dom_blank",
		"visual_broken_asset",
		"visual_stuck_loading",
		"visual_viewport_overflow",
		"visual_offscreen_interactive",
		"visual_text_clipped",
		"visual_focus_zoom_risk",
		"visual_blocking_overlay",
		"visual_status_bar_color_mismatch",
		"visual_safe_area_color_mismatch",
		"visual_unsafe_edge_tap_zone",
	} {
		if !slices.Contains(ids, want) {
			t.Fatalf("Rules() ids = %#v, missing %s", ids, want)
		}
	}
}

func assertFindingCodes(t *testing.T, resp *visualpb.AnalyzeArtifactsResponse, want ...string) {
	t.Helper()
	got := map[string]bool{}
	for _, finding := range resp.GetFindings() {
		got[finding.GetCode()] = true
	}
	for _, code := range want {
		if !got[code] {
			t.Fatalf("findings = %+v, want code %s", resp.GetFindings(), code)
		}
	}
}
