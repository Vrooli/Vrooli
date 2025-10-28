package logging

import (
	"testing"
)

func TestLogFunctions(t *testing.T) {
	// Test that logging functions don't panic
	// We can't easily test output since it goes to slog.Default()
	tests := []struct {
		name    string
		logFunc func(string, ...interface{})
		msg     string
	}{
		{
			name:    "LogInfo",
			logFunc: LogInfo,
			msg:     "test info message",
		},
		{
			name:    "LogDebug",
			logFunc: LogDebug,
			msg:     "test debug message",
		},
		{
			name:    "LogWarn",
			logFunc: LogWarn,
			msg:     "test warn message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			tt.logFunc(tt.msg, "key", "value")
		})
	}
}

func TestLogErrorWithDetails(t *testing.T) {
	// These functions log to slog.Default() which we can't easily capture in tests
	// Just verify they don't panic
	LogError("test error", "error_code", 500, "details", "something went wrong")
	// If we reach here without panic, test passes
}

func TestLogErrorErr(t *testing.T) {
	// These functions log to slog.Default() which we can't easily capture in tests
	// Just verify they don't panic
	testErr := &testError{msg: "test error"}
	LogErrorErr("operation failed", testErr, "operation", "test_op")
	// If we reach here without panic, test passes
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
