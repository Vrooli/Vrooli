package providers

import (
	"context"
	"testing"

	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/updates"
)

// mockHashCalculator is a test double for HashCalculator.
type mockHashCalculator struct {
	hash string
	err  error
}

func (m *mockHashCalculator) CalculateSHA512(filePath string) (string, error) {
	return m.hash, m.err
}

// mockFileStatProvider is a test double for FileStatProvider.
type mockFileStatProvider struct {
	info updates.FileInfo
	err  error
}

func (m *mockFileStatProvider) Stat(path string) (updates.FileInfo, error) {
	return m.info, m.err
}

// mockManifestWriter is a test double for ManifestWriter.
type mockManifestWriter struct {
	written map[string][]byte
	err     error
}

func newMockManifestWriter() *mockManifestWriter {
	return &mockManifestWriter{written: make(map[string][]byte)}
}

func (m *mockManifestWriter) WriteFile(path string, content []byte) error {
	if m.err != nil {
		return m.err
	}
	m.written[path] = content
	return nil
}

func TestGenericProvider_Name(t *testing.T) {
	p := NewGenericProvider(&generation.UpdateConfig{})
	if p.Name() != "generic" {
		t.Errorf("expected name 'generic', got '%s'", p.Name())
	}
}

func TestGenericProvider_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *generation.UpdateConfig
		wantErr bool
	}{
		{
			name: "valid config with URL",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{
					URL: "https://updates.example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "nil generic config",
			config: &generation.UpdateConfig{
				Generic: nil,
			},
			wantErr: true,
		},
		{
			name: "empty URL",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{
					URL: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewGenericProvider(tt.config)
			err := p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenericProvider_GetPublishConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *generation.UpdateConfig
		channel string
		wantURL string
		wantNil bool
	}{
		{
			name: "basic URL with default channel path",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{
					URL: "https://updates.example.com/myapp",
				},
			},
			channel: "stable",
			wantURL: "https://updates.example.com/myapp/stable",
		},
		{
			name: "custom channel path",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{
					URL:         "https://updates.example.com/myapp",
					ChannelPath: "/releases/{channel}/v1",
				},
			},
			channel: "beta",
			wantURL: "https://updates.example.com/myapp/releases/beta/v1",
		},
		{
			name: "nil generic config returns nil",
			config: &generation.UpdateConfig{
				Generic: nil,
			},
			channel: "stable",
			wantNil: true,
		},
		{
			name: "empty URL returns nil",
			config: &generation.UpdateConfig{
				Generic: &generation.GenericUpdateConfig{
					URL: "",
				},
			},
			channel: "stable",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewGenericProvider(tt.config)
			cfg, err := p.GetPublishConfig(tt.channel)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if cfg != nil {
					t.Errorf("expected nil config, got %v", cfg)
				}
				return
			}
			if cfg == nil {
				t.Fatal("expected non-nil config")
			}
			url, ok := cfg["url"].(string)
			if !ok {
				t.Fatal("expected url to be a string")
			}
			if url != tt.wantURL {
				t.Errorf("url = %s, want %s", url, tt.wantURL)
			}
		})
	}
}

