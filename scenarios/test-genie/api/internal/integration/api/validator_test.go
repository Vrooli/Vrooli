package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"test-genie/internal/structure/types"
)

// mockHTTPClient implements HTTPClient for testing.
type mockHTTPClient struct {
	response *http.Response
	err      error
	delay    time.Duration
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.response, m.err
}

func newMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestValidateHealthyEndpoint(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-API-01] healthy endpoint returns success", func(t *testing.T) {
		client := &mockHTTPClient{
			response: newMockResponse(200, `{"status":"healthy"}`),
		}

		v := New(Config{
			BaseURL:        "http://localhost:8080",
			HealthEndpoint: "/health",
			MaxResponseMs:  1000,
		}, WithHTTPClient(client), WithLogger(io.Discard))

		result := v.Validate(context.Background())

		if !result.Success {
			t.Fatalf("expected success, got error: %v", result.Error)
		}
		if result.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", result.StatusCode)
		}
		if result.HealthEndpoint != "/health" {
			t.Errorf("expected endpoint /health, got %s", result.HealthEndpoint)
		}
	})
}

func TestValidateUnhealthyEndpoint(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-API-02] unhealthy endpoint returns failure", func(t *testing.T) {
		client := &mockHTTPClient{
			response: newMockResponse(503, `{"status":"unhealthy"}`),
		}

		v := New(Config{
			BaseURL:        "http://localhost:8080",
			HealthEndpoint: "/health",
		}, WithHTTPClient(client), WithLogger(io.Discard))

		result := v.Validate(context.Background())

		if result.Success {
			t.Fatal("expected failure for unhealthy endpoint")
		}
		if result.StatusCode != 503 {
			t.Errorf("expected status 503, got %d", result.StatusCode)
		}
		if result.FailureClass != types.FailureClassSystem {
			t.Errorf("expected system failure class, got %s", result.FailureClass)
		}
	})
}

func TestValidateUnreachableEndpoint(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-API-03] unreachable endpoint returns failure", func(t *testing.T) {
		client := &mockHTTPClient{
			err: &mockNetError{message: "connection refused"},
		}

		v := New(Config{
			BaseURL:        "http://localhost:8080",
			HealthEndpoint: "/health",
		}, WithHTTPClient(client), WithLogger(io.Discard))

		result := v.Validate(context.Background())

		if result.Success {
			t.Fatal("expected failure for unreachable endpoint")
		}
		if !strings.Contains(result.Error.Error(), "unreachable") {
			t.Errorf("expected unreachable error, got: %v", result.Error)
		}
	})
}

func TestValidateSlowEndpoint(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-API-04] slow endpoint exceeding threshold fails", func(t *testing.T) {
		client := &mockHTTPClient{
			response: newMockResponse(200, `{"status":"healthy"}`),
			delay:    50 * time.Millisecond,
		}

		v := New(Config{
			BaseURL:       "http://localhost:8080",
			MaxResponseMs: 10, // Very low threshold
		}, WithHTTPClient(client), WithLogger(io.Discard))

		result := v.Validate(context.Background())

		if result.Success {
			t.Fatal("expected failure for slow endpoint")
		}
		if !strings.Contains(result.Error.Error(), "exceeds threshold") {
			t.Errorf("expected threshold error, got: %v", result.Error)
		}
		if result.ResponseTimeMs < 50 {
			t.Errorf("expected response time >= 50ms, got %dms", result.ResponseTimeMs)
		}
	})
}

func TestValidateContextCancellation(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-API-05] cancelled context returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		v := New(Config{
			BaseURL: "http://localhost:8080",
		})

		result := v.Validate(ctx)

		if result.Success {
			t.Fatal("expected failure for cancelled context")
		}
		if result.FailureClass != types.FailureClassSystem {
			t.Errorf("expected system failure class, got %s", result.FailureClass)
		}
	})
}

func TestValidateCustomExpectedStatus(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-API-06] custom expected status is respected", func(t *testing.T) {
		client := &mockHTTPClient{
			response: newMockResponse(204, ""),
		}

		v := New(Config{
			BaseURL:        "http://localhost:8080",
			ExpectedStatus: 204,
		}, WithHTTPClient(client), WithLogger(io.Discard))

		result := v.Validate(context.Background())

		if !result.Success {
			t.Fatalf("expected success for status 204, got error: %v", result.Error)
		}
		if result.StatusCode != 204 {
			t.Errorf("expected status 204, got %d", result.StatusCode)
		}
	})
}

func TestValidateDefaultConfig(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-API-07] default config values are applied", func(t *testing.T) {
		client := &mockHTTPClient{
			response: newMockResponse(200, `{}`),
		}

		v := New(Config{
			BaseURL: "http://localhost:8080",
			// No HealthEndpoint, MaxResponseMs, or ExpectedStatus specified
		}, WithHTTPClient(client))

		result := v.Validate(context.Background())

		if !result.Success {
			t.Fatalf("expected success with defaults, got error: %v", result.Error)
		}
		// Verify defaults were applied
		if result.HealthEndpoint != "/health" {
			t.Errorf("expected default endpoint /health, got %s", result.HealthEndpoint)
		}
	})
}

func TestValidateObservationsRecorded(t *testing.T) {
	t.Run("[REQ:TESTGENIE-INT-API-08] observations are properly recorded", func(t *testing.T) {
		client := &mockHTTPClient{
			response: newMockResponse(200, `{"status":"healthy"}`),
		}

		v := New(Config{
			BaseURL: "http://localhost:8080",
		}, WithHTTPClient(client), WithLogger(io.Discard))

		result := v.Validate(context.Background())

		if len(result.Observations) == 0 {
			t.Fatal("expected observations to be recorded")
		}

		// Should have section header
		hasSection := false
		hasSuccess := false
		for _, obs := range result.Observations {
			if obs.Type == types.ObservationSection {
				hasSection = true
			}
			if obs.Type == types.ObservationSuccess {
				hasSuccess = true
			}
		}
		if !hasSection {
			t.Error("expected section observation")
		}
		if !hasSuccess {
			t.Error("expected success observation")
		}
	})
}

// mockNetError implements net.Error for testing.
type mockNetError struct {
	message string
}

func (e *mockNetError) Error() string   { return e.message }
func (e *mockNetError) Timeout() bool   { return false }
func (e *mockNetError) Temporary() bool { return false }
