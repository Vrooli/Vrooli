package capture

import (
	"context"
	"strconv"

	"github.com/vrooli/browser-automation-studio/services/media"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
)

// Keep measured media facts separate from requested CSS layout and DPR. A
// missing probe does not discard the raw recording or manufacture dimensions.
func measureCaptureVideos(ctx context.Context, request *capturev1.CaptureRequest, width, height int32, artifacts []*capturev1.CaptureArtifact, probe func(context.Context, string) (media.Video, error)) {
	for _, artifact := range artifacts {
		if artifact.GetType() != capturev1.CaptureType_CAPTURE_TYPE_VIDEO || artifact.GetMetadata()["unavailable"] == "true" {
			continue
		}
		if artifact.Metadata == nil {
			artifact.Metadata = map[string]string{}
		}
		metadata := artifact.Metadata
		metadata["requested_css_width"] = strconv.Itoa(int(width))
		metadata["requested_css_height"] = strconv.Itoa(int(height))
		if dimensions := request.GetDimensions(); dimensions != nil && dimensions.DeviceScaleFactor != nil {
			metadata["requested_device_scale_factor"] = strconv.FormatFloat(dimensions.GetDeviceScaleFactor(), 'g', -1, 64)
		}
		measured, err := probe(ctx, artifact.GetPath())
		if err != nil {
			metadata["measurement_status"] = "unavailable"
			metadata["measurement_reason"] = "ffprobe did not return a valid raster and duration within the probe budget"
			continue
		}
		metadata["measurement_status"] = "measured"
		metadata["measured_width"] = strconv.Itoa(int(measured.Width))
		metadata["measured_height"] = strconv.Itoa(int(measured.Height))
		metadata["measured_duration_ms"] = strconv.Itoa(int(measured.DurationMs))
	}
}
