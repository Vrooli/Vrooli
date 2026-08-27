package hostinventory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/envkit-go"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostfacts"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	collectorNoDevices      = "no_devices"
	collectorUnsupported    = "unsupported"
	collectorToolNotPresent = "tool_not_present"
)

const (
	collectorParameterA = 2
)

type CommandRunner = shell.Runner

type EnvironmentCommandRunner interface {
	RunWithEnv(ctx context.Context, env []string, name string, args ...string) ([]byte, error)
}

type FileReader interface {
	ReadFile(name string) ([]byte, error)
}

type EnvReader interface {
	Getenv(key string) string
}

type Clock = TimeSource

type TimeSource interface {
	Now() time.Time
}

type Collector struct {
	Commands CommandRunner
	Files    FileReader
	Env      EnvReader
	Clock    Clock
	GOOS     string
	GOARCH   string
	CPUCount func() int
	// DeviceRoots overrides the filesystem locations device-tree enumeration
	// reads. The zero value means the live host.
	DeviceRoots DeviceRoots
}

type (
	osCommandRunner struct{}
	osFileReader    struct{}
	osEnvReader     struct{}
	systemClock     struct{}
)

func (osCommandRunner) LookPath(file string) (string, error) { return exec.LookPath(file) }

func (osCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return runOSCommand(ctx, nil, name, args...)
}

func (osCommandRunner) RunWithEnv(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	return runOSCommand(ctx, env, name, args...)
}

func runOSCommand(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		command.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.Resource, envkit.Env(env))
	}
	configureCommandProcessGroup(command)
	// The context must cancel the whole process group, not only a wrapper
	// process, or a wedged child can retain CombinedOutput's pipe indefinitely.
	command.Cancel = func() error { return terminateCommandProcessGroup(command) }
	command.WaitDelay = tuning.FastHealthPollInterval
	return command.CombinedOutput()
}

func configureCommandProcessGroup(command *exec.Cmd) {
	_ = platform.ConfigureCommand(command, platform.ProcessOptions{Detached: true})
}

func terminateCommandProcessGroup(command *exec.Cmd) error {
	if command == nil || command.Process == nil {
		return nil
	}
	return platform.KillProcess(command.Process.Pid, true)
}

func (osFileReader) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (osFileReader) Glob(pattern string) []string {
	paths, _ := filepath.Glob(pattern)
	return paths
}
func (osEnvReader) Getenv(key string) string { return os.Getenv(key) }
func (systemClock) Now() time.Time           { return time.Now() }

func SystemCollector() Collector {
	return Collector{
		Commands: osCommandRunner{},
		Files:    osFileReader{},
		Env:      osEnvReader{},
		Clock:    systemClock{},
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
		CPUCount: runtime.NumCPU,
	}
}

func Collect(ctx context.Context) (Snapshot, error) {
	var snapshot Snapshot
	if raw, err := sharedFactsReader().Read(ctx, "inventory"); err == nil && json.Unmarshal(raw, &snapshot) == nil {
		return snapshot, nil
	}
	return SystemCollector().Collect(ctx)
}

var (
	factsReaderMu sync.Mutex
	factsReader   *hostfacts.Reader
)

