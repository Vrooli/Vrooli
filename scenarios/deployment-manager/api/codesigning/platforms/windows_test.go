package platforms

import (
	"context"
	"testing"

	"deployment-manager/codesigning"
	"deployment-manager/codesigning/mocks"
)

// testableWindowsDetector wraps WindowsDetector to enable testing on any platform
type testableWindowsDetector struct {
	*WindowsDetector
	mockEnabled bool
}

func newTestableWindowsDetector(fs codesigning.FileSystem, cmd codesigning.CommandRunner, env codesigning.EnvironmentReader) *testableWindowsDetector {
	return &testableWindowsDetector{
		WindowsDetector: NewWindowsDetector(fs, cmd, env),
		mockEnabled:     true,
	}
}

func (d *testableWindowsDetector) DetectTools(ctx context.Context) []codesigning.ToolDetectionResult {
	var results []codesigning.ToolDetectionResult
	results = append(results, d.detectSigntool(ctx))
	return results
}

func (d *testableWindowsDetector) DiscoverCertificates(ctx context.Context) ([]DiscoveredCertificate, error) {
	return d.listStoreCertificates(ctx)
}

func TestWindowsDetector_DetectSigntool_Found(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Mock signtool found in PATH
	cmd.AddLookPath("signtool.exe", `C:\Program Files (x86)\Windows Kits\10\bin\x64\signtool.exe`)
	cmd.AddCommand("signtool.exe", []byte("SignTool Version 10.0.22000.0\n"), nil, nil)

	detector := newTestableWindowsDetector(fs, cmd, env)
	results := detector.DetectTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if !result.Installed {
		t.Error("expected signtool to be installed")
	}
	if result.Tool != "signtool.exe" {
		t.Errorf("expected tool signtool.exe, got %s", result.Tool)
	}
	if result.Path == "" {
		t.Error("expected path to be set")
	}
}

func TestWindowsDetector_DetectSigntool_NotFound(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Don't add signtool to LookPath - it won't be found

	detector := newTestableWindowsDetector(fs, cmd, env)
	results := detector.DetectTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Installed {
		t.Error("expected signtool to not be installed")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
	if result.Remediation == "" {
		t.Error("expected remediation message")
	}
}

func TestWindowsDetector_DetectSigntool_FoundInSDK(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Don't add to LookPath, but add SDK path to filesystem
	sdkPath := `C:\Program Files (x86)\Windows Kits\10\bin\x64\signtool.exe`
	fs.AddFile(sdkPath, []byte{})

	detector := newTestableWindowsDetector(fs, cmd, env)
	results := detector.DetectTools(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]
	if !result.Installed {
		t.Error("expected signtool to be installed")
	}
	if result.Path != sdkPath {
		t.Errorf("expected path %s, got %s", sdkPath, result.Path)
	}
}

func TestWindowsDetector_ListStoreCertificates(t *testing.T) {
	ctx := context.Background()

	fs := mocks.NewMockFileSystem()
	cmd := mocks.NewMockCommandRunner()
	env := mocks.NewMockEnvironmentReader()

	// Mock PowerShell output
	psOutput := `CERT:ABC123DEF456789|CN=Test Company, O=Test Company|CN=DigiCert Code Signing CA|2026-03-15|Test Code Signing`
	cmd.AddCommand("powershell", []byte(psOutput), nil, nil)

	detector := newTestableWindowsDetector(fs, cmd, env)
	certs, err := detector.DiscoverCertificates(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(certs) != 1 {
		t.Fatalf("expected 1 certificate, got %d", len(certs))
	}

	cert := certs[0]
	if cert.ID != "ABC123DEF456789" {
		t.Errorf("expected ID ABC123DEF456789, got %s", cert.ID)
	}
	if !cert.IsCodeSign {
		t.Error("expected IsCodeSign to be true")
	}
	if cert.Platform != codesigning.PlatformWindows {
		t.Errorf("expected platform windows, got %s", cert.Platform)
	}
}

func TestExtractSigntoolVersion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "standard version line",
			input:  "SignTool Version 10.0.22000.0\nCopyright",
			expect: "10.0.22000.0",
		},
		{
			name:   "lowercase",
			input:  "signtool version 10.0.19041.0",
			expect: "10.0.19041.0",
		},
		{
			name:   "no version",
			input:  "Some other output",
			expect: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractSigntoolVersion(tc.input)
			if got != tc.expect {
				t.Errorf("expected %q, got %q", tc.expect, got)
			}
		})
	}
}

func TestExtractCNFromSubject(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		expect  string
	}{
		{
			name:    "standard CN",
			subject: "CN=Test Company, O=Test Org, C=US",
			expect:  "Test Company",
		},
		{
			name:    "CN with spaces",
			subject: "CN=My Company Inc, O=My Company Inc",
			expect:  "My Company Inc",
		},
		{
			name:    "no CN",
			subject: "O=Test Org, C=US",
			expect:  "O=Test Org, C=US",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractCNFromSubject(tc.subject)
			if got != tc.expect {
				t.Errorf("expected %q, got %q", tc.expect, got)
			}
		})
	}
}
