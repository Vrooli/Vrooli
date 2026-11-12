package runtimeapp

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/vrooli/repo-contract-go/cliinvoke"

	deployment "github.com/vrooli/vrooli/packages/resource-deployment"

	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
)

// LightingActions composes fixed setup and runtime operations. Acquisition and
// source adoption remain resource-owner operations invoked only by Setup.
type LightingActions struct {
	// NativePrepare belongs only to the explicit machine-publication operation.
	NativePrepare func(context.Context) error
	// NativePublishArtifact seals the explicitly selected qualified update bytes
	// while preserving the existing service, device scope and lighting intent.
	NativePublishArtifact func(context.Context) error
	// OperatorAuthority is checked before ordinary native lifecycle effects.
	OperatorAuthority        func() error
	ResourceStorageAuthority func(context.Context) error
	Control                  *LightingControl
	Resources                *resources.Controller
	// AcquireManaged must admit exact dependency policy through the SDA owner before staging.
	AcquireManaged func(context.Context, string) (resources.ManagedServiceArtifact, error)
}

func (a LightingActions) Status(ctx context.Context) (*setupv1.LightingRuntimeStatus, error) {
	if err := a.validateResourceStorage(ctx); err != nil {
		return nil, err
	}
	pinned, err := a.pinnedSelection()
	if errors.Is(err, errLightingArtifactUnavailable) {
		intent, readErr := a.Control.controller.Current()
		if readErr != nil {
			return nil, readErr
		}
		return &setupv1.LightingRuntimeStatus{Phase: string(intent.Phase), State: string(LightingStateDegraded), Degraded: true, Message: err.Error(), LaunchBlocker: err.Error()}, nil
	}
	if err != nil {
		return nil, err
	}
	status, err := a.Control.Status(ctx)
	if err != nil {
		return nil, err
	}
	result := &setupv1.LightingRuntimeStatus{
		Phase: string(status.Intent.Phase), State: string(status.State),
		Installed: status.Installed, Running: status.Running, Ready: status.Ready,
		Degraded: status.Degraded, Message: status.Message, ResourceObserved: status.ResourceObserved,
		PinnedSelection: pinned,
	}
	access, err := a.Resources.CurrentOpenRGBDeviceSetupStatus(ctx)
	if err != nil {
		return nil, err
	}
	result.DeviceAccessPlanId = access.PlanID
	result.DeviceAccessPending = access.Pending
	source, found, err := a.Resources.ManagedServiceSource(ctx, openRGBResource)
	if err != nil {
		return nil, err
	}
	if found {
		result.Selection = &setupv1.LightingSelection{
			Source:  setupv1.InstallationSource_INSTALLATION_SOURCE_MANAGED,
			Version: pinned.Version, Sha256: pinned.Sha256, DevicePolicyId: pinned.DevicePolicyId,
		}
		if source.Mode == "host" {
			result.Selection.Source = setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING
			result.Selection.Version, result.Selection.Sha256 = source.Version, source.SHA256
		}
	}
	if prepared := status.Intent.Preparation; prepared != nil && found {
		sourceMatches := prepared.Source == lightingruntime.PreparationManaged && source.Mode == "managed" && prepared.Selection.Version == pinned.Version && prepared.Selection.SHA256 == pinned.Sha256 || prepared.Source == lightingruntime.PreparationExisting && source.Mode == "host" && prepared.Selection.Version == source.Version && prepared.Selection.SHA256 == source.SHA256
		if sourceMatches {
			result.SetupVerified = prepared.Qualified && prepared.Selection.DevicePolicyID == pinned.DevicePolicyId && (access.PolicyID == "" || prepared.Selection.DevicePolicyID == access.PolicyID)
			if status.Intent.Phase == lightingruntime.Disabled {
				result.Message = "Lighting is disabled. The last setup result is retained; enable rechecks the selected installation and device access."
			}
			result.LaunchBlocker = prepared.LaunchBlocker
			result.Selection.Version = prepared.Selection.Version
			result.Selection.Sha256 = prepared.Selection.SHA256
			result.Selection.DevicePolicyId = prepared.Selection.DevicePolicyID
			if prepared.Qualified && !result.SetupVerified {
				result.LaunchBlocker = "The device policy changed after setup. Verify the selected installation and device access again."
			}
		} else {
			result.LaunchBlocker = "The selected installation changed after setup. Run setup again."
		}
	}
	if status.Intent.Selection != nil {
		if result.Selection == nil {
			result.Selection = &setupv1.LightingSelection{}
		}
		result.Selection.Version = status.Intent.Selection.Version
		result.Selection.Sha256 = status.Intent.Selection.SHA256
		result.Selection.DevicePolicyId = status.Intent.Selection.DevicePolicyID
	}
	if access.Pending {
		result.SetupVerified = false
		result.Degraded = true
		result.State = string(LightingStateDegraded)
		result.LaunchBlocker = "Device access changes are incomplete. Recover or roll back the retained permission plan before enabling lighting."
		result.Message = result.LaunchBlocker
	}
	return result, nil
}

