package resources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	openRGBDeviceSetupPendingState    = "pending"
	openRGBDeviceSetupAppliedState    = "applied"
	openRGBDeviceSetupRolledBackState = "rolled_back"
)

type OpenRGBDeviceSetupCandidate struct {
	ID, Label, Kind, Detail string
	Selected                bool
}
type OpenRGBDeviceSetupDiscovery struct {
	Devices       []OpenRGBDeviceSetupCandidate
	Message       string
	AppliedPlanID string
}
type OpenRGBDeviceSetupPlan struct {
	PlanID, Summary string
	DeviceIDs       []string
	Changes         []string
	NeedsSudo       bool
}
type OpenRGBDeviceSetupResult struct {
	PlanID, PolicyID, Message      string
	Applied, RolledBack, NeedsSudo bool
	Pending                        bool
}

type openRGBDeviceSetupStoredPlan struct {
	OwnerUID             uint32
	StateDir             string
	DeviceIDs            []string
	Plan                 hostreqkit.OpenRGBPermissionPlan
	OwnerAccount         string                          `json:",omitempty"`
	NativePolicy         *hostreqkit.OpenRGBNativePolicy `json:",omitempty"`
	PreviousNativePolicy *hostreqkit.OpenRGBNativePolicy `json:",omitempty"`
	PreviousNativePlanID string                          `json:",omitempty"`
}

type openRGBDeviceSetupApplied struct {
	PlanID   string
	PolicyID string
	Pending  bool
}

func (c *Controller) openRGBDeviceSetupPath(name string) (string, error) {
	policyPath, err := openRGBDevicePolicyPath(c.Home)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(policyPath), "device-access", name), nil
}

func (c *Controller) discoverOpenRGBDeviceChoices(ctx context.Context) (OpenRGBDeviceSetupDiscovery, map[string][]hostreqkit.OpenRGBLinuxDevice, error) {
	var result OpenRGBDeviceSetupDiscovery
	nodes, err := hostreqkit.DiscoverLinuxOpenRGBDevices(ctx)
	if err != nil {
		return result, nil, err
	}
	groups := map[string][]hostreqkit.OpenRGBLinuxDevice{}
	for _, node := range nodes {
		d := node.Selector
		physical := d.Topology
		if d.Kind == hostreqkit.OpenRGBLinuxDeviceHIDRaw {
			physical = filepath.Dir(physical)
		}
		groups[d.Kind+"\x00"+physical+"\x00"+d.Detector] = append(groups[d.Kind+"\x00"+physical+"\x00"+d.Detector], d)
	}
	selected, _ := c.LoadOpenRGBDevicePolicy()
	choices := map[string][]hostreqkit.OpenRGBLinuxDevice{}
	for _, devices := range groups {
		slices.SortFunc(devices, func(a, b hostreqkit.OpenRGBLinuxDevice) int {
			return strings.Compare(a.Topology+"\x00"+a.Adapter+"\x00"+a.HardwareID, b.Topology+"\x00"+b.Adapter+"\x00"+b.HardwareID)
		})
		encoded, _ := json.Marshal(devices)
		id := fmt.Sprintf("%x", sha256.Sum256(encoded))
		choices[id] = devices
		candidate := OpenRGBDeviceSetupCandidate{ID: id, Label: devices[0].Detector, Kind: devices[0].Kind, Detail: fmt.Sprintf("%s, %d supported routes", devices[0].Topology, len(devices)), Selected: true}
		for _, d := range devices {
			candidate.Selected = candidate.Selected && slices.Contains(selected.Devices, d)
		}
		result.Devices = append(result.Devices, candidate)
	}
	slices.SortFunc(result.Devices, func(a, b OpenRGBDeviceSetupCandidate) int { return strings.Compare(a.Label+a.Detail, b.Label+b.Detail) })
	result.AppliedPlanID, err = c.CurrentOpenRGBDeviceSetupPlanID(ctx)
	if err != nil {
		return result, nil, err
	}
	result.Message = "Kernel metadata discovery only; select physical devices before reviewing persistent access."
	return result, choices, nil
}

