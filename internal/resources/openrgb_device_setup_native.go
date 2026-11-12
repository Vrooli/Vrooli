package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func (c *Controller) discoverNativeOpenRGBDeviceChoices(ctx context.Context) (OpenRGBDeviceSetupDiscovery, map[string]hostreqkit.OpenRGBNativeDevice, error) {
	var result OpenRGBDeviceSetupDiscovery
	if err := c.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
		return result, nil, err
	}
	if err := validateOpenRGBNativeSetupStorage(); err != nil {
		return result, nil, err
	}
	nodes, err := hostreqkit.DiscoverNativeOpenRGBDevices(ctx)
	if err != nil {
		return result, nil, err
	}
	selected, loadErr := c.LoadOpenRGBNativeDevicePolicy()
	if loadErr != nil && !os.IsNotExist(loadErr) {
		return result, nil, loadErr
	}
	choices := map[string]hostreqkit.OpenRGBNativeDevice{}
	for _, node := range nodes {
		data, _ := json.Marshal(node.Selector)
		id := fmt.Sprintf("%x", sha256.Sum256(data))
		if _, ok := choices[id]; ok {
			return result, nil, errors.New("native HID metadata has duplicate stable attachments")
		}
		choices[id] = node.Selector
		result.Devices = append(result.Devices, OpenRGBDeviceSetupCandidate{ID: id, Label: node.Selector.Detector, Kind: "hid", Detail: node.Selector.StableID, Selected: slices.Contains(selected.Devices, node.Selector)})
	}
	slices.SortFunc(result.Devices, func(a, b OpenRGBDeviceSetupCandidate) int { return strings.Compare(a.ID, b.ID) })
	result.AppliedPlanID, err = c.CurrentOpenRGBDeviceSetupPlanID(ctx)
	result.Message = "Native HID metadata only. All attached interfaces matched by an enabled detector must be selected. SMBus, RAM and privileged bus drivers are unsupported; no administrator fallback."
	return result, choices, err
}

func nativeOpenRGBSetupPlanID(stored openRGBDeviceSetupStoredPlan) (string, error) {
	if stored.NativePolicy == nil || stored.OwnerAccount == "" || stored.OwnerUID != 0 {
		return "", errors.New("invalid native HID setup owner")
	}
	plan := stored.Plan
	plan.ID = ""
	if !reflect.DeepEqual(plan, hostreqkit.OpenRGBPermissionPlan{}) {
		return "", errors.New("native HID enrollment cannot contain a Linux permission recipe")
	}
	if _, err := stored.NativePolicy.ID(); err != nil {
		return "", err
	}
	if stored.PreviousNativePolicy != nil {
		if _, err := stored.PreviousNativePolicy.ID(); err != nil {
			return "", err
		}
	}
	if (stored.PreviousNativePolicy == nil) != (stored.PreviousNativePlanID == "") {
		return "", errors.New("native HID previous enrollment reference is incomplete")
	}
	if stored.PreviousNativePlanID != "" && !updateDigest(stored.PreviousNativePlanID) {
		return "", errors.New("invalid previous native HID plan identity")
	}
	stored.Plan = hostreqkit.OpenRGBPermissionPlan{}
	data, err := json.Marshal(stored)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func (c *Controller) planNativeOpenRGBDeviceSetup(ctx context.Context, ids []string) (OpenRGBDeviceSetupPlan, error) {
	var result OpenRGBDeviceSetupPlan
	_, choices, err := c.discoverNativeOpenRGBDeviceChoices(ctx)
	if err != nil {
		return result, err
	}
	if len(ids) == 0 || len(ids) > 16 {
		return result, errors.New("select one to sixteen native lighting HID interfaces")
	}
	ids = slices.Clone(ids)
	slices.Sort(ids)
	var devices []hostreqkit.OpenRGBNativeDevice
	for i, id := range ids {
		d, ok := choices[id]
		if !ok || i > 0 && id == ids[i-1] {
			return result, errors.New("native HID selection is stale, duplicate or unknown")
		}
		devices = append(devices, d)
	}
	m, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return result, err
	}
	artifact, err := m.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		return result, err
	}
	operator, err := c.operatorStateService().Load(ctx)
	if err != nil {
		return result, err
	}
	if source := operator.Resources[openRGBResourceName].Source; source != nil && source.Mode == "host" {
		profile, e := openRGBReleaseProfileFor(runtime.GOOS, runtime.GOARCH)
		if e != nil || !profile.acceptsArtifactSHA256(source.SHA256) {
			return result, errors.Join(errors.New("adopted native OpenRGB source requires its exact verified release digest before device enrollment"), e)
		}
		artifact.SHA256 = source.SHA256
	}
	catalog, err := readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), m.ManagedService.Config.ContentFile)
	if err != nil {
		return result, err
	}
	account, err := c.openRGBNativeRuntimeAccount(ctx)
	if err != nil {
		return result, err
	}
	p, err := hostreqkit.PlanNativeOpenRGBDeviceAccessForAccount(ctx, artifact.SHA256, fmt.Sprintf("%x", sha256.Sum256(catalog)), account, devices)
	if err != nil {
		return result, err
	}
	if err = validateNativeOpenRGBPolicyManifest(p, m, c.Root); err != nil {
		return result, err
	}
	_, ownerAccount, err := hostreqkit.NativeOpenRGBIdentity()
	if err != nil {
		return result, err
	}
	stored := openRGBDeviceSetupStoredPlan{OwnerAccount: ownerAccount, NativePolicy: &p, DeviceIDs: ids}
	current, err := c.readOpenRGBDeviceSetupReference(ctx)
	if err != nil {
		return result, err
	}
	if current.Pending {
		return result, errors.New("finish pending native HID enrollment recovery before reviewing another selection")
	}
	stored.PreviousNativePlanID = current.PlanID
	previous, err := c.readOpenRGBDevicePolicyRecord()
	if err != nil {
		return result, err
	}
	if previous != nil {
		if previous.NativePolicy == nil {
			return result, errors.New("Linux enrollment cannot be adopted as native HID policy")
		}
		stored.PreviousNativePolicy = previous.NativePolicy
	}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		return result, err
	}
	stored.StateDir = paths.StateDir
	id, err := nativeOpenRGBSetupPlanID(stored)
	if err != nil {
		return result, err
	}
	stored.Plan.ID = id
	path, err := c.openRGBDeviceSetupPath(id + ".json")
	if err != nil {
		return result, err
	}
	if err = ensureOpenRGBDevicePolicyDirectory(filepath.Dir(path), true); err != nil {
		return result, err
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return result, err
	}
	if err = writeResourceOwnedFileDurable(path, data, 0o600); err != nil {
		return result, err
	}
	changes := []string{"Record the selected USB HID interfaces for this machine and release.", "Verify selected-device access under the actual operator token before every governed backend start. Privileged bus transports remain disabled."}
	return OpenRGBDeviceSetupPlan{PlanID: id, Summary: "Enroll selected native HID interfaces for the actual operator backend", DeviceIDs: ids, Changes: changes}, nil
}