func sharedFactsReader() *hostfacts.Reader {
	factsReaderMu.Lock()
	defer factsReaderMu.Unlock()
	if factsReader != nil {
		return factsReader
	}
	// The repository contract designates ~/.vrooli as the cross-process
	// operator runtime home. Do not use os.UserConfigDir here: that would put
	// the cache under ~/.config on Linux and a different platform-specific
	// location elsewhere, preventing the short-lived CLI processes from
	// sharing the same facts file as the rest of the control plane.
	root, err := os.UserHomeDir()
	if err != nil || root == "" {
		root = os.TempDir()
	}
	factsReader = &hostfacts.Reader{Path: filepath.Join(root, repocontractmeta.ProjectConfigDir, "cache", "hostfacts.json"), TTL: map[string]time.Duration{"inventory": tuning.StandardOperationTimeout, "platform": tuning.LongOperationTimeout, "gpu": tuning.ExtendedOperationTimeout, "workloads": tuning.LongOperationTimeout}, BootID: bootID, Probe: func(ctx context.Context, class string) (json.RawMessage, error) {
		var s Snapshot
		var err error
		switch class {
		case "gpu":
			s, err = SystemCollector().CollectGPUFacts(ctx)
		case "platform":
			s, err = SystemCollector().CollectPlatformFacts(ctx)
		case "workloads":
			w, workloadErr := SystemCollector().CollectWorkloads(ctx)
			if workloadErr != nil {
				return nil, workloadErr
			}
			return json.Marshal(w)
		default:
			s, err = SystemCollector().Collect(ctx)
		}
		if err != nil {
			return nil, err
		}
		return json.Marshal(s)
	}}
	return factsReader
}

func bootID() string {
	b, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return scenarioruntime.HealthStatusUnknown
	}
	return strings.TrimSpace(string(b))
}

// CollectGPUFacts performs only the NVIDIA and Docker GPU probes needed to
// decide whether a resource should receive its GPU runtime overlay. It avoids
// unrelated platform probes (notably credential-store discovery) so a slow
// desktop probe cannot make a healthy GPU resource fall back to a CPU image.
func CollectGPUFacts(ctx context.Context) (Snapshot, error) {
	var snapshot Snapshot
	if raw, err := sharedFactsReader().Read(ctx, "gpu"); err == nil && json.Unmarshal(raw, &snapshot) == nil {
		return snapshot, nil
	}
	return SystemCollector().CollectGPUFacts(ctx)
}

func (c Collector) CollectGPUFacts(ctx context.Context) (Snapshot, error) {
	c = c.withDefaults()
	now := c.Clock.Now()
	snap := Snapshot{OS: c.GOOS, Arch: c.GOARCH, RuntimeTools: map[string]Tool{}, ProbeStatuses: map[string]string{}, FieldProvenance: map[string]Provenance{}}
	c.collectDevices(ctx, &snap, now)
	c.collectNvidiaGPUs(ctx, &snap, now)
	c.linkNvidiaDevices(ctx, &snap, now)
	c.collectDockerGPU(ctx, &snap, now)
	c.collectDarwinGPUs(ctx, &snap, now)
	c.collectWindowsGPUs(ctx, &snap, now)
	c.collectNvidiaNodeAccess(&snap, now)
	c.collectROCm(ctx, &snap, now)
	c.collectVulkanICDs(&snap, now)
	c.collectMemory(&snap, now)
	c.collectAppleSiliconGPU(&snap, now)
	return snap, nil
}

func (c Collector) Collect(ctx context.Context) (Snapshot, error) {
	c = c.withDefaults()
	now := c.Clock.Now()
	snap := Snapshot{
		OS:              c.GOOS,
		Arch:            c.GOARCH,
		RuntimeTools:    map[string]Tool{},
		ProbeStatuses:   map[string]string{},
		FieldProvenance: map[string]Provenance{},
	}
	if c.CPUCount != nil {
		snap.CPU.Cores = c.CPUCount()
		snap.FieldProvenance["cpu.cores"] = Provenance{
			SourceKind: SourceKindRuntime,
			Source:     "runtime.NumCPU",
			ObservedAt: now,
			Confidence: "high",
		}
	}

	c.collectPlatformFacts(ctx, &snap, now)
	c.collectMemory(&snap, now)
	c.collectLoad(&snap, now)
	c.collectDevices(ctx, &snap, now)
	c.collectNvidiaGPUs(ctx, &snap, now)
	c.linkNvidiaDevices(ctx, &snap, now)
	c.collectDarwinGPUs(ctx, &snap, now)
	c.collectWindowsGPUs(ctx, &snap, now)
	c.collectDockerGPU(ctx, &snap, now)
	c.collectNvidiaNodeAccess(&snap, now)
	c.collectROCm(ctx, &snap, now)
	c.collectVulkanICDs(&snap, now)
	c.collectAppleSiliconGPU(&snap, now)
	c.collectAppleToolchain(ctx, &snap, now)
	c.collectAndroidToolchain(ctx, &snap, now)
	return snap, nil
}

