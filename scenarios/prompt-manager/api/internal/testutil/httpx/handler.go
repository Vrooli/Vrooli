package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func Request(t testing.TB, method, target string, body io.Reader, vars map[string]string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, target, body)
	if len(vars) > 0 {
		req = mux.SetURLVars(req, vars)
	}
	return req
}

func JSONRequest(t testing.TB, method, target string, body any, vars map[string]string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}
	req := Request(t, method, target, &buf, vars)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func Recorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func DecodeJSON[T any](t testing.TB, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.NewDecoder(recorder.Body).Decode(&out); err != nil {
		t.Fatalf("decode response body: %v\nbody: %s", err, recorder.Body.String())
	}
	return out
}
