package agentsessioncontainment

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// The safeguard's one unit and its four typed settings; see doc.go.
const (
	SliceName   = "vrooli-agents"
	SliceUnit   = SliceName + ".slice"
	unitRelPath = ".config/systemd/user/" + SliceUnit
	// The sibling slice long-running services live in. A service placed
	// here is not charged to the session that started it, which is what
	// made the agent ceiling shrink with every session that ran one.
	ServicesSliceName   = "vrooli-services"
	ServicesSliceUnit   = ServicesSliceName + ".slice"
	servicesUnitRelPath = ".config/systemd/user/" + ServicesSliceUnit
	cgroupMount         = "/sys/fs/cgroup"
	percent             = 100

	// Defaults are the plan's D3 values; the operator changes them through
	// the typed config, never here.
	DefaultCPUWeight         = 50
	DefaultMemoryHighPercent = 50
	DefaultMemoryMaxPercent  = 60
	DefaultTasksMax          = 16384
	// Occupancy thresholds: a ceiling is a safety property only while there
	// is room under it, so the safeguard reports how full it is. Warn early
	// enough that an operator can act before the kernel starts refusing
	// forks, which it does without warning at 100%.
	DefaultOccupancyWarnPercent     = 70
	DefaultOccupancyCriticalPercent = 85
	// Services outrank agents for CPU; their memory ceiling throttles and
	// never kills.
	DefaultServicesCPUWeight         = 200
	DefaultServicesMemoryHighPercent = 70
	DefaultServicesTasksMax          = 16384

	maxCPUWeight    = 10000
	minimumTasksMax = 16
	// memoryTolerancePercent absorbs systemd's page rounding when it turns a
	// percentage into bytes.
	memoryTolerancePercent = 1
)

// Settings are the slice's ceilings as the operator configured them.
type Settings struct {
	CPUWeight         int
	MemoryHighPercent int
	MemoryMaxPercent  int
	TasksMax          int
	// OccupancyWarnPercent and OccupancyCriticalPercent are the fullness a
	// ceiling reaches before the safeguard says so.
	OccupancyWarnPercent     int
	OccupancyCriticalPercent int
	// The services slice's ceilings.
	ServicesCPUWeight         int
	ServicesMemoryHighPercent int
	ServicesTasksMax          int
}

// DefaultSettings returns the D3 defaults.
func DefaultSettings() Settings {
	return Settings{
		CPUWeight: DefaultCPUWeight, MemoryHighPercent: DefaultMemoryHighPercent,
		MemoryMaxPercent: DefaultMemoryMaxPercent, TasksMax: DefaultTasksMax,
		OccupancyWarnPercent: DefaultOccupancyWarnPercent, OccupancyCriticalPercent: DefaultOccupancyCriticalPercent,
		ServicesCPUWeight: DefaultServicesCPUWeight, ServicesMemoryHighPercent: DefaultServicesMemoryHighPercent,
		ServicesTasksMax: DefaultServicesTasksMax,
	}
}

// ResolveSettings reads the typed config; a value outside its bounds keeps
// the default so a typo never removes the ceiling.
func ResolveSettings(config map[string]any) Settings {
	s := DefaultSettings()
	if v, ok := intFromConfig(config["cpu_weight"]); ok && v >= 1 && v <= maxCPUWeight {
		s.CPUWeight = v
	}
	if v, ok := intFromConfig(config["memory_high_percent"]); ok && v >= 1 && v <= percent {
		s.MemoryHighPercent = v
	}
	if v, ok := intFromConfig(config["memory_max_percent"]); ok && v >= 1 && v <= percent {
		s.MemoryMaxPercent = v
	}
	if s.MemoryHighPercent > s.MemoryMaxPercent {
		s.MemoryHighPercent = s.MemoryMaxPercent
	}
	if v, ok := intFromConfig(config["tasks_max"]); ok && v >= minimumTasksMax {
		s.TasksMax = v
	}
	if v, ok := intFromConfig(config["occupancy_warn_percent"]); ok && v >= 1 && v <= percent {
		s.OccupancyWarnPercent = v
	}
	if v, ok := intFromConfig(config["occupancy_critical_percent"]); ok && v >= 1 && v <= percent {
		s.OccupancyCriticalPercent = v
	}
	if s.OccupancyWarnPercent > s.OccupancyCriticalPercent {
		s.OccupancyWarnPercent = s.OccupancyCriticalPercent
	}
	if v, ok := intFromConfig(config["services_cpu_weight"]); ok && v >= 1 && v <= maxCPUWeight {
		s.ServicesCPUWeight = v
	}
	if v, ok := intFromConfig(config["services_memory_high_percent"]); ok && v >= 1 && v <= percent {
		s.ServicesMemoryHighPercent = v
	}
	if v, ok := intFromConfig(config["services_tasks_max"]); ok && v >= minimumTasksMax {
		s.ServicesTasksMax = v
	}
	return s
}

