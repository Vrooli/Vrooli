package runtimehandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

func TestLightingStatusNormalManifestReturnsUnobservedDisabled(t *testing.T) {
	storagetest.IsolateStorage(t)
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	out := &bytes.Buffer{}
	command := &rootcli.CommandContext{HomeDirFn: func() (string, error) { return home, nil }, ResolveRootFn: func() (string, error) { return root, nil }, Stdout: out, Stderr: &bytes.Buffer{}}
	if err := runLightingManifest(context.Background(), &runtimeapp.Service{}, command, []string{"status", "--json"}); err != nil {
		t.Fatal(err)
	}
	var status struct {
		Phase            string `json:"phase"`
		ResourceObserved bool   `json:"resource_observed"`
	}
	if err := json.Unmarshal(out.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Phase != "disabled" || status.ResourceObserved {
		t.Fatalf("status=%s", out)
	}
}

func TestLightingManifestRejectsCallerCommandAndUnknownSourceBeforeEffects(t *testing.T) {
	storagetest.IsolateStorage(t)
	for _, args := range [][]string{{"exec", "--command", "ignored"}, {"enable", "--source", "shell", "--version", "1.1", "--sha256", "bad", "--device-policy-id", "policy"}} {
		root, err := filepath.Abs("../../..")
		if err != nil {
			t.Fatal(err)
		}
		home := t.TempDir()
		command := &rootcli.CommandContext{HomeDirFn: func() (string, error) { return home, nil }, ResolveRootFn: func() (string, error) { return root, nil }, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
		if err := runLightingManifest(context.Background(), &runtimeapp.Service{}, command, args); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}

func TestLightingPreparePublicManifestHasBoundedTypedRoute(t *testing.T) {
	res, err := rootcli.ResolveArgv([]string{"runtime", "lighting", "prepare", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Command != "runtime" {
		t.Fatalf("wrong prepare owner %+v", res)
	}
	// A payload/path/selection is never part of machine preparation.
	for _, args := range [][]string{{"prepare", "--command", "anything"}, {"prepare", "--source", "managed"}} {
		home := t.TempDir()
		command := &rootcli.CommandContext{HomeDirFn: func() (string, error) { t.Fatal("invalid arguments resolved operator home"); return home, nil }, ResolveRootFn: func() (string, error) { t.Fatal("invalid arguments resolved source"); return "", nil }, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}
		if err := runLightingManifest(context.Background(), &runtimeapp.Service{}, command, args); err == nil {
			t.Fatalf("accepted arbitrary preparation input %v", args)
		}
	}
}
