package audioformat

import (
	"testing"

	"github.com/stretchr/testify/require"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

func TestOutputFormatContentType(t *testing.T) {
	require.Equal(t, "audio/mpeg", OutputMP3.ContentType())
	require.Equal(t, "audio/wav", OutputWAV.ContentType())
	require.Equal(t, "audio/ogg", OutputOpus.ContentType())
	require.Equal(t, "audio/flac", OutputFLAC.ContentType())
}

func TestOutputFormatFromString(t *testing.T) {
	for _, f := range []OutputFormat{OutputMP3, OutputWAV, OutputOpus, OutputFLAC} {
		got, ok := OutputFormatFromString(string(f))
		require.True(t, ok)
		require.Equal(t, f, got)
	}
	_, ok := OutputFormatFromString("aiff")
	require.False(t, ok)
	_, ok = OutputFormatFromString("")
	require.False(t, ok)
}

func TestOutputFormatFromProto(t *testing.T) {
	got, ok := OutputFormatFromProto(commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC)
	require.True(t, ok)
	require.Equal(t, OutputFLAC, got)
	_, ok = OutputFormatFromProto(commonv1.ResponseFormat_RESPONSE_FORMAT_UNSPECIFIED)
	require.False(t, ok)
}

func TestOutputFormatProtoRoundTrip(t *testing.T) {
	for _, f := range AllOutputFormats() {
		got, ok := OutputFormatFromProto(OutputFormatToProto(f))
		require.True(t, ok, "format %q must round-trip through proto", f)
		require.Equal(t, f, got)
	}
	require.Equal(t,
		commonv1.ResponseFormat_RESPONSE_FORMAT_UNSPECIFIED,
		OutputFormatToProto(OutputFormat("aiff")))
}

func TestAllOutputFormatsIsTheFromStringVocabulary(t *testing.T) {
	// AllOutputFormats is the single vocabulary source; every member must
	// be accepted by OutputFormatFromString and vice-versa.
	for _, f := range AllOutputFormats() {
		_, ok := OutputFormatFromString(string(f))
		require.True(t, ok, "AllOutputFormats member %q must be a valid format string", f)
	}
}