var errLightingArtifactUnavailable = errors.New("OpenRGB has no governed release for this target")

func (a LightingActions) pinnedArtifact() (deployment.ServiceArtifact, error) {
	manifest, err := a.Resources.ResourceManifest(openRGBResource)
	if err != nil {
		return deployment.ServiceArtifact{}, err
	}
	if manifest.ManagedService == nil {
		return deployment.ServiceArtifact{}, errors.New("OpenRGB has no governed installation declaration")
	}
	artifact := manifest.ManagedService.Artifact
	if err = artifact.Validate(); err != nil {
		return deployment.ServiceArtifact{}, err
	}
	target, err := deployment.CanonicalPlatform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return deployment.ServiceArtifact{}, err
	}
	if artifact.SHA256 == "" {
		if _, exists := artifact.SHA256ByPlatform[target.String()]; !exists {
			return deployment.ServiceArtifact{}, fmt.Errorf("%w: %s", errLightingArtifactUnavailable, target)
		}
	}
	return artifact.ForCurrentPlatform()
}

func (a LightingActions) pinnedSelection() (*setupv1.LightingSelection, error) {
	artifact, err := a.pinnedArtifact()
	if err != nil {
		return nil, err
	}
	selection := &setupv1.LightingSelection{Source: setupv1.InstallationSource_INSTALLATION_SOURCE_MANAGED, Version: artifact.Version, Sha256: artifact.SHA256}
	if id, err := a.Resources.CurrentOpenRGBDevicePolicyID(); err == nil {
		selection.DevicePolicyId = id
	}
	return selection, nil
}

// Setup stages exactly the selected release or adopts a compatible existing
// executable, verifies metadata and retains the disabled lighting intent.
func (a LightingActions) Setup(ctx context.Context, requested *setupv1.LightingSelection) (*setupv1.LightingRuntimeStatus, error) {
	if err := a.validateResourceStorage(ctx); err != nil {
		return nil, err
	}
	if a.OperatorAuthority != nil {
		if err := a.OperatorAuthority(); err != nil {
			return nil, err
		}
	}
	if err := validateLightingSelection(requested, true); err != nil {
		return nil, err
	}
	var selected lightingruntime.Selection
	var setupVerified bool
	var launchBlocker string
	err := a.Control.controller.PrepareDisabledResult(ctx, func() (*lightingruntime.Preparation, error) {
		pinned, err := a.pinnedSelection()
		if err != nil {
			return nil, err
		}
		selection := lightingruntime.Selection{
			Version: requested.Version, SHA256: strings.ToLower(requested.Sha256), DevicePolicyID: requested.DevicePolicyId,
		}
		if selection.DevicePolicyID == "" {
			selection.DevicePolicyID = pinned.DevicePolicyId
		}
		switch requested.Source {
		case setupv1.InstallationSource_INSTALLATION_SOURCE_MANAGED:
			if pinned.Sha256 == "" {
				return nil, errors.New("governed OpenRGB installation is unavailable on this platform")
			}
			if requested.Version != pinned.Version || selection.SHA256 != pinned.Sha256 {
				return nil, errors.New("selected managed OpenRGB release differs from the exact governed platform artifact")
			}
			if _, err := a.Resources.SetManagedServiceSource(ctx, openRGBResource, "managed"); err != nil {
				return nil, err
			}
			artifact, err := a.Resources.InspectManagedServiceArtifact(ctx, openRGBResource)
			if err != nil {
				if a.AcquireManaged == nil {
					return nil, fmt.Errorf("governed installation is unavailable for the selected OpenRGB release: %w", err)
				}
				artifact, err = a.AcquireManaged(ctx, requested.Version)
			}
			if err != nil {
				return nil, err
			}
			if artifact.Version != selection.Version || artifact.SHA256 != selection.SHA256 {
				return nil, errors.New("OpenRGB artifact changed during setup")
			}
			if _, err := a.Resources.PrepareOpenRGBAppImageStage(ctx, artifact); err != nil {
				return nil, fmt.Errorf("prepare selected OpenRGB runtime: %w", nativeLightingElevationCommand(err, cliinvoke.RuntimeLightingPrepare()))
			}
		case setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING:
			choice, err := a.Resources.SetManagedServiceSource(ctx, openRGBResource, "host")
			if err != nil {
				return nil, err
			}
			if choice.Version != requested.Version || requested.Sha256 != "" && choice.SHA256 != selection.SHA256 {
				return nil, errors.New("compatible existing OpenRGB differs from the selected version or executable digest")
			}
			selection.SHA256 = choice.SHA256
			artifact, err := a.Resources.InspectManagedServiceArtifact(ctx, openRGBResource)
			if err != nil {
				return nil, err
			}
			if _, err := a.Resources.PrepareOpenRGBAppImageStage(ctx, artifact); err != nil {
				return nil, fmt.Errorf("prepare exact existing OpenRGB profile: %w", err)
			}
		}

		selected = selection
		if selection.DevicePolicyID == "" {
			launchBlocker = "Installation metadata is verified. Enroll narrowly scoped device access before enabling lighting."
		} else if err := a.Control.Setup(ctx, selection); err != nil {
			launchBlocker = err.Error()
		} else {
			setupVerified = true
			launchBlocker = "Background launch still requires verified access to the selected devices."
		}
		source := lightingruntime.PreparationManaged
		if requested.Source == setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING {
			source = lightingruntime.PreparationExisting
		}
		if len(launchBlocker) > 1024 {
			launchBlocker = launchBlocker[:1024]
		}
		return &lightingruntime.Preparation{Source: source, Selection: selected, Qualified: setupVerified, LaunchBlocker: launchBlocker}, nil
	})
	if err != nil {
		return nil, err
	}
	result, err := a.Status(ctx)
	if err != nil {
		return nil, err
	}
	result.Installed, result.SetupVerified = true, setupVerified
	result.Selection = &setupv1.LightingSelection{Source: requested.Source, Version: selected.Version, Sha256: selected.SHA256, DevicePolicyId: selected.DevicePolicyID}
	if result.DeviceAccessPending {
		result.SetupVerified = false
		return result, nil
	}
	result.Message = "Selected OpenRGB installation metadata verified."
	if setupVerified {
		result.Message = "Selected OpenRGB installation and machine-local policy metadata verified."
	}
	result.LaunchBlocker = launchBlocker
	return result, nil
}

