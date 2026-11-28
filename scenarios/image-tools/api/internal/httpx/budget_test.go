package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSyncBudgetOutlivesTheOrdinaryWriteDeadline pins the regression where a
// handler slower than api-core's write deadline lost its response (a bare EOF at
// the client) although the work succeeded.
func TestSyncBudgetOutlivesTheOrdinaryWriteDeadline(t *testing.T) {
	slow := func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(400 * time.Millisecond)
		_, _ = w.Write([]byte("done"))
	}
	read := func(h http.Handler) (string, error) {
		server := httptest.NewUnstartedServer(h)
		server.Config.WriteTimeout = 100 * time.Millisecond
		server.Start()
		defer server.Close()
		resp, err := http.Get(server.URL)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		return string(body), err
	}
	if body, err := read(http.HandlerFunc(slow)); err == nil && body == "done" {
		t.Errorf("control: a handler slower than the write deadline should lose its response, got %q", body)
	}
	if body, err := read(SyncBudgetFunc(slow)); err != nil || body != "done" {
		t.Errorf("budgeted handler lost its response: body %q, err %v", body, err)
	}
}

// TestEdgesThatWaitOnJobsAreMountedWithTheBudget is the class guard: a REST edge
// whose handler blocks on the job manager (`Jobs.Wait(`) must be mounted through
// SyncBudget in its module, or a slow job loses the response again.
func TestEdgesThatWaitOnJobsAreMountedWithTheBudget(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join("..", "..", "handlers", "*"))
	if err != nil || len(dirs) == 0 {
		t.Fatalf("locate handlers: %v (%d dirs)", err, len(dirs))
	}
	for _, dir := range dirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		waits := false
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") || strings.HasSuffix(file, "connect_handler.go") {
				continue
			}
			src, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(src), "Jobs.Wait(") {
				waits = true
			}
		}
		if !waits {
			continue
		}
		module, err := os.ReadFile(filepath.Join(dir, "module.go"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(module), "SyncBudget") {
			t.Errorf("%s waits on jobs but its module.go does not mount the edge through httpx.SyncBudget", filepath.Base(dir))
		}
	}
}
