package buildinfo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBinaryInstallLockRefusesAliasesWithoutChangingForeignFile(t *testing.T) {
	for _, hard := range []bool{false, true} {
		t.Run(fmt.Sprint(hard), func(t *testing.T) {
			dir := t.TempDir()
			foreign := filepath.Join(dir, "foreign")
			binary := filepath.Join(dir, "vrooli")
			if err := os.WriteFile(foreign, []byte("foreign"), 0o600); err != nil {
				t.Fatal(err)
			}
			var err error
			if hard {
				err = os.Link(foreign, binary+".lock")
			} else {
				err = os.Symlink(foreign, binary+".lock")
			}
			if err != nil {
				t.Fatal(err)
			}
			if release, err := AcquireBinaryInstallLock(binary); err == nil {
				release()
				t.Fatal("root installer accepted aliased lock")
			}
			info, err := os.Stat(foreign)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(foreign)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o600 || string(data) != "foreign" {
				t.Fatal("root install lock mutated foreign file")
			}
		})
	}
}

func TestBinaryInstallLockWaitHonorsConvergenceContext(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "vrooli")
	release, err := AcquireBinaryInstallLock(binary)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if other, err := AcquireBinaryInstallLockContext(ctx, binary); !errors.Is(err, context.DeadlineExceeded) {
		if other != nil {
			other()
		}
		t.Fatalf("bounded convergence lock wait: %v", err)
	}
	release()
	other, err := AcquireBinaryInstallLockContext(context.Background(), binary)
	if err != nil {
		t.Fatal(err)
	}
	other()
}