func intFromConfig(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return n, err == nil
	}
	return 0, false
}

// Containment is the platform-go ceiling the launcher applies per session.
func (s Settings) Containment() platformgo.Containment {
	return platformgo.Containment{
		CPUWeight:  s.CPUWeight,
		MemoryHigh: strconv.Itoa(s.MemoryHighPercent) + "%",
		MemoryMax:  strconv.Itoa(s.MemoryMaxPercent) + "%",
		TasksMax:   s.TasksMax,
		Slice:      SliceUnit,
		// An agent session is worth killing to save the host; that is what
		// the slice is for.
		ManagedOOMKill: true,
	}
}

// Definition is the slice as a platform-go service definition.
func (s Settings) Definition() platformgo.ServiceDefinition {
	c := s.Containment()
	c.Slice = ""
	return platformgo.ServiceDefinition{
		Name:        SliceName,
		Description: "Vrooli coding-agent sessions",
		Kind:        platformgo.KindSlice,
		Scope:       platformgo.ScopeUser,
		Restart:     platformgo.RestartPolicy{Mode: platformgo.RestartNever},
		Protections: platformgo.Protections{Containment: c},
	}
}

// ServicesContainment is the ceiling long-running scenario and resource
// services carry. It is deliberately not the agent ceiling.
//
// Services outrank agent sessions for CPU: the fleet is what the host exists
// to run. There is no MemoryMax and no ManagedOOMMemoryPressure=kill, only a
// MemoryHigh that throttles: a hard ceiling here would have the kernel kill
// llama-server or qdrant, which is the outcome the agent slice exists to
// prevent, not one to reproduce for services. TasksMax is a ceiling a runaway
// service stops at, well clear of the fleet's real use.
func (s Settings) ServicesContainment() platformgo.Containment {
	return platformgo.Containment{
		CPUWeight:  s.ServicesCPUWeight,
		MemoryHigh: strconv.Itoa(s.ServicesMemoryHighPercent) + "%",
		TasksMax:   s.ServicesTasksMax,
	}
}

// ServicesDefinition is the services slice as a service definition.
func (s Settings) ServicesDefinition() platformgo.ServiceDefinition {
	return platformgo.ServiceDefinition{
		Name:        ServicesSliceName,
		Description: "Vrooli scenario and resource services",
		Kind:        platformgo.KindSlice,
		Scope:       platformgo.ScopeUser,
		Restart:     platformgo.RestartPolicy{Mode: platformgo.RestartNever},
		Protections: platformgo.Protections{Containment: s.ServicesContainment()},
	}
}

// Render produces the unit file content.
func Render(s Settings) (platformgo.RenderedArtifact, error) {
	return platformgo.RenderSystemdSlice(s.Definition())
}

// RenderServices produces the services slice unit file content.
func RenderServices(s Settings) (platformgo.RenderedArtifact, error) {
	return platformgo.RenderSystemdSlice(s.ServicesDefinition())
}

// Seams stubbed by tests. Every host interaction goes through one so a unit
// test never touches the real user manager.
var (
	homeDir = func() (string, error) {
		return config.HomeDir()
	}
	validateFn    = platformgo.ValidateSystemd
	installFileFn = hostreqkit.InstallUserFile
)

