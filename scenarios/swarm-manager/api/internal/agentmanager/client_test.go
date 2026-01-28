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
