package platform

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

// ServiceDefinition is the one typed description of a native scheduler unit.
// Every systemd unit, launchd plist and Windows task the control plane or a
// scenario installs is rendered from one of these by the pure renderers in
// render_systemd.go, render_launchd.go and render_windowstask.go, and accepted by
// the matching validator before it is enabled. Nothing else in the tree writes
// a unit body by hand.
type ServiceDefinition struct {
	// Name is the systemd unit base name ("vrooli-autoheal" renders as
	// vrooli-autoheal.service and, for timers, vrooli-autoheal.timer).
	Name string
	// Label is the launchd label (reverse-DNS). Windows task and service
	// names are passed to the installer, not rendered; see CoreUnit.Windows.
	Label string
	// Description is the human-readable summary the native manager displays.
	Description string
	// DocumentationURL is the https URL of the owning source file; see
	// DocumentationURL(ownerPath).
	DocumentationURL string
	// Executable is the absolute path the unit runs.
	Executable string
	// Args are the process arguments after Executable, one token each.
	Args []string
	// Env is the environment the process receives, rendered in key order.
	Env map[string]string
	// WorkingDirectory is optional; empty leaves the manager's default.
	WorkingDirectory string
	// Kind decides the unit shape: a long-lived daemon, a run-to-completion
	// oneshot, or a timer that runs a oneshot on a schedule.
	Kind ServiceKind
	// Schedule is required when Kind is KindTimer and ignored otherwise.
	Schedule *Schedule
	// Restart is the supervision policy for daemons.
	Restart RestartPolicy
	// OnFailureUnit is the systemd unit activated when this one fails
	// (OnFailure=); other targets have no equivalent and ignore it.
	OnFailureUnit string
	// Scope selects the user or the system manager.
	Scope Scope
	// Protections keeps the process schedulable under host pressure.
	Protections Protections
	// StopTimeout bounds the graceful stop before the manager kills the
	// process; zero leaves the manager's default.
	StopTimeout time.Duration
	// Username is the OS principal the unit runs as. It is required for the
	// windows target (the task principal) and optional elsewhere.
	Username string
	// Logs are the stdout/stderr destinations; empty means the manager's
	// default (the journal on systemd, /dev/null on launchd).
	Logs LogPaths
}

// ServiceKind is the unit shape.
type ServiceKind string

const (
	KindDaemon  ServiceKind = "daemon"
	KindOneshot ServiceKind = "oneshot"
	KindTimer   ServiceKind = "timer"
	// KindSlice is a resource-control parent with no process of its own; only
	// systemd renders it. Sessions are born inside it through ContainedCommand.
	KindSlice ServiceKind = "slice"
)

// Scope is the manager namespace a unit is installed into.
type Scope string

const (
	ScopeUser   Scope = "user"
	ScopeSystem Scope = "system"
)

// RestartMode is the supervision policy vocabulary shared by every target.
type RestartMode string

const (
	RestartAlways    RestartMode = "always"
	RestartOnFailure RestartMode = "on-failure"
	RestartNever     RestartMode = "never"
)

// RestartPolicy says when and how quickly a daemon is restarted, and how many
// restarts inside BurstWindow the manager tolerates before it stops trying.
type RestartPolicy struct {
	Mode        RestartMode
	Delay       time.Duration
	BurstLimit  int
	BurstWindow time.Duration
}

// Schedule is a timer's cadence.
type Schedule struct {
	// OnBoot is the delay after boot before the first run.
	OnBoot time.Duration
	// Every is the interval between runs; at least one minute.
	Every time.Duration
	// Persistent runs a missed window as soon as the host is back.
	Persistent bool
}

// Containment is the ceiling vocabulary shared by units, slices and
// sessions: the resources a process tree may consume, expressed once in
// systemd's syntax and rendered per platform (nice and rlimits on launchd,
// a Job Object with quotas on Windows).
type Containment struct {
	// CPUWeight is systemd's CPUWeight= (1..10000, 100 is neutral). Off
	// systemd it maps to nice / task priority through niceForWeight.
	CPUWeight int
	// MemoryHigh is systemd's MemoryHigh= throttling ceiling ("50%", "8G").
	MemoryHigh string
	// MemoryMax is systemd's MemoryMax= hard ceiling ("60%", "8G").
	MemoryMax string
	// TasksMax bounds the number of tasks (threads and processes) in the tree.
	TasksMax int
	// Slice is the parent slice (systemd only): "vrooli-agents.slice".
	Slice string
}

