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

func TestHTTPClient_GetObservedReceipts(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		&mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/v1/runs/run-1/observed-receipts" {
				t.Fatalf("path = %q", req.URL.Path)
			}
			if req.URL.Query().Get("limit") != "100" {
				t.Fatalf("limit = %q, want default 100", req.URL.Query().Get("limit"))
			}
			return makeResponse(http.StatusOK, `{"status":"available","observations":[{"eventId":"receipt-1","correlationId":"run-1"}]}`), nil
		}},
	)

	result, err := client.GetObservedReceipts(context.Background(), "run-1", 0)
	if err != nil {
		t.Fatalf("GetObservedReceipts: %v", err)
	}
	if result.Status != "available" || len(result.Observations) != 1 || result.Observations[0].EventID != "receipt-1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestHTTPClient_GetObservedReceipts_EmptyID(t *testing.T) {
	client := NewHTTPClientWithResolver(func(_ context.Context) (string, error) { return "http://localhost:12345", nil }, &mockHTTPDoer{})
	if _, err := client.GetObservedReceipts(context.Background(), " ", 1); !errors.Is(err, ErrRequestFailed) {
		t.Fatalf("expected ErrRequestFailed, got %v", err)
	}
}

func TestAgentServiceGetRunMessagesPagesAndPreservesEventIdentity(t *testing.T) {
	var requests int
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		&mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
			requests++
			if req.URL.Path != "/api/v1/runs/run-paged/events" {
				t.Fatalf("path = %q", req.URL.Path)
			}
			if req.URL.Query().Get("limit") != "200" {
				t.Fatalf("limit = %q, want 200", req.URL.Query().Get("limit"))
			}
			switch requests {
			case 1:
				if got := req.URL.Query().Get("after_sequence"); got != "" {
					t.Fatalf("first after_sequence = %q, want empty", got)
				}
				return makeResponse(http.StatusOK, `{
				  "events": [
				    {"id":"event-final","runId":"run-paged","sequence":"10","eventType":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"assistant","content":"{\"operating_mode_result\":{\"verdict\":\"accepted\"}}"}},
				    {"id":"event-user","runId":"run-paged","sequence":"11","eventType":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"user","content":"ignored"}}
				  ],
				  "hasMore": true
				}`), nil
			case 2:
				if got := req.URL.Query().Get("after_sequence"); got != "11" {
					t.Fatalf("second after_sequence = %q, want 11", got)
				}
				return makeResponse(http.StatusOK, `{
				  "events": [
				    {"id":"event-trailing","runId":"run-paged","sequence":"20","eventType":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"assistant","content":"trailing subagent noise"}}
				  ],
				  "hasMore": false
				}`), nil
			default:
				t.Fatalf("unexpected request %d", requests)
				return nil, nil
			}
		}},
	)
	service := &AgentService{client: client, enabled: true}
	messages, err := service.GetRunMessages(context.Background(), "run-paged")
	if err != nil {
		t.Fatalf("GetRunMessages: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if len(messages) != 2 {
		t.Fatalf("messages = %#v, want two assistant messages", messages)
	}
	if messages[0].EventID != "event-final" || messages[0].Sequence != 10 || messages[1].EventID != "event-trailing" || messages[1].Sequence != 20 {
		t.Fatalf("messages = %#v, want stable identities across pages", messages)
	}
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

func TestReadErrorResponse_IncludesBody(t *testing.T) {
	resp := makeResponse(http.StatusBadRequest, `{"code":"VALIDATION_FIELD","message":"description must be 16384 characters or less"}`)
	err := readErrorResponse(resp)
	if !errors.Is(err, ErrRequestFailed) {
		t.Fatalf("expected ErrRequestFailed, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "status 400") {
		t.Errorf("expected status 400 in error, got %q", msg)
	}
	if !strings.Contains(msg, "VALIDATION_FIELD") {
		t.Errorf("expected response body in error, got %q", msg)
	}
}

func TestReadErrorResponse_EmptyBody(t *testing.T) {
	resp := makeResponse(http.StatusInternalServerError, "")
	err := readErrorResponse(resp)
	if !errors.Is(err, ErrRequestFailed) {
		t.Fatalf("expected ErrRequestFailed, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "status 500") {
		t.Errorf("expected status 500 in error, got %q", msg)
	}
	// Should not have trailing colon or body text
	if strings.HasSuffix(msg, ": ") {
		t.Errorf("should not have trailing colon for empty body: %q", msg)
	}
}

// buildTestTask creates a proto Task for testing.
func buildTestTask(title string) *domainpb.Task {
	return &domainpb.Task{
		Title: title,
	}
}
