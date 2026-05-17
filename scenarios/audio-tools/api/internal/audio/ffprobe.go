package audio

import (
	"context"
	"fmt"
	"os"
)

// runFfprobeJSON writes audio to a temp file and runs ffprobe against
// it, returning the JSON output. ffprobe needs seekable input so we
// can't simply pipe stdin.
func runFfprobeJSON(ctx context.Context, audio []byte) ([]byte, error) {
	f, err := os.CreateTemp("", "audio-tools-probe-*.bin")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(audio); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return DefaultRunner.Run(ctx, "ffprobe", nil,
		"-v", "error",
		"-show_streams", "-show_format",
		"-of", "json",
		f.Name(),
	)
}

// tempInputs holds temporary file paths the caller must clean up.
type tempInputs struct {
	paths []string
}

func (t *tempInputs) cleanup() {
	for _, p := range t.paths {
		_ = os.Remove(p)
	}
}

func writeTempInputs(sources [][]byte) (*tempInputs, error) {
	out := &tempInputs{}
	for i, src := range sources {
		f, err := os.CreateTemp("", fmt.Sprintf("audio-tools-merge-%d-*.bin", i))
		if err != nil {
			out.cleanup()
			return nil, err
		}
		if _, err := f.Write(src); err != nil {
			f.Close()
			out.cleanup()
			return nil, err
		}
		if err := f.Close(); err != nil {
			out.cleanup()
			return nil, err
		}
		out.paths = append(out.paths, f.Name())
	}
	return out, nil
}
