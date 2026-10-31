package capture

import (
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	"testing"
)

func TestVideoCaptureMIMEIsIndependentOfHostExtensionDatabase(t *testing.T) {
	if got := captureContentType(capturev1.CaptureType_CAPTURE_TYPE_VIDEO, "take.webm"); got != "video/webm" {
		t.Fatalf("video capture MIME = %q", got)
	}
	if got := captureContentType(capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, "frame.png"); got != "image/png" {
		t.Fatalf("screenshot MIME = %q", got)
	}
}
