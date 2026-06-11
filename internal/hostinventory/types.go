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
	OS              string            `json:"os"`
	Arch            string            `json:"arch"`
	CPU             CPU               `json:"cpu"`
	Load            Load              `json:"load,omitempty"`
	Memory          Memory            `json:"memory"`
	Swap            Swap              `json:"swap"`
	GPUs            []GPU             `json:"gpus"`
	GPUProcesses    []GPUProcess      `json:"gpu_processes,omitempty"`
	RuntimeTools    map[string]Tool   `json:"runtime_tools,omitempty"`
	DockerGPU       DockerGPU         `json:"docker_gpu"`
	Warnings        []string          `json:"warnings,omitempty"`
	ProbeStatuses   map[string]string `json:"probe_statuses,omitempty"`
	FieldProvenance map[string]Provenance
}

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
