package capacity

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestUseGateSharedReadersAndExclusiveMutation(t *testing.T) {
	root := t.TempDir()
	one, two := &UseGate{Root: root}, &UseGate{Root: root}
	first, err := one.Acquire("model")
	if err != nil {
		t.Fatal(err)
	}
	defer first()
	second, err := two.Acquire("model")
	if err != nil {
		t.Fatal(err)
	}
	defer second()
	if _, err := two.AcquireExclusive("model"); !errors.Is(err, ErrAssetInUse) {
		t.Fatalf("mutation while two readers hold leases: %v", err)
	}
	first()
	if _, err := one.AcquireExclusive("model"); !errors.Is(err, ErrAssetInUse) {
		t.Fatalf("first reader release released another reader: %v", err)
	}
	second()
	writer, err := two.AcquireExclusive("model")
	if err != nil {
		t.Fatal(err)
	}
	defer writer()
	if _, err := one.Acquire("model"); !errors.Is(err, ErrAssetInUse) {
		t.Fatalf("read while another gate changes model: %v", err)
	}
	other, err := one.Acquire("other-model")
	if err != nil {
		t.Fatal(err)
	}
	other()
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() { defer wg.Done(); writer() }()
	}
	wg.Wait()
	reader, err := one.Acquire("model")
	if err != nil {
		t.Fatal(err)
	}
	reader()
}

func TestUseGateProcessLease(t *testing.T) {
	if root := os.Getenv("VROOLI_CAPACITY_USE_TEST_ROOT"); root != "" {
		gate := &UseGate{Root: root}
		var release func()
		var err error
		if os.Getenv("VROOLI_CAPACITY_USE_TEST_MODE") == "write" {
			release, err = gate.AcquireExclusive("model")
		} else {
			release, err = gate.Acquire("model")
		}
		if err != nil {
			fmt.Fprintln(os.Stdout, "denied:", err)
			os.Exit(2)
		}
		defer release()
		fmt.Fprintln(os.Stdout, "acquired")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}
	root := t.TempDir()
	gate := &UseGate{Root: root}
	for _, mode := range []string{"read", "write"} {
		t.Run(mode, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestUseGateProcessLease$")
			cmd.Env = append(os.Environ(), "VROOLI_CAPACITY_USE_TEST_ROOT="+root, "VROOLI_CAPACITY_USE_TEST_MODE="+mode)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer stdin.Close()
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = cmd.Process.Kill() }()
			line, err := bufio.NewReader(stdout).ReadString('\n')
			if err != nil || strings.TrimSpace(line) != "acquired" {
				t.Fatalf("holder startup: %q %v", line, err)
			}
			if release, err := gate.AcquireExclusive("model"); !errors.Is(err, ErrAssetInUse) {
				if release != nil {
					release()
				}
				t.Fatalf("another process did not exclude writer: %v", err)
			}
			reader, err := gate.Acquire("model")
			if mode == "read" {
				if err != nil {
					t.Fatal(err)
				}
				reader()
			} else if !errors.Is(err, ErrAssetInUse) {
				if reader != nil {
					reader()
				}
				t.Fatalf("another process's writer did not exclude reader: %v", err)
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			writer, err := gate.AcquireExclusive("model")
			if err != nil {
				t.Fatalf("crashed owner retained its lease: %v", err)
			}
			writer()
		})
	}
}

func TestUseGateRejectsUnknownRootAndAliases(t *testing.T) {
	for _, gate := range []*UseGate{nil, {}} {
		if release, err := gate.Acquire("model"); err == nil {
			release()
			t.Fatal("unscoped artifact lease was accepted")
		}
	}
	root := t.TempDir()
	gate := &UseGate{Root: root}
	release, err := gate.Acquire("model")
	if err != nil {
		t.Fatal(err)
	}
	release()
	files, err := filepath.Glob(filepath.Join(root, ".capacity-use-leases", "*.lock"))
	if err != nil || len(files) != 1 {
		t.Fatalf("lock files: %v %v", files, err)
	}
	if err := os.Remove(files[0]); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(root, "foreign")
	if err := os.WriteFile(foreign, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, files[0]); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if release, err := gate.Acquire("model"); err == nil {
		release()
		t.Fatal("lease followed a substituted symlink")
	}
}
