package render

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/internal/testutil/videofixture"
	"github.com/vrooli/browser-automation-studio/storage"
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
			if got, want := media.ContentType, "video/"+test.format; got != want {
				t.Fatalf("content type=%q, want %q", got, want)
			}
			actual, err := os.ReadFile(media.Path)
			if err != nil || !bytes.Equal(actual, test.data) {
				t.Fatal("raw source bytes changed", err)
			}
		})
	}
}

func TestRecordedContainerValidationUsesSharedStorageSniffer(t *testing.T) {
	for _, tc := range []struct {
		name, extension, wantSniff string
		data                       []byte
	}{
		{name: "webm renamed mp4", extension: ".mp4", wantSniff: "video/webm", data: videofixture.WebM()},
		{name: "mp4 renamed webm", extension: ".webm", wantSniff: "video/mp4", data: videofixture.MP4()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "recording"+tc.extension)
			if err := os.WriteFile(path, tc.data, 0600); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			gotSniff := storage.SniffContentType(file)
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if gotSniff != tc.wantSniff {
				t.Fatalf("shared storage sniffer=%q, want %q", gotSniff, tc.wantSniff)
			}
			if err := validateRecordedMedia(t.Context(), path); err == nil || !strings.Contains(err.Error(), "recorded container does not match") {
				t.Fatalf("mismatched recording validation error=%v", err)
			}
		})
	}
}
