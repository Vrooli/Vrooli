package preview

import (
	"path/filepath"
	"testing"
)

func TestPreviewRuntimeStoreRootWithLookupEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     string
	}{
		{
			name:     "configured root is trimmed and preferred",
			envValue: "  /tmp/rcl-preview-store  ",
			want:     "/tmp/rcl-preview-store",
		},
		{
			name: "empty root uses the governed default",
			want: filepath.Join("repo", "scenarios", "react-component-library", "tools", "preview-runtime"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := previewRuntimeStoreRootWithLookupEnv("repo", func(key string) string {
				if key != previewRuntimeStoreEnv {
					t.Fatalf("environment lookup key = %q, want %q", key, previewRuntimeStoreEnv)
				}
				return test.envValue
			})
			if got != test.want {
				t.Fatalf("previewRuntimeStoreRootWithLookupEnv() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPreviewRuntimeStoreRootWithNilLookupUsesDefault(t *testing.T) {
	want := filepath.Join("repo", "scenarios", "react-component-library", "tools", "preview-runtime")
	if got := previewRuntimeStoreRootWithLookupEnv("repo", nil); got != want {
		t.Fatalf("previewRuntimeStoreRootWithLookupEnv(nil) = %q, want %q", got, want)
	}
}
