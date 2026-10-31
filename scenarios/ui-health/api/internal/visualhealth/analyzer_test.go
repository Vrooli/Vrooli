package visualhealth

import (
	"bytes"
	"fmt"
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

func TestAnalyzeDoesNotTreatMeaningfulLightDOMAsBlank(t *testing.T) {
	a := NewAnalyzer(pixel.DefaultThresholds())
	resp := a.Analyze(&visualpb.AnalyzeArtifactsRequest{Scenario: "demo", Steps: []*visualpb.VisualStepArtifact{{
		StepId:        "light-ui",
		ScreenshotPng: testSolidPNG(t),
		DomHtml:       "<main><h1>Audio Tools</h1><button>Open diagnostics</button></main>",
	}}})
	if resp.GetVerdict() != "passed" {
		t.Fatalf("verdict = %q, want passed; findings=%+v", resp.GetVerdict(), resp.GetFindings())
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

func TestAnalyzeRequiresSameNodeVisibleKeyboardFocusBeforeClearingOffscreenFinding(t *testing.T) {
	const base = `{"viewport":{"width":390,"height":844},"elements":[{"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"doc-1:1","selector":"a.skip-link","tag":"a","rect":{"x":8,"y":-56,"width":117,"height":32}}],"keyboardFocus":%s}`
	cases := []struct {
		name        string
		focus       string
		wantFinding bool
	}{
		{
			name:        "same initially hidden node becomes keyboard visible in viewport",
			focus:       `{"focused":true,"focusVisible":true,"visible":true,"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"doc-1:1","rect":{"x":8,"y":0,"width":117,"height":32}}`,
			wantFinding: false,
		},
		{
			name:        "permanently offscreen control remains an error while another node receives focus",
			focus:       `{"focused":true,"focusVisible":true,"visible":true,"nodeKey":"#different-control","nodeIdentity":"doc-1:2","rect":{"x":8,"y":0,"width":117,"height":32}}`,
			wantFinding: true,
		},
		{
			name:        "no focus evidence",
			focus:       `{}`,
			wantFinding: true,
		},
		{
			name:        "focus is not keyboard visible",
			focus:       `{"focused":true,"focusVisible":false,"visible":true,"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"doc-1:1","rect":{"x":8,"y":0,"width":117,"height":32}}`,
			wantFinding: true,
		},
		{
			name:        "same node remains outside viewport after focus",
			focus:       `{"focused":true,"focusVisible":true,"visible":true,"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"doc-1:1","rect":{"x":8,"y":-56,"width":117,"height":32}}`,
			wantFinding: true,
		},
		{
			name:        "matching selector but replaced DOM node",
			focus:       `{"focused":true,"focusVisible":true,"visible":true,"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"new-document:1","rect":{"x":8,"y":0,"width":117,"height":32}}`,
			wantFinding: true,
		},
		{
			name:        "same node is partially clipped by viewport",
			focus:       `{"focused":true,"focusVisible":true,"visible":true,"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"doc-1:1","rect":{"x":-1,"y":0,"width":117,"height":32}}`,
			wantFinding: true,
		},
		{
			name:        "zero-sized focused element",
			focus:       `{"focused":true,"focusVisible":true,"visible":true,"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"doc-1:1","rect":{"x":8,"y":0,"width":0,"height":0}}`,
			wantFinding: true,
		},
		{
			name:        "invisible focused element",
			focus:       `{"focused":true,"focusVisible":true,"visible":false,"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"doc-1:1","rect":{"x":8,"y":0,"width":117,"height":32}}`,
			wantFinding: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			layout := fmt.Sprintf(base, tc.focus)
			resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
				StepId:     "layout",
				Viewport:   &visualpb.Viewport{Width: 390, Height: 844},
				LayoutJson: layout,
			}}})
			found := false
			for _, finding := range resp.GetFindings() {
				if finding.GetCode() == "visual_offscreen_interactive" {
					found = true
					break
				}
			}
			if found != tc.wantFinding {
				t.Fatalf("offscreen finding = %v, want %v (%+v)", found, tc.wantFinding, resp.GetFindings())
			}
		})
	}
}

