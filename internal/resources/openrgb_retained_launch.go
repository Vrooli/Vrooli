package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

// A staged release does not implicitly replace the enrolled serving release.
// Explicit rollback can keep using a qualified frozen profile even when the
// checkout now declares a newer version/catalog.
func (a *OpenRGBHeadless) inspectRetainedOpenRGBSelection(ctx context.Context, selection lightingruntime.Selection) (openRGBHeadlessLaunch, bool, error) {
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		return openRGBHeadlessLaunch{}, false, err
	}
	supervisor := newManagedServiceSupervisor(filepath.Join(paths.StateDir, managedServiceStateFile), filepath.Join(paths.LogsDir, "service.log"))
	state, err := supervisor.readState()
	if os.IsNotExist(err) {
		return openRGBHeadlessLaunch{}, false, nil
	}
	if err != nil {
		return openRGBHeadlessLaunch{}, false, err
	}
	if state.ArtifactSHA256 != selection.SHA256 || state.ArtifactVersion != selection.Version {
		return openRGBHeadlessLaunch{}, false, nil
	}
	if state.InitialLaunch == nil && state.UpdateLaunch == nil {
		return openRGBHeadlessLaunch{}, false, nil
	}
	operator, err := a.controller.operatorStateService().Load(ctx)
	if err != nil {
		return openRGBHeadlessLaunch{}, true, err
	}
	choice := operator.Resources[openRGBResourceName]
	if choice.Enabled == nil || !*choice.Enabled || choice.Source == nil {
		return openRGBHeadlessLaunch{}, true, errors.New("OpenRGB requires explicit resource enablement and source selection")
	}
	if choice.Source.SHA256 != "" && choice.Source.SHA256 != state.ArtifactSHA256 {
		return openRGBHeadlessLaunch{}, true, fmt.Errorf("%w: newer adopted OpenRGB source replaced retained selection", ErrManagedUpdateConflict)
	}
	contract, _, err := openRGBOriginalLaunch(state)
	if err != nil {
		return openRGBHeadlessLaunch{}, true, err
	}
	role, err := a.controller.openRGBFrozenOriginalRole(ctx, state)
	if err != nil {
		return openRGBHeadlessLaunch{}, true, err
	}
	id, _, catalog, err := role.policyIdentity()
	if err != nil {
		return openRGBHeadlessLaunch{}, true, errors.Join(errors.New("retained OpenRGB policy is invalid"), err)
	}
	selected, err := a.controller.readOpenRGBDevicePolicyRecord()
	if err != nil || selected == nil {
		return openRGBHeadlessLaunch{}, true, errors.Join(errors.New("retained OpenRGB policy is no longer selected"), err)
	}
	selectedID, err := selected.ID()
	if err != nil {
		return openRGBHeadlessLaunch{}, true, err
	}
	if id != selection.DevicePolicyID {
		// A completed stop does not bind future explicit enrollment to its
		// obsolete grant. Only the currently selected policy and durable
		// enabled intent can choose fresh qualification; old records remain
		// available for recovery and never authorize another execution.
		if state.StoppedAt != nil && !state.StoppedAt.IsZero() && selectedID == selection.DevicePolicyID {
			owner, e := lightingruntime.NewController(a.controller.Home, lightingruntime.Owners{})
			if e != nil {
				return openRGBHeadlessLaunch{}, true, e
			}
			current, e := owner.Current()
			if e != nil {
				return openRGBHeadlessLaunch{}, true, e
			}
			if current.Phase == lightingruntime.Enabled && current.Selection != nil && *current.Selection == selection {
				return openRGBHeadlessLaunch{}, false, nil
			}
		}
		return openRGBHeadlessLaunch{}, true, errors.New("retained OpenRGB policy differs from selected setup")
	}
	if selectedID != id {
		return openRGBHeadlessLaunch{}, true, errors.New("newer OpenRGB policy selection replaced retained setup")
	}
	settings := contract
	if state.StoppedAt == nil {
		settings, err = freezeManagedUpdateServingConfig(state, paths.ConfigDir, contract)
		if err != nil {
			return openRGBHeadlessLaunch{}, true, err
		}
	}
	endpoint, ready, err := openRGBHeadlessEndpoint(contract.Manifest)
	if err != nil {
		return openRGBHeadlessLaunch{}, true, err
	}
	check := OpenRGBDevicePolicyCheck{Artifact: contract.Artifact, PolicyID: id, DetectorSHA256: catalog, EffectiveUID: os.Geteuid()}
	if role.NativePolicy != nil {
		check.Account = role.NativePolicy.Account
		check.EffectiveUID = 0
	}
	return openRGBHeadlessLaunch{manifest: contract.Manifest, endpoint: endpoint, ready: ready, policy: check, retained: &role, settings: settings.ConfigBytes}, true, nil
}