func (c *Controller) DiscoverOpenRGBDeviceSetup(ctx context.Context) (OpenRGBDeviceSetupDiscovery, error) {
	if runtime.GOOS != "linux" {
		result, _, err := c.discoverNativeOpenRGBDeviceChoices(ctx)
		return result, err
	}
	result, _, err := c.discoverOpenRGBDeviceChoices(ctx)
	return result, err
}

func (c *Controller) PlanOpenRGBDeviceSetup(ctx context.Context, ids []string) (OpenRGBDeviceSetupPlan, error) {
	if runtime.GOOS != "linux" {
		return c.planNativeOpenRGBDeviceSetup(ctx, ids)
	}
	var result OpenRGBDeviceSetupPlan
	uid, err := c.openRGBDeviceSetupUID()
	if err != nil {
		return result, err
	}
	policyPath, err := openRGBDevicePolicyPath(c.Home)
	if err != nil {
		return result, err
	}
	if err := ensureOpenRGBDevicePolicyDirectory(filepath.Dir(policyPath), true); err != nil {
		return result, err
	}
	_, choices, err := c.discoverOpenRGBDeviceChoices(ctx)
	if err != nil {
		return result, err
	}
	if len(ids) == 0 || len(ids) > 16 {
		return result, errors.New("select at least one discovered physical lighting device")
	}
	ids = slices.Clone(ids)
	slices.Sort(ids)
	var devices []hostreqkit.OpenRGBLinuxDevice
	for i, id := range ids {
		if len(choices[id]) == 0 || i > 0 && ids[i-1] == id {
			return result, errors.New("lighting device selection is stale, duplicate or unknown")
		}
		devices = append(devices, choices[id]...)
	}
	manifest, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return result, err
	}
	artifact, err := manifest.ManagedService.Artifact.ForCurrentPlatform()
	if err != nil {
		return result, err
	}
	catalog, err := readManagedUpdateConfigFile(filepath.Join(c.Root, "resources", openRGBResourceName), manifest.ManagedService.Config.ContentFile)
	if err != nil {
		return result, err
	}
	plan, err := hostreqkit.PlanLinuxOpenRGBDeviceAccess(ctx, hostreqkit.OpenRGBPermissionRequest{ArtifactSHA256: artifact.SHA256, DetectorSHA256: fmt.Sprintf("%x", sha256.Sum256(catalog)), OwnerUID: uid, Devices: devices})
	if err != nil {
		return result, err
	}
	path, err := c.openRGBDeviceSetupPath(plan.ID + ".json")
	if err != nil {
		return result, err
	}
	if err := ensureOpenRGBDevicePolicyDirectory(filepath.Dir(path), true); err != nil {
		return result, err
	}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil {
		return result, err
	}
	data, err := json.Marshal(openRGBDeviceSetupStoredPlan{OwnerUID: uid, StateDir: paths.StateDir, DeviceIDs: ids, Plan: plan})
	if err != nil {
		return result, err
	}
	if err := writeResourceOwnedFileDurable(path, data, 0o600); err != nil {
		return result, err
	}
	return OpenRGBDeviceSetupPlan{PlanID: plan.ID, Summary: "Persistent owner access to the selected physical lighting routes", DeviceIDs: ids, Changes: []string{plan.RuleContent}, NeedsSudo: os.Geteuid() != 0}, nil
}

func (c *Controller) openRGBDeviceSetupUID() (uint32, error) {
	uid := os.Geteuid()
	if uid == 0 {
		var ok bool
		uid, _, ok = hostreqkit.InvokingUserIDs()
		if !ok || uid <= 0 {
			return 0, errors.New("lighting permission execution requires the original invoking operator")
		}
		home, err := hostreqkit.InvokingUserHomeDir()
		if err != nil || home != c.Home {
			return 0, errors.Join(errors.New("lighting permission owner home differs from the invoking operator"), err)
		}
	}
	if uid <= 0 {
		return 0, errors.New("lighting permissions require an unprivileged operator identity")
	}
	return uint32(uid), nil
}

