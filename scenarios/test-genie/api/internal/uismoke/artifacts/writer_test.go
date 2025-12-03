package artifacts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"

	"test-genie/internal/uismoke/orchestrator"
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
	expectedDir := "/scenarios/test/coverage/test/ui-smoke"
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

	// With empty response, paths should be empty
	if paths.Screenshot != "" {
		t.Error("Screenshot path should be empty for no screenshot")
	}
	if paths.Console != "" {
		t.Error("Console path should be empty for no console entries")
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

	expectedPath := "/scenarios/test/coverage/test/ui-smoke/latest.json"
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

func TestRelPath(t *testing.T) {
	tests := []struct {
		baseDir string
		path    string
		want    string
	}{
		{
			baseDir: "/scenarios/test",
			path:    "/scenarios/test/coverage/test/ui-smoke/screenshot.png",
			want:    "coverage/test/ui-smoke/screenshot.png",
		},
		{
			baseDir: "/a/b/c",
			path:    "/a/b/c/d/e.txt",
			want:    "d/e.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := relPath(tt.baseDir, tt.path); got != tt.want {
				t.Errorf("relPath() = %q, want %q", got, tt.want)
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
	dir := coverageDir("/scenarios/test", "test-scenario")
	expected := "/scenarios/test/coverage/test-scenario/ui-smoke"
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
