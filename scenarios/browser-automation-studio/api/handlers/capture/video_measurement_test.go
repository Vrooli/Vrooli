package capture

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/services/media"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	"google.golang.org/protobuf/proto"
)

func TestCaptureVideoReportsMeasuredRasterInsteadOfRequestedDPR(t *testing.T) {
	artifact := &capturev1.CaptureArtifact{Type: capturev1.CaptureType_CAPTURE_TYPE_VIDEO, Path: "recording.webm"}
	request := &capturev1.CaptureRequest{Dimensions: &capturev1.Dimensions{DeviceScaleFactor: proto.Float64(2)}}
	measureCaptureVideos(context.Background(), request, 1728, 972, []*capturev1.CaptureArtifact{artifact}, func(context.Context, string) (media.Video, error) {
		return media.Video{Width: 1728, Height: 972, DurationMs: 8000}, nil
	})
	require.Equal(t, "1728", artifact.Metadata["measured_width"])
	require.Equal(t, "972", artifact.Metadata["measured_height"])
	require.Equal(t, "2", artifact.Metadata["requested_device_scale_factor"])
	require.Equal(t, "8000", artifact.Metadata["measured_duration_ms"])
	require.Equal(t, "measured", artifact.Metadata["measurement_status"])
}

func TestCaptureVideoProbeFailureNeverInventsGeometry(t *testing.T) {
	artifact := &capturev1.CaptureArtifact{Type: capturev1.CaptureType_CAPTURE_TYPE_VIDEO, Path: "recording.webm"}
	measureCaptureVideos(context.Background(), &capturev1.CaptureRequest{}, 1920, 1080, []*capturev1.CaptureArtifact{artifact}, func(context.Context, string) (media.Video, error) { return media.Video{}, errors.New("missing tool") })
	require.Equal(t, "unavailable", artifact.Metadata["measurement_status"])
	require.NotContains(t, artifact.Metadata, "measured_width")
	require.NotContains(t, artifact.Metadata, "measured_height")
}