func TestGenericProvider_GenerateManifest(t *testing.T) {
	mockHash := &mockHashCalculator{hash: "testhash123=="}
	mockStat := &mockFileStatProvider{info: updates.FileInfo{Size: 1024, Name: "app.exe"}}
	mockWriter := newMockManifestWriter()

	config := &generation.UpdateConfig{
		Channel: "stable",
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	p := NewGenericProvider(config,
		WithHashCalculator(mockHash),
		WithFileStatProvider(mockStat),
		WithManifestWriter(mockWriter),
	)

	req := &updates.ManifestRequest{
		Version: "1.2.3",
		Channel: "stable",
		Artifacts: map[string]string{
			"win": "/path/to/app.exe",
		},
		BaseURL:   "https://updates.example.com/myapp/stable",
		OutputDir: "/output",
	}

	result, err := p.GenerateManifest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check manifest was generated for Windows
	if _, ok := result.Manifests["latest.yml"]; !ok {
		t.Error("expected latest.yml manifest")
	}

	// Check file was written
	if len(mockWriter.written) != 1 {
		t.Errorf("expected 1 file written, got %d", len(mockWriter.written))
	}

	// Check manifest path was recorded
	if result.ManifestPaths["latest.yml"] != "/output/latest.yml" {
		t.Errorf("unexpected manifest path: %s", result.ManifestPaths["latest.yml"])
	}
}

func TestGenericProvider_GenerateManifest_MultiPlatform(t *testing.T) {
	mockHash := &mockHashCalculator{hash: "testhash123=="}
	mockStat := &mockFileStatProvider{info: updates.FileInfo{Size: 1024, Name: "app"}}
	mockWriter := newMockManifestWriter()

	config := &generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	p := NewGenericProvider(config,
		WithHashCalculator(mockHash),
		WithFileStatProvider(mockStat),
		WithManifestWriter(mockWriter),
	)

	req := &updates.ManifestRequest{
		Version: "1.0.0",
		Channel: "stable",
		Artifacts: map[string]string{
			"win":   "/path/to/app.exe",
			"mac":   "/path/to/app.dmg",
			"linux": "/path/to/app.AppImage",
		},
		BaseURL:   "https://updates.example.com/myapp/stable",
		OutputDir: "/output",
	}

	result, err := p.GenerateManifest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check all platform manifests were generated
	expectedFiles := []string{"latest.yml", "latest-mac.yml", "latest-linux.yml"}
	for _, filename := range expectedFiles {
		if _, ok := result.Manifests[filename]; !ok {
			t.Errorf("expected %s manifest", filename)
		}
	}

	if len(mockWriter.written) != 3 {
		t.Errorf("expected 3 files written, got %d", len(mockWriter.written))
	}
}

func TestGenericProvider_GenerateManifest_EmptyArtifacts(t *testing.T) {
	config := &generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com",
		},
	}

	p := NewGenericProvider(config)

	req := &updates.ManifestRequest{
		Version:   "1.0.0",
		Artifacts: map[string]string{},
	}

	result, err := p.GenerateManifest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Manifests) != 0 {
		t.Errorf("expected no manifests for empty artifacts, got %d", len(result.Manifests))
	}

	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning for empty artifacts, got %d", len(result.Warnings))
	}

	if result.Warnings[0].Code != "EMPTY_ARTIFACTS" {
		t.Errorf("expected EMPTY_ARTIFACTS warning, got %s", result.Warnings[0].Code)
	}
}

func TestGenericProvider_RequiresManifestUpload(t *testing.T) {
	p := NewGenericProvider(&generation.UpdateConfig{})
	if !p.RequiresManifestUpload() {
		t.Error("generic provider should require manifest upload")
	}
}

// =============================================================================
// Edge case tests
// =============================================================================

