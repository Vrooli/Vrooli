package agentmanager

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
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

func TestHTTPClient_CreateResearchRun_Success(t *testing.T) {
	resolver := func(_ context.Context) (string, error) {
		return "http://localhost:12345", nil
	}

	mockHTTP := &mockHTTPDoer{
		doFunc: func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/v1/tasks":
				return makeResponse(http.StatusCreated, `{"task":{"id":"task-123"}}`), nil
			case "/api/v1/runs":
				return makeResponse(http.StatusCreated, `{"run":{"id":"run-456"}}`), nil
			default:
				return makeResponse(http.StatusNotFound, `{}`), nil
			}
		},
	}

	client := NewHTTPClientWithResolver(resolver, mockHTTP)

	resp, err := client.CreateResearchRun(context.Background(), ResearchRequest{
		Title:       "Research Idea",
		Description: "Test",
		ScopePath:   "/tmp",
		ProjectRoot: ".",
		Prompt:      "Focus on feasibility",
		Tag:         "swarm-manager:test",
		CreatedBy:   "swarm-manager",
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if resp.TaskID != "task-123" {
		t.Errorf("TaskID = %q, want %q", resp.TaskID, "task-123")
	}
	if resp.RunID != "run-456" {
		t.Errorf("RunID = %q, want %q", resp.RunID, "run-456")
	}
	if resp.BaseURL != "http://localhost:12345" {
		t.Errorf("BaseURL = %q, want %q", resp.BaseURL, "http://localhost:12345")
	}
	if resp.CreatedAt == "" {
		t.Error("CreatedAt should not be empty")
	}
}

func TestHTTPClient_CreateResearchRun_ResolverError(t *testing.T) {
	resolver := func(_ context.Context) (string, error) {
		return "", ErrNotAvailable
	}

	client := NewHTTPClientWithResolver(resolver, &mockHTTPDoer{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return makeResponse(http.StatusOK, `{}`), nil
		},
	})

	_, err := client.CreateResearchRun(context.Background(), ResearchRequest{
		Title:     "Research",
		ScopePath: "/tmp",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("expected ErrNotAvailable, got %v", err)
	}
}

func TestHTTPClient_CreateResearchRun_RequestError(t *testing.T) {
	resolver := func(_ context.Context) (string, error) {
		return "http://localhost:12345", nil
	}

	client := NewHTTPClientWithResolver(resolver, &mockHTTPDoer{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		},
	})

	_, err := client.CreateResearchRun(context.Background(), ResearchRequest{
		Title:     "Research",
		ScopePath: "/tmp",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("expected ErrNotAvailable, got %v", err)
	}
}

func TestHTTPClient_CreateTaskErrors(t *testing.T) {
	client := NewHTTPClientWithResolver(func(_ context.Context) (string, error) {
		return "http://localhost:12345", nil
	}, &mockHTTPDoer{})

	client.httpClient = &mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusInternalServerError, `{}`), nil
	}}
	if _, err := client.createTask(context.Background(), "http://localhost:12345", ResearchRequest{Title: "t", ScopePath: "/tmp"}); err == nil {
		t.Fatalf("expected error for non-2xx status")
	}

	client.httpClient = &mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusOK, `{"task":{}}`), nil
	}}
	if _, err := client.createTask(context.Background(), "http://localhost:12345", ResearchRequest{Title: "t", ScopePath: "/tmp"}); err == nil {
		t.Fatalf("expected error for missing task id")
	}

	client.httpClient = &mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusOK, `invalid`), nil
	}}
	if _, err := client.createTask(context.Background(), "http://localhost:12345", ResearchRequest{Title: "t", ScopePath: "/tmp"}); err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestHTTPClient_CreateRunErrors(t *testing.T) {
	client := NewHTTPClientWithResolver(func(_ context.Context) (string, error) {
		return "http://localhost:12345", nil
	}, &mockHTTPDoer{})

	client.httpClient = &mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusInternalServerError, `{}`), nil
	}}
	if _, err := client.createRun(context.Background(), "http://localhost:12345", "task-1", ResearchRequest{Prompt: "p"}); err == nil {
		t.Fatalf("expected error for non-2xx status")
	}

	client.httpClient = &mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
		return makeResponse(http.StatusOK, `{"run":{}}`), nil
	}}
	if _, err := client.createRun(context.Background(), "http://localhost:12345", "task-1", ResearchRequest{}); err == nil {
		t.Fatalf("expected error for missing run id")
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

func TestHTTPClient_CreateRunRequestFields(t *testing.T) {
	t.Run("omits empty prompt and tag", func(t *testing.T) {
		var body string
		client := NewHTTPClientWithResolver(func(_ context.Context) (string, error) {
			return "http://localhost:12345", nil
		}, &mockHTTPDoer{
			doFunc: func(req *http.Request) (*http.Response, error) {
				bytes, _ := io.ReadAll(req.Body)
				body = string(bytes)
				return makeResponse(http.StatusOK, `{"run":{"id":"run-1"}}`), nil
			},
		})

		if _, err := client.createRun(context.Background(), "http://localhost:12345", "task-1", ResearchRequest{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(body, `"task_id":"task-1"`) {
			t.Fatalf("expected task_id in body, got %s", body)
		}
		if strings.Contains(body, "prompt") || strings.Contains(body, "tag") {
			t.Fatalf("expected prompt/tag omitted, got %s", body)
		}
	})

	t.Run("includes prompt and tag", func(t *testing.T) {
		var body string
		client := NewHTTPClientWithResolver(func(_ context.Context) (string, error) {
			return "http://localhost:12345", nil
		}, &mockHTTPDoer{
			doFunc: func(req *http.Request) (*http.Response, error) {
				bytes, _ := io.ReadAll(req.Body)
				body = string(bytes)
				return makeResponse(http.StatusOK, `{"run":{"id":"run-2"}}`), nil
			},
		})

		if _, err := client.createRun(context.Background(), "http://localhost:12345", "task-2", ResearchRequest{
			Prompt: "focus",
			Tag:    "demo",
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(body, `"prompt":"focus"`) || !strings.Contains(body, `"tag":"demo"`) {
			t.Fatalf("expected prompt and tag in body, got %s", body)
		}
	})
}

func TestHTTPClient_CreateTaskRequestFields(t *testing.T) {
	var body string
	client := NewHTTPClientWithResolver(func(_ context.Context) (string, error) {
		return "http://localhost:12345", nil
	}, &mockHTTPDoer{
		doFunc: func(req *http.Request) (*http.Response, error) {
			bytes, _ := io.ReadAll(req.Body)
			body = string(bytes)
			return makeResponse(http.StatusOK, `{"task":{"id":"task-1"}}`), nil
		},
	})

	_, err := client.createTask(context.Background(), "http://localhost:12345", ResearchRequest{
		Title:       "  Title  ",
		Description: "  Desc  ",
		ScopePath:   " /tmp ",
		ProjectRoot: " . ",
		CreatedBy:   " swarm ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(body, `"title":"Title"`) {
		t.Fatalf("expected trimmed title, got %s", body)
	}
	if !strings.Contains(body, `"description":"Desc"`) {
		t.Fatalf("expected trimmed description, got %s", body)
	}
	if !strings.Contains(body, `"scope_path":"/tmp"`) {
		t.Fatalf("expected trimmed scope path, got %s", body)
	}
	if !strings.Contains(body, `"project_root":"."`) {
		t.Fatalf("expected trimmed project root, got %s", body)
	}
	if !strings.Contains(body, `"created_by":"swarm"`) {
		t.Fatalf("expected trimmed created_by, got %s", body)
	}
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