func userSystemctl(args ...string) ([]byte, error) {
	name, commandArgs := hostreqkit.InvokingUserCommand("systemctl", append([]string{"--user"}, args...)...)
	return hostreqkit.CombinedOutputFn(name, commandArgs...)
}

// live is what the user manager and the kernel say about the slice now.
type live struct {
	ActiveState  string
	ControlGroup string
	MemoryMax    int64
	TasksMax     int64
	CPUWeight    int64
	CgroupMemMax string
	CgroupPids   string
	CgroupWeight string
	// Occupancy is how full the ceilings are now. A ceiling with no reading
	// is the shape that let the slice reach its task limit on 2026-09-04
	// with every host-level bar green.
	Occupancy platformgo.Occupancy
	// Sessions is what the slice holds: one entry per session scope.
	Sessions []agentscope.Entry
}

// observeSlice reads the live slice. The error means the probe itself could
// not run, which the caller reports as undetermined, never as ok.
func observeSlice() (live, error) {
	out, err := userSystemctl("show", SliceUnit, "-p", "ActiveState", "-p", "ControlGroup", "-p", "MemoryMax", "-p", "TasksMax", "-p", "CPUWeight")
	if err != nil {
		return live{}, fmt.Errorf("systemctl --user show %s: %w: %s", SliceUnit, err, strings.TrimSpace(string(out)))
	}
	var l live
	for _, line := range strings.Split(string(out), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "ActiveState":
			l.ActiveState = value
		case "ControlGroup":
			l.ControlGroup = value
		case "MemoryMax":
			l.MemoryMax = parseSystemdNumber(value)
		case "TasksMax":
			l.TasksMax = parseSystemdNumber(value)
		case "CPUWeight":
			l.CPUWeight = parseSystemdNumber(value)
		}
	}
	if l.ControlGroup != "" {
		dir := filepath.Join(cgroupMount, l.ControlGroup)
		l.CgroupMemMax = readTrimmed(filepath.Join(dir, "memory.max"))
		l.CgroupPids = readTrimmed(filepath.Join(dir, "pids.max"))
		l.CgroupWeight = readTrimmed(filepath.Join(dir, "cpu.weight"))
		slice := agentscope.Ref(l.ControlGroup)
		// A failed read leaves every reading Unknown. A zero value would
		// publish "0 tasks of 0" as evidence, which is the shape that lets
		// an unreadable ceiling be mistaken for an empty one.
		l.Occupancy = unknownOccupancy()
		if occupancy, err := platformgo.ScopeOccupancy(slice); err == nil {
			l.Occupancy = occupancy
		}
		l.Sessions, _ = agentscope.NewReader().Census(slice)
	}
	return l, nil
}

// unknownOccupancy is the reading of a ceiling that could not be read.
func unknownOccupancy() platformgo.Occupancy {
	return platformgo.Occupancy{
		Tasks: platformgo.Unknown, TasksMax: platformgo.Unknown,
		MemoryBytes: platformgo.Unknown, MemoryHighBytes: platformgo.Unknown,
		MemoryMaxBytes: platformgo.Unknown, MemoryHighEvents: platformgo.Unknown,
	}
}

// parseSystemdNumber reads a systemd numeric property; "infinity" and
// "[not set]" are -1 so they never equal a configured value.
func parseSystemdNumber(value string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return -1
	}
	return n
}

func readTrimmed(path string) string {
	data, err := hostreqkit.ReadFileFn(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// physicalMemoryBytes reads MemTotal from /proc/meminfo.
func physicalMemoryBytes() int64 {
	data, err := hostreqkit.ReadFileFn("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "MemTotal:"); ok {
			fields := strings.Fields(rest)
			if len(fields) > 0 {
				if kb, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
					return kb * 1024
				}
			}
		}
	}
	return 0
}

