package audio

import (
	"fmt"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

// contentTypeFor maps the canonical audio format short-code to its
// matching HTTP content-type. The empty string and "wav" both map to
// "audio/wav"; unknown formats fall back to "application/octet-stream".
func contentTypeFor(format string) string {
	switch format {
	case "wav", "":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "aac":
		return "audio/aac"
	case "ogg":
		return "audio/ogg"
	case "webm":
		return "audio/webm"
	case "opus":
		return "audio/opus"
	}
	return "application/octet-stream"
}

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
	}
	return ""
}

// audioFormatFromString is the inverse of audioFormatString, used by
// the ExtractMetadata response path to populate AudioMetadata.format
// from ffmpeg's reported container short-code. Unknown short-codes map
// to AUDIO_FORMAT_UNSPECIFIED.
func audioFormatFromString(s string) commonv1.AudioFormat {
	switch s {
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
	}
	return commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED
}

// normalizationMethodString maps the audio-domain NormalizationMethod
// enum to the ffmpeg-filter selector string the internal pipeline uses.
// UNSPECIFIED maps to "" so the pipeline applies its EBU_R128 default.
func normalizationMethodString(m audiov1.NormalizationMethod) string {
	switch m {
	case audiov1.NormalizationMethod_NORMALIZATION_METHOD_EBU_R128:
		return "ebu_r128"
	case audiov1.NormalizationMethod_NORMALIZATION_METHOD_RMS:
		return "rms"
	case audiov1.NormalizationMethod_NORMALIZATION_METHOD_PEAK:
		return "peak"
	}
	return ""
}

// atoiOr parses s as a decimal integer; on empty input or parse error
// it returns def.
func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil {
		return def
	}
	return v
}
