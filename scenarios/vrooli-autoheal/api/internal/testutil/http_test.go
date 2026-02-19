package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAssertStatusMatchesExpected(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusAccepted)

	AssertStatus(t, rec, http.StatusAccepted)
}

func TestMustDecodeJSONDecodesBody(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.Body.WriteString(`{"status":"ok","count":3}`)

	type response struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}

	decoded := MustDecodeJSON[response](t, rec)
	if decoded.Status != "ok" || decoded.Count != 3 {
		t.Fatalf("unexpected decoded payload: %+v", decoded)
	}
}
