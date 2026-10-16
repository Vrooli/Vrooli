package render

import (
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestRecordedVideoKeepsPresentationAndFullSourceTimeline(t *testing.T) {
	original := &ReplayMovieSpec{Decor: &exportsv1.ReplayDecor{ChromeTheme: "minimal", DeviceFrameTheme: "slim", ShowDesktop: proto.Bool(true), ShowBrowserFrame: proto.Bool(false), ShowDeviceFrame: proto.Bool(true), CursorTheme: "arrowNeon", BrowserScale: proto.Float64(.7)}, Presentation: &exportsv1.ReplayPresentation{Canvas: &exportsv1.ReplayDimensions{Width: 1080, Height: 1920}}, Frames: []*exportsv1.ReplayFrame{{DurationMs: 1000}}}
	spec, err := recordedVideoSpec(original, "http://localhost:1234/api/v1/recordings/assets/id/artifacts/videos/page.webm", 8446, 430, 880)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Decor.GetDeviceFrameTheme() != "slim" || spec.Decor.GetShowBrowserFrame() || !spec.Decor.GetShowDeviceFrame() || spec.Decor.GetBrowserScale() != .7 {
		t.Fatalf("lost styling: %v", spec.Decor)
	}
	if spec.Summary.TotalDurationMs != 8446 || spec.Frames[0].DurationMs != 8446 || spec.RecordedVideo.DurationMs != 8446 {
		t.Fatal("continuous timeline shortened to replay duration")
	}
	if spec.Presentation.Viewport.Width != 430 || spec.Presentation.Canvas.Width != 1080 {
		t.Fatal("raster/canvas confused")
	}
	if spec.Decor.CursorTheme != "disabled" || original.Decor.CursorTheme != "arrowNeon" {
		t.Fatal("must preserve original and avoid invented cursor overlay")
	}
}
func TestRecordedVideoDoesNotSilentlyIgnoreCards(t *testing.T) {
	_, err := recordedVideoSpec(&ReplayMovieSpec{IntroCard: &exportsv1.ReplayIntroCard{Enabled: true}}, "http://localhost/source", 1000, 400, 300)
	if err == nil {
		t.Fatal("unsupported cards silently ignored")
	}
}
