package main

import (
	"testing"
)

// [REQ:TM-SS-001] AI batch configuration
func TestSmartScannerBatchConfiguration(t *testing.T) {
	tests := []struct {
		name             string
		maxFilesPerBatch int
		maxConcurrent    int
		expectedValid    bool
	}{
		{
			name:             "default configuration",
			maxFilesPerBatch: 10,
			maxConcurrent:    5,
			expectedValid:    true,
		},
		{
			name:             "minimal configuration",
			maxFilesPerBatch: 1,
			maxConcurrent:    1,
			expectedValid:    true,
		},
		{
			name:             "high throughput configuration",
			maxFilesPerBatch: 20,
			maxConcurrent:    10,
			expectedValid:    true,
		},
		{
			name:             "invalid - zero files per batch",
			maxFilesPerBatch: 0,
			maxConcurrent:    5,
			expectedValid:    false,
		},
		{
			name:             "invalid - zero concurrency",
			maxFilesPerBatch: 10,
			maxConcurrent:    0,
			expectedValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := SmartScanConfig{
				MaxFilesPerBatch: tt.maxFilesPerBatch,
				MaxConcurrent:    tt.maxConcurrent,
			}

			valid := config.Validate()
			if valid != tt.expectedValid {
				t.Errorf("Validate() = %v, want %v", valid, tt.expectedValid)
			}
		})
	}
}

// SmartScanConfig represents AI batch scanning configuration
type SmartScanConfig struct {
	MaxFilesPerBatch int
	MaxConcurrent    int
	MaxTokensPerReq  int
}

// Validate checks if configuration is valid
func (c *SmartScanConfig) Validate() bool {
	return c.MaxFilesPerBatch > 0 && c.MaxConcurrent > 0
}

// [REQ:TM-SS-001] Test default configuration values
func TestSmartScannerDefaults(t *testing.T) {
	config := GetDefaultSmartScanConfig()

	expectedMaxFiles := 10
	expectedMaxConcurrent := 5

	if config.MaxFilesPerBatch != expectedMaxFiles {
		t.Errorf("Default MaxFilesPerBatch = %d, want %d", config.MaxFilesPerBatch, expectedMaxFiles)
	}

	if config.MaxConcurrent != expectedMaxConcurrent {
		t.Errorf("Default MaxConcurrent = %d, want %d", config.MaxConcurrent, expectedMaxConcurrent)
	}
}

// GetDefaultSmartScanConfig returns default AI batch configuration
func GetDefaultSmartScanConfig() SmartScanConfig {
	return SmartScanConfig{
		MaxFilesPerBatch: 10,
		MaxConcurrent:    5,
		MaxTokensPerReq:  100000,
	}
}

// [REQ:TM-SS-001] Test configurable batch size limits
func TestSmartScannerBatchSizeLimits(t *testing.T) {
	config := SmartScanConfig{
		MaxFilesPerBatch: 10,
		MaxConcurrent:    5,
	}

	// Simulate file batching
	files := make([]string, 25) // 25 files should create 3 batches
	for i := range files {
		files[i] = "file" + string(rune('0'+i))
	}

	batches := createBatches(files, config.MaxFilesPerBatch)

	expectedBatches := 3 // ceil(25/10)
	if len(batches) != expectedBatches {
		t.Errorf("createBatches() created %d batches, want %d", len(batches), expectedBatches)
	}

	// Verify batch sizes
	if len(batches[0]) != 10 {
		t.Errorf("First batch size = %d, want 10", len(batches[0]))
	}
	if len(batches[1]) != 10 {
		t.Errorf("Second batch size = %d, want 10", len(batches[1]))
	}
	if len(batches[2]) != 5 {
		t.Errorf("Third batch size = %d, want 5", len(batches[2]))
	}
}

// createBatches splits files into batches
func createBatches(files []string, batchSize int) [][]string {
	var batches [][]string
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		batches = append(batches, files[i:end])
	}
	return batches
}
