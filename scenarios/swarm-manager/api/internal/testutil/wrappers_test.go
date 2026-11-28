package testutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/apihttptest"
)

func TestStatusCompanionResponses(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{name: "ok", status: http.StatusOK},
		{name: "created", status: http.StatusCreated},
		{name: "not_found", status: http.StatusNotFound},
		{name: "bad_request", status: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rec.WriteHeader(tc.status)
			apihttptest.AssertStatus(t, rec.Result(), tc.status)
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
