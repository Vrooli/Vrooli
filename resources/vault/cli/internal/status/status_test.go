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

func TestStatusSurfacesManagedRuntimeError(t *testing.T) {
	h := &Handlers{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Run: func(string, ...string) ([]byte, error) {
			return nil, errors.New("service unavailable")
		},
	}
	if err := h.Status(nil); err == nil {
		t.Fatal("expected managed runtime error")
	}
}

func TestStatusUsesNativeVaultCommandByDefault(t *testing.T) {
	var gotName string
	h := &Handlers{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Run: func(name string, args ...string) ([]byte, error) {
			gotName = name
			return []byte(`{"initialized":true,"sealed":false,"storage_type":"file"}`), nil
		},
	}
	if err := h.Status(nil); err != nil {
		t.Fatal(err)
	}
	if gotName != "vault" {
		t.Fatalf("native invocation = %q, want vault", gotName)
	}
}

func TestStatusRejectsDirectRemoteVrooliAccess(t *testing.T) {
	t.Setenv("VROOLI_MANAGED_PROVIDER", "remote-vrooli")
	h := &Handlers{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Run: func(string, ...string) ([]byte, error) {
		t.Fatal("remote mode must not invoke Vault directly")
		return nil, nil
	}}
	if err := h.Status(nil); err == nil || !strings.Contains(err.Error(), "scenario API") {
		t.Fatalf("Status() error = %v, want scenario API boundary", err)
	}
}
