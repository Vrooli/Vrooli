package audio

import (
	"context"
	"fmt"
	"os"
)

// createTempFile is the temp-file seam. Production uses os.CreateTemp in
// the OS temp dir; tests override it to force creation/write failures and
// assert that partially created files are removed.
var createTempFile = func(pattern string) (*os.File, error) {
	return os.CreateTemp("", pattern)
}

// runFfprobeJSON writes audio to a temp file and runs ffprobe against
// it, returning the JSON output. ffprobe needs seekable input so we
// can't simply pipe stdin; because the payload is passed as a file and
// stdin is left nil, no interactive stdin handling is needed (ffprobe has
// no -nostdin option; passing one is a hard error).
//
// -hide_banner keeps CI/headless logs banner-free.
// -select_streams a:0 keeps the result audio-only: containers that also
// carry video would otherwise put a video stream first and the metadata
// parser would report its codec/sample-rate instead of the audio stream's.
func runFfprobeJSON(ctx context.Context, audio []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := createTempFile("audio-tools-probe-*.bin")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(audio); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	out, err := DefaultRunner.Run(ctx, "ffprobe", nil,
		"-hide_banner",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_streams", "-show_format",
		"-of", "json",
		f.Name(),
	)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("%w: %w", ctxErr, err)
		}
		return nil, err
	}
	return out, nil
}

// tempInputs holds temporary file paths the caller must clean up.
type tempInputs struct {
	paths []string
}

func (t *tempInputs) cleanup() {
	for _, p := range t.paths {
		_ = os.Remove(p)
	}
	t.paths = nil
}

// writeTempInputs materializes each source into its own temp file for
// ffmpeg's seekable-input operations (merge). Paths are recorded
// immediately after creation so a failure while writing or closing a
// file — including the file being written at that moment — still gets
// removed by cleanup. On success the caller owns cleanup().
func writeTempInputs(sources [][]byte) (*tempInputs, error) {
	out := &tempInputs{}
	for i, src := range sources {
		f, err := createTempFile(fmt.Sprintf("audio-tools-merge-%d-*.bin", i))
		if err != nil {
			out.cleanup()
			return nil, err
		}
		out.paths = append(out.paths, f.Name())
		if _, err := f.Write(src); err != nil {
			f.Close()
			out.cleanup()
			return nil, err
		}
		if err := f.Close(); err != nil {
			out.cleanup()
			return nil, err
		}
	}
	return out, nil
}
