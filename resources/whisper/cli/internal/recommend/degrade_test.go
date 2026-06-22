package recommend

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pinEnv returns a GetEnv that points the pin file at a temp path so tests never
// touch the real runtime-home state dir.
func pinEnv(t *testing.T) (func(string) string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "whisper-model.pin")
	getEnv := func(k string) string {
		if k == EnvPinFile {
			return path
		}
		return ""
	}
	return getEnv, path
}

func newDegradeHandlers(getEnv func(string) string, exec func(context.Context, string, ...string) (string, error)) (*DegradeHandlers, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return &DegradeHandlers{
		Stdout: out,
		Stderr: &bytes.Buffer{},
		GetEnv: getEnv,
		Exec:   exec,
	}, out
}

func TestDegradeWritesPinAndRecreates(t *testing.T) {
	getEnv, pinPath := pinEnv(t)
	var restarted bool
	exec := func(_ context.Context, name string, args ...string) (string, error) {
		switch {
		case name == "vrooli" && len(args) > 1 && args[1] == "list":
			return `{"claims":[{"owner_id":"whisper","activity_state":"idle"}]}`, nil
		case name == "vrooli" && len(args) > 1 && args[1] == "restart":
			restarted = true
			return "restarted", nil
		}
		return "", nil
	}
	h, _ := newDegradeHandlers(getEnv, exec)

	if err := h.Run([]string{"--to", "medium"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	data, err := os.ReadFile(pinPath)
	if err != nil {
		t.Fatalf("pin not written: %v", err)
	}
	if got, ok := ValidModel(string(data)); !ok || got != ModelMedium {
		t.Errorf("pin = %q, want medium", strings.TrimSpace(string(data)))
	}
	if !restarted {
		t.Error("whisper was not recreated")
	}
}

func TestDegradeRefusesWhileActive(t *testing.T) {
	getEnv, pinPath := pinEnv(t)
	exec := func(_ context.Context, name string, args ...string) (string, error) {
		if name == "vrooli" && len(args) > 1 && args[1] == "list" {
			return `{"claims":[{"owner_id":"whisper","activity_state":"active"}]}`, nil
		}
		t.Errorf("unexpected exec while active: %s %v", name, args)
		return "", nil
	}
	h, _ := newDegradeHandlers(getEnv, exec)

	err := h.Run([]string{"--to", "small"})
	if err == nil || !strings.Contains(err.Error(), "active") {
		t.Fatalf("Run() error = %v, want refusal-while-active", err)
	}
	if _, statErr := os.Stat(pinPath); statErr == nil {
		t.Error("pin must not be written when refusing an active claim")
	}
}

func TestDegradeUpshiftClearsPin(t *testing.T) {
	getEnv, pinPath := pinEnv(t)
	if err := WritePin(getEnv, ModelSmall); err != nil {
		t.Fatal(err)
	}
	exec := func(_ context.Context, name string, args ...string) (string, error) {
		if name == "vrooli" && len(args) > 1 && args[1] == "list" {
			return `{"claims":[{"owner_id":"whisper","activity_state":"idle"}]}`, nil
		}
		return "ok", nil
	}
	h, _ := newDegradeHandlers(getEnv, exec)

	if err := h.Run([]string{"--upshift", "--to", "large-v3"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, statErr := os.Stat(pinPath); statErr == nil {
		t.Error("upshift must clear the pin")
	}
}

func TestDegradeRejectsUnknownModel(t *testing.T) {
	getEnv, _ := pinEnv(t)
	h, _ := newDegradeHandlers(getEnv, func(context.Context, string, ...string) (string, error) { return "", nil })
	if err := h.Run([]string{"--to", "ginormous"}); err == nil {
		t.Fatal("Run() accepted an invalid model")
	}
}

// recommend --env honors a pin written by capacity-degrade.
func TestRecommendEnvHonorsPin(t *testing.T) {
	getEnv, _ := pinEnv(t)
	if err := WritePin(getEnv, ModelSmall); err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	h := &Handlers{
		Probe:  fakeHostProbe{},
		Stdout: out,
		Stderr: &bytes.Buffer{},
		GetEnv: getEnv,
	}
	if err := h.Run([]string{"--env"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "WHISPER_MODEL_SIZE=small") {
		t.Errorf("env output = %q, want pinned WHISPER_MODEL_SIZE=small", out.String())
	}
}
