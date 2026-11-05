//go:build journey

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestJ14_SecondAppAdoptsIt(t *testing.T) {
	evidence := os.Getenv("INTEGRATIONS_JOURNEY_EVIDENCE")
	if evidence == "" {
		t.Fatal("INTEGRATIONS_JOURNEY_EVIDENCE is required")
	}
	dist := filepath.Join(evidence, "gct-ui-dist")
	if _, err := os.Stat(filepath.Join(dist, "index.html")); err != nil {
		t.Fatal(err)
	}
	pythonSite, err := exec.Command("python3", "-c", "import site; print(site.getusersitepackages())").Output()
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VROOLI_HOME", filepath.Join(home, ".vrooli"))
	t.Setenv("API_PORT", "0")
	for _, class := range []string{"config", "data", "cache", "logs", "state"} {
		t.Setenv("VROOLI_"+strings.ToUpper(class)+"_ROOT", filepath.Join(home, class))
	}
	srv, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if srv.periodicCapture != nil {
			srv.periodicCapture.Stop()
		}
		_ = srv.db.Close()
	})
	api := srv.Router()
	files := http.FileServer(http.Dir(dist))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/vrooli.") || r.URL.Path == "/health" {
			api.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}))
	defer server.Close()
	cmd := exec.CommandContext(t.Context(), "python3", "../test/integrations_journey.py", server.URL, evidence)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+strings.TrimSpace(string(pythonSite)))
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Fatal(err)
	}
}