func TestGenericProvider_GenerateManifest_ContextCancellation(t *testing.T) {
	mockHash := &mockHashCalculator{hash: "testhash123=="}
	mockStat := &mockFileStatProvider{info: updates.FileInfo{Size: 1024, Name: "app.exe"}}
	mockWriter := newMockManifestWriter()

	config := &generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	p := NewGenericProvider(config,
		WithHashCalculator(mockHash),
		WithFileStatProvider(mockStat),
		WithManifestWriter(mockWriter),
	)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &updates.ManifestRequest{
		Version: "1.0.0",
		Channel: "stable",
		Artifacts: map[string]string{
			"win": "/path/to/app.exe",
		},
		OutputDir: "/output",
	}

	_, err := p.GenerateManifest(ctx, req)
	if err == nil {
		t.Error("expected error for cancelled context")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestGenericProvider_GenerateManifest_FileStatError(t *testing.T) {
	statErr := &testError{msg: "file not found"}
	mockHash := &mockHashCalculator{hash: "testhash123=="}
	mockStat := &mockFileStatProvider{err: statErr}
	mockWriter := newMockManifestWriter()

	config := &generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	p := NewGenericProvider(config,
		WithHashCalculator(mockHash),
		WithFileStatProvider(mockStat),
		WithManifestWriter(mockWriter),
	)

	req := &updates.ManifestRequest{
		Version: "1.0.0",
		Channel: "stable",
		Artifacts: map[string]string{
			"win": "/path/to/app.exe",
		},
		OutputDir: "/output",
	}

	result, err := p.GenerateManifest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have a warning for the failed platform
	if len(result.Warnings) == 0 {
		t.Error("expected warning for file stat error")
	}

	hasWarning := false
	for _, w := range result.Warnings {
		if w.Code == "MANIFEST_GENERATION_FAILED" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected MANIFEST_GENERATION_FAILED warning")
	}

	// No manifest should be generated for the failed platform
	if len(result.Manifests) != 0 {
		t.Errorf("expected no manifests on stat error, got %d", len(result.Manifests))
	}
}

func TestGenericProvider_GenerateManifest_HashCalculationError(t *testing.T) {
	hashErr := &testError{msg: "hash calculation failed"}
	mockHash := &mockHashCalculator{err: hashErr}
	mockStat := &mockFileStatProvider{info: updates.FileInfo{Size: 1024, Name: "app.exe"}}
	mockWriter := newMockManifestWriter()

	config := &generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	p := NewGenericProvider(config,
		WithHashCalculator(mockHash),
		WithFileStatProvider(mockStat),
		WithManifestWriter(mockWriter),
	)

	req := &updates.ManifestRequest{
		Version: "1.0.0",
		Channel: "stable",
		Artifacts: map[string]string{
			"win": "/path/to/app.exe",
		},
		OutputDir: "/output",
	}

	result, err := p.GenerateManifest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have a warning for the failed platform
	hasWarning := false
	for _, w := range result.Warnings {
		if w.Code == "MANIFEST_GENERATION_FAILED" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected MANIFEST_GENERATION_FAILED warning for hash error")
	}

	// No manifest should be generated
	if len(result.Manifests) != 0 {
		t.Errorf("expected no manifests on hash error, got %d", len(result.Manifests))
	}
}

func TestGenericProvider_GenerateManifest_WriteError(t *testing.T) {
	writeErr := &testError{msg: "disk full"}
	mockHash := &mockHashCalculator{hash: "testhash123=="}
	mockStat := &mockFileStatProvider{info: updates.FileInfo{Size: 1024, Name: "app.exe"}}
	mockWriter := &mockManifestWriter{written: make(map[string][]byte), err: writeErr}

	config := &generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	p := NewGenericProvider(config,
		WithHashCalculator(mockHash),
		WithFileStatProvider(mockStat),
		WithManifestWriter(mockWriter),
	)

	req := &updates.ManifestRequest{
		Version: "1.0.0",
		Channel: "stable",
		Artifacts: map[string]string{
			"win": "/path/to/app.exe",
		},
		OutputDir: "/output",
	}

	result, err := p.GenerateManifest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have a warning for the write failure
	hasWarning := false
	for _, w := range result.Warnings {
		if w.Code == "MANIFEST_WRITE_FAILED" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected MANIFEST_WRITE_FAILED warning")
	}
}

func TestGenericProvider_GenerateManifest_MultiPlatformPartialFailure(t *testing.T) {
	// Create a mock stat provider that fails for specific platforms
	mockStat := &platformSpecificMockStatProvider{
		results: map[string]statResult{
			"/path/to/app.exe":      {info: updates.FileInfo{Size: 1024, Name: "app.exe"}},
			"/path/to/app.dmg":      {err: &testError{msg: "file not found"}},
			"/path/to/app.AppImage": {info: updates.FileInfo{Size: 2048, Name: "app.AppImage"}},
		},
	}
	mockHash := &mockHashCalculator{hash: "testhash123=="}
	mockWriter := newMockManifestWriter()

	config := &generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	p := NewGenericProvider(config,
		WithHashCalculator(mockHash),
		WithFileStatProvider(mockStat),
		WithManifestWriter(mockWriter),
	)

	req := &updates.ManifestRequest{
		Version: "1.0.0",
		Channel: "stable",
		Artifacts: map[string]string{
			"win":   "/path/to/app.exe",
			"mac":   "/path/to/app.dmg",
			"linux": "/path/to/app.AppImage",
		},
		BaseURL:   "https://updates.example.com/myapp/stable",
		OutputDir: "/output",
	}

	result, err := p.GenerateManifest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have generated manifests for win and linux, but not mac
	if len(result.Manifests) != 2 {
		t.Errorf("expected 2 manifests (win, linux), got %d", len(result.Manifests))
	}

	// Check that win and linux manifests exist
	if _, ok := result.Manifests["latest.yml"]; !ok {
		t.Error("expected latest.yml manifest for windows")
	}
	if _, ok := result.Manifests["latest-linux.yml"]; !ok {
		t.Error("expected latest-linux.yml manifest for linux")
	}

	// mac should not have a manifest
	if _, ok := result.Manifests["latest-mac.yml"]; ok {
		t.Error("did not expect latest-mac.yml manifest due to stat error")
	}

	// Should have a warning for mac
	hasWarning := false
	for _, w := range result.Warnings {
		if w.Code == "MANIFEST_GENERATION_FAILED" && w.Field == "mac" {
			hasWarning = true
			break
		}
	}
	if !hasWarning {
		t.Error("expected MANIFEST_GENERATION_FAILED warning for mac")
	}
}

func TestGenericProvider_GenerateManifest_NilRequest(t *testing.T) {
	p := NewGenericProvider(&generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com",
		},
	})

	_, err := p.GenerateManifest(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil request")
	}
}

