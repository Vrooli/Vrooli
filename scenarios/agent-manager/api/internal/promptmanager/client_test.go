package promptmanager

import (
	"context"
	"encoding/json"
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

func TestNewHTTPClient_Defaults(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("expected client")
	}
	if client.baseURLResolver == nil {
		t.Fatal("expected baseURLResolver")
	}
	if client.httpClient == nil {
		t.Fatal("expected httpClient")
	}
}

func TestNewHTTPClientWithResolver_Defaults(t *testing.T) {
	client := NewHTTPClientWithResolver(nil, nil)
	if client.baseURLResolver == nil {
		t.Fatal("expected default resolver")
	}
	if client.httpClient == nil {
		t.Fatal("expected default http client")
	}
}

func TestHTTPClient_ReadSkill(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		&mockHTTPDoer{doFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", req.Method)
			}
			if req.URL.Path != "/api/v1/skills/read" {
				t.Errorf("expected /api/v1/skills/read, got %s", req.URL.Path)
			}

			var rr readRequest
			if err := json.NewDecoder(req.Body).Decode(&rr); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(rr.Identifiers) != 1 || rr.Identifiers[0] != "test-skill" {
				t.Errorf("expected identifier test-skill, got %v", rr.Identifiers)
			}
			if rr.Variables["ITEM_NAME"] != "my-item" {
				t.Errorf("expected variable ITEM_NAME=my-item, got %s", rr.Variables["ITEM_NAME"])
			}
			if rr.Output != "combined" {
				t.Errorf("expected output combined, got %s", rr.Output)
			}

			resp := readResponse{Combined: "rendered prompt content"}
			body, _ := json.Marshal(resp)
			return makeResponse(http.StatusOK, string(body)), nil
		}},
	)

	result, err := client.ReadSkill(context.Background(), "test-skill", map[string]string{"ITEM_NAME": "my-item"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "rendered prompt content" {
		t.Errorf("expected 'rendered prompt content', got %q", result)
	}
}

func TestHTTPClient_ReadSkill_ServerError(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) { return "http://localhost:12345", nil },
		&mockHTTPDoer{doFunc: func(_ *http.Request) (*http.Response, error) {
			return makeResponse(http.StatusInternalServerError, "internal error"), nil
		}},
	)

	_, err := client.ReadSkill(context.Background(), "test-skill", nil, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected status 500 in error, got %v", err)
	}
}

func TestHTTPClient_ReadSkill_ResolverError(t *testing.T) {
	client := NewHTTPClientWithResolver(
		func(_ context.Context) (string, error) {
			return "", context.DeadlineExceeded
		},
		nil,
	)

	_, err := client.ReadSkill(context.Background(), "test-skill", nil, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resolve URL") {
		t.Errorf("expected resolve URL in error, got %v", err)
	}
}
