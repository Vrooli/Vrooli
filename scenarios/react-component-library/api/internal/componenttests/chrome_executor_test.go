package componenttests

import (
	"context"
	"errors"
	"testing"
)

func TestNewChromeHarnessExecutorUsesConfiguredAPIAddress(t *testing.T) {
	t.Setenv("RCL_API_BASE_URL", "http://preview.example.test/")
	t.Setenv("API_PORT", "9999")
	executor := NewChromeHarnessExecutor()
	if executor.BaseURL != "http://preview.example.test" {
		t.Fatalf("BaseURL = %q, want configured RCL_API_BASE_URL", executor.BaseURL)
	}
}

func TestNewChromeHarnessExecutorDerivesAddressFromAPIPort(t *testing.T) {
	t.Setenv("RCL_API_BASE_URL", "")
	t.Setenv("API_PORT", "17321")
	executor := NewChromeHarnessExecutor()
	if executor.BaseURL != "http://127.0.0.1:17321" {
		t.Fatalf("BaseURL = %q, want API_PORT-derived address", executor.BaseURL)
	}
}

func TestDecodeRenderedStoryResultPreservesPlantedExpectationFailure(t *testing.T) {
	execution, err := decodeRenderedStoryResult([]byte(`<html><pre id="rcl-story-result">{"passed":false,"failures":[{"message":"expect[0] text Missing label not found"}]}</pre></html>`))
	if err != nil {
		t.Fatalf("decodeRenderedStoryResult() error = %v", err)
	}
	if execution.Passed {
		t.Fatal("planted failing expectation was reported as passed")
	}
	if len(execution.Failures) != 1 || execution.Failures[0] != "expect[0] text Missing label not found" {
		t.Fatalf("failures = %#v", execution.Failures)
	}
}

func TestChromeHarnessExecutorReportsMissingChromeAsUnavailable(t *testing.T) {
	executor := ChromeHarnessExecutor{BaseURL: "http://preview.example.test", ChromePath: "definitely-not-installed-rcl-chrome"}
	_, err := executor.ExecuteStory(context.Background(), "rcl:button", "1.0.0", "default")
	var unavailable ExecutorUnavailableError
	if !errors.As(err, &unavailable) {
		t.Fatalf("ExecuteStory() error = %v, want ExecutorUnavailableError", err)
	}
}
