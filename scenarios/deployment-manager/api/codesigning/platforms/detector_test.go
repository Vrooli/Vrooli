package platforms

import (
	"context"
	"testing"

	"deployment-manager/codesigning"
	"deployment-manager/codesigning/mocks"
)

func TestMultiPlatformDetector_DetectAll(t *testing.T) {
	ctx := context.Background()

	// Create mocks
	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Setup mock responses for GPG (Linux)
	cmd.AddLookPath("gpg", "/usr/bin/gpg")
	cmd.AddCommand("gpg --version", []byte("gpg (GnuPG) 2.2.27\n"), nil, nil)
	cmd.AddCommand("gpg --list-secret-keys --keyid-format long", []byte(`
sec   rsa4096/ABC123DEF456789 2020-01-01 [SC] [expires: 2025-01-01]
      1234567890ABCDEF1234567890ABCDEF12345678
uid           [ultimate] Test User <test@example.com>
`), nil, nil)

	// Create detector with mocks
	detector := &MultiPlatformDetector{
		detectors: []PlatformDetector{
			&testableLinuxDetector{LinuxDetector: NewLinuxDetector(fs, cmd, env)},
		},
	}

	results := detector.DetectAll(ctx)

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if results[0].Platform != codesigning.PlatformLinux {
		t.Errorf("expected platform linux, got %s", results[0].Platform)
	}
}

func TestMultiPlatformDetector_DetectPlatform(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	cmd.AddLookPath("gpg", "/usr/bin/gpg")
	cmd.AddCommand("gpg --version", []byte("gpg (GnuPG) 2.2.27\n"), nil, nil)
	cmd.AddCommand("gpg --list-secret-keys --keyid-format long", []byte(""), nil, nil)

	detector := &MultiPlatformDetector{
		detectors: []PlatformDetector{
			&testableLinuxDetector{LinuxDetector: NewLinuxDetector(fs, cmd, env)},
		},
	}

	// Test existing platform
	result := detector.DetectPlatform(ctx, codesigning.PlatformLinux)
	if result == nil {
		t.Fatal("expected result for linux platform")
	}
	if result.Platform != codesigning.PlatformLinux {
		t.Errorf("expected platform linux, got %s", result.Platform)
	}

	// Test non-existent platform
	result = detector.DetectPlatform(ctx, "nonexistent")
	if result != nil {
		t.Error("expected nil for non-existent platform")
	}
}

// testableLinuxDetector wraps LinuxDetector to enable testing on any platform
type testableLinuxDetector struct {
	*LinuxDetector
}

func (d *testableLinuxDetector) Platform() string {
	return codesigning.PlatformLinux
}

func (d *testableLinuxDetector) DetectTools(ctx context.Context) []codesigning.ToolDetectionResult {
	var results []codesigning.ToolDetectionResult
	results = append(results, d.detectGPG(ctx))
	return results
}

func (d *testableLinuxDetector) DiscoverCertificates(ctx context.Context) ([]DiscoveredCertificate, error) {
	return d.listGPGKeys(ctx, "")
}