func (a LightingActions) Enable(ctx context.Context, requested *setupv1.LightingSelection) (*setupv1.LightingRuntimeStatus, error) {
	if err := a.validateResourceStorage(ctx); err != nil {
		return nil, err
	}
	if a.OperatorAuthority != nil {
		if err := a.OperatorAuthority(); err != nil {
			return nil, err
		}
	}
	if err := validateLightingSelection(requested, false); err != nil {
		return nil, err
	}
	if _, err := a.pinnedArtifact(); err != nil {
		return nil, err
	}
	selection := lightingruntime.Selection{Version: requested.Version, SHA256: strings.ToLower(requested.Sha256), DevicePolicyID: requested.DevicePolicyId}
	if err := a.Control.controller.EnablePrepared(ctx, selection, func() error {
		choice, found, err := a.Resources.ManagedServiceSource(ctx, openRGBResource)
		if err != nil {
			return err
		}
		wantSource := "managed"
		if requested.Source == setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING {
			wantSource = "host"
		}
		if !found || choice.Mode != wantSource {
			return errors.New("complete explicit setup for the selected installation before enabling lighting")
		}
		return a.Resources.SetEnabled(openRGBResource, true)
	}); err != nil {
		return nil, err
	}
	if err := a.Control.controller.Reconcile(ctx); err != nil {
		return nil, err
	}
	return a.Status(ctx)
}

func (a LightingActions) Disable(ctx context.Context) (*setupv1.LightingRuntimeStatus, error) {
	if err := a.validateResourceStorage(ctx); err != nil {
		return nil, err
	}
	if a.OperatorAuthority != nil {
		if err := a.OperatorAuthority(); err != nil {
			return nil, err
		}
	}
	if err := a.Control.Disable(ctx); err != nil {
		return nil, err
	}
	return a.Status(ctx)
}

func validateLightingSelection(selection *setupv1.LightingSelection, setup bool) error {
	if selection == nil || selection.Source != setupv1.InstallationSource_INSTALLATION_SOURCE_MANAGED && selection.Source != setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING {
		return errors.New("select the governed managed installation or a compatible existing installation")
	}
	if (strings.TrimSpace(selection.Version) == "" || strings.TrimSpace(selection.Version) != selection.Version) || len(selection.Version) > 64 || strings.ContainsAny(selection.Version, "\x00\r\n") {
		return errors.New("select an exact OpenRGB release")
	}
	if selection.Sha256 != "" || !setup || selection.Source == setupv1.InstallationSource_INSTALLATION_SOURCE_MANAGED {
		digest, err := hex.DecodeString(selection.Sha256)
		if err != nil || len(digest) != 32 {
			return errors.New("select the exact installation SHA-256")
		}
	}
	if !setup && selection.DevicePolicyId == "" || len(selection.DevicePolicyId) > 128 || strings.TrimSpace(selection.DevicePolicyId) != selection.DevicePolicyId || strings.ContainsAny(selection.DevicePolicyId, "\x00\r\n") {
		return errors.New("select an enrolled machine-local device policy")
	}
	return nil
}

func (a LightingActions) validateResourceStorage(ctx context.Context) error {
	if a.ResourceStorageAuthority != nil {
		return a.ResourceStorageAuthority(ctx)
	}
	return nil
}