// Protections are the resource-control settings that keep a supervisor
// schedulable and alive under the host pressure it exists to report on: a
// floor (MemoryMin, OOMScoreAdjust) plus the shared ceiling vocabulary.
type Protections struct {
	Containment
	// MemoryMin is systemd's MemoryMin= ("128M").
	MemoryMin string
	// OOMScoreAdjust is the process's oom_score_adj (-1000..1000).
	OOMScoreAdjust int
}

// LogPaths are absolute stdout/stderr files.
type LogPaths struct {
	Stdout string
	Stderr string
}

// Validate rejects a definition no renderer can turn into a unit the native
// manager loads: a missing name or executable, a relative executable, a
// newline in any string (which would end a directive early on every target),
// and a timer without a usable schedule.
func (d ServiceDefinition) Validate() error {
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("platform: service definition needs a name")
	}
	if err := d.Protections.Containment.validate(d.Name); err != nil {
		return err
	}
	if d.Kind == KindSlice {
		return d.validateSlice()
	}
	if strings.TrimSpace(d.Executable) == "" {
		return fmt.Errorf("platform: service %s needs an executable", d.Name)
	}
	if !isAbsolutePath(d.Executable) {
		return fmt.Errorf("platform: service %s executable %q is not absolute", d.Name, d.Executable)
	}
	if strings.TrimSpace(d.DocumentationURL) == "" || !(strings.HasPrefix(d.DocumentationURL, "https://") || strings.HasPrefix(d.DocumentationURL, "http://")) {
		return fmt.Errorf("platform: service %s documentation %q is not a URL", d.Name, d.DocumentationURL)
	}
	for field, value := range d.stringFields() {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("platform: service %s %s contains a newline", d.Name, field)
		}
	}
	switch d.Kind {
	case KindDaemon, KindOneshot:
	case KindTimer:
		if d.Schedule == nil {
			return fmt.Errorf("platform: timer %s needs a schedule", d.Name)
		}
		if d.Schedule.Every < time.Minute {
			return fmt.Errorf("platform: timer %s interval %s is under one minute", d.Name, d.Schedule.Every)
		}
	default:
		return fmt.Errorf("platform: service %s has unknown kind %q", d.Name, d.Kind)
	}
	switch d.Scope {
	case ScopeUser, ScopeSystem:
	default:
		return fmt.Errorf("platform: service %s has unknown scope %q", d.Name, d.Scope)
	}
	switch d.Restart.Mode {
	case RestartAlways, RestartOnFailure, RestartNever:
	default:
		return fmt.Errorf("platform: service %s has unknown restart mode %q", d.Name, d.Restart.Mode)
	}
	return nil
}

// validateSlice is the reduced contract of a resource-control parent: a
// name, a description, a scope, no process.
func (d ServiceDefinition) validateSlice() error {
	if strings.TrimSpace(d.Description) == "" {
		return fmt.Errorf("platform: slice %s needs a description", d.Name)
	}
	if d.Executable != "" || len(d.Args) > 0 || d.Schedule != nil {
		return fmt.Errorf("platform: slice %s cannot carry a process", d.Name)
	}
	for field, value := range d.stringFields() {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("platform: slice %s %s contains a newline", d.Name, field)
		}
	}
	switch d.Scope {
	case ScopeUser, ScopeSystem:
	default:
		return fmt.Errorf("platform: slice %s has unknown scope %q", d.Name, d.Scope)
	}
	return nil
}

// minimumTasksMax is the smallest ceiling that leaves a session able to run
// a shell, an agent and one build.
const minimumTasksMax = 16

func (c Containment) validate(name string) error {
	if c.TasksMax != 0 && c.TasksMax < minimumTasksMax {
		return fmt.Errorf("platform: %s TasksMax %d is under %d", name, c.TasksMax, minimumTasksMax)
	}
	if c.CPUWeight < 0 || c.CPUWeight > 10000 {
		return fmt.Errorf("platform: %s CPUWeight %d is outside 1..10000", name, c.CPUWeight)
	}
	high, highPercent, err := parseMemoryCeiling(c.MemoryHigh)
	if err != nil {
		return fmt.Errorf("platform: %s MemoryHigh: %w", name, err)
	}
	max, maxPercent, err := parseMemoryCeiling(c.MemoryMax)
	if err != nil {
		return fmt.Errorf("platform: %s MemoryMax: %w", name, err)
	}
	if high > 0 && max > 0 && highPercent == maxPercent && high > max {
		return fmt.Errorf("platform: %s MemoryHigh %s exceeds MemoryMax %s", name, c.MemoryHigh, c.MemoryMax)
	}
	return nil
}

