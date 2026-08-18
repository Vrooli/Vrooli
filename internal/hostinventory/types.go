package hostinventory

import "time"

type SourceKind string

const (
	SourceKindCommand SourceKind = "command"
	SourceKindFile    SourceKind = "file"
	SourceKindRuntime SourceKind = "runtime"
	SourceKindDerived SourceKind = "derived"
)

type Provenance struct {
	SourceKind SourceKind `json:"source_kind"`
	Source     string     `json:"source"`
	ObservedAt time.Time  `json:"observed_at"`
	Confidence string     `json:"confidence"`
	Command    string     `json:"command,omitempty"`
	File       string     `json:"file,omitempty"`
}

type Snapshot struct {
	OS                      string                  `json:"os"`
	Arch                    string                  `json:"arch"`
	CPU                     CPU                     `json:"cpu"`
	Load                    Load                    `json:"load,omitempty"`
	Memory                  Memory                  `json:"memory"`
	Swap                    Swap                    `json:"swap"`
	GPUs                    []GPU                   `json:"gpus"`
	NvidiaDeviceNodes       []string                `json:"nvidia_device_nodes,omitempty"`
	GPUProcesses            []GPUProcess            `json:"gpu_processes,omitempty"`
	RuntimeTools            map[string]Tool         `json:"runtime_tools,omitempty"`
	DockerGPU               DockerGPU               `json:"docker_gpu"`
	InitSystem              string                  `json:"init_system,omitempty"`
	SessionType             string                  `json:"session_type,omitempty"`
	Seat                    string                  `json:"seat,omitempty"`
	ActiveSessionUser       string                  `json:"active_session_user,omitempty"`
	DisplayManager          string                  `json:"display_manager,omitempty"`
	DisplayServer           string                  `json:"display_server,omitempty"`
	DisplayAttached         bool                    `json:"display_attached"`
	AutoLoginUser           string                  `json:"auto_login_user,omitempty"`
	RemoteDesktop           RemoteDesktopCapability `json:"remote_desktop"`
	Wayland                 WaylandCapability       `json:"wayland"`
	Elevation               ElevationCapability     `json:"elevation"`
	SupportsSysctl          bool                    `json:"supports_sysctl"`
	SupportsSystemd         bool                    `json:"supports_systemd"`
	SupportsLaunchd         bool                    `json:"supports_launchd"`
	SupportsWindowsServices bool                    `json:"supports_windows_services"`
	SupportsRDP             bool                    `json:"supports_rdp"`
	IsHeadless              bool                    `json:"is_headless"`
	IsWSL                   bool                    `json:"is_wsl"`
	SupportsCloudflared     bool                    `json:"supports_cloudflared"`
	Warnings                []string                `json:"warnings,omitempty"`
	ProbeStatuses           map[string]string       `json:"probe_statuses,omitempty"`
	FieldProvenance         map[string]Provenance
}

// WaylandCapability describes both the observed preference and whether the
// host can attain a Wayland session without overriding distribution policy.
type WaylandCapability struct {
	Attainable bool   `json:"attainable"`
	Reason     string `json:"reason,omitempty"`
}

// RemoteDesktopCapability is the shared, read-only classification of host
// remote-desktop providers. Consumers must select from these observations;
// they must not re-run provider-specific probes themselves.
type RemoteDesktopCapability struct {
	Supported        bool                      `json:"supported"`
	Observed         bool                      `json:"observed"`
	Mode             string                    `json:"mode,omitempty"`
	Active           bool                      `json:"active"`
	ListeningPort    int                       `json:"listening_port,omitempty"`
	SelectedProvider string                    `json:"selected_provider,omitempty"`
	Providers        []RemoteDesktopProvider   `json:"providers,omitempty"`
	CredentialStore  CredentialStoreCapability `json:"credential_store"`
}

// CredentialStoreCapability is the result of a real Secret Service read. A
// successful D-Bus peer ping is deliberately not considered evidence that the
// store can serve credentials. State is one of ready, empty, locked,
// unresponsive, unavailable, or unsupported.
type CredentialStoreCapability struct {
	Supported      bool   `json:"supported"`
	Observed       bool   `json:"observed"`
	State          string `json:"state,omitempty"`
	ProbeSucceeded bool   `json:"probe_succeeded"`
	Reason         string `json:"reason,omitempty"`
}

type RemoteDesktopProvider struct {
	Name           string `json:"name"`
	Present        bool   `json:"present"`
	Active         bool   `json:"active"`
	ProbeSucceeded bool   `json:"probe_succeeded"`
	UserSession    bool   `json:"user_session,omitempty"`
}

