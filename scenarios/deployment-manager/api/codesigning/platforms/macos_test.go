package platforms

import (
	"context"
	"testing"

	"deployment-manager/codesigning"
	"deployment-manager/codesigning/mocks"
)

// testableMacOSDetector wraps MacOSDetector to enable testing on any platform
type testableMacOSDetector struct {
	*MacOSDetector
}

func newTestableMacOSDetector(fs codesigning.FileSystem, cmd codesigning.CommandRunner, env codesigning.EnvironmentReader) *testableMacOSDetector {
	return &testableMacOSDetector{
		MacOSDetector: NewMacOSDetector(fs, cmd, env),
	}
}

func (d *testableMacOSDetector) DetectTools(ctx context.Context) []codesigning.ToolDetectionResult {
	var results []codesigning.ToolDetectionResult
	results = append(results, d.detectCodesign(ctx))
	results = append(results, d.detectNotarytool(ctx))
	return results
}

func (d *testableMacOSDetector) DiscoverCertificates(ctx context.Context) ([]DiscoveredCertificate, error) {
	return d.listKeychainIdentities(ctx)
}

func TestMacOSDetector_DetectCodesign_Found(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	cmd.AddLookPath("codesign", "/usr/bin/codesign")
	cmd.AddCommand("codesign --version", []byte(""), nil, nil)
	cmd.AddCommand("xcodebuild -version", []byte("Xcode 15.0\nBuild version 15A240d"), nil, nil)

	detector := newTestableMacOSDetector(fs, cmd, env)
	results := detector.DetectTools(ctx)

	// Should have codesign and notarytool results
	codesignResult := findToolResult(results, "codesign")
	if codesignResult == nil {
		t.Fatal("expected codesign result")
	}

	if !codesignResult.Installed {
		t.Error("expected codesign to be installed")
	}
	if codesignResult.Path != "/usr/bin/codesign" {
		t.Errorf("expected path /usr/bin/codesign, got %s", codesignResult.Path)
	}
}

func TestMacOSDetector_DetectNotarytool_Found(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	cmd.AddLookPath("codesign", "/usr/bin/codesign")
	cmd.AddCommand("xcrun notarytool --version", []byte("notarytool 2.0.0"), nil, nil)

	detector := newTestableMacOSDetector(fs, cmd, env)
	results := detector.DetectTools(ctx)

	notarytoolResult := findToolResult(results, "notarytool")
	if notarytoolResult == nil {
		t.Fatal("expected notarytool result")
	}

	if !notarytoolResult.Installed {
		t.Error("expected notarytool to be installed")
	}
	if notarytoolResult.Version != "notarytool 2.0.0" {
		t.Errorf("expected version 'notarytool 2.0.0', got %s", notarytoolResult.Version)
	}
}

func TestMacOSDetector_ListKeychainIdentities(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Mock security find-identity output
	securityOutput := `Policy: Code Signing
  Matching identities
  1) ABCDEF1234567890ABCDEF1234567890ABCDEF12 "Developer ID Application: Test Company (ABC123XYZ)"
  2) 1234567890ABCDEF1234567890ABCDEF12345678 "Apple Development: test@example.com (XYZ789ABC)"
     2 valid identities found
`
	cmd.AddCommand("security find-identity -v -p codesigning", []byte(securityOutput), nil, nil)

	detector := newTestableMacOSDetector(fs, cmd, env)
	certs, err := detector.DiscoverCertificates(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(certs) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(certs))
	}

	// Check first identity
	cert1 := certs[0]
	if cert1.ID != "ABCDEF1234567890ABCDEF1234567890ABCDEF12" {
		t.Errorf("expected first ID ABCDEF..., got %s", cert1.ID)
	}
	if cert1.Type != "Developer ID Application" {
		t.Errorf("expected type 'Developer ID Application', got %s", cert1.Type)
	}
	if cert1.Platform != codesigning.PlatformMacOS {
		t.Errorf("expected platform macos, got %s", cert1.Platform)
	}

	// Check second identity
	cert2 := certs[1]
	if cert2.Type != "Apple Development" {
		t.Errorf("expected type 'Apple Development', got %s", cert2.Type)
	}
}

func TestMacOSDetector_ValidateIdentity(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	securityOutput := `  1) ABC123 "Developer ID Application: Test Company (TEAMID123)"`
	cmd.AddCommand("security find-identity -v -p codesigning", []byte(securityOutput), nil, nil)

	detector := newTestableMacOSDetector(fs, cmd, env)

	tests := []struct {
		name     string
		identity string
		expect   bool
	}{
		{
			name:     "exact match",
			identity: "Developer ID Application: Test Company (TEAMID123)",
			expect:   true,
		},
		{
			name:     "team ID only",
			identity: "TEAMID123",
			expect:   true,
		},
		{
			name:     "partial match",
			identity: "ABC123",
			expect:   true,
		},
		{
			name:     "no match",
			identity: "Nonexistent Identity",
			expect:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			found, err := detector.ValidateIdentity(ctx, tc.identity)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if found != tc.expect {
				t.Errorf("expected %v for identity %q, got %v", tc.expect, tc.identity, found)
			}
		})
	}
}

func findToolResult(results []codesigning.ToolDetectionResult, tool string) *codesigning.ToolDetectionResult {
	for _, r := range results {
		if r.Tool == tool {
			return &r
		}
	}
	return nil
}
