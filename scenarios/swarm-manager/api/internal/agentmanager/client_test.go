package agentmanager

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type mockHTTPDoer struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func makeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestNewHTTPClient_Defaults(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatalf("expected client")
	}
	if client.baseURLResolver == nil {
		t.Fatalf("expected baseURLResolver")
	}
	if client.httpClient == nil {
		t.Fatalf("expected httpClient")
	}
}

func TestNewHTTPClientWithResolver_Defaults(t *testing.T) {
	client := NewHTTPClientWithResolver(nil, nil)
	if client.baseURLResolver == nil {
		t.Fatalf("expected default resolver")
	}
	if client.httpClient == nil {
		t.Fatalf("expected default http client")
	}
}

func TestHTTPClient_Health(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		client := NewHTTPClientWithResolver(
			func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
			&mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/health" {
					return makeResponse(http.StatusNotFound, ""), nil
				}
				return makeResponse(http.StatusOK, `{}`), nil
			}},
		)
		ok, err := client.Health(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected healthy")
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		client := NewHTTPClientWithResolver(
			func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
			&mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
				return makeResponse(http.StatusServiceUnavailable, ""), nil
			}},
		)
		ok, err := client.Health(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected unhealthy")
		}
	})

	t.Run("resolver error", func(t *testing.T) {
		client := NewHTTPClientWithResolver(
			func(_ context.Context) (string, error) { return "", ErrNotAvailable },
			&mockHTTPDoer{},
		)
		_, err := client.Health(context.Background())
		if !errors.Is(err, ErrNotAvailable) {
			t.Fatalf("expected ErrNotAvailable, got %v", err)
		}
	})
}

func TestHTTPClient_CreateTask_Proto(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var capturedBody string
		client := NewHTTPClientWithResolver(
			func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
			&mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
				bodyBytes, _ := io.ReadAll(req.Body)
				capturedBody = string(bodyBytes)
				return makeResponse(http.StatusCreated, `{"task":{"id":"task-proto-1","title":"test"}}`), nil
			}},
		)

		task, err := client.CreateTask(context.Background(), buildTestTask("Test Proto Task"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.Id != "task-proto-1" {
			t.Errorf("got task ID %q, want %q", task.Id, "task-proto-1")
		}
		if capturedBody == "" {
			t.Fatal("expected non-empty request body")
		}
	})

	t.Run("server error", func(t *testing.T) {
		client := NewHTTPClientWithResolver(
			func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
			&mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
				return makeResponse(http.StatusInternalServerError, `{}`), nil
			}},
		)
		_, err := client.CreateTask(context.Background(), buildTestTask("fail"))
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, ErrRequestFailed) {
			t.Errorf("expected ErrRequestFailed, got %v", err)
		}
	})
}

func TestHTTPClient_GetRun(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		client := NewHTTPClientWithResolver(
			func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
			&mockHTTPDoer{},
		)
		_, err := client.GetRun(context.Background(), "")
		if err == nil {
			t.Fatal("expected error for empty run ID")
		}
	})

	t.Run("success", func(t *testing.T) {
		client := NewHTTPClientWithResolver(
			func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
			&mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
				return makeResponse(http.StatusOK, `{"run":{"id":"run-1","taskId":"task-1","status":"RUN_STATUS_RUNNING"}}`), nil
			}},
		)
		run, err := client.GetRun(context.Background(), "run-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.Id != "run-1" {
			t.Errorf("got run ID %q, want %q", run.Id, "run-1")
		}
	})
}

func TestHTTPClient_StopRun(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		client := NewHTTPClientWithResolver(
			func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
			&mockHTTPDoer{},
		)
		err := client.StopRun(context.Background(), "  ")
		if err == nil {
			t.Fatal("expected error for empty run ID")
		}
	})

	t.Run("success", func(t *testing.T) {
		client := NewHTTPClientWithResolver(
			func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
			&mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
				return makeResponse(http.StatusOK, `{}`), nil
			}},
		)
		err := client.StopRun(context.Background(), "run-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResolveAgentManagerBaseURL(t *testing.T) {
	url, err := resolveAgentManagerBaseURL(context.Background())
	if err != nil {
		t.Skipf("agent-manager not available: %v", err)
	}
	if !strings.HasPrefix(url, "http") {
		t.Fatalf("expected base URL to be http(s), got %q", url)
	}
}

// buildTestTask creates a proto Task for testing.
func buildTestTask(title string) *domainpb.Task {
	return &domainpb.Task{
		Title: title,
	}
}
