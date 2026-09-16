package matrix

import (
	"context"
	"net/url"
	"testing"
)

type fakeHTTPClient struct {
	method string
	path   string
	body   interface{}
	result []byte
}

func (f *fakeHTTPClient) DoWithContext(_ context.Context, method, path string, _ url.Values, body interface{}) ([]byte, error) {
	f.method, f.path, f.body = method, path, body
	return f.result, nil
}

func TestRequestUsesMatrixHTTPContract(t *testing.T) {
	client := &fakeHTTPClient{result: []byte(`{"run_id":"matrix-1","state":"completed"}`)}
	value, err := request(nil, client, "POST", "/validation/matrices/matrix-1/start", nil)
	if err != nil {
		t.Fatalf("request() error = %v", err)
	}
	if client.method != "POST" || client.path != "/validation/matrices/matrix-1/start" {
		t.Fatalf("request() called %s %s, want POST /validation/matrices/matrix-1/start", client.method, client.path)
	}
	if got := matrixID(value); got != "run_id=matrix-1" {
		t.Fatalf("matrixID() = %q, want run_id=matrix-1", got)
	}
	if got := matrixState(value); got != "completed" {
		t.Fatalf("matrixState() = %q, want completed", got)
	}
}