func TestAnalyzeRejectsAmbiguousDuplicateFocusIdentity(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:   "layout",
		Viewport: &visualpb.Viewport{Width: 390, Height: 844},
		LayoutJson: `{"viewport":{"width":390,"height":844},"elements":[` +
			`{"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"duplicate","tag":"a","rect":{"x":8,"y":-56,"width":117,"height":32}},` +
			`{"nodeKey":"html > body > a:nth-of-type(2)","nodeIdentity":"duplicate","tag":"a","rect":{"x":8,"y":-56,"width":117,"height":32}}],` +
			`"keyboardFocus":{"focused":true,"focusVisible":true,"visible":true,"nodeKey":"html > body > a:nth-of-type(1)","nodeIdentity":"duplicate","rect":{"x":8,"y":0,"width":117,"height":32}}}`,
	}}})
	count := 0
	for _, finding := range resp.GetFindings() {
		if finding.GetCode() == "visual_offscreen_interactive" {
			count++
			if finding.GetLocation() != "html > body > a:nth-of-type(1)" && finding.GetLocation() != "html > body > a:nth-of-type(2)" {
				t.Fatalf("offscreen location is not exact enough: %q", finding.GetLocation())
			}
		}
	}
	if count != 2 {
		t.Fatalf("ambiguous focus identity must retain both offscreen findings, got %d: %+v", count, resp.GetFindings())
	}
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

func TestAnalyzeIgnoresScreenReaderOnlyClippedText(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:     "layout",
		LayoutJson: `{"elements":[{"selector":"#preview-label","text":"Preview frame","clientWidth":1,"scrollWidth":80,"clientHeight":1,"scrollHeight":16,"overflowX":"hidden","overflowY":"hidden","whiteSpace":"nowrap","position":"absolute","rect":{"x":30,"y":21,"width":1,"height":1}}]}`,
	}}})
	for _, finding := range resp.GetFindings() {
		if finding.GetCode() == "visual_text_clipped" {
			t.Fatalf("screen-reader-only text must not be reported as visible clipping: %+v", resp.GetFindings())
		}
	}
}

func TestAnalyzeReadsComputedStylesForScreenReaderOnlyText(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:     "layout",
		LayoutJson: `{"elements":[{"selector":"#preview-label","text":"Preview frame","clientWidth":1,"scrollWidth":80,"clientHeight":1,"scrollHeight":16,"computed":{"overflow":"hidden","position":"absolute","whiteSpace":"nowrap"},"rect":{"x":30,"y":21,"width":1,"height":1}}]}`,
	}}})
	for _, finding := range resp.GetFindings() {
		if finding.GetCode() == "visual_text_clipped" {
			t.Fatalf("computed screen-reader-only text must not be reported as visible clipping: %+v", resp.GetFindings())
		}
	}
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

