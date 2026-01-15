package state

import (
	"strings"
	"testing"
)

func TestCompressDecompressLog(t *testing.T) {
	tests := []struct {
		name      string
		serviceID string
		content   string
		wantLines int
	}{
		{
			name:      "simple single line",
			serviceID: "test-service",
			content:   "Hello, World!",
			wantLines: 1,
		},
		{
			name:      "multiple lines",
			serviceID: "preflight-checker",
			content:   "line 1\nline 2\nline 3",
			wantLines: 3,
		},
		{
			name:      "empty string",
			serviceID: "empty-service",
			content:   "",
			wantLines: 0,
		},
		{
			name:      "large log with repeating content",
			serviceID: "large-service",
			content:   strings.Repeat("Log entry: some data here\n", 1000),
			wantLines: 1001, // 1000 newlines + trailing content
		},
		{
			name:      "unicode content",
			serviceID: "unicode-service",
			content:   "Hello, World! unicode chars\nMore text",
			wantLines: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress
			compressed, err := CompressLog(tt.serviceID, tt.content)
			if err != nil {
				t.Fatalf("CompressLog() error = %v", err)
			}

			// Verify metadata
			if compressed.ServiceID != tt.serviceID {
				t.Errorf("ServiceID = %v, want %v", compressed.ServiceID, tt.serviceID)
			}
			if compressed.OriginalLines != tt.wantLines {
				t.Errorf("OriginalLines = %v, want %v", compressed.OriginalLines, tt.wantLines)
			}
			if compressed.CapturedAt == "" {
				t.Error("CapturedAt should not be empty")
			}

			// Decompress and verify content
			decompressed, err := DecompressLog(compressed)
			if err != nil {
				t.Fatalf("DecompressLog() error = %v", err)
			}
			if decompressed != tt.content {
				t.Errorf("DecompressLog() = %q, want %q", decompressed, tt.content)
			}
		})
	}
}

func TestCompressLogCompression(t *testing.T) {
	// Test that compressible content actually gets compressed
	content := strings.Repeat("AAAAAAAAAA", 1000) // Highly compressible

	compressed, err := CompressLog("test", content)
	if err != nil {
		t.Fatalf("CompressLog() error = %v", err)
	}

	// Compressed size should be significantly smaller than original
	originalSize := len(content)
	if compressed.CompressedSize >= originalSize/2 {
		t.Errorf("Expected significant compression, got %d bytes from %d bytes",
			compressed.CompressedSize, originalSize)
	}
}

func TestCompressLogs(t *testing.T) {
	inputs := []LogTailInput{
		{ServiceID: "svc1", Content: "log 1\nlog 2"},
		{ServiceID: "svc2", Content: "log 3"},
		{ServiceID: "svc3", Content: ""}, // Empty should be skipped
	}

	result, err := CompressLogs(inputs)
	if err != nil {
		t.Fatalf("CompressLogs() error = %v", err)
	}

	// Should have 2 results (empty content skipped)
	if len(result) != 2 {
		t.Errorf("CompressLogs() returned %d logs, want 2", len(result))
	}

	// Verify service IDs
	ids := make(map[string]bool)
	for _, cl := range result {
		ids[cl.ServiceID] = true
	}
	if !ids["svc1"] || !ids["svc2"] {
		t.Error("Expected svc1 and svc2 in results")
	}
	if ids["svc3"] {
		t.Error("svc3 with empty content should be skipped")
	}
}

func TestCompressLogsNil(t *testing.T) {
	result, err := CompressLogs(nil)
	if err != nil {
		t.Fatalf("CompressLogs(nil) error = %v", err)
	}
	if result != nil {
		t.Errorf("CompressLogs(nil) = %v, want nil", result)
	}
}

func TestFindCompressedLog(t *testing.T) {
	logs := []CompressedLog{
		{ServiceID: "svc1", CompressedData: "data1"},
		{ServiceID: "svc2", CompressedData: "data2"},
		{ServiceID: "svc3", CompressedData: "data3"},
	}

	tests := []struct {
		name      string
		serviceID string
		wantFound bool
	}{
		{"find first", "svc1", true},
		{"find middle", "svc2", true},
		{"find last", "svc3", true},
		{"not found", "svc4", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindCompressedLog(logs, tt.serviceID)
			if (result != nil) != tt.wantFound {
				t.Errorf("FindCompressedLog() found = %v, want %v", result != nil, tt.wantFound)
			}
			if tt.wantFound && result.ServiceID != tt.serviceID {
				t.Errorf("FindCompressedLog() serviceID = %v, want %v", result.ServiceID, tt.serviceID)
			}
		})
	}
}

func TestMergeLogs(t *testing.T) {
	existing := []CompressedLog{
		{ServiceID: "svc1", CompressedData: "old1"},
		{ServiceID: "svc2", CompressedData: "old2"},
	}

	newLogs := []CompressedLog{
		{ServiceID: "svc2", CompressedData: "new2"}, // Should replace
		{ServiceID: "svc3", CompressedData: "new3"}, // Should add
	}

	result := MergeLogs(existing, newLogs)

	// Should have 3 unique logs
	if len(result) != 3 {
		t.Errorf("MergeLogs() returned %d logs, want 3", len(result))
	}

	// Verify svc2 was replaced
	found := FindCompressedLog(result, "svc2")
	if found == nil || found.CompressedData != "new2" {
		t.Error("svc2 should be replaced with new data")
	}

	// Verify svc1 preserved
	found = FindCompressedLog(result, "svc1")
	if found == nil || found.CompressedData != "old1" {
		t.Error("svc1 should be preserved")
	}

	// Verify svc3 added
	found = FindCompressedLog(result, "svc3")
	if found == nil || found.CompressedData != "new3" {
		t.Error("svc3 should be added")
	}
}

func TestMergeLogsEmptyNew(t *testing.T) {
	existing := []CompressedLog{
		{ServiceID: "svc1", CompressedData: "data1"},
	}

	result := MergeLogs(existing, nil)

	if len(result) != 1 {
		t.Errorf("MergeLogs() with nil new should return existing")
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"single line", 1},
		{"line1\nline2", 2},
		{"line1\nline2\n", 3}, // Trailing newline counts as another line
		{"line1\nline2\nline3", 3},
		{"\n\n\n", 4}, // 3 newlines = 4 "lines"
	}

	for _, tt := range tests {
		got := countLines(tt.input)
		if got != tt.want {
			t.Errorf("countLines(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestCompressionRatio(t *testing.T) {
	tests := []struct {
		name           string
		compressedSize int
		originalSize   int
		wantRatio      float64
	}{
		{"normal compression", 100, 1000, 0.9}, // 90% compression
		{"no compression", 1000, 1000, 0},      // 0% compression ratio is correct
		{"zero original", 100, 0, 0},           // Division by zero case returns 0
		{"zero compressed", 0, 100, 0},         // Zero compressed returns 0
		{"50% compression", 500, 1000, 0.5},    // 50% compression
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := CompressedLog{CompressedSize: tt.compressedSize}
			ratio := CompressionRatio(cl, tt.originalSize)

			if ratio != tt.wantRatio {
				t.Errorf("CompressionRatio() = %v, want %v", ratio, tt.wantRatio)
			}
		})
	}
}
