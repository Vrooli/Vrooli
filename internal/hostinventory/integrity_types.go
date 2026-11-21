package hostinventory

import (
	"regexp"
	"strings"
	"time"
)

var pciAddressIdentityPattern = regexp.MustCompile(`^(?:[0-9a-f]{4}:)?[0-9a-f]{2}:[0-9a-f]{2}\.[0-9a-f]$`)

// DriverBindingRepairRoute points consumers back to the control-plane owner
// for supported host changes. Runtime scenarios report evidence and never
// install a driver or write host configuration themselves.
const DriverBindingRepairRoute = "Run `vrooli setup status` to identify supported host requirements, then `vrooli setup explain <requirement>` before applying a supported safeguard. Do not infer or install a driver from a device name alone."

// CanonicalPCIIdentity returns the shared stable identity used by control-plane
// inventory and telemetry consumers. lspci omits the domain unless requested;
// a missing domain is the conventional zero domain, while sysfs always carries
// the full domain:bus:device.function address.
func CanonicalPCIIdentity(address string) string {
	address = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(address)), "pci:")
	if !pciAddressIdentityPattern.MatchString(address) {
		return ""
	}
	if len(address) == 7 {
		address = "0000:" + address
	}
	return "pci:" + address
}

// IntegrityProbeState describes the quality of one host-integrity observation.
// It is intentionally separate from the capability snapshot probe statuses:
// integrity checks need to distinguish an unsupported platform from a probe
// that ran but degraded.
type IntegrityProbeState string

const (
	IntegrityProbeOK          IntegrityProbeState = "ok"
	IntegrityProbeUnsupported IntegrityProbeState = "unsupported"
	IntegrityProbeDegraded    IntegrityProbeState = "degraded"
	IntegrityProbeFailed      IntegrityProbeState = "failed"
)

type HostInventory struct {
	ID             string                         `json:"id,omitempty"`
	CollectedAt    time.Time                      `json:"collectedAt"`
	Platform       string                         `json:"platform"`
	OS             string                         `json:"os"`
	Arch           string                         `json:"arch"`
	BootID         string                         `json:"bootId,omitempty"`
	Kernel         KernelInfo                     `json:"kernel"`
	Devices        []DeviceInfo                   `json:"devices"`
	Runtimes       []RuntimeToolInfo              `json:"runtimes"`
	Packages       PackageState                   `json:"packages"`
	SecureBoot     SecureBootState                `json:"secureBoot,omitempty"`
	ResetReasons   []ResetReason                  `json:"resetReasons,omitempty"`
	CrashEvidence  CrashEvidenceProbeState        `json:"crashEvidence,omitempty"`
	Signals        []HostSignal                   `json:"signals"`
	ProbeStatus    map[string]IntegrityProbeState `json:"probeStatus"`
	ProbeErrors    map[string]string              `json:"probeErrors,omitempty"`
	Unsupported    []string                       `json:"unsupportedCapabilities,omitempty"`
	Fingerprint    string                         `json:"fingerprint"`
	CollectedParts map[string]map[string]any      `json:"collectedParts,omitempty"`
}

type KernelInfo struct {
	Release              string   `json:"release,omitempty"`
	Version              string   `json:"version,omitempty"`
	ModuleTreePresent    bool     `json:"moduleTreePresent"`
	InstalledModuleTrees []string `json:"installedModuleTrees,omitempty"`
	LoadedModules        []string `json:"loadedModules,omitempty"`
}

type DeviceInfo struct {
	ID               string           `json:"id,omitempty"`
	BusType          string           `json:"busType"`
	Address          string           `json:"address,omitempty"`
	SysPath          string           `json:"sysPath,omitempty"`
	Class            string           `json:"class,omitempty"`
	VendorID         string           `json:"vendorId,omitempty"`
	VendorName       string           `json:"vendorName,omitempty"`
	DeviceID         string           `json:"deviceId,omitempty"`
	DeviceName       string           `json:"deviceName,omitempty"`
	BoundDriver      string           `json:"boundDriver,omitempty"`
	BindingState     string           `json:"bindingState,omitempty"`
	Modalias         string           `json:"modalias,omitempty"`
	Functions        []DeviceFunction `json:"functions,omitempty"`
	Readiness        DeviceReadiness  `json:"readiness"`
	AvailableModules []string         `json:"availableKernelModules,omitempty"`
}

// DeviceFunction records a separately bound interface below an enumerated
// physical device, such as a USB Bluetooth function. The parent USB device's
// generic "usb" driver does not stand in for its functional interface driver.
type DeviceFunction struct {
	ID           string `json:"id"`
	Class        string `json:"class,omitempty"`
	Modalias     string `json:"modalias,omitempty"`
	BoundDriver  string `json:"boundDriver,omitempty"`
	BindingState string `json:"bindingState"`
}

// DeviceReadiness keeps firmware, OS-interface and function evidence separate
// from physical enumeration and driver binding. Unknown and unverified are
// explicit: a bound driver alone proves none of these stages.
type DeviceReadiness struct {
	FirmwareState   string            `json:"firmwareState"`
	FirmwareReason  string            `json:"firmwareReason,omitempty"`
	InterfaceState  string            `json:"interfaceState"`
	InterfaceNames  []string          `json:"interfaceNames,omitempty"`
	InterfaceStatus map[string]string `json:"interfaceStatus,omitempty"`
	InterfaceReason string            `json:"interfaceReason,omitempty"`
	FunctionState   string            `json:"functionState"`
	FunctionReason  string            `json:"functionReason,omitempty"`
}

