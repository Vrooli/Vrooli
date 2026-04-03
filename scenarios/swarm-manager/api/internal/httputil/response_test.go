package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		wantBody string
	}{
		{
			name:     "simple object",
			data:     map[string]string{"status": "ok"},
			wantBody: `{"status":"ok"}`,
		},
		{
			name:     "struct",
			data:     struct{ Name string }{Name: "test"},
			wantBody: `{"Name":"test"}`,
		},
		{
			name:     "array",
			data:     []int{1, 2, 3},
			wantBody: `[1,2,3]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := JSON(w, tt.data)
			if err != nil {
				t.Fatalf("JSON() error = %v", err)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "application/json")
			}

			got := strings.TrimSpace(w.Body.String())
			if got != tt.wantBody {
				t.Errorf("Body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

func TestJSONWithStatus(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "created"}

	err := JSONWithStatus(w, http.StatusCreated, data)
	if err != nil {
		t.Fatalf("JSONWithStatus() error = %v", err)
	}

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusCreated)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), "application/json")
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if result["message"] != "created" {
		t.Errorf("message = %q, want %q", result["message"], "created")
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name         string
		baseDir      string
		relativePath string
		want         bool
	}{
		{
			name:         "valid simple path",
			baseDir:      "/data/ideas",
			relativePath: "my-idea/file.txt",
			want:         true,
		},
		{
			name:         "valid nested path",
			baseDir:      "/data/ideas",
			relativePath: "my-idea/docs/readme.md",
			want:         true,
		},
		{
			name:         "path traversal attack",
			baseDir:      "/data/ideas",
			relativePath: "../../../etc/passwd",
			want:         false,
		},
		{
			name:         "path traversal in middle",
			baseDir:      "/data/ideas",
			relativePath: "my-idea/../../../etc/passwd",
			want:         false,
		},
		{
			name:         "empty relative path",
			baseDir:      "/data/ideas",
			relativePath: "",
			want:         true,
		},
		{
			name:         "base path only",
			baseDir:      "/data/ideas/my-idea",
			relativePath: ".",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePath(tt.baseDir, tt.relativePath)
			if got != tt.want {
				t.Errorf("ValidatePath(%q, %q) = %v, want %v", tt.baseDir, tt.relativePath, got, tt.want)
			}
		})
	}
}

func TestSafeFilePath(t *testing.T) {
	tests := []struct {
		name         string
		baseDir      string
		relativePath string
		wantPath     string
		wantValid    bool
	}{
		{
			name:         "valid path",
			baseDir:      "/data/ideas",
			relativePath: "my-idea/file.txt",
			wantPath:     "/data/ideas/my-idea/file.txt",
			wantValid:    true,
		},
		{
			name:         "path traversal returns empty",
			baseDir:      "/data/ideas",
			relativePath: "../../../etc/passwd",
			wantPath:     "",
			wantValid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotValid := SafeFilePath(tt.baseDir, tt.relativePath)
			if gotValid != tt.wantValid {
				t.Errorf("SafeFilePath() valid = %v, want %v", gotValid, tt.wantValid)
			}
			if gotPath != tt.wantPath {
				t.Errorf("SafeFilePath() path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

// TestJSON_ErrorCase tests JSON encoding error handling
func TestJSON_ErrorCase(t *testing.T) {
	w := httptest.NewRecorder()

	// Create an unmarshalable type (channel cannot be JSON encoded)
	ch := make(chan int)

	err := JSON(w, ch)
	if err == nil {
		t.Error("expected error for unmarshalable type, got nil")
	}
}

// TestJSONWithStatus_Various tests JSONWithStatus with different HTTP status codes
func TestJSONWithStatus_Various(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       any
		wantStatus int
	}{
		{
			name:       "200 OK",
			status:     http.StatusOK,
			data:       map[string]string{"status": "ok"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "201 Created",
			status:     http.StatusCreated,
			data:       map[string]string{"id": "123"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "202 Accepted",
			status:     http.StatusAccepted,
			data:       map[string]string{"task": "queued"},
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "400 Bad Request",
			status:     http.StatusBadRequest,
			data:       map[string]string{"error": "invalid input"},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := JSONWithStatus(w, tt.status, tt.data)
			if err != nil {
				t.Fatalf("JSONWithStatus() error = %v", err)
			}

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want 'application/json'", w.Header().Get("Content-Type"))
			}
		})
	}
}

// TestValidatePath_AdditionalCases tests additional edge cases for path validation
func TestValidatePath_AdditionalCases(t *testing.T) {
	tests := []struct {
		name         string
		baseDir      string
		relativePath string
		want         bool
	}{
		{
			name:         "double dot in filename",
			baseDir:      "/data",
			relativePath: "file..txt",
			want:         true, // Valid - not a traversal, just a filename
		},
		{
			name:         "hidden file",
			baseDir:      "/data",
			relativePath: ".hidden",
			want:         true,
		},
		{
			name:         "deeply nested valid",
			baseDir:      "/data",
			relativePath: "a/b/c/d/e/file.txt",
			want:         true,
		},
		{
			name:         "mixed traversal attempt",
			baseDir:      "/data/ideas",
			relativePath: "foo/../../../etc/passwd",
			want:         false,
		},
		{
			name:         "absolute path treated as relative by filepath.Join",
			baseDir:      "/data/ideas",
			relativePath: "/etc/passwd",
			want:         true, // filepath.Join("/data/ideas", "/etc/passwd") = "/data/ideas/etc/passwd"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePath(tt.baseDir, tt.relativePath)
			if got != tt.want {
				t.Errorf("ValidatePath(%q, %q) = %v, want %v", tt.baseDir, tt.relativePath, got, tt.want)
			}
		})
	}
}
