package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsPublishedLauncherAcceptsExactRetainedCopyWithoutBootActor(t *testing.T) {
	image := windowsLauncherPEFixture("kernel32.dll")
	sum := sha256.Sum256(image)
	digest := hex.EncodeToString(sum[:])
	root := t.TempDir()
	path := filepath.Join(root, digest, "vrooli.exe")
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, image, 0600); e != nil {
		t.Fatal(e)
	}
	proof := NativeLauncherProof{SchemaVersion: NativeLauncherProofVersion, Profile: NativeLauncherProfileOpenRGB, SHA256: digest}
	pinned, opened, released := false, false, false
	native := func(ctx context.Context, path, digest string) (*os.File, error) {
		return openWindowsPublishedLauncherWith(ctx, path, digest, "amd64", windowsPublishedLauncherOpenEffects{
			PinParents: func(dir string) (func(), error) {
				if dir != filepath.Dir(path) {
					t.Fatal(dir)
				}
				pinned = true
				return func() { released = true }, nil
			},
			OpenImage: func(path string) (*os.File, error) {
				if !pinned || released {
					t.Fatal("image opened outside pinned ancestor interval")
				}
				opened = true
				return os.Open(path)
			},
		})
	}
	retained, e := openPublishedNativeLauncherWith(context.Background(), path, proof, "vrooli.exe", func() (string, error) { return root, nil }, native)
	if e != nil {
		t.Fatal(e)
	}
	defer retained.Close()
	if !pinned || !opened || !released {
		t.Fatal("missing native pin/open/release", pinned, opened, released)
	}
	// No installed task/ledger accessor exists at this boundary. Only exact
	// retained publisher proof and sealed image are needed for NEW admission.
	position, e := retained.Seek(0, 1)
	if e != nil || position != 0 {
		t.Fatal(position, e)
	}
}
func TestWindowsPublishedLauncherRejectsTamperedImageAndLoaderBeforeReturn(t *testing.T) {
	valid := windowsLauncherPEFixture("kernel32.dll")
	sum := sha256.Sum256(valid)
	originalDigest := hex.EncodeToString(sum[:])
	for _, tc := range []struct {
		name   string
		image  []byte
		digest string
	}{
		{"same identity different bytes", append([]byte("changed"), valid[7:]...), originalDigest},
		{"native SHA with unqualified DLL", windowsLauncherPEFixture("ambient.dll"), ""},
		{"native SHA with foreign ABI", func() []byte { v := append([]byte(nil), valid...); v[0x84] = 0x64; v[0x85] = 0xaa; return v }(), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			digest := tc.digest
			if digest == "" {
				sum := sha256.Sum256(tc.image)
				digest = hex.EncodeToString(sum[:])
			}
			path := filepath.Join(t.TempDir(), "vrooli.exe")
			if e := os.WriteFile(path, tc.image, 0600); e != nil {
				t.Fatal(e)
			}
			released := false
			var opened *os.File
			result, e := openWindowsPublishedLauncherWith(context.Background(), path, digest, "amd64", windowsPublishedLauncherOpenEffects{
				PinParents: func(string) (func(), error) { return func() { released = true }, nil },
				OpenImage:  func(path string) (*os.File, error) { var e error; opened, e = os.Open(path); return opened, e },
			})
			if e == nil || result != nil {
				t.Fatal("unqualified image returned", e)
			}
			if !released {
				t.Fatal("ancestor handles leaked")
			}
			if _, e = opened.Stat(); !errors.Is(e, os.ErrClosed) {
				t.Fatal("failed image handle leaked", e)
			}
		})
	}
}
func TestWindowsPublishedLauncherRefusesUnsealedParentsAndImageWithoutFallback(t *testing.T) {
	for _, where := range []string{"parents", "image"} {
		opened, released := false, false
		denied := errors.New("native immutable ACL failure")
		result, e := openWindowsPublishedLauncherWith(context.Background(), "/fixture/vrooli.exe", "unused", "amd64", windowsPublishedLauncherOpenEffects{
			PinParents: func(string) (func(), error) {
				if where == "parents" {
					return nil, denied
				}
				return func() { released = true }, nil
			},
			OpenImage: func(string) (*os.File, error) { opened = true; return nil, denied },
		})
		if result != nil || !errors.Is(e, denied) {
			t.Fatal("native ACL failure hidden", e)
		}
		if where == "parents" && opened {
			t.Fatal("opened image below unproven parent")
		}
		if where == "image" && !released {
			t.Fatal("parent handles leaked after denied image")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	_, e := openWindowsPublishedLauncherWith(ctx, "/fixture/vrooli.exe", "unused", "amd64", windowsPublishedLauncherOpenEffects{
		PinParents: func(string) (func(), error) { calls++; return nil, nil }, OpenImage: func(string) (*os.File, error) { calls++; return nil, nil },
	})
	if !errors.Is(e, context.Canceled) || calls != 0 {
		t.Fatal("canceled request reached native path effects", calls, e)
	}
}
