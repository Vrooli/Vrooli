package bundle

import (
	"testing"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

func TestDefaultServiceCompilerCompile(t *testing.T) {
	compiler := &defaultServiceCompiler{
		platform: &defaultPlatformResolver{},
		fileOps:  &defaultFileOperations{},
	}

	t.Run("unsupported build type", func(t *testing.T) {
		svc := bundlemanifest.Service{
			ID: "test-svc",
			Build: &bundlemanifest.BuildConfig{
				Type: "unsupported",
			},
		}
		_, err := compiler.Compile(svc, "linux-amd64", "/tmp")
		if err == nil {
			t.Error("expected error for unsupported build type")
		}
	})
}

func TestRustTarget(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		goarch    string
		expected  string
		expectErr bool
	}{
		{
			name:     "linux amd64",
			goos:     "linux",
			goarch:   "amd64",
			expected: "x86_64-unknown-linux-gnu",
		},
		{
			name:     "linux arm64",
			goos:     "linux",
			goarch:   "arm64",
			expected: "aarch64-unknown-linux-gnu",
		},
		{
			name:     "darwin amd64",
			goos:     "darwin",
			goarch:   "amd64",
			expected: "x86_64-apple-darwin",
		},
		{
			name:     "darwin arm64",
			goos:     "darwin",
			goarch:   "arm64",
			expected: "aarch64-apple-darwin",
		},
		{
			name:     "windows amd64",
			goos:     "windows",
			goarch:   "amd64",
			expected: "x86_64-pc-windows-msvc",
		},
		{
			name:     "windows arm64",
			goos:     "windows",
			goarch:   "arm64",
			expected: "aarch64-pc-windows-msvc",
		},
		{
			name:      "unsupported os",
			goos:      "freebsd",
			goarch:    "amd64",
			expectErr: true,
		},
		{
			name:      "unsupported arch",
			goos:      "linux",
			goarch:    "386",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rustTarget(tt.goos, tt.goarch)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("rustTarget(%q, %q) = %q, want %q", tt.goos, tt.goarch, result, tt.expected)
			}
		})
	}
}
