package ai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckProviderResponse_OK(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}
	if err := checkProviderResponse(resp, "test"); err != nil {
		t.Errorf("200 should return nil, got %v", err)
	}
}

func TestCheckProviderResponse_NonOK(t *testing.T) {
	rec := httptest.NewRecorder()
	_, _ = rec.WriteString("rate limited")
	resp := rec.Result()
	resp.StatusCode = http.StatusTooManyRequests
	err := checkProviderResponse(resp, "openrouter")
	if err == nil {
		t.Fatal("non-200 should return error")
	}
	if !strings.Contains(err.Error(), "openrouter") {
		t.Errorf("error should mention provider name, got %v", err)
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("error should mention status code, got %v", err)
	}
}