func (c Collector) withDefaults() Collector {
	if c.Commands == nil {
		c.Commands = osCommandRunner{}
	}
	if c.Files == nil {
		c.Files = osFileReader{}
	}
	if c.Env == nil {
		c.Env = osEnvReader{}
	}
	if c.Clock == nil {
		c.Clock = systemClock{}
	}
	if c.GOOS == "" {
		c.GOOS = runtime.GOOS
	}
	if c.GOARCH == "" {
		c.GOARCH = runtime.GOARCH
	}
	if c.CPUCount == nil {
		c.CPUCount = runtime.NumCPU
	}
	return c
}

func (c Collector) collectLoad(snap *Snapshot, observedAt time.Time) {
	if hostreqspec.PlatformFromGOOS(snap.OS) != hostreqspec.PlatformLinux {
		if collectPlatformLoad(snap, observedAt) {
			return
		}
		snap.ProbeStatuses["load"] = collectorUnsupported
		return
	}
	data, err := c.Files.ReadFile("/proc/loadavg")
	if err != nil {
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("read /proc/loadavg: %v", err))
		snap.ProbeStatuses["load"] = scenarioruntime.StatusFailed
		return
	}
	load, err := ParseLinuxLoadavg(string(data), snap.CPU.Cores)
	if err != nil {
		snap.Warnings = append(snap.Warnings, err.Error())
		snap.ProbeStatuses["load"] = scenarioruntime.StatusFailed
		return
	}
	snap.Load = load
	snap.ProbeStatuses["load"] = "ok"
	snap.FieldProvenance["load"] = Provenance{
		SourceKind: SourceKindFile,
		Source:     "linux procfs",
		ObservedAt: observedAt,
		Confidence: "high",
		File:       "/proc/loadavg",
	}
}

func (c Collector) collectMemory(snap *Snapshot, observedAt time.Time) {
	switch hostreqspec.PlatformFromGOOS(snap.OS) {
	case hostreqspec.PlatformLinux:
		data, err := c.Files.ReadFile("/proc/meminfo")
		if err != nil {
			snap.Warnings = append(snap.Warnings, fmt.Sprintf("read /proc/meminfo: %v", err))
			snap.ProbeStatuses["memory"] = scenarioruntime.StatusFailed
			return
		}
		mem, swap, err := ParseLinuxMeminfo(string(data))
		if err != nil {
			snap.Warnings = append(snap.Warnings, err.Error())
			snap.ProbeStatuses["memory"] = scenarioruntime.StatusFailed
			return
		}
		snap.Memory = mem
		snap.Swap = swap
		snap.ProbeStatuses["memory"] = "ok"
		snap.FieldProvenance["memory.total_bytes"] = Provenance{
			SourceKind: SourceKindFile,
			Source:     "linux procfs",
			ObservedAt: observedAt,
			Confidence: "high",
			File:       "/proc/meminfo",
		}
	case hostreqspec.PlatformMacOS:
		out, err := c.Commands.Run(context.Background(), "sysctl", "-n", "hw.memsize")
		if err != nil {
			snap.Warnings = append(snap.Warnings, fmt.Sprintf("sysctl hw.memsize: %v", err))
			snap.ProbeStatuses["memory"] = scenarioruntime.StatusFailed
			return
		}
		total, err := ParseUintBytes(strings.TrimSpace(string(out)))
		if err != nil {
			snap.Warnings = append(snap.Warnings, fmt.Sprintf("parse hw.memsize: %v", err))
			snap.ProbeStatuses["memory"] = scenarioruntime.StatusFailed
			return
		}
		snap.Memory.TotalBytes = total
		snap.ProbeStatuses["memory"] = "ok"
		snap.FieldProvenance["memory.total_bytes"] = Provenance{
			SourceKind: SourceKindCommand,
			Source:     "darwin sysctl",
			ObservedAt: observedAt,
			Confidence: "high",
			Command:    "sysctl -n hw.memsize",
		}
	case hostreqspec.PlatformWindows:
		out, err := c.Commands.Run(context.Background(), "wmic", "ComputerSystem", "get", "TotalPhysicalMemory", "/Value")
		if err != nil {
			snap.Warnings = append(snap.Warnings, fmt.Sprintf("wmic TotalPhysicalMemory: %v", err))
			snap.ProbeStatuses["memory"] = scenarioruntime.StatusFailed
			return
		}
		total, err := ParseWindowsTotalPhysicalMemory(string(out))
		if err != nil {
			snap.Warnings = append(snap.Warnings, err.Error())
			snap.ProbeStatuses["memory"] = scenarioruntime.StatusFailed
			return
		}
		snap.Memory.TotalBytes = total
		snap.ProbeStatuses["memory"] = "ok"
		snap.FieldProvenance["memory.total_bytes"] = Provenance{
			SourceKind: SourceKindCommand,
			Source:     "windows wmic",
			ObservedAt: observedAt,
			Confidence: "medium",
			Command:    "wmic ComputerSystem get TotalPhysicalMemory /Value",
		}
	default:
		snap.ProbeStatuses["memory"] = collectorUnsupported
	}
}

