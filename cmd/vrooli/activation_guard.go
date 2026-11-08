package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

// This guard precedes session registration, shim repair and normal command
// initialization. Version probes are handled earlier and remain read-only.
func guardRootActivationEntrypoint(nativeSupervisor bool) error {
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	var store *coreactivation.Store
	if buildinfo.RuntimeHomeSnapshotBase64 == "" {
		// Source-built development commands remain subject to an admitted
		// operation's fence. They use the source owner rather than guessing a
		// runtime layout, and cannot stand in for either retained native image.
		store, err = coreactivation.NewStore(home)
	} else {
		state, resolveErr := buildinfo.CompiledRuntimeHomeEntryPath(home, repocontract.HomeKeyState)
		if resolveErr != nil {
			return resolveErr
		}
		store, err = coreactivation.NewStoreAt(home, filepath.Join(state, "core-activation"))
	}
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.GuardCurrentEntrypoint(ctx, buildinfo.Fingerprint, runtimeapp.NativePersistentStateContract(), nativeSupervisor); err != nil {
		return fmt.Errorf("root update initialization fence: %w", err)
	}
	return nil
}
