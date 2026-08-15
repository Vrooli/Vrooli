package hostinventory

import "time"

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
	BusType          string   `json:"busType"`
	Address          string   `json:"address,omitempty"`
	Class            string   `json:"class,omitempty"`
	VendorID         string   `json:"vendorId,omitempty"`
	VendorName       string   `json:"vendorName,omitempty"`
	DeviceID         string   `json:"deviceId,omitempty"`
	DeviceName       string   `json:"deviceName,omitempty"`
	BoundDriver      string   `json:"boundDriver,omitempty"`
	AvailableModules []string `json:"availableKernelModules,omitempty"`
}

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
