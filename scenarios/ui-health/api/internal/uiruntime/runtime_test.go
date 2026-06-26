package uiruntime

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"ui-health/internal/evidence"
	"ui-health/internal/services/manifestvalidation"

	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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

func TestCheckUsesVisualHealthForBlankScreenshot(t *testing.T) {
	bas := &fakeBAS{res: &runResult{
		loaded:            true,
		handshakeSignaled: true,
		screenshotRef:     "/api/v1/storage/runtime.png",
		screenshotPNG:     solidPNG(t, 80, 60, color.RGBA{R: 255, G: 255, B: 255, A: 255}),
	}}
	r := newRunner("http://localhost:5173", nil, bas)
	finds := r.Check(context.Background(), Input{Scenario: "demo"})
	if len(finds) == 0 || finds[0].Code != "runtime_render_broken" {
		t.Fatalf("want runtime_render_broken first, got %v", codes(finds))
	}
	if finds[0].Severity != manifestvalidation.SeverityError {
		t.Fatalf("blank screenshot must be an error, got %s", finds[0].Severity)
	}
}

func TestCheckSurfacesVisualHealthDOMFindings(t *testing.T) {
	bas := &fakeBAS{res: &runResult{
		loaded:            true,
		handshakeSignaled: true,
		domHTML:           "<main><script>boot()</script><style>.x{}</style><div> </div></main>",
	}}
	r := newRunner("http://localhost:5173", nil, bas)
	finds := r.Check(context.Background(), Input{Scenario: "demo"})
	got := codes(finds)
	if len(got) < 2 || got[0] != "runtime_render_broken" || got[1] != "visual_dom_blank" {
		t.Fatalf("want runtime summary plus visual_dom_blank detail, got %v", got)
	}
	if finds[1].Suggestion == "" {
		t.Fatal("visual detail finding must carry analyzer remediation")
	}
}

func TestCheckSurfacesVisualHealthLayoutFindings(t *testing.T) {
	bas := &fakeBAS{res: &runResult{
		loaded:            true,
		handshakeSignaled: true,
		viewportWidth:     360,
		viewportHeight:    640,
		layoutJSON:        `{"document":{"scrollWidth":460,"scrollHeight":640},"elements":[]}`,
	}}
	r := newRunner("http://localhost:5173", nil, bas)
	got := codes(r.Check(context.Background(), Input{Scenario: "demo"}))
	if len(got) < 2 || got[0] != "runtime_render_broken" || got[1] != "visual_viewport_overflow" {
		t.Fatalf("want runtime summary plus visual_viewport_overflow detail, got %v", got)
	}
}

func TestReadTimelineExtractsVisualArtifacts(t *testing.T) {
	node := nodeArtifacts
	tl := &bastimeline.ExecutionTimeline{Entries: []*bastimeline.TimelineEntry{{
		NodeId: &node,
		Context: &basbase.EventContext{ExtractedData: map[string]*commonpb.JsonValue{
			"visual_artifacts": objectValue(map[string]*commonpb.JsonValue{
				"domHtml": stringValue("<main>Ready</main>"),
				"viewport": objectValue(map[string]*commonpb.JsonValue{
					"width":  intValue(390),
					"height": intValue(844),
				}),
				"layout": objectValue(map[string]*commonpb.JsonValue{
					"document": objectValue(map[string]*commonpb.JsonValue{
						"scrollWidth":  intValue(460),
						"scrollHeight": intValue(844),
					}),
				}),
				"network": listValue([]*commonpb.JsonValue{
					objectValue(map[string]*commonpb.JsonValue{
						"url":          stringValue("https://example.test/logo.png"),
						"method":       stringValue("GET"),
						"resourceType": stringValue("image"),
						"status":       intValue(404),
					}),
				}),
			}),
		}},
	}}}
	res := readTimeline(tl)
	if res.domHTML != "<main>Ready</main>" {
		t.Fatalf("domHTML = %q", res.domHTML)
	}
	if res.viewportWidth != 390 || res.viewportHeight != 844 {
		t.Fatalf("viewport = %dx%d, want 390x844", res.viewportWidth, res.viewportHeight)
	}
	if res.layoutJSON == "" {
		t.Fatal("layoutJSON was not extracted")
	}
	if len(res.network) != 1 || res.network[0].Status == nil || *res.network[0].Status != 404 {
		t.Fatalf("network = %+v, want one 404 image entry", res.network)
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

func solidPNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return buf.Bytes()
}

func stringValue(v string) *commonpb.JsonValue {
	return &commonpb.JsonValue{Kind: &commonpb.JsonValue_StringValue{StringValue: v}}
}

func intValue(v int64) *commonpb.JsonValue {
	return &commonpb.JsonValue{Kind: &commonpb.JsonValue_IntValue{IntValue: v}}
}

func objectValue(fields map[string]*commonpb.JsonValue) *commonpb.JsonValue {
	return &commonpb.JsonValue{Kind: &commonpb.JsonValue_ObjectValue{ObjectValue: &commonpb.JsonObject{Fields: fields}}}
}

func listValue(values []*commonpb.JsonValue) *commonpb.JsonValue {
	return &commonpb.JsonValue{Kind: &commonpb.JsonValue_ListValue{ListValue: &commonpb.JsonList{Values: values}}}
}
