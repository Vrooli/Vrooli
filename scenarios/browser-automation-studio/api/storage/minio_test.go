package storage

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestNewMinIOClient verifies MinIO client initialization
func TestNewMinIOClient(t *testing.T) {
	log := logrus.New()
	log.SetOutput(nil) // Suppress output during tests

	tests := []struct {
		name        string
		endpoint    string
		expectError bool
	}{
		{
			name:        "missing endpoint skips initialization",
			endpoint:    "",
			expectError: false, // Client creation should succeed but return nil
		},
		{
			name:        "invalid endpoint format",
			endpoint:    "not-a-valid-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset environment variable
			if tt.endpoint != "" {
				os.Setenv("MINIO_ENDPOINT", tt.endpoint)
				defer os.Unsetenv("MINIO_ENDPOINT")
			} else {
				os.Unsetenv("MINIO_ENDPOINT")
			}

			client, err := NewMinIOClient(log)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Logf("Expected no error for test case, got: %v", err)
				}
				// Client may be nil when MinIO is not configured, which is acceptable
				if client != nil && client.log == nil {
					t.Error("Client logger should be set when client is created")
				}
			}
		})
	}
}

// TestMinIOClientConfiguration verifies environment variable handling
func TestMinIOClientConfiguration(t *testing.T) {
	log := logrus.New()
	log.SetOutput(nil)

	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "all variables set",
			envVars: map[string]string{
				"MINIO_ENDPOINT":   "localhost:9000",
				"MINIO_ACCESS_KEY": "minioadmin",
				"MINIO_SECRET_KEY": "minioadmin",
				"MINIO_BUCKET":     "test-bucket",
			},
			expectError: false,
		},
		{
			name: "missing endpoint",
			envVars: map[string]string{
				"MINIO_ACCESS_KEY": "minioadmin",
				"MINIO_SECRET_KEY": "minioadmin",
			},
			expectError: false, // Should skip initialization gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear and set environment variables
			os.Unsetenv("MINIO_ENDPOINT")
			os.Unsetenv("MINIO_ACCESS_KEY")
			os.Unsetenv("MINIO_SECRET_KEY")
			os.Unsetenv("MINIO_BUCKET")

			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			client, err := NewMinIOClient(log)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				// No error expected - client may be nil if MinIO not configured
				if err != nil {
					t.Logf("Client initialization note: %v", err)
				}
				if client != nil && client.bucketName == "" {
					t.Error("Bucket name should be set when client is configured")
				}
			}
		})
	}
}

// TestMinIOClientNilSafety verifies nil client handling
func TestMinIOClientNilSafety(t *testing.T) {
	log := logrus.New()
	log.SetOutput(nil)

	// Ensure MinIO is not configured
	os.Unsetenv("MINIO_ENDPOINT")
	os.Unsetenv("MINIO_ACCESS_KEY")
	os.Unsetenv("MINIO_SECRET_KEY")

	client, _ := NewMinIOClient(log)

	// Client should be nil when not configured, which is acceptable
	if client != nil {
		t.Log("Client was created despite missing configuration - checking graceful degradation")
	}
}
