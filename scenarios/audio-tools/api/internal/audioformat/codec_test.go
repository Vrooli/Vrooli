package audioformat

import (
	"testing"

	"github.com/stretchr/testify/require"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

func TestCodecStringRoundTrip(t *testing.T) {
	for _, c := range []Codec{
		CodecPCMS16LE, CodecWAV, CodecMP3, CodecFLAC, CodecOGG, CodecWebM, CodecOpus, CodecAAC,
	} {
		require.Equal(t, c, CodecFromString(c.String()), "round-trip %v", c)
	}
	require.Equal(t, CodecUnknown, CodecFromString(""))
	require.Equal(t, CodecUnknown, CodecFromString("flv"))
	require.Equal(t, CodecPCMS16LE, CodecFromString("pcm"))
}

func TestCodecProtoRoundTrip(t *testing.T) {
	for _, c := range []Codec{
		CodecPCMS16LE, CodecWAV, CodecMP3, CodecFLAC, CodecOGG, CodecWebM, CodecOpus, CodecAAC,
	} {
		got, ok := FromProto(ToProto(c))
		require.True(t, ok, "proto round-trip ok %v", c)
		require.Equal(t, c, got)
	}
	_, ok := FromProto(commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED)
	require.False(t, ok)
	require.Equal(t, commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE, ToProto(CodecPCMS16LE))
}

func TestIsCanonicalPCM(t *testing.T) {
	require.True(t, CodecPCMS16LE.IsCanonicalPCM())
	require.False(t, CodecWebM.IsCanonicalPCM())
}