func TestGenericProvider_GenerateManifest_NoOutputDir(t *testing.T) {
	mockHash := &mockHashCalculator{hash: "testhash123=="}
	mockStat := &mockFileStatProvider{info: updates.FileInfo{Size: 1024, Name: "app.exe"}}
	mockWriter := newMockManifestWriter()

	config := &generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	p := NewGenericProvider(config,
		WithHashCalculator(mockHash),
		WithFileStatProvider(mockStat),
		WithManifestWriter(mockWriter),
	)

	req := &updates.ManifestRequest{
		Version: "1.0.0",
		Channel: "stable",
		Artifacts: map[string]string{
			"win": "/path/to/app.exe",
		},
		// No OutputDir - should generate in-memory only
	}

	result, err := p.GenerateManifest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Manifest should be in memory
	if _, ok := result.Manifests["latest.yml"]; !ok {
		t.Error("expected latest.yml manifest in memory")
	}

	// But no file should be written
	if len(mockWriter.written) != 0 {
		t.Errorf("expected no files written without OutputDir, got %d", len(mockWriter.written))
	}

	// ManifestPaths should be empty
	if len(result.ManifestPaths) != 0 {
		t.Errorf("expected no manifest paths without OutputDir, got %d", len(result.ManifestPaths))
	}
}

func TestGenericProvider_GenerateManifest_EmptyFilename(t *testing.T) {
	mockHash := &mockHashCalculator{hash: "testhash123=="}
	// FileInfo with empty name - should use filepath.Base of artifact path
	mockStat := &mockFileStatProvider{info: updates.FileInfo{Size: 1024, Name: ""}}
	mockWriter := newMockManifestWriter()

	config := &generation.UpdateConfig{
		Generic: &generation.GenericUpdateConfig{
			URL: "https://updates.example.com/myapp",
		},
	}

	p := NewGenericProvider(config,
		WithHashCalculator(mockHash),
		WithFileStatProvider(mockStat),
		WithManifestWriter(mockWriter),
	)

	req := &updates.ManifestRequest{
		Version: "1.0.0",
		Channel: "stable",
		Artifacts: map[string]string{
			"win": "/path/to/myapp-1.0.0.exe",
		},
		BaseURL:   "https://updates.example.com/myapp/stable",
		OutputDir: "/output",
	}

	result, err := p.GenerateManifest(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The manifest should use the basename from the artifact path
	manifestContent := string(result.Manifests["latest.yml"])
	if manifestContent == "" {
		t.Fatal("expected manifest content")
	}

	// Should contain the filename from the path
	if !containsString(manifestContent, "myapp-1.0.0.exe") {
		t.Error("expected manifest to contain filename from artifact path")
	}
}

// =============================================================================
// Test helpers
// =============================================================================

// testError is a simple error for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

// statResult holds the result for a stat operation.
type statResult struct {
	info updates.FileInfo
	err  error
}

// platformSpecificMockStatProvider returns different results based on path.
type platformSpecificMockStatProvider struct {
	results map[string]statResult
}

func (m *platformSpecificMockStatProvider) Stat(path string) (updates.FileInfo, error) {
	if result, ok := m.results[path]; ok {
		return result.info, result.err
	}
	return updates.FileInfo{}, &testError{msg: "unexpected path: " + path}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