func (c Collector) collectNvidiaGPUs(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	hostOS := hostreqspec.PlatformFromGOOS(snap.OS)
	if hostOS != hostreqspec.PlatformLinux && hostOS != hostreqspec.PlatformMacOS && hostOS != hostreqspec.PlatformWindows {
		snap.ProbeStatuses["nvidia_gpu"] = collectorUnsupported
		return
	}
	path, err := c.Commands.LookPath("nvidia-smi")
	snap.RuntimeTools["nvidia-smi"] = Tool{Present: err == nil, Path: path}
	if err != nil {
		snap.ProbeStatuses["nvidia_gpu"] = "not_present"
		return
	}
	query := "--query-gpu=index,name,uuid,driver_version,utilization.gpu,utilization.memory,memory.total,memory.used,temperature.gpu,fan.speed,power.draw,power.limit,clocks.sm,clocks.mem"
	out, err := c.Commands.Run(ctx, "nvidia-smi", query, "--format=csv,noheader,nounits")
	if err != nil {
		if strings.Contains(string(out), "No devices were found") {
			snap.ProbeStatuses["nvidia_gpu"] = collectorNoDevices
			return
		}
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("nvidia-smi query gpu: %v", err))
		snap.ProbeStatuses["nvidia_gpu"] = scenarioruntime.StatusFailed
		return
	}
	gpus, warnings, err := ParseNvidiaDetailedGPUCSV(string(out))
	if err != nil {
		snap.Warnings = append(snap.Warnings, err.Error())
		snap.ProbeStatuses["nvidia_gpu"] = scenarioruntime.StatusFailed
		return
	}
	snap.GPUs = append(snap.GPUs, gpus...)
	c.collectNvidiaComputeCapabilities(ctx, snap, observedAt)
	if hostreqspec.PlatformFromGOOS(snap.OS) == hostreqspec.PlatformLinux {
		snap.NvidiaDeviceNodes = linuxNvidiaDeviceNodes()
	}
	snap.Warnings = append(snap.Warnings, warnings...)
	snap.ProbeStatuses["nvidia_gpu"] = "ok"
	snap.FieldProvenance["gpus"] = Provenance{
		SourceKind: SourceKindCommand,
		Source:     "nvidia-smi",
		ObservedAt: observedAt,
		Confidence: "high",
		Command:    "nvidia-smi " + query + " --format=csv,noheader,nounits",
	}
	c.collectNvidiaGPUProcesses(ctx, snap, observedAt)
}