// occupancyNotes reports how full the ceilings are and what is holding them.
// A ceiling is only a safety property while there is room under it, and
// until 2026-09-04 nothing on this host read the other side of the ratio: a
// slice reached its task ceiling with every host-level bar green, and the
// first symptom was an unrelated component reporting its database corrupt.
//
// These are notes, not mismatches: a full slice is a correctly configured
// slice, so it must not read as drift. It must read as a warning.
func occupancyNotes(l live, s Settings) []string {
	var out []string
	tasks := l.Occupancy.TaskSaturation()
	switch {
	case tasks == platformgo.SaturationUnknown:
		out = append(out, "task occupancy of "+SliceUnit+" is undetermined: the pids controller reported no reading")
	case tasks >= float64(s.OccupancyCriticalPercent)/percent:
		out = append(out, fmt.Sprintf("%s holds %d of %d tasks (%.0f%%): at this ceiling the kernel refuses the fork of every new agent session, which surfaces to the operator as an unrelated startup failure", SliceUnit, l.Occupancy.Tasks, l.Occupancy.TasksMax, tasks*percent))
	case tasks >= float64(s.OccupancyWarnPercent)/percent:
		out = append(out, fmt.Sprintf("%s holds %d of %d tasks (%.0f%%)", SliceUnit, l.Occupancy.Tasks, l.Occupancy.TasksMax, tasks*percent))
	}
	memory := l.Occupancy.MemorySaturation()
	if memory >= float64(s.OccupancyWarnPercent)/percent {
		out = append(out, fmt.Sprintf("%s holds %d of %d memory bytes (%.0f%%); ManagedOOMMemoryPressure=kill acts inside the slice above the ceiling", SliceUnit, l.Occupancy.MemoryBytes, l.Occupancy.MemoryMaxBytes, memory*percent))
	}
	if orphans := agentscope.Agentless(l.Sessions); len(orphans) > 0 {
		held := agentscope.AgentlessTasks(l.Sessions)
		out = append(out, fmt.Sprintf("%d of %d session scopes hold %d tasks with no coding agent left in them: those sessions ended and left long-running children, and nothing releases their share of the ceiling", len(orphans), len(l.Sessions), held))
	}
	return out
}

// liveMismatches compares the live slice with the configured ceilings and
// names every difference.
func liveMismatches(l live, s Settings) []string {
	var out []string
	if l.ActiveState != "active" {
		out = append(out, fmt.Sprintf("%s is %q, not active", SliceUnit, l.ActiveState))
	}
	if l.TasksMax != int64(s.TasksMax) {
		out = append(out, fmt.Sprintf("live TasksMax %d, want %d", l.TasksMax, s.TasksMax))
	}
	if l.CPUWeight != int64(s.CPUWeight) {
		out = append(out, fmt.Sprintf("live CPUWeight %d, want %d", l.CPUWeight, s.CPUWeight))
	}
	if physical := physicalMemoryBytes(); physical > 0 {
		want := physical * int64(s.MemoryMaxPercent) / percent
		tolerance := physical * memoryTolerancePercent / percent
		if l.MemoryMax < want-tolerance || l.MemoryMax > want+tolerance {
			out = append(out, fmt.Sprintf("live MemoryMax %d bytes, want %d%% of %d (%d)", l.MemoryMax, s.MemoryMaxPercent, physical, want))
		}
	}
	if l.ControlGroup == "" {
		out = append(out, "the user manager reports no control group for the slice")
		return out
	}
	if l.CgroupPids != strconv.Itoa(s.TasksMax) {
		out = append(out, fmt.Sprintf("cgroup pids.max %q, want %d", l.CgroupPids, s.TasksMax))
	}
	if l.CgroupWeight != strconv.Itoa(s.CPUWeight) {
		out = append(out, fmt.Sprintf("cgroup cpu.weight %q, want %d", l.CgroupWeight, s.CPUWeight))
	}
	if l.CgroupMemMax != strconv.FormatInt(l.MemoryMax, 10) {
		out = append(out, fmt.Sprintf("cgroup memory.max %q differs from the manager's MemoryMax %d", l.CgroupMemMax, l.MemoryMax))
	}
	return out
}

