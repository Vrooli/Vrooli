package capture

import (
	"testing"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

func TestParseCaptureFlags_BasicURL(t *testing.T) {
	f, err := parseCaptureFlags([]string{"--url", "https://example.com"})
	if err != nil {
		t.Fatalf("parseCaptureFlags: %v", err)
	}
	if f.url != "https://example.com" {
		t.Fatalf("url=%q", f.url)
	}
}

func TestParseCaptureFlags_CSVCaptures(t *testing.T) {
	f, err := parseCaptureFlags([]string{"--url", "https://x", "--capture", "screenshot, console-logs ,network"})
	if err != nil {
		t.Fatalf("parseCaptureFlags: %v", err)
	}
	want := []string{"screenshot", "console-logs", "network"}
	if len(f.captures) != len(want) {
		t.Fatalf("captures=%v", f.captures)
	}
	for i, w := range want {
		if f.captures[i] != w {
			t.Errorf("captures[%d]=%q want %q", i, f.captures[i], w)
		}
	}
}

func TestParseCaptureFlags_DimensionsAndExplicitOverride(t *testing.T) {
	f, err := parseCaptureFlags([]string{"--url", "u", "--dimensions", "mobile", "--width", "1200", "--height", "800"})
	if err != nil {
		t.Fatalf("parseCaptureFlags: %v", err)
	}
	req, err := buildCaptureRequest(f)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if req.Dimensions == nil {
		t.Fatal("dimensions nil")
	}
	if req.Dimensions.Width == nil || *req.Dimensions.Width != 1200 {
		t.Errorf("width: %+v", req.Dimensions.Width)
	}
	if req.Dimensions.Height == nil || *req.Dimensions.Height != 800 {
		t.Errorf("height: %+v", req.Dimensions.Height)
	}
	// Explicit wins → preset must remain UNSPECIFIED.
	if req.Dimensions.Preset != capturev1.DimensionsPreset_DIMENSIONS_PRESET_UNSPECIFIED {
		t.Errorf("preset should be unspecified when width/height given, got %v", req.Dimensions.Preset)
	}
}

func TestParseCaptureFlags_DimensionsPresetOnly(t *testing.T) {
	f, err := parseCaptureFlags([]string{"--url", "u", "--dimensions", "mobile"})
	if err != nil {
		t.Fatalf("parseCaptureFlags: %v", err)
	}
	req, err := buildCaptureRequest(f)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if req.Dimensions == nil || req.Dimensions.Preset != capturev1.DimensionsPreset_DIMENSIONS_PRESET_MOBILE {
		t.Errorf("expected mobile preset, got %+v", req.Dimensions)
	}
}

func TestParseCaptureFlags_WaitForVariants(t *testing.T) {
	cases := map[string]func(*capturev1.WaitFor) bool{
		"500":         func(w *capturev1.WaitFor) bool { return w.GetTimeoutMs() == 500 },
		"networkidle": func(w *capturev1.WaitFor) bool { return w.GetNetworkidle() },
		"#root":       func(w *capturev1.WaitFor) bool { return w.GetSelector() == "#root" },
	}
	for input, check := range cases {
		f, err := parseCaptureFlags([]string{"--url", "u", "--wait-for", input})
		if err != nil {
			t.Fatalf("%q parse: %v", input, err)
		}
		req, err := buildCaptureRequest(f)
		if err != nil {
			t.Fatalf("%q build: %v", input, err)
		}
		if req.WaitFor == nil || !check(req.WaitFor) {
			t.Errorf("wait-for %q produced %+v", input, req.WaitFor)
		}
	}
}

func TestParseCaptureFlags_UnknownCaptureType(t *testing.T) {
	f, err := parseCaptureFlags([]string{"--url", "u", "--capture", "screenshot,wat"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := buildCaptureRequest(f); err == nil {
		t.Fatal("expected error for unknown capture type")
	}
}

func TestParseCaptureFlags_BoolFlagsAndUnknown(t *testing.T) {
	f, err := parseCaptureFlags([]string{"--url", "u", "--json", "--dry-run"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !f.json || !f.dryRun {
		t.Fatalf("flags: %+v", f)
	}
	if _, err := parseCaptureFlags([]string{"--bogus"}); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestProtoToJSON_Shape(t *testing.T) {
	w := int32(390)
	h := int32(844)
	resp := &capturev1.CaptureResponse{
		ExecutionId: "exec-1",
		OutDir:      "/tmp/x",
		DurationMs:  1234,
		Artifacts: []*capturev1.CaptureArtifact{{
			Type:      capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT,
			Path:      "/tmp/x/shot.png",
			SizeBytes: 4096,
			Metadata:  map[string]string{"width": "390", "height": "844"},
		}},
	}
	_ = w
	_ = h
	m := protoToJSON(resp)
	if m["execution_id"] != "exec-1" {
		t.Errorf("execution_id wrong: %v", m["execution_id"])
	}
	arts, ok := m["artifacts"].([]map[string]interface{})
	if !ok || len(arts) != 1 || arts[0]["path"] != "/tmp/x/shot.png" {
		t.Errorf("artifacts shape wrong: %+v", m["artifacts"])
	}
}
