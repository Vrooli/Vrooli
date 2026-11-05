package apihttp_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/projectmeta"
)

func writeMode(t *testing.T, mode string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{"mode":"` + mode + `"}`
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "service.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return dir
}

// observingHandler records whether IsTestMode was set on each request.
func observingHandler(observed *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*observed = database.IsTestMode(r.Context())
		w.WriteHeader(http.StatusOK)
	})
}

func TestTestModeMiddleware_DevelopmentInjectsCtx(t *testing.T) {
	projectmeta.SetStartDirForTesting(writeMode(t, "development"))
	t.Setenv(apihttp.TestModeForceEnableEnv, "")

	var observed bool
	h := apihttp.TestModeMiddleware(observingHandler(&observed))

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(apihttp.TestModeHeader, "1")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if !observed {
		t.Fatalf("expected IsTestMode true, got false")
	}

	// Header missing → no test-mode.
	observed = false
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))
	if observed {
		t.Fatalf("expected IsTestMode false when header missing, got true")
	}
}

func TestTestModeMiddleware_DevelopmentRejectsNonOneValue(t *testing.T) {
	projectmeta.SetStartDirForTesting(writeMode(t, "development"))
	t.Setenv(apihttp.TestModeForceEnableEnv, "")

	var observed bool
	h := apihttp.TestModeMiddleware(observingHandler(&observed))

	for _, val := range []string{"true", "yes", "0", ""} {
		observed = false
		req := httptest.NewRequest("GET", "/x", nil)
		if val != "" {
			req.Header.Set(apihttp.TestModeHeader, val)
		}
		h.ServeHTTP(httptest.NewRecorder(), req)
		if observed {
			t.Fatalf("value %q: expected IsTestMode false", val)
		}
	}
}

func TestTestModeMiddleware_ProductionNoOp(t *testing.T) {
	projectmeta.SetStartDirForTesting(writeMode(t, "production"))
	t.Setenv(apihttp.TestModeForceEnableEnv, "")

	var observed bool
	h := apihttp.TestModeMiddleware(observingHandler(&observed))

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(apihttp.TestModeHeader, "1")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if observed {
		t.Fatalf("production mode: expected IsTestMode false, got true")
	}
}

func TestTestModeMiddleware_ForceEnableOverridesProduction(t *testing.T) {
	projectmeta.SetStartDirForTesting(writeMode(t, "production"))
	t.Setenv(apihttp.TestModeForceEnableEnv, "1")

	var observed bool
	h := apihttp.TestModeMiddleware(observingHandler(&observed))

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(apihttp.TestModeHeader, "1")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if !observed {
		t.Fatalf("force-enable: expected IsTestMode true, got false")
	}
}