func readOpenRGBDeviceSetupJSON(path string, output any) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > 256<<10 {
		return errors.New("unsafe lighting permission owner record")
	}
	if err := verifyManagedUpdatePrivatePath(path, info); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (256<<10)+1))
	if err != nil {
		return err
	}
	if len(data) > 256<<10 {
		return errors.New("lighting permission owner record exceeds its bound")
	}
	if err := validateManagedUpdateJSON(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	return decoder.Decode(output)
}

func (c *Controller) CurrentOpenRGBDeviceSetupPlanID(ctx context.Context) (string, error) {
	current, err := c.readOpenRGBDeviceSetupReference(ctx)
	return current.PlanID, err
}

func (c *Controller) OpenRGBDeviceSetupPending(ctx context.Context) (bool, error) {
	current, err := c.readOpenRGBDeviceSetupReference(ctx)
	return current.Pending, err
}

func (c *Controller) CurrentOpenRGBDeviceSetupStatus(ctx context.Context) (OpenRGBDeviceSetupResult, error) {
	current, err := c.readOpenRGBDeviceSetupReference(ctx)
	return OpenRGBDeviceSetupResult{PlanID: current.PlanID, PolicyID: current.PolicyID, Pending: current.Pending}, err
}

func (c *Controller) readOpenRGBDeviceSetupReference(ctx context.Context) (openRGBDeviceSetupApplied, error) {
	var current openRGBDeviceSetupApplied
	if err := ctx.Err(); err != nil {
		return current, err
	}
	path, err := c.openRGBDeviceSetupPath("applied.json")
	if err != nil {
		return current, err
	}
	if err := readOpenRGBDeviceSetupJSON(path, &current); os.IsNotExist(err) {
		return current, nil
	} else if err != nil {
		return current, err
	}
	if current.PlanID == "" && current.PolicyID == "" && !current.Pending {
		return current, nil
	}
	if !updateDigest(current.PlanID) || !updateDigest(current.PolicyID) {
		return current, errors.New("invalid applied lighting permission reference")
	}
	return current, nil
}

func (c *Controller) ApplyOpenRGBDeviceSetup(ctx context.Context, planID, sudoMode string) (OpenRGBDeviceSetupResult, error) {
	return c.executeOpenRGBDeviceSetup(ctx, planID, sudoMode, false)
}
func (c *Controller) RollbackOpenRGBDeviceSetup(ctx context.Context, planID, sudoMode string) (OpenRGBDeviceSetupResult, error) {
	return c.executeOpenRGBDeviceSetup(ctx, planID, sudoMode, true)
}

