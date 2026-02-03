package smoketest

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileForContent creates a MockFileSystem that returns the given content.
func mockFileForContent(content string) *MockFileSystem {
	return &MockFileSystem{
		OpenFunc: func(path string) (*os.File, error) {
			return createTempFileWithContent(content)
		},
	}
}

// createTempFileWithContent creates a temp file with given content for testing.
func createTempFileWithContent(content string) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "telemetry-test-*.jsonl")
	if err != nil {
		return nil, err
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, err
	}
	// Seek to beginning for reading
	if _, err := tmpFile.Seek(0, 0); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, err
	}
	return tmpFile, nil
}

func TestDefaultTelemetryErrorExtractor_ExtractErrors(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		limit          int
		wantCount      int
		wantFirstEvent string
		wantFirstMsg   string
		wantErr        bool
		category       string
	}{
		{
			name: "extracts smoke_test_failed event",
			content: `{"timestamp":"2026-02-02T15:29:17.042Z","event":"smoke_test_failed","level":"error","session_id":"d41a6b0c-1b88-41bb-abe6-2d89a3538eac","deploymentMode":"bundled","details":{"error":"Error: Bundled payload is missing"}}
`,
			limit:          10,
			wantCount:      1,
			wantFirstEvent: "smoke_test_failed",
			wantFirstMsg:   "Error: Bundled payload is missing",
			category:       "happy_path",
		},
		{
			name: "extracts app_session_failed event",
			content: `{"timestamp":"2026-02-02T15:29:17.042Z","event":"app_session_failed","level":"error","session_id":"abc123","details":{"reason":"app_exit_before_ready"}}
`,
			limit:          10,
			wantCount:      1,
			wantFirstEvent: "app_session_failed",
			wantFirstMsg:   "app_exit_before_ready",
			category:       "happy_path",
		},
		{
			name: "returns most recent error first",
			content: `{"timestamp":"2026-02-02T15:00:00.000Z","event":"smoke_test_failed","details":{"error":"First error"}}
{"timestamp":"2026-02-02T15:00:01.000Z","event":"smoke_test_failed","details":{"error":"Second error"}}
{"timestamp":"2026-02-02T15:00:02.000Z","event":"smoke_test_failed","details":{"error":"Third error"}}
`,
			limit:          10,
			wantCount:      3,
			wantFirstEvent: "smoke_test_failed",
			wantFirstMsg:   "Third error",
			category:       "ordering",
		},
		{
			name: "respects limit parameter",
			content: `{"event":"smoke_test_failed","details":{"error":"Error 1"}}
{"event":"smoke_test_failed","details":{"error":"Error 2"}}
{"event":"smoke_test_failed","details":{"error":"Error 3"}}
`,
			limit:          2,
			wantCount:      2,
			wantFirstEvent: "smoke_test_failed",
			wantFirstMsg:   "Error 3",
			category:       "boundary",
		},
		{
			name: "skips non-error events",
			content: `{"event":"smoke_test_started","details":{"deploymentMode":"bundled"}}
{"event":"smoke_test_failed","details":{"error":"The real error"}}
{"event":"app_session_succeeded","details":{}}
`,
			limit:          10,
			wantCount:      1,
			wantFirstEvent: "smoke_test_failed",
			wantFirstMsg:   "The real error",
			category:       "filtering",
		},
		{
			name: "skips events without error message",
			content: `{"event":"smoke_test_failed","details":{}}
{"event":"smoke_test_failed","details":{"error":"Valid error"}}
`,
			limit:        10,
			wantCount:    1,
			wantFirstMsg: "Valid error",
			category:     "edge_case",
		},
		{
			name:      "handles empty file",
			content:   "",
			limit:     10,
			wantCount: 0,
			category:  "edge_case",
		},
		{
			name: "handles malformed JSON lines gracefully",
			content: `not valid json
{"event":"smoke_test_failed","details":{"error":"Valid error"}}
also not valid
`,
			limit:        10,
			wantCount:    1,
			wantFirstMsg: "Valid error",
			category:     "error_handling",
		},
		{
			name:      "handles file with only non-error events",
			content:   `{"event":"smoke_test_started","details":{}}`,
			limit:     10,
			wantCount: 0,
			category:  "edge_case",
		},
		{
			name: "extracts deployment mode",
			content: `{"event":"smoke_test_failed","deploymentMode":"external-server","details":{"error":"Test error"}}
`,
			limit:          10,
			wantCount:      1,
			wantFirstMsg:   "Test error",
			wantFirstEvent: "smoke_test_failed",
			category:       "happy_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := mockFileForContent(tc.content)
			extractor := NewTelemetryErrorExtractor(fs)

			errors, err := extractor.ExtractErrors("/fake/path.jsonl", tc.limit)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, errors, tc.wantCount, "error count mismatch")

			if tc.wantCount > 0 {
				if tc.wantFirstEvent != "" {
					assert.Equal(t, tc.wantFirstEvent, errors[0].Event, "first event type mismatch")
				}
				if tc.wantFirstMsg != "" {
					assert.Equal(t, tc.wantFirstMsg, errors[0].Message, "first error message mismatch")
				}
			}
		})
	}
}

