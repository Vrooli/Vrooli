// Package hostwireless prepares unsigned, reviewed host wireless artifacts.
package hostwireless

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
)

const (
	NobleProfile          = "noble-mt7927"
	NobleBluetoothProfile = "noble-mt6639"
	nobleABI              = "7.0.0-30-generic"
	prepareLimit          = 4 * time.Minute
	buildAllowance        = 256 << 20
)

//go:embed noble-mt7927.patch
var nobleDelta []byte

//go:embed build-noble.sh
var nobleBuild []byte

//go:embed noble-mt6639.patch
var bluetoothDelta []byte

//go:embed build-noble-bluetooth.sh
var bluetoothBuild []byte

type buildProfile struct {
	patch, recipe []byte
	modules       []string
	newExports    map[string]string
	openItems     []string
}

var preparationProfiles = map[string]buildProfile{
	NobleProfile: {
		patch: nobleDelta, recipe: nobleBuild, modules: nobleModules,
		newExports: map[string]string{
			"mt792x_dma_alloc_queues": "mt792x-lib",
			"mt7925_mcu_set_dbdc":     "mt7925-common",
		},
		openItems: []string{
			"Qualify MT7927 PCIe loss containment and native recovery",
			"Qualify module signing and trust, host transaction and selectable boot recovery",
			"Coordinate Wi-Fi activation and verify firmware, association and traffic",
		},
	},
	NobleBluetoothProfile: {
		patch: bluetoothDelta, recipe: bluetoothBuild, modules: []string{"btmtk", "btusb"},
		openItems: []string{
			"Establish eligible MT6639 Bluetooth firmware with authenticated bytes",
			"Review the adapted transfer and reset behavior against later upstream changes",
			"Qualify module signing and trust, host transaction and selectable boot recovery",
			"Verify controller initialization, BlueZ peer discovery and pairing, and resume",
		},
	},
}

// Receipt identifies an unsigned candidate and the checks performed.
type Receipt struct {
	SchemaVersion       int               `json:"schema_version"`
	Profile             string            `json:"profile"`
	State               string            `json:"state"`
	Reason              string            `json:"reason,omitempty"`
	KernelRelease       string            `json:"kernel_release,omitempty"`
	PCIDevices          []string          `json:"pci_devices,omitempty"`
	USBDevices          []string          `json:"usb_devices,omitempty"`
	Modaliases          []string          `json:"modaliases,omitempty"`
	SourceDirectory     string            `json:"source_directory,omitempty"`
	SourceSHA256        string            `json:"source_profile_sha256,omitempty"`
	PatchSHA256         string            `json:"patch_sha256"`
	BuildRecipeSHA256   string            `json:"build_recipe_sha256"`
	ReservedDiskBytes   int64             `json:"reserved_disk_bytes,omitempty"`
	ArtifactDirectory   string            `json:"artifact_directory,omitempty"`
	ArtifactSHA256      map[string]string `json:"artifact_sha256,omitempty"`
	StockABI            ABIProof          `json:"stock_abi"`
	CandidateABI        ABIProof          `json:"candidate_abi"`
	CopiedTreeRollback  bool              `json:"copied_tree_rollback"`
	ActivationEligible  bool              `json:"activation_eligible"`
	ActivationOpenItems []string          `json:"activation_open_items"`
}

// Prepare inspects the explicit profile by default. Apply publishes an unsigned
// candidate under the user's artifact root after the stock and rollback checks.
func Prepare(ctx context.Context, profile string, apply bool) (Receipt, error) {
	return prepare(ctx, profile, apply, nil)
}

