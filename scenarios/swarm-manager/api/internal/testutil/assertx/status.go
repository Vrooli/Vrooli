package assertx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Status(tb testing.TB, rec *httptest.ResponseRecorder, expected int) {
	tb.Helper()
	if rec.Code != expected {
		tb.Errorf("expected status %d, got %d: %s", expected, rec.Code, rec.Body.String())
	}
}

func StatusOK(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	Status(tb, rec, http.StatusOK)
}

func StatusCreated(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	Status(tb, rec, http.StatusCreated)
}

func StatusNotFound(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	Status(tb, rec, http.StatusNotFound)
}

func StatusBadRequest(tb testing.TB, rec *httptest.ResponseRecorder) {
	tb.Helper()
	Status(tb, rec, http.StatusBadRequest)
}
