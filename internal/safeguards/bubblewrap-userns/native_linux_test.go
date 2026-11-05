package bubblewrapuserns

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var (
	liveNativeSandbox = flag.Bool("live-native-codex-sandbox", false, "Run the installed no-model Codex sandbox containment check")
	nativeHelper      = flag.Bool("native-sandbox-helper", false, "Execute the sandbox child assertions")
	nativeMode        = flag.String("native-sandbox-mode", "", "Child sandbox mode")
	nativeWork        = flag.String("native-sandbox-work", "", "Child writable fixture")
	nativeOutside     = flag.String("native-sandbox-outside", "", "Child read-only fixture")
	nativeAddress     = flag.String("native-sandbox-address", "", "Host loopback positive control")
)

// Explicit opt-in: this exercises the installed native sandbox, not a model.
// The fixture is outside /tmp because workspace-write intentionally grants /tmp.
func TestLiveNativeCodexSandbox(t *testing.T) {
	if !*liveNativeSandbox {
		t.Skip("pass -live-native-codex-sandbox for the no-model host containment check")
	}
	userDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.MkdirTemp(userDir, ".vrooli-codex-containment-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(root); err != nil {
			t.Error(err)
		}
	})
	work := filepath.Join(root, "work")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.WriteFile(outside, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(work, "escape")); err != nil {
		t.Fatal(err)
	}
	// An owner policy must override a permissive inherited user configuration.
	// Keep the fixture private; never read or modify the operator's credentials.
	nativeHome := filepath.Join(root, "codex-home")
	if err := os.Mkdir(nativeHome, 0700); err != nil {
		t.Fatal(err)
	}
	seed := fmt.Sprintf("sandbox_mode = \"danger-full-access\"\napproval_policy = \"on-request\"\n[sandbox_workspace_write]\nnetwork_access = true\nwritable_roots = [%q]\n", root)
	if err := os.WriteFile(filepath.Join(nativeHome, "config.toml"), []byte(seed), 0600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	conn, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("host positive network control: %v", err)
	}
	_ = conn.Close()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"workspace-write", "read-only"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "codex", "sandbox",
				"-c", "sandbox_mode=\""+mode+"\"", "-c", "sandbox_workspace_write.network_access=false",
				"-c", "sandbox_workspace_write.writable_roots=[]", "-c", "approval_policy=\"never\"",
				"--", self, "-test.run=^TestNativeSandboxHelper$", "-test.v", "-native-sandbox-helper",
				"-native-sandbox-mode="+mode, "-native-sandbox-work="+work,
				"-native-sandbox-outside="+outside, "-native-sandbox-address="+listener.Addr().String())
			cmd.Dir = work
			cmd.Env = append(os.Environ(), "CODEX_HOME="+nativeHome)
			output, err := cmd.CombinedOutput()
			t.Log(string(output))
			if err != nil {
				t.Fatalf("native %s containment: %v", mode, err)
			}
			bytes, err := os.ReadFile(outside)
			if err != nil || string(bytes) != "preserve" {
				t.Fatalf("outside fixture changed: %q %v", bytes, err)
			}
		})
	}
}

func TestNativeSandboxHelper(t *testing.T) {
	if !*nativeHelper {
		t.Skip("native sandbox child only")
	}
	work, outside := *nativeWork, *nativeOutside
	err := os.WriteFile(filepath.Join(work, *nativeMode), []byte("allowed"), 0600)
	if *nativeMode == "workspace-write" && err != nil {
		t.Fatalf("allowed workspace write: %v", err)
	}
	if *nativeMode == "read-only" && err == nil {
		t.Fatal("read-only workspace write succeeded")
	}
	for _, path := range []string{outside, filepath.Join(work, "escape")} {
		if err := os.WriteFile(path, []byte("forbidden"), 0600); err == nil {
			t.Fatalf("write escaped through %s", path)
		}
	}
	conn, err := net.DialTimeout("tcp4", *nativeAddress, time.Second)
	if err == nil {
		_ = conn.Close()
		t.Fatal("host loopback reachable from no-network sandbox")
	}
	t.Log("workspace policy, outside-file denial, symlink denial and host-loopback denial passed")
}
