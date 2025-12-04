package artifacts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"test-genie/internal/smoke/orchestrator"
)

// mockFileSystem records filesystem operations for testing.
type mockFileSystem struct {
	files    map[string][]byte
	dirs     map[string]bool
	writeErr error
	mkdirErr error
}

func newMockFileSystem() *mockFileSystem {
	return &mockFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *mockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.files[path] = data
	return nil
}

func (m *mockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if m.mkdirErr != nil {
		return m.mkdirErr
	}
	m.dirs[path] = true
	return nil
}

func TestWriter_WriteAll(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter(WithFileSystem(fs))

	// Create a simple PNG (minimal valid PNG header for testing)
	screenshotData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	screenshotB64 := base64.StdEncoding.EncodeToString(screenshotData)

	response := &orchestrator.BrowserResponse{
		Success: true,
		Console: []orchestrator.ConsoleEntry{
			{Level: "log", Message: "hello"},
		},
		Network: []orchestrator.NetworkEntry{
			{URL: "http://localhost/api", ErrorText: "connection refused"},
		},
		HTML:       "<html><body>test</body></html>",
		Screenshot: screenshotB64,
		Raw:        json.RawMessage(`{"success": true}`),
	}

	paths, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}

	// Check paths were set
	if paths.Screenshot == "" {
		t.Error("Screenshot path should be set")
	}
	if paths.Console == "" {
		t.Error("Console path should be set")
	}
	if paths.Network == "" {
		t.Error("Network path should be set")
	}
	if paths.HTML == "" {
		t.Error("HTML path should be set")
	}
	if paths.Raw == "" {
		t.Error("Raw path should be set")
	}

	// Check directory was created
	expectedDir := "/scenarios/test/coverage/ui-smoke"
	if !fs.dirs[expectedDir] {
		t.Errorf("expected directory %s to be created", expectedDir)
	}

	// Check files were written
	expectedFiles := []string{
		expectedDir + "/screenshot.png",
		expectedDir + "/console.json",
		expectedDir + "/network.json",
		expectedDir + "/dom.html",
		expectedDir + "/raw.json",
	}

	for _, f := range expectedFiles {
		if _, exists := fs.files[f]; !exists {
			t.Errorf("expected file %s to be written", f)
		}
	}
}

func TestWriter_WriteAll_MinimalResponse(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter(WithFileSystem(fs))

	response := &orchestrator.BrowserResponse{
		Success: true,
	}

	paths, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}

	// With empty response, most paths should be empty
	if paths.Screenshot != "" {
		t.Error("Screenshot path should be empty for no screenshot")
	}
	if paths.Console != "" {
		t.Error("Console path should be empty for no console entries")
	}
	// Network is always written (even if empty) for visibility
	if paths.Network == "" {
		t.Error("Network path should be set even for empty response")
	}
}

func TestWriter_WriteResultJSON(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter(WithFileSystem(fs))

	result := &orchestrator.Result{
		Scenario:   "test",
		Status:     orchestrator.StatusPassed,
		Message:    "UI loaded successfully",
		DurationMs: 5000,
	}

	err := w.WriteResultJSON(context.Background(), "/scenarios/test", "test", result)
	if err != nil {
		t.Fatalf("WriteResultJSON() error = %v", err)
	}

	expectedPath := "/scenarios/test/coverage/ui-smoke/latest.json"
	if _, exists := fs.files[expectedPath]; !exists {
		t.Errorf("expected file %s to be written", expectedPath)
	}

	// Verify the content
	var written orchestrator.Result
	if err := json.Unmarshal(fs.files[expectedPath], &written); err != nil {
		t.Fatalf("failed to unmarshal written result: %v", err)
	}

	if written.Status != orchestrator.StatusPassed {
		t.Errorf("Status = %v, want %v", written.Status, orchestrator.StatusPassed)
	}
	if written.Message != "UI loaded successfully" {
		t.Errorf("Message = %q, want %q", written.Message, "UI loaded successfully")
	}
}

func TestAbsPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			path: "/scenarios/test/coverage/ui-smoke/screenshot.png",
			want: "/scenarios/test/coverage/ui-smoke/screenshot.png",
		},
		{
			path: "/a/b/c/d/e.txt",
			want: "/a/b/c/d/e.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := absPath(tt.path); got != tt.want {
				t.Errorf("absPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriter_WriteAll_MkdirError(t *testing.T) {
	fs := newMockFileSystem()
	fs.mkdirErr = os.ErrPermission
	w := NewWriter(WithFileSystem(fs))

	response := &orchestrator.BrowserResponse{Success: true}

	_, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err == nil {
		t.Error("WriteAll() should return error when mkdir fails")
	}
}

func TestWriter_WriteAll_InvalidScreenshot(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter(WithFileSystem(fs))

	response := &orchestrator.BrowserResponse{
		Success:    true,
		Screenshot: "not-valid-base64!!!",
	}

	_, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err == nil {
		t.Error("WriteAll() should return error for invalid base64 screenshot")
	}
}

func TestWriter_WriteAll_WriteScreenshotError(t *testing.T) {
	fs := &selectiveErrorFileSystem{
		mockFileSystem: newMockFileSystem(),
		errorOnFile:    "screenshot.png",
	}
	w := NewWriter(WithFileSystem(fs))

	screenshotData := []byte{0x89, 0x50, 0x4E, 0x47}
	screenshotB64 := base64.StdEncoding.EncodeToString(screenshotData)

	response := &orchestrator.BrowserResponse{
		Success:    true,
		Screenshot: screenshotB64,
	}

	_, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err == nil {
		t.Error("WriteAll() should return error when writing screenshot fails")
	}
}

func TestWriter_WriteAll_WriteConsoleError(t *testing.T) {
	fs := &selectiveErrorFileSystem{
		mockFileSystem: newMockFileSystem(),
		errorOnFile:    "console.json",
	}
	w := NewWriter(WithFileSystem(fs))

	response := &orchestrator.BrowserResponse{
		Success: true,
		Console: []orchestrator.ConsoleEntry{
			{Level: "log", Message: "test"},
		},
	}

	_, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err == nil {
		t.Error("WriteAll() should return error when writing console fails")
	}
}

func TestWriter_WriteAll_WriteNetworkError(t *testing.T) {
	fs := &selectiveErrorFileSystem{
		mockFileSystem: newMockFileSystem(),
		errorOnFile:    "network.json",
	}
	w := NewWriter(WithFileSystem(fs))

	response := &orchestrator.BrowserResponse{
		Success: true,
		Network: []orchestrator.NetworkEntry{
			{URL: "http://localhost/api"},
		},
	}

	_, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err == nil {
		t.Error("WriteAll() should return error when writing network fails")
	}
}

func TestWriter_WriteAll_WriteHTMLError(t *testing.T) {
	fs := &selectiveErrorFileSystem{
		mockFileSystem: newMockFileSystem(),
		errorOnFile:    "dom.html",
	}
	w := NewWriter(WithFileSystem(fs))

	response := &orchestrator.BrowserResponse{
		Success: true,
		HTML:    "<html></html>",
	}

	_, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err == nil {
		t.Error("WriteAll() should return error when writing html fails")
	}
}

func TestWriter_WriteAll_WriteRawError(t *testing.T) {
	fs := &selectiveErrorFileSystem{
		mockFileSystem: newMockFileSystem(),
		errorOnFile:    "raw.json",
	}
	w := NewWriter(WithFileSystem(fs))

	response := &orchestrator.BrowserResponse{
		Success: true,
		Raw:     json.RawMessage(`{"key": "value"}`),
	}

	_, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err == nil {
		t.Error("WriteAll() should return error when writing raw fails")
	}
}

func TestWriter_WriteResultJSON_MkdirError(t *testing.T) {
	fs := newMockFileSystem()
	fs.mkdirErr = os.ErrPermission
	w := NewWriter(WithFileSystem(fs))

	result := &orchestrator.Result{Status: orchestrator.StatusPassed}

	err := w.WriteResultJSON(context.Background(), "/scenarios/test", "test", result)
	if err == nil {
		t.Error("WriteResultJSON() should return error when mkdir fails")
	}
}

func TestWriter_WriteResultJSON_WriteError(t *testing.T) {
	fs := &selectiveErrorFileSystem{
		mockFileSystem: newMockFileSystem(),
		errorOnFile:    "latest.json",
	}
	w := NewWriter(WithFileSystem(fs))

	result := &orchestrator.Result{Status: orchestrator.StatusPassed}

	err := w.WriteResultJSON(context.Background(), "/scenarios/test", "test", result)
	if err == nil {
		t.Error("WriteResultJSON() should return error when write fails")
	}
}

func TestWriter_WriteAll_InvalidRawJSON(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter(WithFileSystem(fs))

	response := &orchestrator.BrowserResponse{
		Success: true,
		Raw:     json.RawMessage(`not valid json`),
	}

	// Should still work even with invalid JSON (writes as-is)
	_, err := w.WriteAll(context.Background(), "/scenarios/test", "test", response)
	if err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}
}