func prepare(ctx context.Context, profile string, apply bool, observed *hostinventory.HostInventory) (Receipt, error) {
	receipt := Receipt{SchemaVersion: 1, Profile: profile, State: "unsupported"}
	spec, ok := preparationProfiles[profile]
	if !ok {
		receipt.Reason = "unknown wireless preparation profile"
		return completeInspection(receipt, apply)
	}
	receipt.PatchSHA256, receipt.BuildRecipeSHA256 = digest(spec.patch), digest(spec.recipe)
	receipt.ActivationOpenItems = slices.Clone(spec.openItems)
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		receipt.Reason = "this profile requires Linux amd64"
		return completeInspection(receipt, apply)
	}
	osRelease, err := os.ReadFile("/etc/os-release")
	if err != nil {
		receipt.State, receipt.Reason = "unverified", fmt.Sprintf("read host distribution: %v", err)
		return completeInspection(receipt, apply)
	}
	inventory, err := hostinventory.NewIntegrityCollector(hostinventory.IntegrityCollectorOptions{}).Collect(ctx)
	if err != nil {
		receipt.State, receipt.Reason = "unverified", fmt.Sprintf("read host inventory: %v", err)
		return completeInspection(receipt, apply)
	}
	if observed != nil {
		*observed = inventory
	}
	selectNobleHost(&receipt, osRelease, inventory)
	if receipt.State != "eligible" {
		return completeInspection(receipt, apply)
	}

	source, _ := mt7927source.Lookup(mt7927source.NobleName)
	receipt.SourceDirectory, err = source.StagedPath()
	if err != nil {
		return receipt, err
	}
	if err := source.VerifyManifest(filepath.Join(receipt.SourceDirectory, mt7927source.Manifest)); err != nil {
		receipt.State = "source_required"
		receipt.Reason = fmt.Sprintf("stage SDA source/%s@%s on the host-tool surface: %v", mt7927source.NobleName, source.Version(), err)
		return completeInspection(receipt, apply)
	}
	if !apply {
		return receipt, nil
	}
	if os.Geteuid() == 0 {
		receipt.State, receipt.Reason = "unsupported", "wireless preparation requires an unprivileged user"
		return completeInspection(receipt, apply)
	}
	receipt, err = prepareArtifacts(ctx, receipt, source, spec)
	if err != nil && receipt.State != preparedArtifactState {
		receipt.State, receipt.Reason = "failed", err.Error()
	}
	return receipt, err
}

func completeInspection(r Receipt, apply bool) (Receipt, error) {
	if apply {
		return r, errors.New(r.Reason)
	}
	return r, nil
}

func selectNobleHost(r *Receipt, osRelease []byte, inventory hostinventory.HostInventory) {
	fields := map[string]string{}
	for _, line := range strings.Split(string(osRelease), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			fields[key] = strings.Trim(value, "\"'")
		}
	}
	r.KernelRelease = inventory.Kernel.Release
	switch {
	case fields["ID"] != "ubuntu" || fields["VERSION_CODENAME"] != "noble":
		r.State, r.Reason = "unsupported", "this profile requires Ubuntu Noble"
	case r.KernelRelease == "":
		r.State, r.Reason = "unverified", "running kernel release unavailable"
	case r.KernelRelease != nobleABI:
		r.State, r.Reason = "unsupported", "running kernel release differs from the reviewed "+nobleABI+" ABI"
	default:
		if r.Profile == NobleBluetoothProfile {
			selectBluetoothDevice(r, inventory)
			return
		}
		for _, device := range inventory.Devices {
			if device.BusType == "pci" && device.VendorID == "14c3" && device.DeviceID == "7927" {
				r.PCIDevices = append(r.PCIDevices, device.ID)
				if strings.HasPrefix(strings.ToLower(device.Modalias), "pci:v000014c3d00007927") {
					r.Modaliases = append(r.Modaliases, device.Modalias)
				}
			}
		}
		slices.Sort(r.Modaliases)
		r.Modaliases = slices.Compact(r.Modaliases)
		slices.Sort(r.PCIDevices)
		if len(r.Modaliases) > 0 {
			r.State, r.Reason = "eligible", ""
		} else if len(r.PCIDevices) > 0 || inventory.ProbeStatus["pciDevices"] != hostinventory.IntegrityProbeOK {
			r.State, r.Reason = "unverified", "MT7927 PCI modalias could not be verified"
		} else {
			r.State, r.Reason = "unsupported", "PCI device 14c3:7927 was not observed"
		}
	}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
