package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	intaudio "audio-tools/internal/audio"
	audiomocks "audio-tools/internal/audio/mocks"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

// silenceWAV builds a mono 16-bit PCM WAV of the given length at 16 kHz.
// Silence is enough: the test is about the reported duration, which ffprobe
// derives from the produced container, not from the sample content.
func silenceWAV(seconds float64) []byte {
	const sampleRate = 16000
	data := make([]byte, int(seconds*sampleRate)*2)
	var buf bytes.Buffer
	write := func(v any) { _ = binary.Write(&buf, binary.LittleEndian, v) }
	buf.WriteString("RIFF")
	write(uint32(36 + len(data)))
	buf.WriteString("WAVEfmt ")
	write(uint32(16))
	write(uint16(1))
	write(uint16(1))
	write(uint32(sampleRate))
	write(uint32(sampleRate * 2))
	write(uint16(2))
	write(uint16(16))
	buf.WriteString("data")
	write(uint32(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

const outputProbeJSON = `{"streams":[{"codec_name":"pcm_s16le","sample_rate":"44100","channels":1}],"format":{"format_name":"wav","duration":"2.75"}}`

// probeAwareRunner answers the transform with fixed output bytes and the
// duration probe with a canned ffprobe document.
func probeAwareRunner() *audiomocks.FakeRunner {
	return &audiomocks.FakeRunner{Respond: func(name string, _ []string) ([]byte, error) {
		if name == "ffprobe" {
			return []byte(outputProbeJSON), nil
		}
		return []byte("OUT"), nil
	}}
}

func probeCalls(calls []audiomocks.Call) int {
	n := 0
	for _, c := range calls {
		if c.Name == "ffprobe" {
			n++
		}
	}
	return n
}

func TestTranscode_ReportsOutputDuration(t *testing.T) {
	fake := probeAwareRunner()
	withRunner(t, fake)
	c := newClient(t)

	res, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
		Audio: []byte("X"), OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}))
	require.NoError(t, err)
	require.InDelta(t, 2.75, res.Msg.GetDurationSeconds(), 1e-9)
	require.Equal(t, 1, probeCalls(fake.Calls), "duration must be probed from the produced output")
}

func TestTrim_ReportsOutputDuration(t *testing.T) {
	fake := probeAwareRunner()
	withRunner(t, fake)
	c := newClient(t)

	res, err := c.Trim(context.Background(), connect.NewRequest(&audiov1.TrimRequest{
		Audio: []byte("X"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV, StartSeconds: 0, EndSeconds: 2,
	}))
	require.NoError(t, err)
	require.InDelta(t, 2.75, res.Msg.GetDurationSeconds(), 1e-9)
	// One probe bounds the requested window against the input, one reports the
	// duration of the produced output.
	require.Equal(t, 2, probeCalls(fake.Calls))
}

func TestMerge_ReportsOutputDuration(t *testing.T) {
	fake := probeAwareRunner()
	withRunner(t, fake)
	c := newClient(t)

	res, err := c.Merge(context.Background(), connect.NewRequest(&audiov1.MergeRequest{
		Sources: []*audiov1.MergeSource{
			{Audio: []byte("A"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV},
			{Audio: []byte("B"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV},
		},
		OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}))
	require.NoError(t, err)
	require.InDelta(t, 2.75, res.Msg.GetDurationSeconds(), 1e-9)
	require.Equal(t, 1, probeCalls(fake.Calls))
}

// TestTransforms_FailingProbeKeepsZeroDuration: duration is best-effort, so
// a failed probe must leave the successful transform intact with duration
// unknown (0) rather than turning the response into an error.
func TestTransforms_FailingProbeKeepsZeroDuration(t *testing.T) {
	failing := func() *audiomocks.FakeRunner {
		return &audiomocks.FakeRunner{Respond: func(name string, _ []string) ([]byte, error) {
			if name == "ffprobe" {
				return nil, errors.New("probe exploded")
			}
			return []byte("OUT"), nil
		}}
	}

	t.Run("transcode", func(t *testing.T) {
		withRunner(t, failing())
		c := newClient(t)
		res, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
			Audio: []byte("X"), OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
		}))
		require.NoError(t, err)
		require.Zero(t, res.Msg.GetDurationSeconds())
		require.Equal(t, []byte("OUT"), res.Msg.GetAudio())
	})

	t.Run("trim", func(t *testing.T) {
		withRunner(t, failing())
		c := newClient(t)
		res, err := c.Trim(context.Background(), connect.NewRequest(&audiov1.TrimRequest{
			Audio: []byte("X"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV, StartSeconds: 0, EndSeconds: 2,
		}))
		require.NoError(t, err)
		require.Zero(t, res.Msg.GetDurationSeconds())
		require.Equal(t, []byte("OUT"), res.Msg.GetAudio())
	})

	t.Run("merge", func(t *testing.T) {
		withRunner(t, failing())
		c := newClient(t)
		res, err := c.Merge(context.Background(), connect.NewRequest(&audiov1.MergeRequest{
			Sources: []*audiov1.MergeSource{
				{Audio: []byte("A"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV},
				{Audio: []byte("B"), Format: commonv1.AudioFormat_AUDIO_FORMAT_WAV},
			},
			OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
		}))
		require.NoError(t, err)
		require.Zero(t, res.Msg.GetDurationSeconds())
		require.Equal(t, []byte("OUT"), res.Msg.GetAudio())
	})
}

// TestTransforms_MissingFfprobeKeepsZeroDuration: no ffprobe on the host is
// also not a transform failure.
func TestTransforms_MissingFfprobeKeepsZeroDuration(t *testing.T) {
	withRunner(t, audiomocks.NewFakeRunner([]byte("OUT"), nil))
	restore := intaudio.SetFfmpegAvailableForTest(true, false)
	t.Cleanup(restore)

	c := newClient(t)
	res, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
		Audio: []byte("X"), OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}))
	require.NoError(t, err)
	require.Zero(t, res.Msg.GetDurationSeconds())
}

// TestTranscode_RealFfmpegReportsPlausibleDuration closes the loop on real
// binaries: duration comes from probing the audio ffmpeg actually produced.
func TestTranscode_RealFfmpegReportsPlausibleDuration(t *testing.T) {
	if !intaudio.HasFfmpeg() || !intaudio.HasFfprobe() {
		t.Skip("ffmpeg/ffprobe not available")
	}
	c := newClient(t)

	res, err := c.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
		Audio: silenceWAV(0.5), OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
	}))
	require.NoError(t, err)
	require.InDelta(t, 0.5, res.Msg.GetDurationSeconds(), 0.1)
}
