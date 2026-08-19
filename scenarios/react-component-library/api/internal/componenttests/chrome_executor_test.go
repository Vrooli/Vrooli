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

func TestDecodeStoryResultJSONAcceptsPlaywrightOutput(t *testing.T) {
	execution, err := decodeStoryResultJSON([]byte(`{"passed":true,"failures":[]}`))
	if err != nil {
		t.Fatalf("decodeStoryResultJSON() error = %v", err)
	}
	if !execution.Passed {
		t.Fatal("Playwright success result was reported as failed")
	}
}

func TestDecodeStoryResultJSONPreservesConsoleAndPerformanceEvidence(t *testing.T) {
	execution, err := decodeStoryResultJSON([]byte(`{"passed":true,"failures":[],"console":{"consoleErrors":["warning"],"pageErrors":["uncaught"],"failedRequests":["/api"]},"performance":{"mountMs":12.5,"commitCount":3,"rerenderCount":2,"longTasks":[55.25],"nodeCount":17}}`))
	if err != nil {
		t.Fatalf("decodeStoryResultJSON() error = %v", err)
	}
	if len(execution.Console.ConsoleErrors) != 1 || execution.Console.ConsoleErrors[0] != "warning" {
		t.Fatalf("console errors = %#v", execution.Console.ConsoleErrors)
	}
	if len(execution.Console.PageErrors) != 1 || execution.Console.PageErrors[0] != "uncaught" {
		t.Fatalf("page errors = %#v", execution.Console.PageErrors)
	}
	if len(execution.Console.FailedRequests) != 1 || execution.Console.FailedRequests[0] != "/api" {
		t.Fatalf("failed requests = %#v", execution.Console.FailedRequests)
	}
	if execution.Performance.MountMS != 12.5 || execution.Performance.CommitCount != 3 || execution.Performance.RerenderCount != 2 || execution.Performance.NodeCount != 17 {
		t.Fatalf("performance = %#v", execution.Performance)
	}
	if len(execution.Performance.LongTasks) != 1 || execution.Performance.LongTasks[0] != 55.25 {
		t.Fatalf("long tasks = %#v", execution.Performance.LongTasks)
	}
}

func TestEnvironWithChromeReplacesInheritedValue(t *testing.T) {
	environ := environWithChrome([]string{"PATH=/bin", "RCL_CHROME_BIN=google-chrome", "RCL_CHROME_BIN=stale"}, "/usr/bin/google-chrome")
	for _, entry := range environ {
		if entry == "RCL_CHROME_BIN=google-chrome" || entry == "RCL_CHROME_BIN=stale" {
			t.Fatalf("inherited Chrome value remained in environment: %q", entry)
		}
	}
	if environ[len(environ)-1] != "RCL_CHROME_BIN=/usr/bin/google-chrome" {
		t.Fatalf("normalized Chrome value = %q", environ[len(environ)-1])
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