func (c Collector) collectNvidiaComputeCapabilities(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	const query = "--query-gpu=index,compute_cap"
	out, err := c.Commands.Run(ctx, "nvidia-smi", query, "--format=csv,noheader,nounits")
	if err != nil {
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("nvidia-smi query compute capability: %v", err))
		snap.ProbeStatuses["nvidia_gpu_compute_capability"] = scenarioruntime.StatusFailed
		return
	}
	capabilities := ParseNvidiaComputeCapabilityCSV(string(out))
	for index := range snap.GPUs {
		if capability, ok := capabilities[snap.GPUs[index].Index]; ok {
			snap.GPUs[index].CUDAComputeCapability = capability
		}
	}
	snap.ProbeStatuses["nvidia_gpu_compute_capability"] = "ok"
	snap.FieldProvenance["gpus.cuda_compute_capability"] = Provenance{
		SourceKind: SourceKindCommand,
		Source:     "nvidia-smi",
		ObservedAt: observedAt,
		Confidence: "high",
		Command:    "nvidia-smi " + query + " --format=csv,noheader,nounits",
	}
}

func linuxNvidiaDeviceNodes() []string {
	paths, _ := filepath.Glob("/dev/nvidia*")
	nodes := make([]string, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err == nil && info.Mode()&os.ModeCharDevice != 0 {
			nodes = append(nodes, path)
		}
	}
	sort.Strings(nodes)
	return nodes
}

func (c Collector) collectNvidiaGPUProcesses(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	out, err := c.Commands.Run(ctx, "nvidia-smi", "--query-compute-apps=pid,process_name,used_memory,gpu_uuid", "--format=csv,noheader,nounits")
	if err != nil {
		if strings.Contains(string(out), "No running processes found") {
			return
		}
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("nvidia-smi query compute apps: %v", err))
		snap.ProbeStatuses["nvidia_gpu_processes"] = scenarioruntime.StatusFailed
		return
	}
	processes, warnings, err := ParseNvidiaComputeAppsCSV(string(out))
	if err != nil {
		snap.Warnings = append(snap.Warnings, err.Error())
		snap.ProbeStatuses["nvidia_gpu_processes"] = scenarioruntime.StatusFailed
		return
	}
	indexByUUID := map[string]int{}
	for _, gpu := range snap.GPUs {
		if gpu.UUID != "" {
			indexByUUID[gpu.UUID] = gpu.Index
		}
	}
	for i := range processes {
		if index, ok := indexByUUID[processes[i].GPUUUID]; ok {
			processes[i].GPUIndex = index
		}
	}
	snap.GPUProcesses = append(snap.GPUProcesses, processes...)
	snap.Warnings = append(snap.Warnings, warnings...)
	snap.ProbeStatuses["nvidia_gpu_processes"] = "ok"
	snap.FieldProvenance["gpu_processes"] = Provenance{
		SourceKind: SourceKindCommand,
		Source:     "nvidia-smi",
		ObservedAt: observedAt,
		Confidence: "medium",
		Command:    "nvidia-smi --query-compute-apps=pid,process_name,used_memory,gpu_uuid --format=csv,noheader,nounits",
	}
}

func (c Collector) collectDockerGPU(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	path, err := c.Commands.LookPath("docker")
	snap.RuntimeTools["docker"] = Tool{Present: err == nil, Path: path}
	if err != nil {
		snap.ProbeStatuses["docker_gpu"] = "docker_not_present"
		return
	}
	out, err := c.Commands.Run(ctx, "docker", "info")
	if err != nil {
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("docker info: %v", err))
		snap.ProbeStatuses["docker_gpu"] = scenarioruntime.StatusFailed
		return
	}
	snap.DockerGPU.NvidiaRuntime = DockerInfoHasNvidiaRuntime(string(out))
	snap.ProbeStatuses["docker_gpu"] = "ok"
	snap.FieldProvenance["docker_gpu.nvidia_runtime"] = Provenance{
		SourceKind: SourceKindCommand,
		Source:     "docker info",
		ObservedAt: observedAt,
		Confidence: "medium",
		Command:    "docker info",
	}
}

