// Package testutil contains helpers shared by API tests.
package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Serve executes a request and returns the recorder for concise handler tests.
func Serve(t testing.TB, handler http.Handler, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}
