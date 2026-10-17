package render

import (
	"bytes"
	"github.com/vrooli/browser-automation-studio/internal/testutil/videofixture"
	"os"
	"path/filepath"
	"testing"
)

func TestRawRecordingRequiresDecodableMatchingMedia(t *testing.T) {
	for _, test := range []struct {
		name, format string
		data         []byte
		valid        bool
	}{
		{"valid.mp4", "mp4", videofixture.MP4(), true},
		{"valid.webm", "webm", videofixture.WebM(), true},
		{"text.mp4", "mp4", []byte("not video"), false},
		{"text.webm", "webm", []byte("not video"), false},
		{"truncated.mp4", "mp4", videofixture.MP4()[:len(videofixture.MP4())/2], false},
		{"truncated.webm", "webm", videofixture.WebM()[:len(videofixture.WebM())/2], false},
		{"renamed.mp4", "mp4", videofixture.WebM(), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), test.name)
			if err := os.WriteFile(path, test.data, 0600); err != nil {
				t.Fatal(err)
			}
			media, err := ConvertRecordedMedia(t.Context(), path, test.format)
			if !test.valid {
				if err == nil {
					media.Cleanup()
					t.Fatal("delivered invalid/mislabeled recording")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer media.Cleanup()
			actual, err := os.ReadFile(media.Path)
			if err != nil || !bytes.Equal(actual, test.data) {
				t.Fatal("raw source bytes changed", err)
			}
		})
	}
}
