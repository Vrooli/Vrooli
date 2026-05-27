package audioformat

import commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"

// OutputFormat is a TTS-egress container the substrate can produce. The
// set is narrower than Codec because synthesis adapters only ever emit
// these four; it mirrors the proto common.ResponseFormat enum.
type OutputFormat string

const (
	OutputMP3  OutputFormat = "mp3"
	OutputWAV  OutputFormat = "wav"
	OutputOpus OutputFormat = "opus"
	OutputFLAC OutputFormat = "flac"
)

// ContentType returns the HTTP content type for the encoded container.
func (f OutputFormat) ContentType() string {
	switch f {
	case OutputMP3:
		return "audio/mpeg"
	case OutputWAV:
		return "audio/wav"
	case OutputOpus:
		return "audio/ogg"
	case OutputFLAC:
		return "audio/flac"
	default:
		return "application/octet-stream"
	}
}

// OutputFormatFromString maps the internal wire vocabulary to an
// OutputFormat. The bool is false for empty/unrecognized input.
func OutputFormatFromString(s string) (OutputFormat, bool) {
	switch OutputFormat(s) {
	case OutputMP3, OutputWAV, OutputOpus, OutputFLAC:
		return OutputFormat(s), true
	default:
		return "", false
	}
}

// OutputFormatFromProto maps a common.ResponseFormat enum value to an
// OutputFormat. The bool is false for RESPONSE_FORMAT_UNSPECIFIED.
func OutputFormatFromProto(f commonv1.ResponseFormat) (OutputFormat, bool) {
	switch f {
	case commonv1.ResponseFormat_RESPONSE_FORMAT_MP3:
		return OutputMP3, true
	case commonv1.ResponseFormat_RESPONSE_FORMAT_WAV:
		return OutputWAV, true
	case commonv1.ResponseFormat_RESPONSE_FORMAT_OPUS:
		return OutputOpus, true
	case commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC:
		return OutputFLAC, true
	default:
		return "", false
	}
}