func TestCoverageDir(t *testing.T) {
	dir := coverageDir("/scenarios/test")
	expected := "/scenarios/test/coverage/ui-smoke"
	if dir != expected {
		t.Errorf("coverageDir() = %q, want %q", dir, expected)
	}
}

func TestNewWriter_DefaultFileSystem(t *testing.T) {
	w := NewWriter()
	if w.fs == nil {
		t.Error("default filesystem should not be nil")
	}
}

// selectiveErrorFileSystem returns an error only for specific files
type selectiveErrorFileSystem struct {
	*mockFileSystem
	errorOnFile string
}

func (s *selectiveErrorFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if containsSubstring(path, s.errorOnFile) {
		return os.ErrPermission
	}
	return s.mockFileSystem.WriteFile(path, data, perm)
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestWriter_WriteReadme_Success(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter(WithFileSystem(fs))

	result := &orchestrator.Result{
		Scenario:   "test-scenario",
		Status:     orchestrator.StatusPassed,
		Message:    "UI loaded successfully",
		DurationMs: 5000,
		UIURL:      "http://localhost:3000",
		Handshake: orchestrator.HandshakeResult{
			Signaled:   true,
			DurationMs: 500,
		},
		Artifacts: orchestrator.ArtifactPaths{
			Screenshot: "coverage/ui-smoke/screenshot.png",
			Console:    "coverage/ui-smoke/console.json",
		},
	}

	readmePath, err := w.WriteReadme(context.Background(), "/scenarios/test", "test-scenario", result)
	if err != nil {
		t.Fatalf("WriteReadme() error = %v", err)
	}

	expectedPath := "/scenarios/test/coverage/ui-smoke/README.md"
	if _, exists := fs.files[expectedPath]; !exists {
		t.Errorf("expected file %s to be written", expectedPath)
	}
	if readmePath != expectedPath {
		t.Errorf("WriteReadme() returned path = %q, want %q", readmePath, expectedPath)
	}

	content := string(fs.files[expectedPath])
	if !containsSubstring(content, "test-scenario") {
		t.Error("README should contain scenario name")
	}
	if !containsSubstring(content, "✅") {
		t.Error("README should contain success emoji for passed status")
	}
}

func TestWriter_WriteReadme_FailedStatus(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter(WithFileSystem(fs))

	result := &orchestrator.Result{
		Scenario: "test-scenario",
		Status:   orchestrator.StatusFailed,
		Message:  "Iframe bridge never signaled ready",
		UIURL:    "http://localhost:3000",
		Handshake: orchestrator.HandshakeResult{
			Signaled: false,
			TimedOut: true,
		},
	}

	_, err := w.WriteReadme(context.Background(), "/scenarios/test", "test-scenario", result)
	if err != nil {
		t.Fatalf("WriteReadme() error = %v", err)
	}

	expectedPath := "/scenarios/test/coverage/ui-smoke/README.md"
	content := string(fs.files[expectedPath])
	if !containsSubstring(content, "❌") {
		t.Error("README should contain failure emoji for failed status")
	}
	if !containsSubstring(content, "Troubleshooting") {
		t.Error("README should contain Troubleshooting section for failed status")
	}
}

func TestWriter_WriteReadme_SkippedStatus(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter(WithFileSystem(fs))

	result := &orchestrator.Result{
		Scenario: "test-scenario",
		Status:   orchestrator.StatusSkipped,
		Message:  "No UI port detected",
	}

	_, err := w.WriteReadme(context.Background(), "/scenarios/test", "test-scenario", result)
	if err != nil {
		t.Fatalf("WriteReadme() error = %v", err)
	}

	expectedPath := "/scenarios/test/coverage/ui-smoke/README.md"
	content := string(fs.files[expectedPath])
	if !containsSubstring(content, "⏭️") {
		t.Error("README should contain skip emoji for skipped status")
	}
}

func TestWriter_WriteReadme_BlockedStatus(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter(WithFileSystem(fs))

	result := &orchestrator.Result{
		Scenario: "test-scenario",
		Status:   orchestrator.StatusBlocked,
		Message:  "Browserless resource is offline",
	}

	_, err := w.WriteReadme(context.Background(), "/scenarios/test", "test-scenario", result)
	if err != nil {
		t.Fatalf("WriteReadme() error = %v", err)
	}

	expectedPath := "/scenarios/test/coverage/ui-smoke/README.md"
	content := string(fs.files[expectedPath])
	if !containsSubstring(content, "🚫") {
		t.Error("README should contain blocked emoji for blocked status")
	}
	if !containsSubstring(content, "Troubleshooting") {
		t.Error("README should contain Troubleshooting section for blocked status")
	}
}

func TestWriter_WriteReadme_MkdirError(t *testing.T) {
	fs := newMockFileSystem()
	fs.mkdirErr = os.ErrPermission
	w := NewWriter(WithFileSystem(fs))

	result := &orchestrator.Result{Status: orchestrator.StatusPassed}

	_, err := w.WriteReadme(context.Background(), "/scenarios/test", "test-scenario", result)
	if err == nil {
		t.Error("WriteReadme() should return error when mkdir fails")
	}
}

func TestWriter_WriteReadme_WriteError(t *testing.T) {
	fs := &selectiveErrorFileSystem{
		mockFileSystem: newMockFileSystem(),
		errorOnFile:    "README.md",
	}
	w := NewWriter(WithFileSystem(fs))

	result := &orchestrator.Result{Status: orchestrator.StatusPassed}

	_, err := w.WriteReadme(context.Background(), "/scenarios/test", "test-scenario", result)
	if err == nil {
		t.Error("WriteReadme() should return error when write fails")
	}
}

func TestGenerateReadme_PassedWithAllArtifacts(t *testing.T) {
	result := &orchestrator.Result{
		Scenario:   "test-scenario",
		Status:     orchestrator.StatusPassed,
		Message:    "UI loaded successfully",
		DurationMs: 5000,
		UIURL:      "http://localhost:3000",
		Handshake: orchestrator.HandshakeResult{
			Signaled:   true,
			DurationMs: 500,
		},
		Artifacts: orchestrator.ArtifactPaths{
			Screenshot: "screenshot.png",
			Console:    "console.json",
			Network:    "network.json",
			HTML:       "dom.html",
			Raw:        "raw.json",
		},
	}

	content := generateReadme(result)

	// Check header
	if !containsSubstring(content, "# test-scenario UI Smoke Test Results") {
		t.Error("README should have header with scenario name")
	}

	// Check status
	if !containsSubstring(content, "**Status:** ✅ passed") {
		t.Error("README should show passed status")
	}

	// Check test info table
	if !containsSubstring(content, "| Scenario | `test-scenario` |") {
		t.Error("README should have scenario in table")
	}
	if !containsSubstring(content, "| Duration | 5000ms |") {
		t.Error("README should have duration in table")
	}
	if !containsSubstring(content, "| URL Tested | http://localhost:3000 |") {
		t.Error("README should have URL in table")
	}

	// Check handshake section
	if !containsSubstring(content, "iframe-bridge signaled ready") {
		t.Error("README should mention successful handshake")
	}
	if !containsSubstring(content, "500ms") {
		t.Error("README should show handshake duration")
	}

	// Check artifacts section
	if !containsSubstring(content, "### Screenshot") {
		t.Error("README should have Screenshot section")
	}
	if !containsSubstring(content, "### Console Logs") {
		t.Error("README should have Console Logs section")
	}
	if !containsSubstring(content, "### Network Failures") {
		t.Error("README should have Network Failures section")
	}
	if !containsSubstring(content, "### DOM Snapshot") {
		t.Error("README should have DOM Snapshot section")
	}
	if !containsSubstring(content, "### Raw Response") {
		t.Error("README should have Raw Response section")
	}

	// Check footer
	if !containsSubstring(content, "Generated by test-genie UI smoke test") {
		t.Error("README should have footer")
	}
}

func TestGenerateReadme_HandshakeTimeout(t *testing.T) {
	result := &orchestrator.Result{
		Scenario: "test-scenario",
		Status:   orchestrator.StatusFailed,
		Message:  "Iframe bridge never signaled ready",
		UIURL:    "http://localhost:3000",
		Handshake: orchestrator.HandshakeResult{
			Signaled:   false,
			TimedOut:   true,
			DurationMs: 15000,
		},
	}

	content := generateReadme(result)

	if !containsSubstring(content, "Handshake timed out") {
		t.Error("README should mention handshake timeout")
	}
	if !containsSubstring(content, "15000ms") {
		t.Error("README should show timeout duration")
	}
	if !containsSubstring(content, "@vrooli/iframe-bridge") {
		t.Error("README should mention iframe-bridge package in troubleshooting")
	}
}

func TestGenerateReadme_HandshakeError(t *testing.T) {
	result := &orchestrator.Result{
		Scenario: "test-scenario",
		Status:   orchestrator.StatusFailed,
		Message:  "Handshake failed",
		UIURL:    "http://localhost:3000",
		Handshake: orchestrator.HandshakeResult{
			Signaled: false,
			Error:    "Connection refused",
		},
	}

	content := generateReadme(result)

	if !containsSubstring(content, "Handshake error") {
		t.Error("README should mention handshake error")
	}
	if !containsSubstring(content, "Connection refused") {
		t.Error("README should show the error message")
	}
}

func TestGenerateReadme_BundleStale(t *testing.T) {
	result := &orchestrator.Result{
		Scenario: "test-scenario",
		Status:   orchestrator.StatusBlocked,
		Message:  "UI bundle is stale",
		Bundle: &orchestrator.BundleStatus{
			Fresh:  false,
			Reason: "Source files newer than dist",
		},
	}

	content := generateReadme(result)

	if !containsSubstring(content, "Bundle Status") {
		t.Error("README should have Bundle Status section")
	}
	if !containsSubstring(content, "UI bundle is stale") {
		t.Error("README should show bundle is stale")
	}
	if !containsSubstring(content, "Source files newer than dist") {
		t.Error("README should show stale reason")
	}
	if !containsSubstring(content, "vrooli scenario restart") {
		t.Error("README should suggest restart command")
	}
}

func TestGenerateReadme_BundleFresh(t *testing.T) {
	result := &orchestrator.Result{
		Scenario: "test-scenario",
		Status:   orchestrator.StatusPassed,
		Message:  "UI loaded successfully",
		UIURL:    "http://localhost:3000",
		Handshake: orchestrator.HandshakeResult{
			Signaled: true,
		},
		Bundle: &orchestrator.BundleStatus{
			Fresh: true,
		},
	}

	content := generateReadme(result)

	if !containsSubstring(content, "UI bundle is fresh") {
		t.Error("README should mention bundle is fresh")
	}
}

func TestGenerateReadme_NoArtifacts(t *testing.T) {
	result := &orchestrator.Result{
		Scenario: "test-scenario",
		Status:   orchestrator.StatusSkipped,
		Message:  "No UI port detected",
	}

	content := generateReadme(result)

	if !containsSubstring(content, "No artifacts were collected") {
		t.Error("README should mention no artifacts collected")
	}
	if !containsSubstring(content, "expected for skipped tests") {
		t.Error("README should explain why no artifacts for skipped tests")
	}
}

func TestGenerateReadme_BlockedWithMessage(t *testing.T) {
	result := &orchestrator.Result{
		Scenario: "test-scenario",
		Status:   orchestrator.StatusBlocked,
		Message:  "Browserless resource is offline. Run 'resource-browserless manage start' then rerun the smoke test.",
	}

	content := generateReadme(result)

	if !containsSubstring(content, "Blocked Test") {
		t.Error("README should have Blocked Test section")
	}
	if !containsSubstring(content, "Browserless resource is offline") {
		t.Error("README should show the blocked message")
	}
}