type handler struct{ manifest hostreqkit.SafeguardManifest }

// NewHandler is the registry constructor.
func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

func isLinux(host hostreqkit.Host) bool {
	return hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformLinux
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported
	s := ResolveSettings(requirement.Config)
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if !isLinux(host) {
		// No slice off Linux; the launcher applies the ceilings per session
		// and this safeguard only reports what they are.
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, fmt.Sprintf("no slice on %s; the launcher applies per-session ceilings (tasks %d, memory %d%% of physical) through the rlimit shim on macOS and a Job Object on Windows; fixture-verified", host.OS, s.TasksMax, s.MemoryMaxPercent))
		return status
	}
	if !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "requires a systemd user manager to own the agent slice")
		return status
	}
	home, err := homeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "could not determine the invoking user's home directory")
		return status
	}
	artifact, err := Render(s)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "render "+SliceUnit+": "+err.Error())
		return status
	}
	verdict := validateFn(artifact, platformgo.ScopeUser)
	recordVerdict(&status, verdict)
	content := artifact.Primary().Content
	unitPath := filepath.Join(home, filepath.FromSlash(unitRelPath))
	var pending []string
	if verdict.Rejected() {
		pending = append(pending, "the native validator rejected the rendered slice: "+verdict.Output)
	}
	if !hostreqkit.FileContentMatches(unitPath, content) {
		pending = append(pending, unitPath+" missing or stale")
	}
	servicesArtifact, err := RenderServices(s)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "render "+ServicesSliceUnit+": "+err.Error())
		return status
	}
	servicesPath := filepath.Join(home, filepath.FromSlash(servicesUnitRelPath))
	if !hostreqkit.FileContentMatches(servicesPath, servicesArtifact.Primary().Content) {
		pending = append(pending, servicesPath+" missing or stale")
	}
	l, err := observeSlice()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Evidence["probe"] = "undetermined"
		status.Notes = append(status.Notes, "undetermined: "+err.Error())
		status.Notes = append(status.Notes, pending...)
		return status
	}
	status.Evidence["live"] = map[string]any{
		"active_state": l.ActiveState, "control_group": l.ControlGroup,
		"memory_max": l.MemoryMax, "tasks_max": l.TasksMax, "cpu_weight": l.CPUWeight,
		"cgroup_memory_max": l.CgroupMemMax, "cgroup_pids_max": l.CgroupPids, "cgroup_cpu_weight": l.CgroupWeight,
	}
	status.Evidence["occupancy"] = map[string]any{
		"tasks": l.Occupancy.Tasks, "tasks_max": l.Occupancy.TasksMax,
		"task_saturation": l.Occupancy.TaskSaturation(),
		"memory_bytes":    l.Occupancy.MemoryBytes, "memory_max_bytes": l.Occupancy.MemoryMaxBytes,
		"memory_saturation":  l.Occupancy.MemorySaturation(),
		"memory_high_events": l.Occupancy.MemoryHighEvents,
		"scopes":             len(l.Sessions), "agentless_scopes": len(agentscope.Agentless(l.Sessions)),
		"agentless_tasks": agentscope.AgentlessTasks(l.Sessions),
	}
	occupancy := occupancyNotes(l, s)
	pending = append(pending, liveMismatches(l, s)...)
	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes,
			fmt.Sprintf("%s is active with CPUWeight=%d MemoryHigh=%d%% MemoryMax=%d%% TasksMax=%d", SliceUnit, s.CPUWeight, s.MemoryHighPercent, s.MemoryMaxPercent, s.TasksMax),
			fmt.Sprintf("%s carries CPUWeight=%d MemoryHigh=%d%% TasksMax=%d; long-running services are placed there so a service is never charged to the session that started it", ServicesSliceUnit, s.ServicesCPUWeight, s.ServicesMemoryHighPercent, s.ServicesTasksMax))
		status.Notes = append(status.Notes, occupancy...)
		return status
	}
	status.Notes = append(status.Notes, occupancy...)
	status.ExecutionState = hostreqkit.ExecutionPending
	status.Notes = append(status.Notes, pending...)
	return status
}

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	s := ResolveSettings(status.Config)
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes,
			fmt.Sprintf("dry-run: would install %s (CPUWeight=%d MemoryHigh=%d%% MemoryMax=%d%% TasksMax=%d) and start it", SliceUnit, s.CPUWeight, s.MemoryHighPercent, s.MemoryMaxPercent, s.TasksMax),
			fmt.Sprintf("dry-run: would install %s (CPUWeight=%d MemoryHigh=%d%% TasksMax=%d, no hard ceiling and no managed-OOM kill) and start it", ServicesSliceUnit, s.ServicesCPUWeight, s.ServicesMemoryHighPercent, s.ServicesTasksMax))
		return status, nil
	}
	fail := func(step string, err error) (hostreqkit.ItemStatus, error) {
		status.Applied = false
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, step+": "+err.Error())
		return status, nil
	}
	home, err := homeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return fail("home", fmt.Errorf("could not determine the invoking user's home directory"))
	}
	artifact, err := Render(s)
	if err != nil {
		return fail("render "+SliceUnit, err)
	}
	verdict := validateFn(artifact, platformgo.ScopeUser)
	recordVerdict(&status, verdict)
	if verdict.Rejected() {
		return fail("validate "+SliceUnit, fmt.Errorf("the native validator rejected the rendered slice; nothing was installed: %s", verdict.Output))
	}
	content := artifact.Primary().Content
	unitPath := filepath.Join(home, filepath.FromSlash(unitRelPath))
	if !hostreqkit.FileContentMatches(unitPath, content) {
		if err := installFileFn(unitPath, content, opts); err != nil {
			return fail("install "+unitPath, err)
		}
	}
	servicesArtifact, err := RenderServices(s)
	if err != nil {
		return fail("render "+ServicesSliceUnit, err)
	}
	servicesVerdict := validateFn(servicesArtifact, platformgo.ScopeUser)
	if servicesVerdict.Rejected() {
		return fail("validate "+ServicesSliceUnit, fmt.Errorf("the native validator rejected the rendered slice; nothing was installed: %s", servicesVerdict.Output))
	}
	servicesContent := servicesArtifact.Primary().Content
	servicesPath := filepath.Join(home, filepath.FromSlash(servicesUnitRelPath))
	if !hostreqkit.FileContentMatches(servicesPath, servicesContent) {
		if err := installFileFn(servicesPath, servicesContent, opts); err != nil {
			return fail("install "+servicesPath, err)
		}
	}
	if out, err := userSystemctl("daemon-reload"); err != nil {
		return fail("systemctl --user daemon-reload", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out))))
	}
	for _, unit := range []string{SliceUnit, ServicesSliceUnit} {
		if out, err := userSystemctl("start", unit); err != nil {
			return fail("systemctl --user start "+unit, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out))))
		}
	}
	// A mutation proves itself: only a re-inspection that reads the live
	// slice back with the configured values counts as applied.
	requirement := hostreqspec.ResolvedRequirement{Name: status.Name, Kind: hostreqspec.KindSafeguard, Required: status.Required, Config: status.Config}
	verified := h.Inspect(host, requirement)
	status.Evidence = verified.Evidence
	if !verified.Applied {
		return fail("verify "+SliceUnit, fmt.Errorf("the slice did not read back with the configured ceilings: %s", strings.Join(verified.Notes, "; ")))
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, verified.Notes...)
	return status, nil
}

// recordVerdict stores the native validator's answer as evidence so the
// readiness phase can read it back from `vrooli setup status --json`.
func recordVerdict(status *hostreqkit.ItemStatus, verdict platformgo.Verdict) {
	if status.Evidence == nil {
		status.Evidence = map[string]any{}
	}
	if verdict.State == "" {
		return
	}
	status.Evidence["validator_verdict"] = verdict
	if verdict.State == platformgo.VerdictUnavailable {
		status.Notes = append(status.Notes, "native validator unavailable; rendered slice is unproven: "+verdict.Output)
	}
}
