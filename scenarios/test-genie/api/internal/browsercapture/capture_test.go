package browsercapture

import (
	"context"
	"errors"
	"testing"

	"test-genie/internal/evidence"

	capturepb "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

func TestCapturePage_MapsScreenshotAndRequestShape(t *testing.T) {
	fake := &FakeCaptureClient{Response: &capturepb.CaptureResponse{
		Artifacts: []*capturepb.CaptureArtifact{
			{Type: capturepb.CaptureType_CAPTURE_TYPE_SCREENSHOT, Path: "/srv/shot.png", SizeBytes: 42},
		},
	}}
	mc := NewMultiCapturer(fake)

	res, err := mc.CapturePage(context.Background(), PageRequest{ScenarioSlug: "demo", Path: "/backlog", Label: "Backlog"})
	if err != nil {
		t.Fatalf("CapturePage error: %v", err)
	}
	if res.ScreenshotPath != "/srv/shot.png" {
		t.Fatalf("screenshot path = %q", res.ScreenshotPath)
	}
	if !res.Evidence.Loaded || !res.Evidence.Handshake.Signaled {
		t.Fatalf("visual capture evidence should be loaded with handshake satisfied: %+v", res.Evidence)
	}
	// Request shape: desktop preset, networkidle, screenshot+console+network.
	req := fake.Requests[0]
	if req.GetUrl() != "scenario=demo,path=/backlog" {
		t.Fatalf("url = %q", req.GetUrl())
	}
	if req.GetDimensions().GetPreset() != capturepb.DimensionsPreset_DIMENSIONS_PRESET_DESKTOP {
		t.Fatalf("expected DESKTOP preset")
	}
	if !req.GetWaitFor().GetNetworkidle() {
		t.Fatalf("expected networkidle wait")
	}
	if len(req.GetCaptures()) != 3 {
		t.Fatalf("expected screenshot+console+network (3), got %d", len(req.GetCaptures()))
	}
}

func TestCapturePage_NetworkFailureCountFeedsVerdict(t *testing.T) {
	fake := &FakeCaptureClient{Response: &capturepb.CaptureResponse{
		Artifacts: []*capturepb.CaptureArtifact{
			{Type: capturepb.CaptureType_CAPTURE_TYPE_NETWORK, Metadata: map[string]string{"failure_count": "2"}},
		},
	}}
	mc := NewMultiCapturer(fake)

	res, _ := mc.CapturePage(context.Background(), PageRequest{ScenarioSlug: "demo", Path: "/"})
	v := evidence.Analyze(res.Evidence)
	if v.Passed() {
		t.Fatalf("network failures should fail the verdict")
	}
	if v.NetworkFailureCount != 2 {
		t.Fatalf("network failure count = %d, want 2", v.NetworkFailureCount)
	}
}

func TestCapturePage_VideoRequestedAndMapped(t *testing.T) {
	fake := &FakeCaptureClient{Response: &capturepb.CaptureResponse{
		Artifacts: []*capturepb.CaptureArtifact{
			{Type: capturepb.CaptureType_CAPTURE_TYPE_VIDEO, Path: "/srv/v.webm"},
		},
	}}
	mc := NewMultiCapturer(fake)

	res, _ := mc.CapturePage(context.Background(), PageRequest{ScenarioSlug: "demo", Path: "/", IncludeVideo: true})
	if res.VideoPath != "/srv/v.webm" {
		t.Fatalf("video path = %q", res.VideoPath)
	}
	hasVideo := false
	for _, c := range fake.Requests[0].GetCaptures() {
		if c == capturepb.CaptureType_CAPTURE_TYPE_VIDEO {
			hasVideo = true
		}
	}
	if !hasVideo {
		t.Fatalf("VIDEO capture type not requested")
	}
}

func TestCapturePage_ErrorYieldsNotLoaded(t *testing.T) {
	fake := &FakeCaptureClient{Err: errors.New("boom")}
	mc := NewMultiCapturer(fake)

	res, err := mc.CapturePage(context.Background(), PageRequest{ScenarioSlug: "demo", Path: "/"})
	if err == nil {
		t.Fatal("expected transport error")
	}
	if res.Evidence.Loaded {
		t.Fatalf("evidence should be not-loaded on capture error")
	}
	if evidence.Analyze(res.Evidence).Passed() {
		t.Fatalf("not-loaded evidence must fail the verdict")
	}
}
