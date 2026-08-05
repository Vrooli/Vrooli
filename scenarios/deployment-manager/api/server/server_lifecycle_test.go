package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestNewBuildsConfiguredServer(t *testing.T) {
	t.Setenv("API_PORT", "0")
	t.Setenv("VROOLI_SOURCE_ROOT", filepath.Clean(filepath.Join("..", "..", "..", "..")))
	t.Setenv("SQLITE_PATH", filepath.Join(t.TempDir(), "deployment-manager.db"))
	srv, err := New()
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	if srv.Router == nil || srv.Handler == nil || len(srv.ConnectRoutes) == 0 {
		t.Fatalf("server was not fully wired: %#v", srv)
	}
	if srv.RoutedDB != nil {
		t.Cleanup(func() { _ = srv.RoutedDB.Close() })
	}
}

func TestServerWriteJSONAndFileRoots(t *testing.T) {
	srv := &Server{}
	rec := httptest.NewRecorder()
	srv.WriteJSON(rec, http.StatusCreated, map[string]string{"status": "ok"})
	if rec.Code != http.StatusCreated || rec.Header().Get("Content-Type") != "application/json" || rec.Body.String() == "" {
		t.Fatalf("JSON response = %d %q", rec.Code, rec.Body.String())
	}
	roots, err := newFileRoots()
	if err != nil || roots == nil {
		t.Fatalf("file roots = %#v, %v", roots, err)
	}
}

func TestServerStartStopsOnSignal(t *testing.T) {
	srv := &Server{Config: &Config{Port: "0"}, Handler: http.NewServeMux()}
	done := make(chan error, 1)
	go func() { done <- srv.Start() }()
	time.Sleep(100 * time.Millisecond)
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start() = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop after SIGTERM")
	}
}
