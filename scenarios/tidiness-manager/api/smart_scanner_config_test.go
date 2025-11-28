package main

import (
	"os"
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
		{
			name:             "invalid - negative files per batch",
			maxFilesPerBatch: -5,
			maxConcurrent:    5,
			expectedValid:    false,
		},
		{
			name:             "invalid - negative concurrency",
			maxFilesPerBatch: 10,
			maxConcurrent:    -3,
			expectedValid:    false,
		},
		{
			name:             "invalid - both zero",
			maxFilesPerBatch: 0,
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

// [REQ:TM-SS-001] Test default configuration values
func TestSmartScannerDefaults(t *testing.T) {
	config := GetDefaultSmartScanConfig()

	expectedMaxFiles := 10
	expectedMaxConcurrent := 5
	expectedMaxTokens := 100000

	if config.MaxFilesPerBatch != expectedMaxFiles {
		t.Errorf("Default MaxFilesPerBatch = %d, want %d", config.MaxFilesPerBatch, expectedMaxFiles)
	}

	if config.MaxConcurrent != expectedMaxConcurrent {
		t.Errorf("Default MaxConcurrent = %d, want %d", config.MaxConcurrent, expectedMaxConcurrent)
	}

	if config.MaxTokensPerReq != expectedMaxTokens {
		t.Errorf("Default MaxTokensPerReq = %d, want %d", config.MaxTokensPerReq, expectedMaxTokens)
	}

	// Verify default configuration is valid
	if !config.Validate() {
		t.Error("Default configuration should be valid")
	}
}

// [REQ:TM-SS-001] Test NewSmartScanner initialization
func TestNewSmartScanner(t *testing.T) {
	tests := []struct {
		name        string
		config      SmartScanConfig
		envVarValue string
		shouldError bool
		expectedURL string
	}{
		{
			name: "valid config with default URL",
			config: SmartScanConfig{
				MaxFilesPerBatch: 10,
				MaxConcurrent:    5,
				MaxTokensPerReq:  100000,
			},
			envVarValue: "",
			shouldError: false,
			expectedURL: "http://localhost:8100",
		},
		{
			name: "valid config with custom URL",
			config: SmartScanConfig{
				MaxFilesPerBatch: 10,
				MaxConcurrent:    5,
				MaxTokensPerReq:  100000,
			},
			envVarValue: "http://custom-claude:9000",
			shouldError: false,
			expectedURL: "http://custom-claude:9000",
		},
		{
			name: "invalid config - zero batch size",
			config: SmartScanConfig{
				MaxFilesPerBatch: 0,
				MaxConcurrent:    5,
				MaxTokensPerReq:  100000,
			},
			envVarValue: "",
			shouldError: true,
		},
		{
			name: "invalid config - zero concurrency",
			config: SmartScanConfig{
				MaxFilesPerBatch: 10,
				MaxConcurrent:    0,
				MaxTokensPerReq:  100000,
			},
			envVarValue: "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envVarValue != "" {
				oldEnv := os.Getenv("CLAUDE_CODE_URL")
				os.Setenv("CLAUDE_CODE_URL", tt.envVarValue)
				defer os.Setenv("CLAUDE_CODE_URL", oldEnv)
			}

			scanner, err := NewSmartScanner(tt.config)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if scanner == nil {
				t.Fatal("Scanner is nil")
			}

			if scanner.resourceURL != tt.expectedURL {
				t.Errorf("resourceURL = %s, want %s", scanner.resourceURL, tt.expectedURL)
			}

			if scanner.sessionID == "" {
				t.Error("sessionID should not be empty")
			}

			if scanner.httpClient == nil {
				t.Error("httpClient should not be nil")
			}

			if scanner.analyzedFiles == nil {
				t.Error("analyzedFiles map should be initialized")
			}
		})
	}
}

// [REQ:TM-SS-001] Test config validation boundary conditions
func TestSmartScanConfig_ValidationBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		config    SmartScanConfig
		wantValid bool
	}{
		{
			name: "max int values",
			config: SmartScanConfig{
				MaxFilesPerBatch: 2147483647,
				MaxConcurrent:    2147483647,
				MaxTokensPerReq:  2147483647,
			},
			wantValid: true,
		},
		{
			name: "boundary - 1 file, 1 concurrent",
			config: SmartScanConfig{
				MaxFilesPerBatch: 1,
				MaxConcurrent:    1,
				MaxTokensPerReq:  1,
			},
			wantValid: true,
		},
		{
			name: "negative max tokens (should be valid - just config)",
			config: SmartScanConfig{
				MaxFilesPerBatch: 10,
				MaxConcurrent:    5,
				MaxTokensPerReq:  -1,
			},
			wantValid: true, // Token limit doesn't affect validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.config.Validate()
			if valid != tt.wantValid {
				t.Errorf("Validate() = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}