func (c RemoteDesktopCapability) Provider(name string) (RemoteDesktopProvider, bool) {
	for _, provider := range c.Providers {
		if provider.Name == name {
			return provider, true
		}
	}
	return RemoteDesktopProvider{}, false
}

// ElevationCapability is the typed answer to whether a privileged operation
// can proceed on the current host. Windows deliberately reports no
// non-interactive elevation mechanism when the process is not elevated.
type ElevationCapability struct {
	Elevated   bool   `json:"elevated"`
	CanElevate bool   `json:"can_elevate"`
	Mechanism  string `json:"mechanism,omitempty"`
}

// DisplayManagerNames is the single display-manager vocabulary shared by the
// control plane and autoheal. xrdp is intentionally absent: it is an RDP
// server, not a display manager.
var DisplayManagerNames = []string{"gdm", "gdm3", "lightdm", "sddm", "lxdm", "xdm"}

type CPU struct {
	Cores int `json:"cores"`
}

type Load struct {
	Load1           float64 `json:"load1,omitempty"`
	Load5           float64 `json:"load5,omitempty"`
	Load15          float64 `json:"load15,omitempty"`
	RunningProcs    int     `json:"running_procs,omitempty"`
	TotalProcs      int     `json:"total_procs,omitempty"`
	LastPID         int     `json:"last_pid,omitempty"`
	NormalizedLoad1 float64 `json:"normalized_load1,omitempty"`
	NormalizedLoad5 float64 `json:"normalized_load5,omitempty"`
}

type Memory struct {
	TotalBytes     uint64 `json:"total_bytes"`
	AvailableBytes uint64 `json:"available_bytes,omitempty"`
	BuffersBytes   uint64 `json:"buffers_bytes,omitempty"`
	CachedBytes    uint64 `json:"cached_bytes,omitempty"`
}

type Swap struct {
	TotalBytes uint64 `json:"total_bytes,omitempty"`
	FreeBytes  uint64 `json:"free_bytes,omitempty"`
}

type GPU struct {
	Index                    int      `json:"index"`
	UUID                     string   `json:"uuid,omitempty"`
	Name                     string   `json:"name"`
	DriverVersion            string   `json:"driver_version,omitempty"`
	CUDAComputeCapability    string   `json:"cuda_compute_capability,omitempty"`
	VRAMBytes                uint64   `json:"vram_bytes"`
	VRAMUsedBytes            uint64   `json:"vram_used_bytes,omitempty"`
	UtilizationPercent       float64  `json:"utilization_percent,omitempty"`
	MemoryUtilizationPercent float64  `json:"memory_utilization_percent,omitempty"`
	TemperatureC             *float64 `json:"temperature_c,omitempty"`
	FanSpeedPercent          *float64 `json:"fan_speed_percent,omitempty"`
	PowerDrawW               *float64 `json:"power_draw_w,omitempty"`
	PowerLimitW              *float64 `json:"power_limit_w,omitempty"`
	SMClockMHz               *float64 `json:"sm_clock_mhz,omitempty"`
	MemoryClockMHz           *float64 `json:"memory_clock_mhz,omitempty"`
	Source                   string   `json:"source"`
}

type GPUProcess struct {
	GPUIndex    int    `json:"gpu_index"`
	GPUUUID     string `json:"gpu_uuid,omitempty"`
	PID         int    `json:"pid"`
	ProcessName string `json:"process_name"`
	UsedBytes   uint64 `json:"used_bytes"`
}

type DockerGPU struct {
	NvidiaRuntime bool `json:"nvidia_runtime"`
}

type Tool struct {
	Present bool   `json:"present"`
	Path    string `json:"path,omitempty"`
	// Version is populated only by probes that can read one cheaply. An empty
	// Version on a present tool means "not probed", never "no version".
	Version string `json:"version,omitempty"`
}

func (s Snapshot) HasNvidiaGPU() bool {
	for _, gpu := range s.GPUs {
		if gpu.Name != "" && gpu.Source == "nvidia-smi" {
			return true
		}
	}
	return false
}

func (s Snapshot) HasDockerNvidiaRuntime() bool {
	return s.DockerGPU.NvidiaRuntime
}

func (s Snapshot) HasDockerAddressableNvidiaGPU() bool {
	return s.HasNvidiaGPU() && s.HasDockerNvidiaRuntime()
}
