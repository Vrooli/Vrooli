package execute

import (
	"strings"
	"testing"
)

func TestParseRunResponseRejectsEmptyBody(t *testing.T) {
	_, err := parseRunResponse(nil)
	if err == nil {
		t.Fatal("expected empty response body to fail")
	}
	if !strings.Contains(err.Error(), "empty response body") {
		t.Fatalf("expected empty-body diagnostic, got %v", err)
	}
}

func TestParseRunResponseSuccess(t *testing.T) {
	resp, err := parseRunResponse([]byte(`{"success":true,"executionId":"exec-123"}`))
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.ExecutionID != "exec-123" {
		t.Fatalf("expected execution id exec-123, got %q", resp.ExecutionID)
	}
}

func TestProcessSSEStreamRejectsEarlyTermination(t *testing.T) {
	client := &Client{}

	_, err := client.processSSEStream(strings.NewReader(""), nil, nil)
	if err == nil {
		t.Fatal("expected early stream termination to fail")
	}
	if !strings.Contains(err.Error(), "stream ended before completion event") {
		t.Fatalf("expected stream termination diagnostic, got %v", err)
	}
}
