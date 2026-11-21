package models

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
)

func TestRawProbeCommandsRequireCurrentRightsBeforeInference(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	image := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(image, []byte("not-used-before-authorization"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"OLLAMA_MODEL_POLICY_PATH":         filepath.Join(root, "resources", "ollama", "model-policy.json"),
		"VROOLI_MODEL_RIGHTS_CONTEXT_PATH": filepath.Join(root, "scenarios", "ai-gateway", "config", "model-rights-context.json"),
	}
	var constructors int
	daemon := &fakeDaemon{models: []string{"gemma4:12b"}, shows: map[string]ensure.ShowResponse{
		"gemma4:12b": {Template: "{{ .Messages }}", Capabilities: []string{"completion", "tools", "vision"}},
	}, toolNames: map[string][]string{"gemma4:12b": {"write_file"}}}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	h := &Handlers{
		NewDaemon: func() Daemon { constructors++; return daemon },
		GetEnv:    func(key string) string { return env[key] },
		Stdout:    stdout,
		Stderr:    stderr,
	}

	if err := h.ProbeTools([]string{"--model", "gemma4:12b"}); err == nil || !strings.Contains(err.Error(), "runtime artifact identity binding is missing") {
		t.Fatalf("unbound direct tool probe returned %v", err)
	}
	if constructors != 1 || daemon.chatCalls != 0 {
		t.Fatalf("tool denial identity reads=%d chat=%d", constructors, daemon.chatCalls)
	}
	if err := h.ProbeVision([]string{"--role", "vision.default", "--image", image}); err == nil || !strings.Contains(err.Error(), "current Ollama inventory has no digest") {
		t.Fatalf("unbound role vision probe returned %v", err)
	}
	if constructors != 2 || daemon.chatCalls != 0 || daemon.generateCalls != 0 {
		t.Fatalf("vision denial identity reads=%d chat=%d generate=%d", constructors, daemon.chatCalls, daemon.generateCalls)
	}
	if err := h.Doctor([]string{"--role", "code.local"}); err == nil {
		t.Fatal("doctor reported a rights-unqualified model as successfully probed")
	}
	if daemon.chatCalls != 0 {
		t.Fatalf("doctor rights denial reached chat %d times", daemon.chatCalls)
	}
}

func TestRawModelInferenceCommandsRequireVerifiedLocalDaemon(t *testing.T) {
	var normalDaemonUsed bool
	h := &Handlers{
		NewDaemon: func() Daemon {
			normalDaemonUsed = true
			return &fakeDaemon{}
		},
		NewLoopbackDaemon: func() (Daemon, error) {
			return nil, errors.New("configured endpoint is not loopback")
		},
	}
	if _, err := h.newLocalDaemon(); err == nil || !strings.Contains(err.Error(), "not loopback") {
		t.Fatalf("unverified model daemon was accepted: %v", err)
	}
	if normalDaemonUsed {
		t.Fatal("unverified endpoint fell back to a normal daemon client")
	}
}

func TestRawModelAuthorizerReloadsContextForEachAttempt(t *testing.T) {
	// The real resource authorizer is exercised by gateway package tests; this
	// command-level case pins the injection boundary to model identity per probe.
	var authorizedModel string
	var checks int
	authorize := func(role, model string) error {
		checks++
		if role != "vision.default" || model != "gemma4:12b" {
			t.Errorf("authorization identity role=%q model=%q", role, model)
		}
		if checks == 2 {
			return os.ErrPermission
		}
		return nil
	}
	client := &fakeDaemon{emptyChat: true}
	_, err := ProbeVision(context.Background(), client, "gemma4:12b", []byte("image"), func() error {
		authorizedModel = "gemma4:12b"
		return authorize("vision.default", authorizedModel)
	})
	if err == nil || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("changed rights before fallback were not surfaced: %v", err)
	}
	if checks != 2 || client.chatCalls != 1 || client.generateCalls != 0 {
		t.Fatalf("checks=%d chat=%d generate=%d", checks, client.chatCalls, client.generateCalls)
	}
}
