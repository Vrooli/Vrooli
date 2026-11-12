package hostreqkit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeHIDTransportRequiresEveryPawnIOModuleOmitted(t *testing.T) {
	root := t.TempDir()
	proof, err := verifyNativeOpenRGBBlockedModules(t.Context(), root, os.Lstat)
	if err != nil || !openRGBHex(proof, 64) {
		t.Fatal("fixed omitted-module scope refused", err)
	}
	for _, name := range openRGBNativeBlockedWindowsModules {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, name)
			if err := os.WriteFile(path, []byte("privileged module"), 0400); err != nil {
				t.Fatal(err)
			}
			if _, err := verifyNativeOpenRGBBlockedModules(t.Context(), root, os.Lstat); err == nil {
				t.Fatal("existing module admitted even when readonly")
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNativeHIDTransportRejectsDeniedAndUnknownModuleMetadata(t *testing.T) {
	for _, failure := range []error{os.ErrPermission, errors.New("I/O failed")} {
		if _, err := verifyNativeOpenRGBBlockedModules(t.Context(), t.TempDir(), func(string) (os.FileInfo, error) { return nil, failure }); err == nil {
			t.Fatal("nonabsence failure treated as omitted-module proof")
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := verifyNativeOpenRGBBlockedModules(ctx, t.TempDir(), func(string) (os.FileInfo, error) {
		t.Fatal("cancelled verification inspected modules")
		return nil, nil
	}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
