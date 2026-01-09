package httpjson

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecode(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	t.Run("valid JSON", func(t *testing.T) {
		body := `{"name":"test","value":42}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()

		var result testStruct
		err := Decode(w, req, &result)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Name != "test" || result.Value != 42 {
			t.Errorf("values not decoded correctly: %+v", result)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(""))
		w := httptest.NewRecorder()

		var result testStruct
		err := Decode(w, req, &result)
		if err == nil {
			t.Error("expected error for empty body")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		body := `{"name":"test",invalid}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()

		var result testStruct
		err := Decode(w, req, &result)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("unknown fields rejected", func(t *testing.T) {
		body := `{"name":"test","value":42,"unknown":"field"}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()

		var result testStruct
		err := Decode(w, req, &result)
		if err == nil {
			t.Error("expected error for unknown field")
		}
	})

	t.Run("multiple JSON objects rejected", func(t *testing.T) {
		body := `{"name":"first","value":1}{"name":"second","value":2}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()

		var result testStruct
		err := Decode(w, req, &result)
		if err == nil {
			t.Error("expected error for multiple JSON objects")
		}
		if err != nil && !strings.Contains(err.Error(), "single JSON object") {
			t.Errorf("expected 'single JSON object' error, got: %v", err)
		}
	})

	t.Run("body size limit", func(t *testing.T) {
		// Create a body larger than MaxBodyBytes (1MB)
		largeBody := bytes.Repeat([]byte("x"), int(MaxBodyBytes())+1)
		req := httptest.NewRequest("POST", "/", bytes.NewReader(largeBody))
		w := httptest.NewRecorder()

		var result testStruct
		err := Decode(w, req, &result)
		if err == nil {
			t.Error("expected error for oversized body")
		}
	})
}

func TestDecodeAllowEmpty(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	t.Run("valid JSON", func(t *testing.T) {
		body := `{"name":"test","value":42}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()

		var result testStruct
		err := DecodeAllowEmpty(w, req, &result)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Name != "test" || result.Value != 42 {
			t.Errorf("values not decoded correctly: %+v", result)
		}
	})

	t.Run("empty body allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(""))
		w := httptest.NewRecorder()

		var result testStruct
		err := DecodeAllowEmpty(w, req, &result)
		if err != nil {
			t.Errorf("expected no error for empty body, got %v", err)
		}
	})

	t.Run("EOF allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", http.NoBody)
		w := httptest.NewRecorder()

		var result testStruct
		err := DecodeAllowEmpty(w, req, &result)
		if err != nil {
			t.Errorf("expected no error for EOF, got %v", err)
		}
	})

	t.Run("invalid JSON still rejected", func(t *testing.T) {
		body := `{"name":"test",invalid}`
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()

		var result testStruct
		err := DecodeAllowEmpty(w, req, &result)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestMaxBodyBytes(t *testing.T) {
	// Test that the default value is 1MB when no config is set
	if MaxBodyBytes() != 1<<20 {
		t.Errorf("MaxBodyBytes() = %d, expected %d (1MB)", MaxBodyBytes(), 1<<20)
	}
}

// Test edge cases with custom io.Reader to simulate various scenarios
type errorReader struct{}

func (e errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestDecodeWithReadError(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest("POST", "/", errorReader{})
	w := httptest.NewRecorder()

	var result testStruct
	err := Decode(w, req, &result)
	if err == nil {
		t.Error("expected error when reading fails")
	}
}
