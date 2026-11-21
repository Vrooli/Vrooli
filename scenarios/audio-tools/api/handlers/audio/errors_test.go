package audio

import (
	"errors"
	"fmt"
	"testing"

	intaudio "audio-tools/internal/audio"

	"connectrpc.com/connect"
)

// TestMapAudioErr locks in the honest-error contract: precondition and
// caller-input failures must surface actionable Connect codes instead of
// flattening to Internal (the bug the Diagnostics page exposed).
func TestMapAudioErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{
			name: "ffmpeg missing is a precondition",
			err:  intaudio.ErrFFmpegMissing,
			want: connect.CodeFailedPrecondition,
		},
		{
			name: "wrapped ffmpeg missing still maps to precondition",
			err:  fmt.Errorf("transcode: %w", intaudio.ErrFFmpegMissing),
			want: connect.CodeFailedPrecondition,
		},
		{
			name: "ffmpeg exec rejection is invalid argument",
			err:  fmt.Errorf("%w: %w", intaudio.ErrFfmpegExec, errors.New("ffmpeg: exit 1: bad input")),
			want: connect.CodeInvalidArgument,
		},
		{
			name: "unknown error stays internal",
			err:  errors.New("genuinely unexpected"),
			want: connect.CodeInternal,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := connect.CodeOf(mapAudioErr(tc.err))
			if got != tc.want {
				t.Fatalf("mapAudioErr(%v) code = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
