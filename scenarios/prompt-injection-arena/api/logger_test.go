//go:build testing
// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestLoggerStructure(t *testing.T) {
	t.Run("LoggerCreation", func(t *testing.T) {
		logger := NewLogger("test-service")
		if logger == nil {
			t.Fatal("NewLogger should return non-nil logger")
		}
		if logger.service != "test-service" {
			t.Errorf("Expected service='test-service', got '%s'", logger.service)
		}
	})

	t.Run("StructuredLogging", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger("test-service")
		logger.Info("test message", map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		})

		// Restore stdout
		w.Close()
		os.Stdout = old

		// Read captured output
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Validate JSON structure
		var entry LogEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("Output should be valid JSON: %v", err)
		}

		if entry.Level != string(LogLevelInfo) {
			t.Errorf("Expected level='INFO', got '%s'", entry.Level)
		}
		if entry.Service != "test-service" {
			t.Errorf("Expected service='test-service', got '%s'", entry.Service)
		}
		if entry.Message != "test message" {
			t.Errorf("Expected message='test message', got '%s'", entry.Message)
		}
		if entry.Fields["key1"] != "value1" {
			t.Errorf("Expected field key1='value1', got '%v'", entry.Fields["key1"])
		}
		if entry.Fields["key2"] != float64(42) {
			t.Errorf("Expected field key2=42, got '%v'", entry.Fields["key2"])
		}
	})
}

func TestLogLevels(t *testing.T) {
	testCases := []struct {
		name     string
		logFunc  func(*Logger, string, ...map[string]interface{})
		expected LogLevel
	}{
		{"Debug", (*Logger).Debug, LogLevelDebug},
		{"Info", (*Logger).Info, LogLevelInfo},
		{"Warn", (*Logger).Warn, LogLevelWarn},
		{"Error", (*Logger).Error, LogLevelError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := NewLogger("test-service")
			tc.logFunc(logger, "test message")

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Validate JSON
			var entry LogEntry
			if err := json.Unmarshal([]byte(output), &entry); err != nil {
				t.Fatalf("Output should be valid JSON: %v", err)
			}

			if entry.Level != string(tc.expected) {
				t.Errorf("Expected level='%s', got '%s'", tc.expected, entry.Level)
			}
		})
	}
}

func TestLoggerWithFields(t *testing.T) {
	t.Run("WithoutFields", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger("test-service")
		logger.Info("test message")

		// Restore stdout
		w.Close()
		os.Stdout = old

		// Read captured output
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Validate JSON
		var entry LogEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("Output should be valid JSON: %v", err)
		}

		if entry.Fields != nil && len(entry.Fields) > 0 {
			t.Errorf("Expected no fields, got %v", entry.Fields)
		}
	})

	t.Run("WithMultipleFields", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger("test-service")
		logger.Info("test message", map[string]interface{}{
			"string":  "value",
			"number":  123,
			"boolean": true,
			"nested": map[string]interface{}{
				"key": "value",
			},
		})

		// Restore stdout
		w.Close()
		os.Stdout = old

		// Read captured output
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Validate JSON
		var entry LogEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("Output should be valid JSON: %v", err)
		}

		if len(entry.Fields) != 4 {
			t.Errorf("Expected 4 fields, got %d", len(entry.Fields))
		}
	})
}

func TestLoggerEdgeCases(t *testing.T) {
	t.Run("EmptyServiceName", func(t *testing.T) {
		logger := NewLogger("")
		if logger.service != "" {
			t.Errorf("Expected empty service name to be preserved")
		}
	})

	t.Run("SpecialCharactersInMessage", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger("test-service")
		specialMsg := "test \"quoted\" message with\nnewlines and\ttabs"
		logger.Info(specialMsg)

		// Restore stdout
		w.Close()
		os.Stdout = old

		// Read captured output
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Validate JSON parsing handles special characters
		var entry LogEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("Output should be valid JSON even with special chars: %v", err)
		}
	})

	t.Run("NilFieldsMap", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		logger := NewLogger("test-service")
		logger.Info("test message", nil)

		// Restore stdout
		w.Close()
		os.Stdout = old

		// Read captured output
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Should not crash
		var entry LogEntry
		if err := json.Unmarshal([]byte(output), &entry); err != nil {
			t.Fatalf("Should handle nil fields map: %v", err)
		}
	})
}

func TestLoggerPerformance(t *testing.T) {
	t.Run("HighThroughput", func(t *testing.T) {
		// Redirect stdout to /dev/null for performance test
		old := os.Stdout
		devNull, _ := os.Open(os.DevNull)
		os.Stdout = devNull
		defer devNull.Close()

		logger := NewLogger("test-service")

		// Log 1000 messages
		for i := 0; i < 1000; i++ {
			logger.Info("performance test", map[string]interface{}{
				"iteration": i,
				"data":      "some data here",
			})
		}

		// Restore stdout
		os.Stdout = old

		// Test passes if no crashes or deadlocks
	})
}

func TestInitLogger(t *testing.T) {
	t.Run("GlobalLoggerInitialization", func(t *testing.T) {
		InitLogger()
		if logger == nil {
			t.Fatal("InitLogger should initialize global logger")
		}
		if logger.service != "prompt-injection-arena-api" {
			t.Errorf("Expected service='prompt-injection-arena-api', got '%s'", logger.service)
		}
	})
}

func TestLogEntry(t *testing.T) {
	t.Run("JSONSerialization", func(t *testing.T) {
		entry := LogEntry{
			Timestamp: "2024-01-01T00:00:00Z",
			Level:     "INFO",
			Service:   "test-service",
			Message:   "test message",
			Fields: map[string]interface{}{
				"key": "value",
			},
		}

		jsonBytes, err := json.Marshal(entry)
		if err != nil {
			t.Fatalf("LogEntry should be JSON serializable: %v", err)
		}

		// Verify JSON contains expected fields
		jsonStr := string(jsonBytes)
		expectedFields := []string{"timestamp", "level", "service", "message", "fields"}
		for _, field := range expectedFields {
			if !strings.Contains(jsonStr, field) {
				t.Errorf("JSON should contain field '%s'", field)
			}
		}
	})

	t.Run("JSONDeserialization", func(t *testing.T) {
		jsonStr := `{
			"timestamp": "2024-01-01T00:00:00Z",
			"level": "INFO",
			"service": "test-service",
			"message": "test message",
			"fields": {"key": "value"}
		}`

		var entry LogEntry
		if err := json.Unmarshal([]byte(jsonStr), &entry); err != nil {
			t.Fatalf("Should deserialize valid JSON: %v", err)
		}

		if entry.Level != "INFO" {
			t.Errorf("Expected level='INFO', got '%s'", entry.Level)
		}
		if entry.Message != "test message" {
			t.Errorf("Expected message='test message', got '%s'", entry.Message)
		}
	})
}