func (c Collector) collectDarwinGPUs(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	if hostreqspec.PlatformFromGOOS(snap.OS) != hostreqspec.PlatformMacOS {
		return
	}
	path, err := c.Commands.LookPath("system_profiler")
	snap.RuntimeTools["system_profiler"] = Tool{Present: err == nil, Path: path}
	if err != nil {
		snap.ProbeStatuses["darwin_gpu"] = collectorToolNotPresent
		return
	}
	out, err := c.Commands.Run(ctx, "system_profiler", "SPDisplaysDataType")
	if err != nil {
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("system_profiler SPDisplaysDataType: %v", err))
		snap.ProbeStatuses["darwin_gpu"] = scenarioruntime.StatusFailed
		return
	}
	gpus := ParseSystemProfilerGPUs(string(out))
	if len(gpus) == 0 {
		snap.ProbeStatuses["darwin_gpu"] = collectorNoDevices
		return
	}
	snap.GPUs = append(snap.GPUs, gpus...)
	snap.ProbeStatuses["darwin_gpu"] = "ok"
	snap.FieldProvenance["gpus.darwin"] = Provenance{
		SourceKind: SourceKindCommand,
		Source:     "system_profiler",
		ObservedAt: observedAt,
		Confidence: "medium",
		Command:    "system_profiler SPDisplaysDataType",
	}
}

func (c Collector) collectWindowsGPUs(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	if hostreqspec.PlatformFromGOOS(snap.OS) != hostreqspec.PlatformWindows {
		return
	}
	path, err := c.Commands.LookPath("wmic")
	snap.RuntimeTools["wmic"] = Tool{Present: err == nil, Path: path}
	if err != nil {
		snap.ProbeStatuses["windows_gpu"] = collectorToolNotPresent
		return
	}
	out, err := c.Commands.Run(ctx, "wmic", "path", "win32_VideoController", "get", "name")
	if err != nil {
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("wmic gpu: %v", err))
		snap.ProbeStatuses["windows_gpu"] = scenarioruntime.StatusFailed
		return
	}
	gpus := ParseWindowsGPUNames(string(out))
	if len(gpus) == 0 {
		snap.ProbeStatuses["windows_gpu"] = collectorNoDevices
		return
	}
	snap.GPUs = append(snap.GPUs, gpus...)
	snap.ProbeStatuses["windows_gpu"] = "ok"
	snap.FieldProvenance["gpus.windows"] = Provenance{
		SourceKind: SourceKindCommand,
		Source:     "windows wmic",
		ObservedAt: observedAt,
		Confidence: "medium",
		Command:    "wmic path win32_VideoController get name",
	}
}

func (c Collector) collectAppleSiliconGPU(snap *Snapshot, observedAt time.Time) {
	if hostreqspec.PlatformFromGOOS(snap.OS) != hostreqspec.PlatformMacOS || snap.Arch != "arm64" || snap.Memory.TotalBytes == 0 || len(snap.GPUs) > 0 {
		return
	}
	snap.GPUs = append(snap.GPUs, GPU{
		Index:     len(snap.GPUs),
		Name:      "Apple GPU (unified)",
		VRAMBytes: snap.Memory.TotalBytes / collectorParameterA,
		Source:    "darwin-unified-memory",
	})
	snap.FieldProvenance["gpus.apple_unified"] = Provenance{
		SourceKind: SourceKindDerived,
		Source:     "darwin arm64 unified memory estimate",
		ObservedAt: observedAt,
		Confidence: "medium",
	}
}
