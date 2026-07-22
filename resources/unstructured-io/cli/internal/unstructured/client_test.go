package unstructured

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestProcess(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/general/v0/general" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if e := r.ParseMultipartForm(1024); e != nil {
			t.Fatal(e)
		}
		w.Write([]byte(`[{"type":"NarrativeText","text":"hello"}]`))
	}))
	defer s.Close()
	p := filepath.Join(t.TempDir(), "sample.txt")
	if e := os.WriteFile(p, []byte("hello"), 0o600); e != nil {
		t.Fatal(e)
	}
	v, e := (Client{BaseURL: s.URL}).Process(context.Background(), p)
	if e != nil || len(v) != 1 || v[0].Text != "hello" {
		t.Fatalf("Process=%+v,%v", v, e)
	}
}

func TestRejectsUnsupportedFormat(t *testing.T) {
	if _, e := (Client{}).Process(context.Background(), "file.exe"); e == nil {
		t.Fatal("expected error")
	}
}
