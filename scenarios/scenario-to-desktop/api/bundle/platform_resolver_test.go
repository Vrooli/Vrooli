package bundle

import (
	"testing"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

func TestExpandShorthandToHostArch(t *testing.T) {
	tests := []struct {
		name       string
		platform   string
		wantOS     string
		shouldFind bool
	}{
		{"windows", "windows", "windows", true},
		{"win", "win", "windows", true},
		{"darwin", "darwin", "darwin", true},
		{"mac", "mac", "darwin", true},
		{"linux unsupported", "linux", "", false},
		{"freebsd unsupported", "freebsd", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os, arch := expandShorthandToHostArch(tt.platform)
			if tt.shouldFind {
				if os != tt.wantOS {
					t.Errorf("expandShorthandToHostArch(%q) os = %q, want %q", tt.platform, os, tt.wantOS)
				}
				if arch == "" {
					t.Errorf("expandShorthandToHostArch(%q) arch should not be empty", tt.platform)
				}
			} else {
				if os != "" || arch != "" {
					t.Errorf("expandShorthandToHostArch(%q) = (%q, %q), want empty", tt.platform, os, arch)
				}
			}
		})
	}
}

func TestDefaultPlatformResolverResolveBinaryForPlatform(t *testing.T) {
	resolver := &defaultPlatformResolver{}

	tests := []struct {
		name            string
		service         bundlemanifest.Service
		platform        string
		expectPath      string
		shouldFindMatch bool
	}{
		{
			name: "exact match",
			service: bundlemanifest.Service{
				ID:       "test-svc",
				Binaries: map[string]bundlemanifest.Binary{"linux": {Path: "/path/linux"}},
			},
			platform:        "linux",
			expectPath:      "/path/linux",
			shouldFindMatch: true,
		},
		{
			name: "alias match (mac-amd64 to darwin-amd64)",
			service: bundlemanifest.Service{
				ID:       "test-svc",
				Binaries: map[string]bundlemanifest.Binary{"darwin-amd64": {Path: "/path/darwin-amd64"}},
			},
			platform:        "mac-amd64",
			expectPath:      "/path/darwin-amd64",
			shouldFindMatch: true,
		},
		{
			name: "alias match (win-amd64 to windows-amd64)",
			service: bundlemanifest.Service{
				ID:       "test-svc",
				Binaries: map[string]bundlemanifest.Binary{"windows-amd64": {Path: "/path/windows-amd64"}},
			},
			platform:        "win-amd64",
			expectPath:      "/path/windows-amd64",
			shouldFindMatch: true,
		},
		{
			name: "no match",
			service: bundlemanifest.Service{
				ID:       "test-svc",
				Binaries: map[string]bundlemanifest.Binary{"linux": {Path: "/path/linux"}},
			},
			platform:        "windows",
			shouldFindMatch: false,
		},
		{
			name: "empty binaries",
			service: bundlemanifest.Service{
				ID:       "test-svc",
				Binaries: map[string]bundlemanifest.Binary{},
			},
			platform:        "linux",
			shouldFindMatch: false,
		},
		{
			name: "architecture-specific match",
			service: bundlemanifest.Service{
				ID: "test-svc",
				Binaries: map[string]bundlemanifest.Binary{
					"linux-amd64": {Path: "/path/linux-amd64"},
					"linux-arm64": {Path: "/path/linux-arm64"},
				},
			},
			platform:        "linux-amd64",
			expectPath:      "/path/linux-amd64",
			shouldFindMatch: true,
		},
		{
			name: "shorthand expansion match (linux finds linux-x64)",
			service: bundlemanifest.Service{
				ID: "test-svc",
				Binaries: map[string]bundlemanifest.Binary{
					"linux-x64": {Path: "/path/linux-x64"},
				},
			},
			platform:        "linux",
			expectPath:      "/path/linux-x64",
			shouldFindMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary, found := resolver.ResolveBinaryForPlatform(tt.service, tt.platform)
			if found != tt.shouldFindMatch {
				t.Errorf("ResolveBinaryForPlatform() found = %v, want %v", found, tt.shouldFindMatch)
				return
			}
			if tt.shouldFindMatch && binary.Path != tt.expectPath {
				t.Errorf("ResolveBinaryForPlatform() path = %q, want %q", binary.Path, tt.expectPath)
			}
		})
	}
}