func (d ServiceDefinition) stringFields() map[string]string {
	fields := map[string]string{
		"name": d.Name, "label": d.Label, "description": d.Description, "documentation": d.DocumentationURL,
		"executable": d.Executable, "working directory": d.WorkingDirectory, "on-failure unit": d.OnFailureUnit,
		"username": d.Username, "stdout log": d.Logs.Stdout, "stderr log": d.Logs.Stderr, "memory floor": d.Protections.MemoryMin,
	}
	for i, arg := range d.Args {
		fields[fmt.Sprintf("argument %d", i)] = arg
	}
	for key, value := range d.Env {
		fields["environment "+key] = key + "=" + value
	}
	return fields
}

// envKeys returns the environment keys in a stable order so a rendered unit
// compares byte-for-byte across runs.
func (d ServiceDefinition) envKeys() []string {
	keys := make([]string, 0, len(d.Env))
	for key := range d.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// isAbsolutePath accepts both POSIX and Windows absolute forms regardless of
// the host, because renderers run for every target on one machine.
func isAbsolutePath(path string) bool {
	if strings.HasPrefix(path, "/") {
		return true
	}
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return true
	}
	return strings.HasPrefix(path, `\\`)
}

// DocumentationURL returns the Documentation= URL for a source file owner
// path such as "internal/safeguards/autoheal-watchdog/handler.go".
func DocumentationURL(ownerPath string) string {
	return "https://github.com/Vrooli/Vrooli/blob/master/" + strings.TrimPrefix(strings.TrimSpace(ownerPath), "/")
}

// RenderedArtifact is what a renderer produced for one target: one file for
// launchd and Windows, one or two for systemd (a timer renders its service
// unit alongside).
type RenderedArtifact struct {
	Target string         `json:"target"`
	Files  []RenderedFile `json:"files"`
}

// RenderedFile is one native unit body with the basename it must be
// installed under.
type RenderedFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// File returns the rendered file with the given basename.
func (a RenderedArtifact) File(name string) (RenderedFile, bool) {
	for _, file := range a.Files {
		if file.Name == name {
			return file, true
		}
	}
	return RenderedFile{}, false
}

// Primary returns the artifact's first file: the plist, the task XML, or the
// systemd unit that carries the ExecStart.
func (a RenderedArtifact) Primary() RenderedFile {
	if len(a.Files) == 0 {
		return RenderedFile{}
	}
	return a.Files[0]
}

// NormalizeTarget maps every accepted platform token onto the renderer
// vocabulary: "linux", "darwin" or "windows". "macos" is accepted because
// safeguard manifests and the autoheal scenario use the product vocabulary.
func NormalizeTarget(target string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "linux":
		return "linux", nil
	case "darwin", "macos":
		return "darwin", nil
	case "windows":
		return "windows", nil
	default:
		return "", fmt.Errorf("platform: unsupported service target %q", target)
	}
}

