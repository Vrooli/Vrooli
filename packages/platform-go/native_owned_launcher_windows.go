//go:build windows

package platform

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/sys/windows"
)

func nativeLauncherPublisherRoot() (string, error) {
	program, e := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if e != nil {
		return "", e
	}
	return filepath.Join(program, "Vrooli", "OpenRGB", "launchers"), nil
}

// Canonical content-addressed publisher identity is checked by the common
// public owner before this native opener. Pin every root-owned ancestor and
// the exact non-reparse image against mutation while checking its own bytes.
// No boot task, running manager, historical ledger or current CLI is consulted.
func openPublishedNativeLauncher(ctx context.Context, path, digest string) (*os.File, error) {
	return openWindowsPublishedLauncherWith(ctx, path, digest, runtime.GOARCH, windowsPublishedLauncherOpenEffects{
		PinParents: func(dir string) (func(), error) {
			held, e := windowsEnsureExistingBrokerParents(dir)
			if e != nil {
				return nil, e
			}
			return func() {
				for _, handle := range held {
					windows.CloseHandle(handle)
				}
			}, nil
		},
		OpenImage: func(path string) (*os.File, error) {
			handle, e := windowsOpenBrokerPath(path, false, windows.GENERIC_READ)
			if e != nil {
				return nil, e
			}
			return os.NewFile(uintptr(handle), path), nil
		},
	})
}

// Independent caller admission and early task bootstrap both select exactly
// the digest-bound proof branch. A failed retained proof never falls back to
// installed-host guessing or historical executable enumeration.
func openWindowsOwnedServiceSpecHost(ctx context.Context, s OperationSpec) (*os.File, error) {
	if s.NativeLauncher != nil {
		return OpenPublishedNativeLauncher(ctx, s.Path, *s.NativeLauncher)
	}
	return openWindowsOwnedServiceHost(s.Path)
}
