// Package hostinventory collects typed host facts for integrity checks.
package hostinventory

import "time"

type ProbeState string

const (
	ProbeOK          ProbeState = "ok"
	ProbeUnsupported ProbeState = "unsupported"
	ProbeDegraded    ProbeState = "degraded"
	ProbeFailed      ProbeState = "failed"
)

type HostInventory struct {
	ID             string                    `json:"id,omitempty"`
	CollectedAt    time.Time                 `json:"collectedAt"`
	Platform       string                    `json:"platform"`
	OS             string                    `json:"os"`
	Arch           string                    `json:"arch"`
	BootID         string                    `json:"bootId,omitempty"`
	Kernel         KernelInfo                `json:"kernel"`
	Devices        []DeviceInfo              `json:"devices"`
	Runtimes       []RuntimeToolInfo         `json:"runtimes"`
	Packages       PackageState              `json:"packages"`
	Signals        []HostSignal              `json:"signals"`
	ProbeStatus    map[string]ProbeState     `json:"probeStatus"`
	ProbeErrors    map[string]string         `json:"probeErrors,omitempty"`
	Unsupported    []string                  `json:"unsupportedCapabilities,omitempty"`
	Fingerprint    string                    `json:"fingerprint"`
	CollectedParts map[string]map[string]any `json:"collectedParts,omitempty"`
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
	Manager           string   `json:"manager,omitempty"`
	Installed         []string `json:"installedRelevantPackages,omitempty"`
	PendingUpgrades   []string `json:"pendingUpgrades,omitempty"`
	BrokenOrHeld      []string `json:"brokenOrHeldPackages,omitempty"`
	KernelModuleDrift []string `json:"kernelModuleDrift,omitempty"`
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
	Snapshot    *SnapshotRecord       `json:"snapshot"`
	Fresh       bool                  `json:"fresh"`
	AgeSeconds  int64                 `json:"ageSeconds"`
	ProbeStatus map[string]ProbeState `json:"probeStatus"`
}
