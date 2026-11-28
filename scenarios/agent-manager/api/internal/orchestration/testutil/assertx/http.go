// Package assertx provides orchestration-facing HTTP test assertions.
package assertx

import (
	"net/http/httptest"
	"testing"
)

// HTTPStatus fails if recorder.Code does not match want.
func HTTPStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder == nil {
		t.Fatalf("HTTPStatus: recorder is nil, want %d", want)
	}
	if recorder.Code != want {
		t.Errorf("HTTPStatus: got %d, want %d: %s", recorder.Code, want, recorder.Body.String())
	}
}
