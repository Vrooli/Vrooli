package runtimeapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
)

// PrepareNative is the explicit public machine preparation operation. It
// provisions no backend process or device scan and leaves enablement unchanged.
func (a *LightingActions) PrepareNative(ctx context.Context) error {
	if a != nil {
		if err := a.validateResourceStorage(ctx); err != nil {
			return err
		}
	}
	if a == nil || a.Control == nil || a.Resources == nil || a.NativePrepare == nil {
		return fmt.Errorf("native lighting preparation: %w", platform.ErrUnsupported)
	}
	if err := a.Control.controller.WithResourceUpdate(ctx, func(intent lightingruntime.Intent) (*lightingruntime.Selection, error) {
		if intent.Phase == lightingruntime.Enabled {
			if a.NativePublishArtifact == nil {
				return nil, fmt.Errorf("native artifact publication: %w", platform.ErrUnsupported)
			}
			return nil, a.NativePublishArtifact(ctx)
		}
		if intent.Phase != lightingruntime.Disabled {
			return nil, errors.New("finish lighting disable before native preparation")
		}
		return nil, a.NativePrepare(ctx)
	}); err != nil {
		return nativeLightingElevationCommand(err, cliinvoke.RuntimeLightingPrepare())
	}
	return nil
}

func prepareNativeLightingMachine(ctx context.Context, c *resources.Controller, home, root string) error {
	if err := validateNativeLightingPublisher(ctx, c); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	var acquired resources.ManagedServiceArtifact
	return prepareNativeLightingMachineWith(ctx, nativeLightingPreparationEffects{inspect: func(ctx context.Context) error {
		var inspectErr error
		acquired, inspectErr = c.InspectManagedServiceArtifact(ctx, openRGBResource)
		if inspectErr != nil {
			return fmt.Errorf("acquire the pinned OpenRGB software through ordinary operator lighting setup before elevated preparation: %w", inspectErr)
		}
		return nil
	}, install: func(ctx context.Context) (runtimesupervisor.ServiceInstallResult, error) {
		return runtimesupervisor.InstallService(ctx, runtimesupervisor.ServiceInstallOptions{HomeDir: home, Executable: exe, SourceRoot: root, User: false})
	}, device: c.PrepareOpenRGBNativeDeviceSetup, artifact: func(ctx context.Context) error {
		_, err := c.PrepareOpenRGBAppImageStage(ctx, acquired)
		return err
	}, launcher: func(ctx context.Context, path string) error { return prepareNativeLightingLauncher(ctx, c, path) }})
}

type nativeLightingPreparationEffects struct {
	inspect  func(context.Context) error
	install  func(context.Context) (runtimesupervisor.ServiceInstallResult, error)
	device   func(context.Context) error
	artifact func(context.Context) error
	launcher func(context.Context, string) error
}

func prepareNativeLightingMachineWith(ctx context.Context, e nativeLightingPreparationEffects) error {
	if err := e.inspect(ctx); err != nil {
		return err
	}
	installed, err := e.install(ctx)
	if err != nil {
		return err
	}
	if installed.Executable == "" || !installed.ExecutableIsCanonical {
		return errors.New("native preparation requires the final sealed installed service host")
	}
	if err = e.device(ctx); err != nil {
		return err
	}
	if err = e.artifact(ctx); err != nil {
		return err
	}
	return e.launcher(ctx, installed.Executable)
}

func nativeLightingElevationCommand(err error, args []string) error {
	return nativeLightingElevationCommandFor(runtime.GOOS, os.Getenv("VROOLI_RESOURCE_STORAGE_ROOT"), err, args)
}

// Transfer only the selected namespace scalar. Sudo/UAC may remove ambient
// environment; the supported command must not silently enroll a different root.
func nativeLightingElevationCommandFor(goos, storageRoot string, err error, args []string) error {
	if !errors.Is(err, hostreqkit.ErrElevationRequired) {
		return err
	}
	if storageRoot != "" && (!filepath.IsAbs(storageRoot) || filepath.Clean(storageRoot) != storageRoot || strings.TrimSpace(storageRoot) != storageRoot || strings.ContainsAny(storageRoot, "\x00\r\n")) {
		return fmt.Errorf("native lighting elevation requires a canonical global resource root: %w", err)
	}
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
	if goos == "windows" {
		quote = func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
	}
	var parts []string
	for _, arg := range append([]string{"vrooli"}, args...) {
		parts = append(parts, quote(arg))
	}
	command := strings.Join(parts, " ")
	if goos == "darwin" {
		return fmt.Errorf("%w: run sudo %s %s", err, quote("VROOLI_RESOURCE_STORAGE_ROOT="+storageRoot), command)
	}
	preserved := "$vrooliLightingStorageRoot"
	command = preserved + " = [Environment]::GetEnvironmentVariable('VROOLI_RESOURCE_STORAGE_ROOT', 'Process'); try { [Environment]::SetEnvironmentVariable('VROOLI_RESOURCE_STORAGE_ROOT', " + quote(storageRoot) + ", 'Process'); & " + command + " } finally { [Environment]::SetEnvironmentVariable('VROOLI_RESOURCE_STORAGE_ROOT', " + preserved + ", 'Process') }"
	return fmt.Errorf("%w: run the following from the same operator's elevated PowerShell prompt: %s", err, command)
}

// Both explicit setup publication and serving-target publication require the
// same governed platform/image and setup authority before any machine effects.
func validateNativeLightingPublisher(ctx context.Context, c *resources.Controller) error {
	if ctx == nil || c == nil {
		return errors.New("native publication requires a resource controller and context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := resources.ValidateOpenRGBNativeHostProfile(ctx); err != nil {
		return err
	}
	manifest, err := c.ResourceManifest(openRGBResource)
	if err != nil {
		return err
	}
	if manifest.ManagedService == nil {
		return errors.New("OpenRGB has no governed native artifact")
	}
	if _, err = manifest.ManagedService.Artifact.ForCurrentPlatform(); err != nil {
		return err
	}
	security, err := platform.ProcessSecurity(os.Getpid())
	if err != nil {
		return err
	}
	if !security.Elevated {
		return hostreqkit.ErrElevationRequired
	}
	// A native machine image must carry structural authority before installation;
	// source-built scratch CLIs cannot masquerade as a supported boot artifact.
	if _, err = buildinfo.RuntimeHomeSnapshotDigest(); err != nil {
		return fmt.Errorf("install a governed stamped CLI before native lighting preparation: %w", err)
	}
	return nil
}