func TestDefaultTelemetryErrorExtractor_ExtractLatestError(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantMsg string
		wantNil bool
		wantErr bool
	}{
		{
			name:    "returns latest error",
			content: `{"event":"smoke_test_failed","details":{"error":"First"}}` + "\n" + `{"event":"smoke_test_failed","details":{"error":"Latest"}}`,
			wantMsg: "Latest",
		},
		{
			name:    "returns nil for empty file",
			content: "",
			wantNil: true,
		},
		{
			name:    "returns nil for no errors",
			content: `{"event":"smoke_test_started","details":{}}`,
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := mockFileForContent(tc.content)
			extractor := NewTelemetryErrorExtractor(fs)

			err, extractErr := extractor.ExtractLatestError("/fake/path.jsonl")

			if tc.wantErr {
				require.Error(t, extractErr)
				return
			}

			require.NoError(t, extractErr)

			if tc.wantNil {
				assert.Nil(t, err)
				return
			}

			require.NotNil(t, err)
			assert.Equal(t, tc.wantMsg, err.Message)
		})
	}
}

func TestDefaultTelemetryErrorExtractor_FileNotFound(t *testing.T) {
	fs := &MockFileSystem{
		OpenFunc: func(path string) (*os.File, error) {
			return nil, os.ErrNotExist
		},
	}
	extractor := NewTelemetryErrorExtractor(fs)

	_, err := extractor.ExtractErrors("/nonexistent/path.jsonl", 10)
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestFormatTelemetryError(t *testing.T) {
	tests := []struct {
		name string
		err  *TelemetryError
		want string
	}{
		{
			name: "formats error message",
			err:  &TelemetryError{Message: "Bundled payload is missing"},
			want: "Bundled payload is missing",
		},
		{
			name: "strips Error prefix",
			err:  &TelemetryError{Message: "Error: Bundled payload is missing"},
			want: "Bundled payload is missing",
		},
		{
			name: "returns empty for nil",
			err:  nil,
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatTelemetryError(tc.err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestExtractErrors_PreservesAllFields(t *testing.T) {
	content := `{"timestamp":"2026-02-02T15:29:17.042Z","event":"smoke_test_failed","level":"error","session_id":"test-session","deploymentMode":"bundled","details":{"error":"Test error message"}}`

	fs := mockFileForContent(content)
	extractor := NewTelemetryErrorExtractor(fs)

	errors, err := extractor.ExtractErrors("/fake/path.jsonl", 10)
	require.NoError(t, err)
	require.Len(t, errors, 1)

	telErr := errors[0]
	assert.Equal(t, "smoke_test_failed", telErr.Event)
	assert.Equal(t, "Test error message", telErr.Message)
	assert.Equal(t, "bundled", telErr.DeploymentMode)
	assert.Equal(t, "test-session", telErr.SessionID)
	assert.Equal(t, "2026-02-02T15:29:17.042Z", telErr.Timestamp)
}

// MockFileSystem for testing - implements FileSystem interface
type MockFileSystem struct {
	StatFunc     func(path string) (os.FileInfo, error)
	ReadDirFunc  func(path string) ([]os.DirEntry, error)
	OpenFunc     func(path string) (*os.File, error)
	ChmodFunc    func(path string, mode os.FileMode) error
	ReadFileFunc func(path string) ([]byte, error)
}

func (m *MockFileSystem) Stat(path string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(path)
	}
	return nil, nil
}

func (m *MockFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	if m.ReadDirFunc != nil {
		return m.ReadDirFunc(path)
	}
	return nil, nil
}

func (m *MockFileSystem) Open(path string) (*os.File, error) {
	if m.OpenFunc != nil {
		return m.OpenFunc(path)
	}
	return nil, nil
}

func (m *MockFileSystem) Chmod(path string, mode os.FileMode) error {
	if m.ChmodFunc != nil {
		return m.ChmodFunc(path, mode)
	}
	return nil
}

func (m *MockFileSystem) ReadFile(path string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(path)
	}
	return nil, nil
}

// Ensure content is written to temp files correctly
func TestTempFileHelper(t *testing.T) {
	content := `{"event":"test","details":{}}`
	file, err := createTempFileWithContent(content)
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	// Read back content
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, content, strings.TrimSpace(string(buf[:n])))
}

func TestDefaultTelemetryErrorExtractor_ExtractLatestErrorForSession(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		sessionID string
		wantMsg   string
		wantNil   bool
	}{
		{
			name: "finds error matching session ID",
			content: `{"event":"smoke_test_failed","session_id":"session-1","details":{"error":"Error 1"}}
{"event":"smoke_test_failed","session_id":"session-2","details":{"error":"Error 2"}}`,
			sessionID: "session-1",
			wantMsg:   "Error 1",
		},
		{
			name: "returns most recent matching error",
			content: `{"event":"smoke_test_failed","session_id":"session-1","details":{"error":"First"}}
{"event":"smoke_test_failed","session_id":"session-2","details":{"error":"Unrelated"}}
{"event":"smoke_test_failed","session_id":"session-1","details":{"error":"Latest"}}`,
			sessionID: "session-1",
			wantMsg:   "Latest",
		},
		{
			name: "returns nil when no matching session",
			content: `{"event":"smoke_test_failed","session_id":"session-other","details":{"error":"Other error"}}
{"event":"smoke_test_failed","session_id":"session-other2","details":{"error":"Another error"}}`,
			sessionID: "session-1",
			wantNil:   true,
		},
		{
			name: "falls back to latest when session ID is empty",
			content: `{"event":"smoke_test_failed","session_id":"session-1","details":{"error":"First"}}
{"event":"smoke_test_failed","session_id":"session-2","details":{"error":"Latest"}}`,
			sessionID: "",
			wantMsg:   "Latest",
		},
		{
			name:      "returns nil for empty file",
			content:   "",
			sessionID: "session-1",
			wantNil:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := mockFileForContent(tc.content)
			extractor := NewTelemetryErrorExtractor(fs)

			err, extractErr := extractor.ExtractLatestErrorForSession("/fake/path.jsonl", tc.sessionID)

			require.NoError(t, extractErr)

			if tc.wantNil {
				assert.Nil(t, err)
				return
			}

			require.NotNil(t, err)
			assert.Equal(t, tc.wantMsg, err.Message)
		})
	}
}