func TestAnalyzeFocusZoomRiskRequiresEnabledMobileTextEntry(t *testing.T) {
	tests := []struct {
		name        string
		width       int32
		elementJSON string
		wantFinding bool
	}{
		{
			name:        "enabled mobile text below threshold",
			width:       390,
			elementJSON: `{"selector":"#enabled-14","tag":"input","type":"text","fontSize":14,"rect":{"x":16,"y":24,"width":240,"height":40}}`,
			wantFinding: true,
		},
		{
			name:        "enabled mobile text at threshold",
			width:       390,
			elementJSON: `{"selector":"#enabled-16","tag":"input","type":"text","fontSize":16,"rect":{"x":16,"y":24,"width":240,"height":40}}`,
		},
		{
			name:        "desktop text below mobile threshold",
			width:       1440,
			elementJSON: `{"selector":"#desktop-14","tag":"input","type":"text","fontSize":14,"rect":{"x":16,"y":24,"width":240,"height":40}}`,
		},
		{
			name:        "disabled mobile text below threshold",
			width:       390,
			elementJSON: `{"selector":"#disabled-14","tag":"input","type":"text","fontSize":14,"disabled":true,"rect":{"x":16,"y":24,"width":240,"height":40}}`,
		},
		{
			name:        "mobile button input below text threshold",
			width:       390,
			elementJSON: `{"selector":"#button-14","tag":"input","type":"button","fontSize":14,"rect":{"x":16,"y":24,"width":240,"height":40}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
				StepId:     tt.name,
				Viewport:   &visualpb.Viewport{Width: tt.width, Height: 844},
				LayoutJson: `{"elements":[` + tt.elementJSON + `]}`,
			}}})
			found := false
			for _, finding := range resp.GetFindings() {
				if finding.GetCode() == "visual_focus_zoom_risk" {
					found = true
				}
			}
			if found != tt.wantFinding {
				t.Fatalf("visual_focus_zoom_risk = %v, want %v; findings: %+v", found, tt.wantFinding, resp.GetFindings())
			}
		})
	}
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
		"visual_heading_not_block",
		"visual_token_fallback_used",
		"visual_adjacent_text_collision",
	} {
		if !slices.Contains(ids, want) {
			t.Fatalf("Rules() ids = %#v, missing %s", ids, want)
		}
	}
}

func TestAnalyzeReportsHeadingThatIsNotBlock(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:     "heading",
		LayoutJson: `{"elements":[{"selector":"#adopted-asset-showcase-title","role":"heading","display":"inline-block","text":"Adopted foundation reference","rect":{"x":10,"y":10,"width":200,"height":30}}]}`,
	}}})
	assertFindingCodes(t, resp, "visual_heading_not_block")
}

func TestAnalyzeKeepsBlockHeadingSilent(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:     "heading",
		LayoutJson: `{"elements":[{"selector":"h1","role":"heading","display":"block","text":"Ready","rect":{"x":10,"y":10,"width":200,"height":30}}]}`,
	}}})
	for _, finding := range resp.GetFindings() {
		if finding.GetCode() == "visual_heading_not_block" {
			t.Fatalf("block heading must be silent: %+v", finding)
		}
	}
}

func TestAnalyzeReportsTokenFallback(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:     "tokens",
		LayoutJson: `{"tokens":["#0f172a","Inter"],"elements":[{"selector":".title","color":"#ff0000","fontFamily":"Arial","display":"block","text":"Literal","rect":{"x":0,"y":0,"width":100,"height":20}}]}`,
	}}})
	assertFindingCodes(t, resp, "visual_token_fallback_used")
}

func TestAnalyzeAllowsTokenBackedVisualValues(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:     "tokens",
		LayoutJson: `{"tokens":["#0f172a","Inter"],"elements":[{"selector":".title","color":"#0f172a","fontFamily":"Inter","display":"block","text":"Tokenized","rect":{"x":0,"y":0,"width":100,"height":20}}]}`,
	}}})
	for _, finding := range resp.GetFindings() {
		if finding.GetCode() == "visual_token_fallback_used" {
			t.Fatalf("token-backed values must be silent: %+v", finding)
		}
	}
}

func TestAnalyzeReportsAdjacentTextCollision(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId: "text",
		LayoutJson: `{"elements":[
			{"nodeType":"text","parentSelector":".label","text":"First","display":"block","rect":{"x":0,"y":10,"width":50,"height":20}},
			{"nodeType":"text","parentSelector":".label","text":"Second","display":"block","rect":{"x":50,"y":10,"width":60,"height":20}}
		]}`,
	}}})
	assertFindingCodes(t, resp, "visual_adjacent_text_collision")
}

func TestAnalyzeKeepsInlineTextSilent(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId: "text",
		LayoutJson: `{"elements":[
			{"nodeType":"text","parentSelector":".label","text":"First","display":"inline","rect":{"x":0,"y":10,"width":50,"height":20}},
			{"nodeType":"text","parentSelector":".label","text":"Second","display":"inline","rect":{"x":50,"y":10,"width":60,"height":20}}
		]}`,
	}}})
	for _, finding := range resp.GetFindings() {
		if finding.GetCode() == "visual_adjacent_text_collision" {
			t.Fatalf("inline text must be silent: %+v", finding)
		}
	}
}

func TestAnalyzeReportsElementViewportOverflow(t *testing.T) {
	resp := NewAnalyzer(pixel.DefaultThresholds()).Analyze(&visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{{
		StepId:     "overflow",
		Viewport:   &visualpb.Viewport{Width: 360, Height: 640},
		LayoutJson: `{"document":{"scrollWidth":360,"scrollHeight":640},"elements":[{"selector":".wide","display":"block","rect":{"x":350,"y":20,"width":40,"height":20}}]}`,
	}}})
	assertFindingCodes(t, resp, "visual_viewport_overflow")
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

func TestChromeMetadataDoesNotTurnPageContentIntoAStatusBar(t *testing.T) {
	content := color.RGBA{R: 239, G: 68, B: 68, A: 255}
	chrome := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	step := &visualpb.VisualStepArtifact{
		ScreenshotPng: testChromePNG(t, content, chrome, chrome, chrome),
		LayoutJson:    `{"chrome":{"themeColor":"#ffffff","statusBarStyle":"black-translucent"},"safeArea":{"top":0,"right":0,"bottom":0,"left":0}}`,
	}
	if got := chromeColorFindings(step, step.ScreenshotPng); len(got) != 0 {
		t.Fatalf("ordinary page content was treated as browser chrome: %+v", got)
	}
	step.LayoutJson = `{"chrome":{"themeColor":"#ffffff","statusBarStyle":"black-translucent"},"safeArea":{"top":12,"right":0,"bottom":0,"left":0}}`
	if got := chromeColorFindings(step, step.ScreenshotPng); len(got) == 0 {
		t.Fatal("a real mismatched safe area must remain a failure")
	}
}

func TestChromeColorsAllowDifferentTopAndBottomOwners(t *testing.T) {
	top := color.RGBA{R: 15, G: 23, B: 42, A: 255}
	bottom := color.RGBA{R: 2, G: 6, B: 23, A: 255}
	step := &visualpb.VisualStepArtifact{
		ScreenshotPng: testChromePNG(t, top, bottom, top, top),
		LayoutJson:    `{"chrome":{"themeColor":"#0f172a","safeAreaColor":"#0f172a","safeAreaColors":{"bottom":"#020617"}},"safeArea":{"top":12,"bottom":10}}`,
	}
	if got := chromeColorFindings(step, step.ScreenshotPng); len(got) != 0 {
		t.Fatalf("independent top/bottom owners must pass: %+v", got)
	}
	step.LayoutJson = `{"chrome":{"themeColor":"#0f172a","safeAreaColor":"#0f172a","safeAreaColors":{"bottom":"#ff0000"}},"safeArea":{"top":12,"bottom":10}}`
	if got := chromeColorFindings(step, step.ScreenshotPng); len(got) == 0 {
		t.Fatal("incorrect bottom owner color must fail")
	}
}

func TestChromeColorSamplingScalesCSSInsetsToScreenshotPixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			shade := color.RGBA{A: 255}
			if y >= 180 && y < 190 {
				shade = color.RGBA{R: 255, A: 255}
			}
			img.SetRGBA(x, y, shade)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	step := &visualpb.VisualStepArtifact{ScreenshotPng: buf.Bytes(), LayoutJson: `{"viewport":{"width":100,"height":100},"chrome":{"safeAreaColor":"#000000"},"safeArea":{"bottom":10}}`}
	if got := chromeColorFindings(step, step.ScreenshotPng); len(got) == 0 {
		t.Fatal("must examine the whole 20-device-pixel bottom inset, not only its last 10 pixels")
	}
}

func TestMobileZeroInsetsReportSafeAreaUnexercised(t *testing.T) {
	step := &visualpb.VisualStepArtifact{LayoutJson: `{"viewport":{"width":390,"height":844},"chrome":{"themeColor":"#020617"},"safeArea":{"top":0,"bottom":0}}`}
	findings, _ := analyzeLayout(step)
	found := false
	for _, f := range findings {
		if f.Code == "visual_safe_area_not_exercised" {
			found = true
		}
	}
	if !found {
		t.Fatal("zero-inset mobile evidence must identify safe-area coverage as unexercised")
	}
	if hasError(findings) {
		t.Fatal("a zero-inset browser is valid; missing notch coverage alone is not a layout failure")
	}
}

func TestInstalledChromeRequirementRejectsEmulatedAndUnknownCapture(t *testing.T) {
	for _, capture := range []string{`{"kind":"browser_viewport"}`, `{"kind":"unknown"}`, `{"kind":"browser_geometry","standalone":true}`} {
		step := &visualpb.VisualStepArtifact{StepId: "installed", LayoutJson: `{"viewport":{"width":393,"height":852},"safeArea":{"top":59,"bottom":34},"chrome":{"themeColor":"#020617","requireNativeScreen":true},"capture":` + capture + `,"elements":[]}`}
		findings, _ := analyzeLayout(step)
		matched := false
		for _, f := range findings {
			if f.Code == "visual_native_screen_unverified" && f.Severity == severityError {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("installed app qualification accepted %s without physical screen evidence: %+v", capture, findings)
		}
	}
}
