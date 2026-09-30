package httpx

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestDoerFunc(t *testing.T) {
	called := false
	doer := DoerFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		if req.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", req.Method)
		}
		return Response(http.StatusCreated, "ok"), nil
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}
	resp, err := doer.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if !called {
		t.Fatal("expected doer function to be called")
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestResponse(t *testing.T) {
	resp := Response(http.StatusAccepted, "accepted")
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "accepted" {
		t.Fatalf("body = %q, want accepted", string(body))
	}
}
