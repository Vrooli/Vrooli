package buildinfo

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/proto/protogen"
)

// The real owner runs its real Buf pipeline and holds its ordinary lock. Only
// a barrier after successful generation is injected, so the footprint can be
// measured deterministically while the verifier's populated work tree exists.
func TestRootFingerprintDuringRealProtoVerificationHostProbe(t *testing.T) {
	root := os.Getenv("VROOLI_ROOT_VERIFIER_INPUT_PROBE_ROOT")
	output := os.Getenv("VROOLI_ROOT_VERIFIER_INPUT_PROBE_OUTPUT")
	if root == "" || output == "" {
		t.Skip("requires an explicit source root and retained qualification directory")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(output) {
		t.Fatal("qualification paths must be absolute")
	}
	if err := os.MkdirAll(output, 0o700); err != nil {
		t.Fatal(err)
	}
	log, err := os.Create(filepath.Join(output, "real-verifier-tool-output.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	paths := FingerprintTargetsForExecutable("vrooli")
	before, err := ComputeSourceFingerprintForPaths(root, paths...)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	generated, release := make(chan string, 1), make(chan struct{})
	var once sync.Once
	unpause := func() { once.Do(func() { close(release) }) }
	defer unpause()
	config := protogen.DefaultConfig(root)
	config.Logger = log
	config.RunTool = func(ctx context.Context, dir, name string, args ...string) error {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir, cmd.Stdout, cmd.Stderr = dir, log, log
		if err := cmd.Run(); err != nil {
			return err
		}
		if name == "buf" && len(args) > 0 && args[0] == "generate" {
			for index, arg := range args {
				if arg == "--output" && index+1 < len(args) {
					generated <- args[index+1]
					break
				}
			}
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
	owner, err := protogen.New(config)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- owner.Verify(ctx) }()
	var stage string
	select {
	case stage = <-generated:
	case err := <-done:
		t.Fatalf("real verifier exited before its generated tree was observed: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	files := 0
	if err := filepath.WalkDir(stage, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			files++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if files == 0 {
		t.Fatal("real verifier produced no Go files")
	}
	during, err := ComputeSourceFingerprintForPaths(root, paths...)
	if err != nil {
		t.Fatal(err)
	}
	unpause()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("real verifier failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatalf("owner did not remove successful verification work tree: %v", err)
	}
	after, err := ComputeSourceFingerprintForPaths(root, paths...)
	if err != nil {
		t.Fatal(err)
	}
	report := struct {
		Stage              string `json:"stage"`
		GeneratedGoFiles   int    `json:"generated_go_files"`
		Before             string `json:"before"`
		During             string `json:"during"`
		After              string `json:"after"`
		VerifierPassed     bool   `json:"verifier_passed"`
		OwnerCleanupProved bool   `json:"owner_cleanup_proved"`
	}{stage, files, before, during, after, true, true}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "real-verifier-fingerprint-proof.json"), append(encoded, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if before != during || before != after {
		t.Fatalf("verifier work changed source identity: before=%s during=%s after=%s populated_go_files=%d", before, during, after, files)
	}
	t.Logf("real_verifier_passed=true populated_go_files=%d fingerprint_stable_during_generation_and_after_owner_cleanup=%s", files, before)
}
