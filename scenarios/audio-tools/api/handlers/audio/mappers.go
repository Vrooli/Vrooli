package audio

import (
	"strings"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

// audioFormatString maps the proto AudioFormat enum to the ffmpeg
// short-code string the internal audio pipeline consumes. UNSPECIFIED
// maps to the empty string ("use pipeline default / passthrough").
func audioFormatString(f commonv1.AudioFormat) string {
	switch f {
	case commonv1.AudioFormat_AUDIO_FORMAT_WAV:
		return "wav"
	case commonv1.AudioFormat_AUDIO_FORMAT_MP3:
		return "mp3"
	case commonv1.AudioFormat_AUDIO_FORMAT_FLAC:
		return "flac"
	case commonv1.AudioFormat_AUDIO_FORMAT_OGG:
		return "ogg"
	case commonv1.AudioFormat_AUDIO_FORMAT_WEBM:
		return "webm"
	case commonv1.AudioFormat_AUDIO_FORMAT_OPUS:
		return "opus"
	case commonv1.AudioFormat_AUDIO_FORMAT_AAC:
		return "aac"
	case commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE:
		return "pcm_s16le"
	}
	return ""
}

// audioFormatFromString is the inverse of audioFormatString, used by
// the ExtractMetadata response path to populate AudioMetadata.format
// from ffmpeg's reported container short-code. ffprobe often reports a
// comma-separated alias list (e.g. "matroska,webm" for WebM), so each
// token is tried in turn. Unknown short-codes map to
// AUDIO_FORMAT_UNSPECIFIED.
func audioFormatFromString(s string) commonv1.AudioFormat {
	for _, token := range strings.Split(s, ",") {
		switch strings.TrimSpace(token) {
		case "wav":
			return commonv1.AudioFormat_AUDIO_FORMAT_WAV
		case "mp3":
			return commonv1.AudioFormat_AUDIO_FORMAT_MP3
		case "flac":
			return commonv1.AudioFormat_AUDIO_FORMAT_FLAC
		case "ogg":
			return commonv1.AudioFormat_AUDIO_FORMAT_OGG
		case "webm":
			return commonv1.AudioFormat_AUDIO_FORMAT_WEBM
		case "opus":
			return commonv1.AudioFormat_AUDIO_FORMAT_OPUS
		case "aac":
			return commonv1.AudioFormat_AUDIO_FORMAT_AAC
		case "pcm_s16le":
			return commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE
		}
	}
	return commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED
}

// normalizationMethodString maps the audio-domain NormalizationMethod
// enum to the ffmpeg-filter selector string the internal pipeline uses.
// UNSPECIFIED maps to "" so the pipeline applies its EBU_R128 default.
var normalizationMethods = map[audiov1.NormalizationMethod]string{
	audiov1.NormalizationMethod_NORMALIZATION_METHOD_EBU_R128: "ebu_r128",
	audiov1.NormalizationMethod_NORMALIZATION_METHOD_RMS:      "rms",
	audiov1.NormalizationMethod_NORMALIZATION_METHOD_PEAK:     "peak",
}

func normalizationMethodString(m audiov1.NormalizationMethod) string { return normalizationMethods[m] }
