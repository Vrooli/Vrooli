package audio

import (
	"testing"

	"github.com/stretchr/testify/require"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

func TestPCMFormatMapping(t *testing.T) {
	format := commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE
	require.Equal(t, "pcm_s16le", audioFormatString(format))
	require.Equal(t, format, audioFormatFromString("pcm_s16le"))
}
