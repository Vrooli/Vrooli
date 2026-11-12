//go:build !windows

package platform

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDarwinPublishedLauncherRequiresSealedExactNativeImage(t *testing.T) {
	for _, kind := range []string{"valid", "wrong-digest", "foreign-dylib", "wrong-arch", "writable", "non-executable", "symlink", "foreign-acl", "replacement", "cancel-after-format"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "vrooli")
			image := darwinBrokerMachFixture("/usr/lib/libSystem.B.dylib")
			if kind == "foreign-dylib" {
				image = darwinBrokerMachFixture("/Users/operator/evil.dylib")
			}
			if err := os.WriteFile(path, image, 0555); err != nil {
				t.Fatal(err)
			}
			digest := fmt.Sprintf("%x", sha256.Sum256(image))
			if kind == "wrong-digest" {
				digest = fmt.Sprintf("%x", sha256.Sum256([]byte("different-host")))
			}
			if kind == "writable" {
				if err := os.Chmod(path, 0755); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "non-executable" {
				if err := os.Chmod(path, 0444); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "symlink" {
				other := filepath.Join(root, "real-vrooli")
				if err := os.Rename(path, other); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, path); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			checks := 0
			reader := darwinPublishedLauncherReader{path: func(context.Context, string, bool) error {
				checks++
				if kind == "foreign-acl" {
					return ErrOperationIdentity
				}
				return nil
			}, format: func(f *os.File) error {
				arch := "arm64"
				if kind == "wrong-arch" {
					arch = "amd64"
				}
				if err := validateDarwinBrokerMachO(f, arch); err != nil {
					return err
				}
				if kind == "replacement" {
					next := filepath.Join(root, "next")
					if err := os.WriteFile(next, image, 0555); err != nil {
						return err
					}
					if err := os.Rename(next, path); err != nil {
						return err
					}
				}
				if kind == "cancel-after-format" {
					cancel()
				}
				return nil
			}}
			file, err := reader.open(ctx, path, digest)
			if kind == "valid" {
				if err != nil || file == nil {
					t.Fatal("sealed exact native image refused", err)
				}
				defer file.Close()
				if checks != 2 {
					t.Fatal("missing post-read ancestry/ACL check", checks)
				}
			} else {
				if err == nil || file != nil {
					if file != nil {
						file.Close()
					}
					t.Fatal("invalid published image admitted", kind, err)
				}
				if kind == "cancel-after-format" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			}
		})
	}
}
