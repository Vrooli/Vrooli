package apijson

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		type payload struct {
			Status string `json:"status"`
		}

		got, err := Parse[payload]([]byte(`{"status":"ok"}`), "parse payload")
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}
		if got.Status != "ok" {
			t.Fatalf("expected status ok, got %q", got.Status)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		_, err := Parse[map[string]any]([]byte(" \n\t "), "parse payload")
		if err == nil {
			t.Fatal("expected empty body to fail")
		}
		if !strings.Contains(err.Error(), "empty response body") {
			t.Fatalf("expected empty-body diagnostic, got %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := Parse[map[string]any]([]byte(`{`), "parse payload")
		if err == nil {
			t.Fatal("expected invalid json to fail")
		}
		if !strings.Contains(err.Error(), "parse payload") {
			t.Fatalf("expected action context in error, got %v", err)
		}
	})
}
