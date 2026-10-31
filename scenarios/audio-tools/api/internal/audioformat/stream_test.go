package audioformat_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
	afmocks "audio-tools/internal/audioformat/mocks"
)

// drain collects frames until the channel closes or the deadline trips.
func drain(t *testing.T, frames <-chan []byte) []byte {
	t.Helper()
	var out []byte
	timeout := time.After(2 * time.Second)
	for {
		select {
		case f, ok := <-frames:
			if !ok {
				return out
			}
			out = append(out, f...)
		case <-timeout:
			t.Fatal("timed out draining frames")
			return out
		}
	}
}

func TestStreamDecoderPCMFastPath(t *testing.T) {
	proc := &afmocks.FakeProcessRunner{}
	e := audioformat.New(audioformat.WithProcessRunner(proc), audioformat.WithFfmpegProbe(func() bool { return true }))

	dec, err := e.NewStreamDecoder(context.Background(), audioformat.CodecPCMS16LE)
	require.NoError(t, err)
	require.Empty(t, proc.Calls, "PCM fast-path must not start a decode process")

	go func() {
		_ = dec.Write([]byte{0x01, 0x02})
		_ = dec.Write([]byte{0x03, 0x04})
		_ = dec.CloseInput()
	}()
	require.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, drain(t, dec.Frames()))
	require.NoError(t, dec.Err())
}

func TestStreamDecoderFfmpegDecode(t *testing.T) {
	proc := &afmocks.FakeProcessRunner{} // identity transform
	e := audioformat.New(audioformat.WithProcessRunner(proc), audioformat.WithFfmpegProbe(func() bool { return true }))

	dec, err := e.NewStreamDecoder(context.Background(), audioformat.CodecWebM)
	require.NoError(t, err)
	require.Len(t, proc.Calls, 1)
	require.Equal(t, "ffmpeg", proc.Calls[0].Name)
	require.Contains(t, proc.Calls[0].Args, "-flush_packets")
	require.Contains(t, proc.Calls[0].Args, "s16le")

	go func() {
		_ = dec.Write([]byte{0xAA, 0xBB, 0xCC, 0xDD})
		_ = dec.CloseInput()
	}()
	require.Equal(t, []byte{0xAA, 0xBB, 0xCC, 0xDD}, drain(t, dec.Frames()))
	require.NoError(t, dec.Err())
}

func TestStreamDecoderPartialFrameBuffering(t *testing.T) {
	proc := &afmocks.FakeProcessRunner{}
	e := audioformat.New(audioformat.WithProcessRunner(proc), audioformat.WithFfmpegProbe(func() bool { return true }))
	dec, err := e.NewStreamDecoder(context.Background(), audioformat.CodecWebM)
	require.NoError(t, err)

	go func() {
		// Odd-length write: the decoder must hold the trailing byte until
		// the next write completes the int16 sample.
		_ = dec.Write([]byte{0xAA, 0xBB, 0xCC})
		_ = dec.Write([]byte{0xDD})
		_ = dec.CloseInput()
	}()
	got := drain(t, dec.Frames())
	require.Equal(t, []byte{0xAA, 0xBB, 0xCC, 0xDD}, got)
	require.Equal(t, 0, len(got)%audioformat.CanonicalBytesPerSample)
}

func TestStreamDecoderCtxCancelKills(t *testing.T) {
	proc := &afmocks.FakeProcessRunner{}
	e := audioformat.New(audioformat.WithProcessRunner(proc), audioformat.WithFfmpegProbe(func() bool { return true }))

	ctx, cancel := context.WithCancel(context.Background())
	dec, err := e.NewStreamDecoder(ctx, audioformat.CodecWebM)
	require.NoError(t, err)

	cancel()
	// Frames must close (no leak) and the terminal error reflects the kill.
	drain(t, dec.Frames())
	require.ErrorIs(t, dec.Err(), context.Canceled)
}

func TestNewStreamDecoderNoFfmpeg(t *testing.T) {
	proc := &afmocks.FakeProcessRunner{}
	e := audioformat.New(audioformat.WithProcessRunner(proc), audioformat.WithFfmpegProbe(func() bool { return false }))

	_, err := e.NewStreamDecoder(context.Background(), audioformat.CodecWebM)
	require.ErrorIs(t, err, audioformat.ErrFfmpegRequired)
	require.Empty(t, proc.Calls)

	// PCM still works without ffmpeg.
	dec, err := e.NewStreamDecoder(context.Background(), audioformat.CodecPCMS16LE)
	require.NoError(t, err)
	require.NoError(t, dec.Close())
}
