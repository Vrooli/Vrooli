package storage

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestStorageNativePinResolvesWithoutSource(t *testing.T) {
	const key = "VROOLI_STORAGE_NATIVE_PIN_FIXTURE"
	if encoded := os.Getenv(key); encoded != "" {
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if err := repocontract.PinRuntimeHomeSnapshot(data); err != nil {
			t.Fatal(err)
		}
		source, home := t.TempDir(), t.TempDir()
		t.Setenv("VROOLI_SOURCE_ROOT", filepath.Join(source, "absent"))
		t.Setenv("VROOLI_ROOT", filepath.Join(source, "absent"))
		prior, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(source); err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(prior)
		r, err := NewResolver(ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil), UserHomeDir: func() (string, error) { return home, nil }})
		if err != nil {
			t.Fatal(err)
		}
		paths, err := r.Resolve(Options{ScenarioID: "agent-recovery"})
		if err != nil {
			t.Fatalf("compiled native authority still required a checkout: %v", err)
		}
		state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
		if err != nil || paths.StateDir != filepath.Join(state, "vrooli", "agent-recovery") {
			t.Fatalf("batch and single-entry authority disagree: %+v %v", paths, err)
		}
		entries, err := os.ReadDir(home)
		if err != nil || len(entries) != 0 {
			t.Fatal("path resolution initialized runtime state")
		}
		return
	}
	contract, _, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		t.Fatal(err)
	}
	data, err := contract.RuntimeHomeSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStorageNativePinResolvesWithoutSource$")
	cmd.Env = append(os.Environ(), key+"="+base64.StdEncoding.EncodeToString(data))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("source-free native storage subprocess: %v\n%s", err, output)
	}
}