func (c *Controller) executeNativeOpenRGBDeviceSetup(ctx context.Context, id string, rollback bool) (OpenRGBDeviceSetupResult, error) {
	result := OpenRGBDeviceSetupResult{PlanID: id}
	if err := c.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
		return result, err
	}
	if !updateDigest(id) {
		return result, errors.New("invalid native HID enrollment plan identity")
	}
	path, err := c.openRGBDeviceSetupPath(id + ".json")
	if err != nil {
		return result, err
	}
	var stored openRGBDeviceSetupStoredPlan
	if err = readOpenRGBDeviceSetupJSON(path, &stored); err != nil {
		return result, err
	}
	derived, err := nativeOpenRGBSetupPlanID(stored)
	if err != nil || id != derived || id != stored.Plan.ID {
		return result, errors.Join(errors.New("native HID plan differs from its reviewed inputs"), err)
	}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil || paths.StateDir != stored.StateDir {
		return result, errors.Join(errors.New("native HID resource owner path changed; review a fresh plan"), err)
	}
	err = c.withOpenRGBDevicePermissionFence(ctx, func() error {
		machine, e := hostreqkit.NativeOpenRGBMachineIdentity()
		if e != nil {
			return e
		}
		operator, e := c.openRGBNativeServiceAccount(ctx)
		if e != nil {
			return e
		}
		owner := operator.Principal
		if machine != stored.NativePolicy.MachineID || owner != stored.OwnerAccount {
			return errors.New("native HID enrollment belongs to another machine or operator")
		}
		account, e := c.openRGBNativeRuntimeAccount(ctx)
		if e != nil {
			return e
		}
		if account != stored.NativePolicy.Account {
			return errors.New("native HID operator principal changed since review")
		}
		if !rollback {
			if _, e := hostreqkit.ValidateNativeOpenRGBDeviceSelection(ctx, *stored.NativePolicy); e != nil {
				return e
			}
		}
		current, e := c.readOpenRGBDeviceSetupReference(ctx)
		if e != nil {
			return e
		}
		if current.Pending && current.PlanID != id {
			return errors.New("another native HID enrollment requires recovery")
		}
		if rollback && current.PlanID != id {
			return errors.New("stale native HID enrollment cannot roll back its successor")
		}
		if !rollback && current.PlanID != id && current.PlanID != stored.PreviousNativePlanID {
			return errors.New("native HID enrollment changed since review")
		}
		p := stored.NativePolicy
		activeID := id
		if rollback {
			p = stored.PreviousNativePolicy
			activeID = stored.PreviousNativePlanID
		}
		policyID, _ := stored.NativePolicy.ID()
		pending := openRGBDeviceSetupApplied{PlanID: id, PolicyID: policyID, Pending: true}
		currentPath, _ := c.openRGBDeviceSetupPath("applied.json")
		write := func(v openRGBDeviceSetupApplied) error {
			data, e := json.Marshal(v)
			if e != nil {
				return e
			}
			return writeResourceOwnedFileDurable(currentPath, data, 0o600)
		}
		if e = write(pending); e != nil {
			return e
		}
		result.Pending = true
		if p != nil {
			if rollback {
				policyID, e = c.writeOpenRGBNativeDevicePolicy(*p)
			} else {
				policyID, e = c.StoreOpenRGBNativeDevicePolicy(*p)
			}
		} else {
			policyPath, pathErr := openRGBDevicePolicyPath(c.Home)
			if pathErr != nil {
				return pathErr
			}
			p, policyErr := resourcePrivatePathPolicy(policyPath)
			if policyErr != nil {
				return policyErr
			}
			e = config.RemovePrivateFileDurable(policyPath, p)
			policyID = ""
		}
		if e != nil {
			return e
		}
		if e = write(openRGBDeviceSetupApplied{PlanID: activeID, PolicyID: policyID}); e != nil {
			return e
		}
		result.PolicyID = policyID
		result.Pending = false
		result.Applied = !rollback
		result.RolledBack = rollback
		result.Message = "Native HID selection enrolled. Access under the actual operator token remains unverified until its owned launch witness succeeds. No OS permissions or drivers were changed."
		return nil
	})
	return result, err
}