func TestIsErrorStale(t *testing.T) {
	// Reference time: 2026-02-02T12:00:00Z
	refTime, _ := time.Parse(time.RFC3339, "2026-02-02T12:00:00Z")

	tests := []struct {
		name      string
		err       *TelemetryError
		startTime time.Time
		wantStale bool
	}{
		{
			name:      "nil error",
			err:       nil,
			startTime: refTime,
			wantStale: false,
		},
		{
			name: "empty timestamp",
			err: &TelemetryError{
				Message:   "Error",
				Timestamp: "",
			},
			startTime: refTime,
			wantStale: false,
		},
		{
			name: "error before start time is stale",
			err: &TelemetryError{
				Message:   "Old error",
				Timestamp: "2026-02-02T11:00:00Z",
			},
			startTime: refTime,
			wantStale: true,
		},
		{
			name: "error after start time is not stale",
			err: &TelemetryError{
				Message:   "New error",
				Timestamp: "2026-02-02T13:00:00Z",
			},
			startTime: refTime,
			wantStale: false,
		},
		{
			name: "error at exact start time is not stale",
			err: &TelemetryError{
				Message:   "Exact error",
				Timestamp: "2026-02-02T12:00:00Z",
			},
			startTime: refTime,
			wantStale: false,
		},
		{
			name: "handles millisecond timestamp format",
			err: &TelemetryError{
				Message:   "Millis error",
				Timestamp: "2026-02-02T11:30:00.123Z",
			},
			startTime: refTime,
			wantStale: true,
		},
		{
			name: "invalid timestamp format returns false",
			err: &TelemetryError{
				Message:   "Bad timestamp",
				Timestamp: "not-a-timestamp",
			},
			startTime: refTime,
			wantStale: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsErrorStale(tc.err, tc.startTime)
			assert.Equal(t, tc.wantStale, result)
		})
	}
}
