package httputil

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	type testPayload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name      string
		input     string
		wantErr   bool
		wantName  string
		wantValue int
	}{
		{
			name:      "valid JSON",
			input:     `{"name": "test", "value": 42}`,
			wantErr:   false,
			wantName:  "test",
			wantValue: 42,
		},
		{
			name:    "invalid JSON",
			input:   `{not valid json`,
			wantErr: true,
		},
		{
			name:    "unknown fields rejected",
			input:   `{"name": "test", "unknown": "field"}`,
			wantErr: true,
		},
		{
			name:    "multiple JSON values rejected",
			input:   `{"name": "test"}{"name": "extra"}`,
			wantErr: true,
		},
		{
			name:      "empty object is valid",
			input:     `{}`,
			wantErr:   false,
			wantName:  "",
			wantValue: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result, err := DecodeJSON[testPayload](reader, 1024)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeJSON() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("DecodeJSON() unexpected error: %v", err)
				return
			}

			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			if result.Value != tt.wantValue {
				t.Errorf("Value = %d, want %d", result.Value, tt.wantValue)
			}
		})
	}
}

func TestDecodeJSON_NilReader(t *testing.T) {
	t.Parallel()

	type payload struct{}
	_, err := DecodeJSON[payload](nil, 1024)
	if err == nil {
		t.Error("DecodeJSON(nil) should return error")
	}
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	payload := map[string]string{"key": "value"}

	WriteJSON(w, http.StatusOK, payload)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
	}

	expected := `{"key":"value"}`
	body := strings.TrimSpace(w.Body.String())
	if body != expected {
		t.Errorf("Body = %q, want %q", body, expected)
	}
}

func TestWriteAPIError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	apiErr := APIError{
		Code:    "test_error",
		Message: "Test error message",
		Hint:    "Try again",
	}

	WriteAPIError(w, http.StatusBadRequest, apiErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := w.Body.String()
	if !strings.Contains(body, "test_error") {
		t.Errorf("Body should contain error code: %s", body)
	}
	if !strings.Contains(body, "Test error message") {
		t.Errorf("Body should contain error message: %s", body)
	}
}