const (
	DeviceBindingBound   = "bound"
	DeviceBindingUnbound = "unbound"
	DeviceBindingUnknown = "unknown"
	DeviceBindingPartial = "partial"

	DeviceReadinessUnknown       = "unknown"
	DeviceReadinessNotApplicable = "not_applicable"
	DeviceReadinessMissing       = "missing"
	DeviceReadinessAbsent        = "absent"
	DeviceReadinessInitialized   = "initialized"
	DeviceReadinessDisabled      = "disabled"
	DeviceReadinessDisconnected  = "disconnected"
	DeviceReadinessUnverified    = "unverified"
	DeviceReadinessVerified      = "verified"
)

type RuntimeToolInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path,omitempty"`
	Version  string `json:"version,omitempty"`
	Callable bool   `json:"callable"`
	Error    string `json:"error,omitempty"`
}

type PackageState struct {
	Manager           string               `json:"manager,omitempty"`
	Installed         []string             `json:"installedRelevantPackages,omitempty"`
	InstalledPackages []PackageInfo        `json:"installedPackages,omitempty"`
	PendingUpgrades   []string             `json:"pendingUpgrades,omitempty"`
	BrokenOrHeld      []string             `json:"brokenOrHeldPackages,omitempty"`
	KernelModuleDrift []string             `json:"kernelModuleDrift,omitempty"`
	Kernel            KernelPackageState   `json:"kernel,omitempty"`
	Drivers           []DriverPackageState `json:"drivers,omitempty"`
}

type PackageInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status,omitempty"`
}

type KernelPackageState struct {
	RunningKernel        string   `json:"runningKernel,omitempty"`
	InstalledMatching    []string `json:"installedMatchingPackages,omitempty"`
	MissingMatching      []string `json:"missingMatchingPackages,omitempty"`
	HeldOrBlocked        []string `json:"heldOrBlockedPackages,omitempty"`
	InstalledOtherKernel []string `json:"installedOtherKernelPackages,omitempty"`
}

type DriverPackageState struct {
	Vendor                   string            `json:"vendor"`
	VendorID                 string            `json:"vendorId,omitempty"`
	Series                   string            `json:"series,omitempty"`
	Flavor                   string            `json:"flavor,omitempty"`
	InstalledPackages        []PackageInfo     `json:"installedPackages,omitempty"`
	LoadedModules            []string          `json:"loadedModules,omitempty"`
	ExpectedModulePackage    string            `json:"expectedModulePackage,omitempty"`
	ExpectedPackageInstalled bool              `json:"expectedPackageInstalled"`
	MissingModulePackage     string            `json:"missingModulePackage,omitempty"`
	Candidate                *PackageCandidate `json:"candidate,omitempty"`
	Applicability            string            `json:"applicability"`
}

type PackageCandidate struct {
	Name      string `json:"name"`
	Version   string `json:"version,omitempty"`
	Available bool   `json:"available"`
	Source    string `json:"source,omitempty"`
	Error     string `json:"error,omitempty"`
}

type SecureBootState struct {
	Supported bool   `json:"supported"`
	Enabled   bool   `json:"enabled"`
	Source    string `json:"source,omitempty"`
	Error     string `json:"error,omitempty"`
}

type ResetReason struct {
	BootID      string    `json:"bootId,omitempty"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
	Source      string    `json:"source"`
	RawMessage  string    `json:"rawMessage"`
	Category    string    `json:"category"`
	Criticality string    `json:"criticality"`
}

type CrashEvidenceProbeState struct {
	PstoreSupported       bool   `json:"pstoreSupported"`
	PstoreReadable        bool   `json:"pstoreReadable"`
	PstoreError           string `json:"pstoreError,omitempty"`
	PstoreExportReadable  bool   `json:"pstoreExportReadable"`
	PstoreExportError     string `json:"pstoreExportError,omitempty"`
	PstoreCoverageGap     bool   `json:"pstoreCoverageGap"`
	RasdaemonPresent      bool   `json:"rasdaemonPresent"`
	RasdaemonPath         string `json:"rasdaemonPath,omitempty"`
	RasdaemonServiceState string `json:"rasdaemonServiceState,omitempty"`
	RasdaemonError        string `json:"rasdaemonError,omitempty"`
}

type HostSignal struct {
	Source    string    `json:"source"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	BootID    string    `json:"bootId,omitempty"`
	Category  string    `json:"category"`
	Message   string    `json:"message"`
}

type Change struct {
	ID             int64          `json:"id,omitempty"`
	FromSnapshotID string         `json:"fromSnapshotId,omitempty"`
	ToSnapshotID   string         `json:"toSnapshotId,omitempty"`
	ChangeType     string         `json:"changeType"`
	Severity       string         `json:"severity"`
	Summary        string         `json:"summary"`
	Details        map[string]any `json:"details,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
}

type SnapshotRecord struct {
	ID            string        `json:"id"`
	CollectedAt   time.Time     `json:"collectedAt"`
	LastSeenAt    time.Time     `json:"lastSeenAt,omitempty"`
	Platform      string        `json:"platform"`
	OS            string        `json:"os"`
	Arch          string        `json:"arch"`
	BootID        string        `json:"bootId,omitempty"`
	KernelRelease string        `json:"kernelRelease,omitempty"`
	Fingerprint   string        `json:"fingerprint"`
	Inventory     HostInventory `json:"inventory"`
}

type InventoryResponse struct {
	Snapshot    *SnapshotRecord                `json:"snapshot"`
	Fresh       bool                           `json:"fresh"`
	AgeSeconds  int64                          `json:"ageSeconds"`
	ProbeStatus map[string]IntegrityProbeState `json:"probeStatus"`
}
