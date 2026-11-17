package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	platform "github.com/vrooli/platform-go"
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
	return guardRootActivationForHome(home, nativeSupervisor)
}

// The installed manager guards the actual operator activation namespace.
func guardRootActivationForHome(home string, nativeSupervisor bool) error {
	return guardRootActivationForImage(home, nativeSupervisor, "")
}

// The descriptor reader has already proved the current native machine image.
// Never replace that OS evidence with a generic mutable-path image probe.
func guardRootActivationForNativeBroker(o platform.NativeSystemServiceBrokerOptions) error {
	if len(o.ExecutableSHA256) != 64 {
		return fmt.Errorf("native installed supervisor activation guard requires verified image evidence")
	}
	return guardRootActivationForImage(o.OperatorHome, true, o.ExecutableSHA256)
}

func guardRootActivationForImage(home string, nativeSupervisor bool, imageSHA string) error {
	var err error
	var store *coreactivation.Store
	if buildinfo.RuntimeHomeSnapshotBase64 == "" {
		// Source-built development commands remain subject to an admitted
		// operation's fence. They use the source owner rather than guessing a
		// runtime layout, and cannot stand in for the installed native image.
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
	if err := guardRootActivationStore(ctx, store, imageSHA, nativeSupervisor); err != nil {
		return fmt.Errorf("root update initialization fence: %w", err)
	}
	return nil
}

type rootActivationEntrypointStore interface {
	GuardEntrypoint(context.Context, string, string, string, bool) error
	GuardCurrentEntrypoint(context.Context, string, string, bool) error
}

func guardRootActivationStore(ctx context.Context, store rootActivationEntrypointStore, imageSHA string, native bool) error {
	if imageSHA != "" {
		return store.GuardEntrypoint(ctx, imageSHA, buildinfo.Fingerprint, runtimeapp.NativePersistentStateContract(), native)
	}
	return store.GuardCurrentEntrypoint(ctx, buildinfo.Fingerprint, runtimeapp.NativePersistentStateContract(), native)
}
