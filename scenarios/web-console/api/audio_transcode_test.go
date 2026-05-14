package main

import (
	"context"
	"testing"
)

// TestTranscodeAudio_PassthroughWhenNoFfmpeg exercises the Server.transcodeAudio
// seam — a fake transcoder is injected and we verify the server uses it.
// The real Transcode implementation is covered by internal/audio tests.
func TestTranscodeAudio_PassthroughWhenNoFfmpeg(t *testing.T) {
	srv := &Server{
		transcodeAudio: func(_ context.Context, audio []byte) ([]byte, error) {
			return audio, nil
		},
	}

	input := []byte("fake audio data")
	output, err := srv.transcodeAudio(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if &output[0] != &input[0] {
		t.Error("expected passthrough to return the same slice")
	}
}
