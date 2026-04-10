package testutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAssertStatusWrappers(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		assertFn func(*testing.T, *httptest.ResponseRecorder)
	}{
		{name: "ok", status: http.StatusOK, assertFn: AssertStatusOK},
		{name: "created", status: http.StatusCreated, assertFn: AssertStatusCreated},
		{name: "not_found", status: http.StatusNotFound, assertFn: AssertStatusNotFound},
		{name: "bad_request", status: http.StatusBadRequest, assertFn: AssertStatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteHeader(tc.status)
			tc.assertFn(t, rec)
		})
	}
}

func TestAssertFileHelpers(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(existing, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	AssertFileExists(t, existing)
	AssertFileNotExists(t, filepath.Join(dir, "missing.txt"))
}