func (c *Controller) executeOpenRGBDeviceSetup(ctx context.Context, id, sudoMode string, rollback bool) (OpenRGBDeviceSetupResult, error) {
	if runtime.GOOS != "linux" {
		return c.executeNativeOpenRGBDeviceSetup(ctx, id, rollback)
	}
	result := OpenRGBDeviceSetupResult{PlanID: id}
	if !updateDigest(id) {
		return result, errors.New("invalid lighting permission plan ID")
	}
	uid, err := c.openRGBDeviceSetupUID()
	if err != nil {
		return result, err
	}
	path, err := c.openRGBDeviceSetupPath(id + ".json")
	if err != nil {
		return result, err
	}
	var stored openRGBDeviceSetupStoredPlan
	if err := readOpenRGBDeviceSetupJSON(path, &stored); err != nil {
		return result, err
	}
	if stored.OwnerUID != uid || stored.Plan.ID != id || stored.Plan.Policy.OwnerUID != uid {
		return result, errors.New("lighting permission plan belongs to a different owner")
	}
	if os.Geteuid() != 0 {
		binary, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{})
		if err != nil {
			return result, err
		}
		args := cliinvoke.RuntimeLightingPermissionApply(id)
		if rollback {
			args = cliinvoke.RuntimeLightingPermissionRollback(id)
		}
		quoted := []string{"'" + strings.ReplaceAll(binary, "'", "'\"'\"'") + "'"}
		for _, arg := range args {
			quoted = append(quoted, "'"+strings.ReplaceAll(arg, "'", "'\"'\"'")+"'")
		}
		result.NeedsSudo, result.Message = true, "Run sudo "+strings.Join(quoted, " ")
		if sudoMode != "ask" {
			return result, nil
		}
		command, elevatedArgs, err := hostreqkit.WithSudo("ask", binary, args)
		if err != nil {
			return result, err
		}
		var output openRGBDeviceSetupOutput
		child := cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: command, Args: elevatedArgs, Stdout: &output})
		return parseOpenRGBDeviceSetupChild(id, output.data, output.overflow, child.Error())
	}
	paths, err := resourceStoragePaths(openRGBResourceName)
	if err != nil || stored.StateDir == "" || stored.StateDir != paths.StateDir {
		return result, errors.Join(errors.New("lighting permission plan resource owner path changed; review a fresh plan"), err)
	}
	var changed OpenRGBDeviceSetupResult
	err = c.withOpenRGBDevicePermissionFence(ctx, func() error {
		var operationErr error
		changed, operationErr = c.applyOpenRGBDeviceSetupOwned(ctx, id, stored, rollback)
		return operationErr
	})
	if changed.PlanID == "" {
		changed.PlanID = id
	}
	return changed, err
}

func (c *Controller) applyOpenRGBDeviceSetupOwned(ctx context.Context, id string, stored openRGBDeviceSetupStoredPlan, rollback bool) (OpenRGBDeviceSetupResult, error) {
	return c.applyOpenRGBDeviceSetupTransaction(ctx, id, stored, func() (hostreqkit.OpenRGBPermissionResult, error) {
		if rollback {
			return hostreqkit.RollbackLinuxOpenRGBDeviceAccess(ctx, id)
		}
		return hostreqkit.ApplyLinuxOpenRGBDeviceAccess(ctx, stored.Plan)
	}, func(policy *hostreqkit.OpenRGBLinuxPolicy) (string, error) {
		if policy != nil {
			return c.storeOpenRGBDevicePolicyRecord(*policy)
		}
		path, err := openRGBDevicePolicyPath(c.Home)
		if err != nil {
			return "", err
		}
		if err = os.Remove(path); os.IsNotExist(err) {
			err = nil
		}
		if err == nil {
			err = confirmOpenRGBPublicationDurability(filepath.Dir(path))
		}
		return "", err
	})
}

