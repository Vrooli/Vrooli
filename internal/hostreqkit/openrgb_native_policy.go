package hostreqkit

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// OpenRGBNativePolicy enrolls native HID access already held by an ordinary
// account. It never authorizes elevation, a driver installation, or a bus scan.
// Linux's existing udev enrollment remains a separate versioned contract.
type OpenRGBNativePolicy struct {
	Platform, MachineID, Account   string
	ArtifactSHA256, DetectorSHA256 string
	Devices                        []OpenRGBNativeDevice
}

// StableID identifies the native attachment; a current registry/devnode birth
// belongs to OpenRGBNativeNode.Generation, never to the durable selector.
type OpenRGBNativeDevice struct {
	Detector, Kind, StableID, HardwareID string
	UsagePage, Usage                     uint16
}

type OpenRGBNativeNode struct {
	Selector         OpenRGBNativeDevice
	Path, Generation string
}

type OpenRGBNativeEvidence struct {
	PolicyID, Account, DeviceGenerationSHA256 string
	SelectedPaths                             []string
	CurrentScopeVerified                      bool
}

func nativeDeviceKey(d OpenRGBNativeDevice) string {
	b, _ := json.Marshal(d)
	return string(b)
}

func (p OpenRGBNativePolicy) ID() (string, error) {
	if (p.Platform != "darwin" && p.Platform != "windows") || !openRGBHex(p.MachineID, 64) || !openRGBHex(p.ArtifactSHA256, 64) || !openRGBHex(p.DetectorSHA256, 64) || !validNativeAccount(p.Platform, p.Account) || len(p.Devices) == 0 || len(p.Devices) > 16 {
		return "", errors.New("incomplete native OpenRGB HID policy")
	}
	p.Devices = slices.Clone(p.Devices)
	slices.SortFunc(p.Devices, func(a, b OpenRGBNativeDevice) int { return strings.Compare(nativeDeviceKey(a), nativeDeviceKey(b)) })
	stable := make(map[string]bool, len(p.Devices))
	for i, d := range p.Devices {
		hardware := strings.Split(d.HardwareID, ":")
		if d.Kind != "hid" || strings.TrimSpace(d.Detector) == "" || strings.ContainsAny(d.Detector, "\r\n\x00") || len(hardware) != 2 || !openRGBHex(hardware[0], 4) || !openRGBHex(hardware[1], 4) || stable[d.StableID] || strings.TrimSpace(d.StableID) == "" || len(d.StableID) > 4096 || strings.ContainsAny(d.StableID, "\r\n\x00*?") || i > 0 && d == p.Devices[i-1] {
			return "", errors.New("unqualified or duplicate native OpenRGB HID selector")
		}
		stable[d.StableID] = true
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}

// Keep ID structural so retained exact revocation survives changes to the
// qualified release/catalog/controller set. Launch qualification is separate.
func ValidateOpenRGBNativeDetectorScope(p OpenRGBNativePolicy) error {
	if _, err := p.ID(); err != nil {
		return err
	}
	if p.DetectorSHA256 != openRGBQualifiedDetectorCatalogSHA256 {
		return errors.New("native HID policy detector catalog has no qualified source footprint")
	}
	for _, d := range p.Devices {
		if !qualifiedNativeHID(d) {
			return errors.New("native HID policy selects an unqualified controller transaction footprint")
		}
	}
	return nil
}

func validNativeAccount(platform, account string) bool {
	if platform == "darwin" {
		if !strings.HasPrefix(account, "uid:") {
			return false
		}
		var id uint32
		_, err := fmt.Sscanf(account, "uid:%d", &id)
		return err == nil && id > 0 && account == fmt.Sprintf("uid:%d", id)
	}
	if platform != "windows" {
		return false
	}
	// Local/domain and Microsoft Entra operator SIDs. Well-known SYSTEM,
	// service, builtin-group and capability identities are excluded. Actual
	// user membership and elevation are independently observed from the token.
	partsCount := 4
	prefix := "sid:S-1-5-21-"
	if strings.HasPrefix(account, "sid:S-1-12-1-") {
		prefix = "sid:S-1-12-1-"
	}
	if !strings.HasPrefix(account, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(account, prefix), "-")
	if len(parts) != partsCount {
		return false
	}
	for _, v := range parts {
		var n uint32
		if _, e := fmt.Sscanf(v, "%d", &n); e != nil || fmt.Sprint(n) != v {
			return false
		}
	}
	return true
}

// ValidateNativeOpenRGBAccount validates a non-system operator principal without
// querying credentials or opening hardware. Runtime token proof is separate.
func ValidateNativeOpenRGBAccount(platform, account string) error {
	if !validNativeAccount(platform, account) {
		return errors.New("native OpenRGB requires an ordinary operator principal")
	}
	return nil
}

// Source footprint: OpenRGB release_1.1 AsusAuraUSBControllerDetect.cpp,
// REGISTER_HID_DETECTOR[_PU] motherboard and external Terminal entries. These callbacks use HIDAPI;
// RAM/PCI/SMBus/PawnIO paths are deliberately outside this policy.
func qualifiedNativeHID(d OpenRGBNativeDevice) bool {
	if d.Kind != "hid" {
		return false
	}
	if d.Detector == "ASUS ROG AURA Terminal" {
		return d.HardwareID == "0b05:1889"
	}
	if d.Detector != "ASUS Aura Motherboard" {
		return false
	}
	switch d.HardwareID {
	case "0b05:18f3", "0b05:1939", "0b05:1aa6", "0b05:1bed":
		return true
	case "0b05:19af":
		return d.UsagePage == 0xff72 && d.Usage == 0xa1
	}
	return false
}

// DiscoverNativeOpenRGBDevices reads OS HID metadata only. It opens no lighting
// transport and is called only by explicit setup, never implicit startup.
func DiscoverNativeOpenRGBDevices(ctx context.Context) ([]OpenRGBNativeNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nativeOpenRGBDiscover(ctx)
}

// PlanNativeOpenRGBDeviceAccess binds an explicit selected scope to the native
// machine/account. Applying it verifies existing HID access without modifying
// OS policy; callers persist only after this verification succeeds.
func PlanNativeOpenRGBDeviceAccess(ctx context.Context, artifact, catalog string, selected []OpenRGBNativeDevice) (OpenRGBNativePolicy, error) {
	machine, account, err := nativeOpenRGBIdentity()
	if err != nil {
		return OpenRGBNativePolicy{}, err
	}
	p := OpenRGBNativePolicy{Platform: nativeOpenRGBPlatform(), MachineID: machine, Account: account, ArtifactSHA256: artifact, DetectorSHA256: catalog, Devices: slices.Clone(selected)}
	if _, err = p.ID(); err != nil {
		return OpenRGBNativePolicy{}, err
	}
	_, err = observeNativeOpenRGB(ctx, p, nativeOpenRGBIdentity, nativeOpenRGBDiscover, nil)
	return p, err
}

// NativeOpenRGBIdentity returns the current caller's native identity. Setup
// uses it for receipt ownership only; it is not evidence of another account's
// device access.
func NativeOpenRGBIdentity() (string, string, error) { return nativeOpenRGBIdentity() }

// NativeOpenRGBMachineIdentity is metadata only and does not attest runtime
// credentials or hardware access. Setup checks it before provisioning effects
// so a Darwin no-CGO executable cannot leave a partially prepared account.
func NativeOpenRGBMachineIdentity() (string, error) { return nativeOpenRGBMachineID() }

// PlanNativeOpenRGBDeviceAccessForAccount binds metadata to an already
// provisioned runtime principal. This performs no transport access and makes
// no claim that the caller's token can prove the target account's HID rights.
func PlanNativeOpenRGBDeviceAccessForAccount(ctx context.Context, artifact, catalog, account string, selected []OpenRGBNativeDevice) (OpenRGBNativePolicy, error) {
	machine, err := nativeOpenRGBMachineID()
	if err != nil {
		return OpenRGBNativePolicy{}, err
	}
	p := OpenRGBNativePolicy{Platform: nativeOpenRGBPlatform(), MachineID: machine, Account: account, ArtifactSHA256: artifact, DetectorSHA256: catalog, Devices: slices.Clone(selected)}
	_, err = ValidateNativeOpenRGBDeviceSelection(ctx, p)
	return p, err
}

// ValidateNativeOpenRGBDeviceSelection verifies metadata scope using the local
// machine without impersonating or asserting access by the enrolled principal.
func ValidateNativeOpenRGBDeviceSelection(ctx context.Context, p OpenRGBNativePolicy) (OpenRGBNativeEvidence, error) {
	return observeNativeOpenRGB(ctx, p, func() (string, string, error) {
		machine, err := nativeOpenRGBMachineID()
		return machine, p.Account, err
	}, nativeOpenRGBDiscover, nil)
}

func ObserveNativeOpenRGBDevicePolicy(ctx context.Context, p OpenRGBNativePolicy) (OpenRGBNativeEvidence, error) {
	return observeNativeOpenRGB(ctx, p, nativeOpenRGBIdentity, nativeOpenRGBDiscover, nativeOpenRGBAccess)
}

// VerifyNativeOpenRGBPolicyOwner checks execution identity without requiring a
// present device, allowing its owner to revoke enrollment after unplugging.
func VerifyNativeOpenRGBPolicyOwner(p OpenRGBNativePolicy) error {
	if _, err := p.ID(); err != nil {
		return err
	}
	machine, account, err := nativeOpenRGBIdentity()
	if err != nil {
		return err
	}
	if machine != p.MachineID || account != p.Account {
		return errors.New("native HID policy belongs to a different machine or account")
	}
	return nil
}

func observeNativeOpenRGB(ctx context.Context, p OpenRGBNativePolicy, identity func() (string, string, error), discover func(context.Context) ([]OpenRGBNativeNode, error), access func(context.Context, OpenRGBNativeNode) error) (OpenRGBNativeEvidence, error) {
	var result OpenRGBNativeEvidence
	if err := ValidateOpenRGBNativeDetectorScope(p); err != nil {
		return result, err
	}
	id, err := p.ID()
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	machine, account, err := identity()
	if err != nil {
		return result, err
	}
	if machine != p.MachineID || account != p.Account {
		return result, errors.New("native OpenRGB machine/account differs from enrollment")
	}
	nodes, err := discover(ctx)
	if err != nil {
		return result, err
	}
	selected := map[OpenRGBNativeDevice]bool{}
	enabled := map[string]bool{}
	for _, d := range p.Devices {
		selected[d] = false
		enabled[d.Detector] = true
	}
	var current []OpenRGBNativeNode
	for _, n := range nodes {
		if !qualifiedNativeHID(n.Selector) || !enabled[n.Selector.Detector] {
			continue
		}
		// An enabled upstream detector enumerates every matching HID interface.
		// Refuse rather than grant access to unselected callback candidates.
		if _, ok := selected[n.Selector]; !ok {
			return result, errors.New("native HID detector would reach an unenrolled attachment; review a complete selection")
		}
		if selected[n.Selector] || n.Path == "" || n.Generation == "" {
			return result, errors.New("ambiguous native HID attachment or missing generation")
		}
		selected[n.Selector] = true
		current = append(current, n)
	}
	for _, found := range selected {
		if !found {
			return result, errors.New("selected native HID attachment is absent or replaced")
		}
	}
	slices.SortFunc(current, func(a, b OpenRGBNativeNode) int {
		return strings.Compare(nativeDeviceKey(a.Selector), nativeDeviceKey(b.Selector))
	})
	for _, n := range current {
		if access != nil {
			if err = access(ctx, n); err != nil {
				return result, err
			}
		}
		result.SelectedPaths = append(result.SelectedPaths, n.Path)
	}
	// Re-enumerate after the access opens, rejecting replacement during proof.
	after, err := discover(ctx)
	if err != nil {
		return result, err
	}
	var relevant []OpenRGBNativeNode
	for _, n := range after {
		if qualifiedNativeHID(n.Selector) && enabled[n.Selector.Detector] {
			relevant = append(relevant, n)
		}
	}
	slices.SortFunc(relevant, func(a, b OpenRGBNativeNode) int {
		return strings.Compare(nativeDeviceKey(a.Selector), nativeDeviceKey(b.Selector))
	})
	if !slices.Equal(current, relevant) {
		return result, errors.New("native HID generation changed during access verification")
	}
	machine, account, err = identity()
	if err != nil || machine != p.MachineID || account != p.Account {
		return result, errors.Join(errors.New("native HID execution identity changed during access verification"), err)
	}
	b, _ := json.Marshal(current)
	result.PolicyID, result.Account, result.DeviceGenerationSHA256 = id, account, fmt.Sprintf("%x", sha256.Sum256(b))
	result.CurrentScopeVerified = access != nil
	return result, nil
}

// Native detector classification follows the exact registered VID/PID footprint.
func nativeOpenRGBHIDDetector(hardware string) string {
	switch hardware {
	case "0b05:1889":
		return "ASUS ROG AURA Terminal"
	case "0b05:18f3", "0b05:1939", "0b05:19af", "0b05:1aa6", "0b05:1bed":
		return "ASUS Aura Motherboard"
	}
	return ""
}