// DefaultPathEntries returns the per-target tool directories a Vrooli
// supervisor needs on PATH, most specific first, without checking that they
// exist. The Go toolchain directory matters: the autoheal recovery floor runs
// `go mod download` from inside the unit, and a PATH without it burned a
// breaker slot per attempt on 2026-09-02.
//
// localAppData is only consulted for the windows target (the WinGet links
// directory lives under it); pass "" elsewhere.
func DefaultPathEntries(target, home, localAppData string) []string {
	home = strings.TrimSpace(home)
	var entries []string
	if target == "windows" {
		if localAppData = strings.TrimSpace(localAppData); localAppData != "" {
			entries = append(entries, windowsJoin(localAppData, "Microsoft", "WinGet", "Links"))
		}
		if home != "" {
			entries = append(entries, windowsJoin(home, "AppData", "Local", "Microsoft", "WinGet", "Links"))
		}
	} else {
		entries = append(entries, "/opt/homebrew/bin", "/usr/local/go/bin", "/usr/local/bin")
		if home != "" {
			entries = append(entries,
				filepath.Join(home, ".cargo", "bin"),
				filepath.Join(home, "go", "bin"),
				filepath.Join(home, ".local", "bin"),
				filepath.Join(home, "bin"),
			)
		}
	}
	if home != "" {
		if runtimeBin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin); err == nil {
			if target == "windows" {
				runtimeBin = strings.ReplaceAll(runtimeBin, "/", `\`)
			}
			entries = append(entries, runtimeBin)
		}
	}
	return entries
}

// targetJoin joins path elements with the target's separator.
func targetJoin(target string, elems ...string) string {
	if target == "windows" {
		return windowsJoin(elems...)
	}
	return filepath.Join(elems...)
}

// windowsJoin joins with backslashes regardless of the rendering host, so a
// Windows PATH rendered on Linux does not carry forward slashes.
func windowsJoin(elems ...string) string {
	return strings.TrimRight(elems[0], `\/`) + `\` + strings.Join(elems[1:], `\`)
}

// DefaultPath is the complete PATH a rendered unit carries for a target: the
// DefaultPathEntries followed by the system directories, joined with the
// target's list separator.
func DefaultPath(target, home string) string {
	entries := DefaultPathEntries(target, home, "")
	separator := ":"
	if target == "windows" {
		separator = ";"
		entries = append(entries, `C:\Windows\System32`, `C:\Windows`)
	} else {
		entries = append(entries, "/usr/bin", "/bin", "/usr/sbin", "/sbin")
	}
	return strings.Join(entries, separator)
}

// CoreUnit is one of the native units the control plane owns on every host,
// with its identity under each manager.
type CoreUnit struct {
	// ID is the stable identity callers switch on.
	ID string `json:"id"`
	// Kind says whether the unit is a long-lived daemon, a oneshot or a timer.
	Kind ServiceKind `json:"kind"`
	// Systemd is the full unit name including its suffix.
	Systemd string `json:"systemd"`
	// Launchd is the label; a timer shares its service's plist and has none.
	Launchd string `json:"launchd,omitempty"`
	// Windows is the SCM service or Task Scheduler task name; a timer shares
	// its service's task and has none.
	Windows string `json:"windows,omitempty"`
	// OwnerPath is the repository path of the code that renders the unit.
	OwnerPath string `json:"owner_path"`
}

const (
	CoreUnitAutohealLoop           = "autoheal-loop"
	CoreUnitRuntimeSupervisor      = "runtime-supervisor"
	CoreUnitEmergencyWatchdog      = "emergency-watchdog"
	CoreUnitEmergencyWatchdogTimer = "emergency-watchdog-timer"
)

// CoreUnits returns the four native units every Vrooli host carries. Lists
// of "our units" are derived from here, never retyped.
func CoreUnits() []CoreUnit {
	return []CoreUnit{
		{ID: CoreUnitAutohealLoop, Kind: KindDaemon, Systemd: "vrooli-autoheal.service", Launchd: "com.vrooli.autoheal", Windows: "VrooliAutoheal", OwnerPath: "internal/safeguards/autoheal-watchdog/handler.go"},
		{ID: CoreUnitRuntimeSupervisor, Kind: KindDaemon, Systemd: "vrooli-runtime-supervisor.service", Launchd: "com.vrooli.runtime-supervisor", Windows: "VrooliRuntimeSupervisor", OwnerPath: "internal/runtimesupervisor/service_install.go"},
		{ID: CoreUnitEmergencyWatchdog, Kind: KindOneshot, Systemd: "vrooli-emergency-watchdog.service", Launchd: "com.vrooli.emergency-watchdog", Windows: "VrooliEmergencyWatchdog", OwnerPath: "internal/safeguards/emergency-watchdog/handler.go"},
		{ID: CoreUnitEmergencyWatchdogTimer, Kind: KindTimer, Systemd: "vrooli-emergency-watchdog.timer", OwnerPath: "internal/safeguards/emergency-watchdog/handler.go"},
	}
}

// CoreUnitByID returns the core unit with the given ID.
func CoreUnitByID(id string) (CoreUnit, bool) {
	for _, unit := range CoreUnits() {
		if unit.ID == id {
			return unit, true
		}
	}
	return CoreUnit{}, false
}

// CoreDaemonUnits returns the systemd names of the long-lived core units, the
// ones whose liveness and binary freshness matter between invocations.
func CoreDaemonUnits() []string {
	var names []string
	for _, unit := range CoreUnits() {
		if unit.Kind == KindDaemon {
			names = append(names, unit.Systemd)
		}
	}
	return names
}

// CoreSystemdUnits returns every core systemd unit name, timers included.
func CoreSystemdUnits() []string {
	names := make([]string, 0, len(CoreUnits()))
	for _, unit := range CoreUnits() {
		names = append(names, unit.Systemd)
	}
	return names
}

// NativeName returns the unit's identity under the target's manager.
func (u CoreUnit) NativeName(target string) string {
	switch target {
	case "darwin":
		return u.Launchd
	case "windows":
		return u.Windows
	default:
		return u.Systemd
	}
}

// LaunchdLogPath is the file a LaunchAgent at plistPath logs to and the file
// the darwin log reader opens for it. Both sides derive it from here so the
// renderer and the reader cannot disagree on the name.
func LaunchdLogPath(plistPath, label string) string {
	return filepath.Join(filepath.Dir(plistPath), "..", "Logs", label+".log")
}