func (c *Controller) applyOpenRGBDeviceSetupTransaction(ctx context.Context, id string, stored openRGBDeviceSetupStoredPlan, execute func() (hostreqkit.OpenRGBPermissionResult, error), publish func(*hostreqkit.OpenRGBLinuxPolicy) (string, error)) (OpenRGBDeviceSetupResult, error) {
	result := OpenRGBDeviceSetupResult{PlanID: id}
	policyID, err := stored.Plan.Policy.ID()
	if err != nil {
		return result, err
	}
	currentPath, err := c.openRGBDeviceSetupPath("applied.json")
	if err != nil {
		return result, err
	}
	if err := ensureOpenRGBDevicePolicyDirectory(filepath.Dir(currentPath), true); err != nil {
		return result, err
	}
	previous, err := c.readOpenRGBDeviceSetupReference(ctx)
	if err != nil {
		return result, err
	}
	if previous.Pending && previous.PlanID != id {
		return result, fmt.Errorf("finish recovery of lighting permission plan %s before applying another plan", previous.PlanID)
	}
	writeReference := func(reference openRGBDeviceSetupApplied) error {
		data, err := json.Marshal(reference)
		if err != nil {
			return err
		}
		return writeResourceOwnedFileDurable(currentPath, data, 0o600)
	}
	// Record recovery intent before the privileged owner can change a node.
	// A crash at any later point keeps start admission closed until retry.
	if err := writeReference(openRGBDeviceSetupApplied{PlanID: id, PolicyID: policyID, Pending: true}); err != nil {
		return result, err
	}
	host, err := execute()
	result.PolicyID = policyID
	result.Applied, result.RolledBack, result.Pending = host.Applied, host.RolledBack, host.Pending || err != nil && (host.Applied || host.RolledBack)
	if err != nil && !host.Pending && !host.Applied && !host.RolledBack {
		if restoreErr := writeReference(previous); restoreErr != nil {
			result.Pending = true
			return result, errors.Join(err, restoreErr)
		}
	}
	if err != nil {
		result.Message = err.Error()
		return result, err
	}
	if host.Applied || host.RolledBack {
		if host.Pending || host.Applied && host.RolledBack || host.PlanID != id || (host.ActivePolicy == nil) != (host.ActivePlanID == "") || host.ActivePlanID != "" && !updateDigest(host.ActivePlanID) || host.Applied && host.ActivePlanID != id {
			result.Pending = true
			return result, errors.New("privileged permission completion has an inconsistent active enrollment")
		}
		result.PolicyID, err = publish(host.ActivePolicy)
		if err != nil {
			result.Pending = true
			return result, err
		}
		data, err := json.Marshal(openRGBDeviceSetupApplied{PlanID: host.ActivePlanID, PolicyID: result.PolicyID})
		if err != nil {
			return result, err
		}
		current, _ := c.openRGBDeviceSetupPath("applied.json")
		if err := writeResourceOwnedFileDurable(current, data, 0o600); err != nil {
			result.Pending = true
			return result, err
		}
	}
	if !host.Applied && !host.RolledBack {
		result.Pending = true
		return result, errors.New("privileged permission owner returned no completed transaction")
	}
	result.Message = "Selected lighting permission transaction completed."
	return result, nil
}

type openRGBDeviceSetupOutput struct {
	data     []byte
	overflow bool
}

func (w *openRGBDeviceSetupOutput) Write(data []byte) (int, error) {
	remaining := (64 << 10) - len(w.data)
	w.overflow = w.overflow || len(data) > remaining
	w.data = append(w.data, data[:min(len(data), remaining)]...)
	return len(data), nil
}

func parseOpenRGBDeviceSetupChild(id string, data []byte, overflow bool, childErr error) (OpenRGBDeviceSetupResult, error) {
	result := OpenRGBDeviceSetupResult{PlanID: id}
	if overflow {
		return result, errors.Join(errors.New("privileged lighting permission receipt exceeds its output bound"), childErr)
	}
	if err := validateManagedUpdateJSON(data); err != nil {
		return result, errors.Join(errors.New("privileged lighting permission receipt is invalid"), err, childErr)
	}
	wire := &setupv1.LightingDeviceAccessResult{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, wire); err != nil || wire.PlanId != id {
		return result, errors.Join(errors.New("privileged lighting receipt differs from the selected plan"), err, childErr)
	}
	result.Message, result.PolicyID, result.NeedsSudo = wire.Message, wire.DevicePolicyId, wire.NeedsSudo
	result.Applied, result.RolledBack, result.Pending = wire.State == openRGBDeviceSetupAppliedState, wire.State == openRGBDeviceSetupRolledBackState, wire.State == openRGBDeviceSetupPendingState
	if !result.Applied && !result.RolledBack && !result.Pending && !result.NeedsSudo {
		return result, errors.Join(errors.New("privileged lighting receipt has no supported outcome"), childErr)
	}
	if result.Pending && childErr == nil {
		childErr = errors.New("lighting permission recovery remains pending")
	}
	return result, childErr
}
