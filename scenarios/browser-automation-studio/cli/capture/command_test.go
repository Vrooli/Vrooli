package capture

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

func TestFlagsFromContext_UsesProductionParser(t *testing.T) {
	rc, err := cliapp.NewTestRunContextFromArgs(captureArgSchema(), []string{
		"--url", "https://example.com", "--capture", "screenshot, console-logs", "--dimensions", "mobile", "--device-scale-factor", "2", "--dry-run",
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	f, err := flagsFromContext(rc)
	if err != nil {
		t.Fatalf("flagsFromContext: %v", err)
	}
	if f.url != "https://example.com" || f.dimensions != "mobile" || !f.hasDeviceScale || f.deviceScaleFactor != 2 || !f.dryRun {
		t.Fatalf("unexpected flags: %+v", f)
	}
	if got := f.captures; len(got) != 2 || got[0] != "screenshot" || got[1] != "console-logs" {
		t.Fatalf("captures: %v", got)
	}
}

func TestBuildCaptureRequest_DimensionsAndExplicitOverride(t *testing.T) {
	f := captureFlags{url: "u", dimensions: "mobile", width: 1200, height: 800, hasWidth: true, hasHeight: true}
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

func TestBuildCaptureRequest_DimensionsPresetOnly(t *testing.T) {
	f := captureFlags{url: "u", dimensions: "mobile"}
	req, err := buildCaptureRequest(f)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if req.Dimensions == nil || req.Dimensions.Preset != capturev1.DimensionsPreset_DIMENSIONS_PRESET_MOBILE {
		t.Errorf("expected mobile preset, got %+v", req.Dimensions)
	}
}

func TestBuildCaptureRequest_WaitForVariants(t *testing.T) {
	cases := map[string]func(*capturev1.WaitFor) bool{
		"500":         func(w *capturev1.WaitFor) bool { return w.GetTimeoutMs() == 500 },
		"networkidle": func(w *capturev1.WaitFor) bool { return w.GetNetworkidle() },
		"#root":       func(w *capturev1.WaitFor) bool { return w.GetSelector() == "#root" },
	}
	for input, check := range cases {
		f := captureFlags{url: "u", waitFor: input}
		req, err := buildCaptureRequest(f)
		if err != nil {
			t.Fatalf("%q build: %v", input, err)
		}
		if req.WaitFor == nil || !check(req.WaitFor) {
			t.Errorf("wait-for %q produced %+v", input, req.WaitFor)
		}
	}
}

func TestBuildCaptureRequest_UnknownCaptureType(t *testing.T) {
	f := captureFlags{url: "u", captures: []string{"screenshot", "wat"}}
	if _, err := buildCaptureRequest(f); err == nil {
		t.Fatal("expected error for unknown capture type")
	}
}

func TestBuildCaptureRequest_DeviceScaleFactor(t *testing.T) {
	f := captureFlags{url: "u", deviceScaleFactor: 2, hasDeviceScale: true}
	req, err := buildCaptureRequest(f)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if req.Dimensions == nil || req.Dimensions.GetDeviceScaleFactor() != 2 {
		t.Fatalf("device scale factor: %+v", req.Dimensions)
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
