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
		if _, err := w.Write([]byte(`[{"type":"NarrativeText","text":"hello"}]`)); err != nil {
			t.Fatal(err)
		}
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

func TestProcessWithStrategyForwardsStrategy(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1024); err != nil {
			t.Fatal(err)
		}
		_, fileHeader, err := r.FormFile("files")
		if err != nil {
			t.Fatal(err)
		}
		if got := fileHeader.Header.Get("Content-Type"); got != "application/pdf" {
			t.Errorf("PDF part Content-Type = %q, want application/pdf", got)
		}
		if got := r.FormValue("strategy"); got != "fast" {
			t.Errorf("strategy = %q, want fast", got)
		}
		if _, err := w.Write([]byte(`[{"type":"NarrativeText","text":"invoice 1007"}]`)); err != nil {
			t.Fatal(err)
		}
	}))
	defer s.Close()
	p := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(p, []byte("pdf fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := (Client{BaseURL: s.URL}).ProcessWithStrategy(context.Background(), p, "fast")
	if err != nil || len(values) != 1 || values[0].Text != "invoice 1007" {
		t.Fatalf("ProcessWithStrategy = %+v, %v", values, err)
	}
}

func TestProcessRejectsEmptyResult(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1024); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer s.Close()
	p := filepath.Join(t.TempDir(), "scan.pdf")
	if err := os.WriteFile(p, []byte("synthetic scan"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (Client{BaseURL: s.URL}).ProcessWithStrategy(context.Background(), p, "fast"); err == nil {
		t.Fatal("empty successful response must fail closed")
	}
}

func TestRejectsUnsupportedFormat(t *testing.T) {
	if _, e := (Client{}).Process(context.Background(), "file.exe"); e == nil {
		t.Fatal("expected error")
	}
}

func TestReadinessChecksHealthAndPartition(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthcheck":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/general/v0/general":
			if err := r.ParseMultipartForm(1024); err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(`[{"type":"NarrativeText","text":"probe"}]`)); err != nil {
				t.Fatal(err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()
	if err := (Client{BaseURL: s.URL}).Readiness(context.Background()); err != nil {
		t.Fatalf("Readiness: %v", err)
	}
}
