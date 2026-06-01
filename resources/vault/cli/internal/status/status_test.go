package status

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestStatusReportsPersistentFileStorage(t *testing.T) {
	h := &Handlers{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Run: func(string, ...string) ([]byte, error) {
			return []byte(`{"initialized":true,"sealed":false,"storage_type":"file","version":"1.17.6"}`), nil
		},
	}
	if err := h.Status(nil); err != nil {
		t.Fatalf("Status: %v", err)
	}
	got := h.Stdout.(*bytes.Buffer).String()
	for _, want := range []string{"Initialized: true", "Sealed: false", "Storage: file", "Persistence safe: true"} {
		if !strings.Contains(got, want) {
			t.Fatalf("status output %q missing %q", got, want)
		}
	}
}

func TestStatusJSON(t *testing.T) {
	var out bytes.Buffer
	h := &Handlers{
		Stdout: &out,
		Stderr: &bytes.Buffer{},
		Run: func(string, ...string) ([]byte, error) {
			return []byte(`{"initialized":true,"sealed":false,"storage_type":"inmem"}`), nil
		},
	}
	if err := h.Status([]string{"--json"}); err != nil {
		t.Fatalf("Status json: %v", err)
	}
	if !strings.Contains(out.String(), `"persistence_safe": false`) {
		t.Fatalf("json output = %q", out.String())
	}
}

func TestStatusSurfacesDockerError(t *testing.T) {
	h := &Handlers{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Run: func(string, ...string) ([]byte, error) {
			return nil, errors.New("docker down")
		},
	}
	if err := h.Status(nil); err == nil {
		t.Fatal("expected docker error")
	}
}
