//go:build linux

package platform

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// cgroupMount is the cgroup v2 mount point. It is a variable so the
// occupancy readers can be exercised against a fixture tree.
var cgroupMount = "/sys/fs/cgroup"

const (
	scopeSuffix     = ".scope"
	agentSlice      = "vrooli-agents.slice"
	testSlicePrefix = "vrooli-test"
	// scopeResolveBudget bounds how long Start waits for the user manager to
	// report the cgroup of a scope systemd-run just created.
	scopeResolveBudget = 5 * time.Second
	scopeResolvePoll   = 50 * time.Millisecond
	selfResolvePoll    = 20 * time.Millisecond
	cgroupDirMode      = 0o755
	cgroupFileMode     = 0o644
)

// containedCommand prefers the user manager: systemd-run creates the scope
// under the slice with the ceilings before the target execs, so no child can
// fork outside it. Without systemd-run the tree is placed by hand in a
// cgroup created under this process's own cgroup, which the kernel enforces
// just the same; the manager does not know that cgroup, so the method is
// reported so a lease row can say so.
func containedCommand(spec ContainedSpec) (*Contained, error) {
	if path, err := exec.LookPath("systemd-run"); err == nil {
		argv := []string{"--user", "--scope", "--quiet", "--unit=" + spec.Scope}
		if spec.Containment.Slice != "" {
			argv = append(argv, "--slice="+spec.Containment.Slice)
		}
		for _, property := range systemdProperties(spec.Containment) {
			argv = append(argv, "-p", property)
		}
		argv = append(argv, "--", spec.Path)
		argv = append(argv, spec.Args...)
		cmd := exec.Command(path, argv...)
		applySpec(cmd, spec)
		// systemd-run must reach the caller's own user manager; a shell entered
		// with su carries another uid's XDG_RUNTIME_DIR and would fail
		// silently. The child inherits the corrected bus, which is its own.
		cmd.Env = userManagerEnv(cmd.Env)
		return &Contained{Cmd: cmd, Method: MethodSystemdRun, after: resolveSystemdScope}, nil
	}
	dir, err := fallbackCgroupDir(spec.Containment.Slice, spec.Scope)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(spec.Path, spec.Args...)
	applySpec(cmd, spec)
	containment := spec.Containment
	return &Contained{
		Cmd:    cmd,
		Method: MethodCgroupWrite,
		after: func(c *Contained) error {
			if err := writeCgroupLimits(dir, containment); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(dir, "cgroup.procs"), []byte(strconv.Itoa(c.Cmd.Process.Pid)), cgroupFileMode); err != nil {
				return fmt.Errorf("platform: move pid %d into %s: %w", c.Cmd.Process.Pid, dir, err)
			}
			c.Scope = ScopeRef{Name: spec.Scope, Kind: ScopeKindCgroup, Path: strings.TrimPrefix(dir, cgroupMount)}
			return nil
		},
		cleanup: func() { _ = os.Remove(dir) },
	}, nil
}

func applySpec(cmd *exec.Cmd, spec ContainedSpec) {
	cmd.Env = spec.Env
	cmd.Dir = spec.Dir
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
}

// systemdProperties renders the ceilings as systemd-run -p arguments.
func systemdProperties(c Containment) []string {
	var out []string
	if c.CPUWeight > 0 {
		out = append(out, "CPUWeight="+strconv.Itoa(c.CPUWeight))
	}
	if c.MemoryHigh != "" {
		out = append(out, "MemoryHigh="+c.MemoryHigh)
	}
	if c.MemoryMax != "" {
		out = append(out, "MemoryMax="+c.MemoryMax)
	}
	if c.TasksMax > 0 {
		out = append(out, "TasksMax="+strconv.Itoa(c.TasksMax))
	}
	return out
}

// resolveSystemdScope asks the user manager for the scope's cgroup after
// systemd-run started it. The child is the systemd-run process's target;
// its own /proc entry is authoritative once it exists, and the manager's
// answer is the fallback while it is still being placed.
func resolveSystemdScope(c *Contained) error {
	scope := scopeNameFromArgs(c.Cmd.Args)
	deadline := time.Now().Add(scopeResolveBudget)
	for {
		if path := systemdScopeCgroup(scope); path != "" {
			c.Scope = ScopeRef{Name: scope, Kind: ScopeKindCgroup, Path: path}
			return nil
		}
		if c.Cmd.ProcessState != nil || !pidIsAlive(c.Cmd.Process.Pid) {
			// A short-lived child finished before the manager answered; the
			// scope was created and is gone. Its path is still determined by
			// the manager's root and the slice, so the ref is exact.
			root := userManagerRoot()
			if root == "" {
				return fmt.Errorf("platform: systemd-run exited before the user manager reported a cgroup for %s%s", scope, scopeSuffix)
			}
			slice := scopeSliceFromArgs(c.Cmd.Args)
			c.Scope = ScopeRef{Name: scope, Kind: ScopeKindCgroup, Path: root + "/" + slice + "/" + scope + scopeSuffix}
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("platform: user manager never reported a cgroup for %s%s", scope, scopeSuffix)
		}
		time.Sleep(scopeResolvePoll)
	}
}

// scopeSliceFromArgs reads the --slice= argument; the manager's default for a
// user scope is app.slice.
func scopeSliceFromArgs(args []string) string {
	for _, arg := range args {
		if slice, ok := strings.CutPrefix(arg, "--slice="); ok {
			return slice
		}
	}
	return "app.slice"
}

// userManagerRoot is the cgroup of the calling uid's user manager, the
// parent of every slice and scope it creates.
func userManagerRoot() string {
	cmd := exec.Command("systemctl", "--user", "show", "-p", "ControlGroup", "--value", "--", "-.slice")
	cmd.Env = userManagerEnv(nil)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func scopeNameFromArgs(args []string) string {
	for _, arg := range args {
		if unit, ok := strings.CutPrefix(arg, "--unit="); ok {
			return unit
		}
	}
	return ""
}

// sliceCgroup asks the user manager where a slice lives, falling back to the
// manager root so an unstarted slice still resolves to the path it will have.
func sliceCgroup(slice string) string {
	cmd := exec.Command("systemctl", "--user", "show", "-p", "ControlGroup", "--value", slice)
	cmd.Env = userManagerEnv(nil)
	if output, err := cmd.Output(); err == nil {
		if path := strings.TrimSpace(string(output)); path != "" {
			return path
		}
	}
	if root := userManagerRoot(); root != "" {
		return root + "/" + slice
	}
	return ""
}

func systemdScopeCgroup(scope string) string {
	cmd := exec.Command("systemctl", "--user", "show", "-p", "ControlGroup", "--value", scope+scopeSuffix)
	cmd.Env = userManagerEnv(nil)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// userManagerEnv returns env (this process's environment when nil) with
// XDG_RUNTIME_DIR and DBUS_SESSION_BUS_ADDRESS pointing at the calling uid's
// own runtime directory when that directory exists. A shell entered through
// su, or a service, otherwise carries another uid's bus and every user-manager
// call fails with "Failed to connect to bus" while the uid's manager is fine.
func userManagerEnv(env []string) []string {
	if env == nil {
		env = os.Environ()
	}
	runtimeDir := "/run/user/" + strconv.Itoa(os.Getuid())
	if _, err := os.Stat(filepath.Join(runtimeDir, "bus")); err != nil {
		return env
	}
	out := make([]string, 0, len(env)+2)
	for _, entry := range env {
		if strings.HasPrefix(entry, "XDG_RUNTIME_DIR=") || strings.HasPrefix(entry, "DBUS_SESSION_BUS_ADDRESS=") {
			continue
		}
		out = append(out, entry)
	}
	return append(out, "XDG_RUNTIME_DIR="+runtimeDir, "DBUS_SESSION_BUS_ADDRESS=unix:path="+runtimeDir+"/bus")
}

// fallbackCgroupDir prepares <sibling of own cgroup>/<slice>/<scope> and
// proves the tree delegates the three controllers; anything less is
// undetermined, never a silent no-op. The slice is a sibling, not a child,
// of the caller's cgroup: cgroup v2 refuses controllers below a cgroup that
// holds processes (the caller), while its parent already delegates them.
func fallbackCgroupDir(slice, scope string) (string, error) {
	own, err := processScope(os.Getpid())
	if err != nil || own == "" || own == "/" {
		return "", fmt.Errorf("platform: undetermined: own cgroup unreadable: %v", err)
	}
	parent := filepath.Join(cgroupMount, filepath.Dir(own))
	if err := enableControllers(parent); err != nil {
		return "", err
	}
	if slice != "" {
		parent = filepath.Join(parent, slice)
		if err := os.MkdirAll(parent, cgroupDirMode); err != nil {
			return "", fmt.Errorf("platform: undetermined: cannot create %s: %w", parent, err)
		}
		if err := enableControllers(parent); err != nil {
			return "", err
		}
	}
	dir := filepath.Join(parent, scope+scopeSuffix)
	if err := os.MkdirAll(dir, cgroupDirMode); err != nil {
		return "", fmt.Errorf("platform: undetermined: cannot create %s: %w", dir, err)
	}
	return dir, nil
}

// enableControllers makes sure cpu, memory and pids are delegated to the
// children of dir, enabling them when the file is writable.
func enableControllers(dir string) error {
	path := filepath.Join(dir, "cgroup.subtree_control")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("platform: undetermined: %s unreadable: %w", path, err)
	}
	enabled := map[string]bool{}
	for _, controller := range strings.Fields(string(data)) {
		enabled[controller] = true
	}
	var missing []string
	for _, controller := range []string{"cpu", "memory", "pids"} {
		if !enabled[controller] {
			missing = append(missing, "+"+controller)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if err := os.WriteFile(path, []byte(strings.Join(missing, " ")), cgroupFileMode); err != nil {
		return fmt.Errorf("platform: undetermined: subtree_control lacks %s and cannot be written: %w", strings.Join(missing, " "), err)
	}
	return nil
}

func writeCgroupLimits(dir string, c Containment) error {
	physical, _ := memTotalBytes()
	writes := map[string]string{}
	if c.CPUWeight > 0 {
		writes["cpu.weight"] = strconv.Itoa(c.CPUWeight)
	}
	if c.TasksMax > 0 {
		writes["pids.max"] = strconv.Itoa(c.TasksMax)
	}
	for file, value := range map[string]string{"memory.high": c.MemoryHigh, "memory.max": c.MemoryMax} {
		bytes, err := memoryCeilingBytes(value, physical)
		if err != nil {
			return err
		}
		if bytes > 0 {
			writes[file] = strconv.FormatInt(bytes, 10)
		}
	}
	for file, value := range writes {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(value), cgroupFileMode); err != nil {
			return fmt.Errorf("platform: write %s=%s: %w", file, value, err)
		}
	}
	return nil
}

// memTotalBytes reads MemTotal from /proc/meminfo.
func memTotalBytes() (int64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kb, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return 0, err
			}
			return kb * kibibyte, nil
		}
	}
	return 0, errors.New("platform: MemTotal absent from /proc/meminfo")
}

// containSelf adopts the calling pid into a new transient scope through the
// user manager's StartTransientUnit (busctl carries the PIDs property that
// systemd-run --scope cannot), and falls back to the hand-made cgroup.
func containSelf(scope string, c Containment) (ScopeRef, string, error) {
	if busctl, err := exec.LookPath("busctl"); err == nil {
		physical, _ := memTotalBytes()
		properties := [][3]string{{"PIDs", "au", "1 " + strconv.Itoa(os.Getpid())}}
		if c.Slice != "" {
			properties = append(properties, [3]string{"Slice", "s", c.Slice})
		}
		if c.CPUWeight > 0 {
			properties = append(properties, [3]string{"CPUWeight", "t", strconv.Itoa(c.CPUWeight)})
		}
		for _, ceiling := range []struct{ name, value string }{{"MemoryHigh", c.MemoryHigh}, {"MemoryMax", c.MemoryMax}} {
			bytes, err := memoryCeilingBytes(ceiling.value, physical)
			if err != nil {
				return ScopeRef{Kind: ScopeKindNone}, MethodNone, err
			}
			if bytes > 0 {
				properties = append(properties, [3]string{ceiling.name, "t", strconv.FormatInt(bytes, 10)})
			}
		}
		if c.TasksMax > 0 {
			properties = append(properties, [3]string{"TasksMax", "t", strconv.Itoa(c.TasksMax)})
		}
		argv := []string{"--user", "call", "org.freedesktop.systemd1", "/org/freedesktop/systemd1", "org.freedesktop.systemd1.Manager", "StartTransientUnit", "ssa(sv)a(sa(sv))", scope + scopeSuffix, "fail", strconv.Itoa(len(properties))}
		for _, property := range properties {
			argv = append(argv, property[0], property[1])
			argv = append(argv, strings.Fields(property[2])...)
		}
		argv = append(argv, "0")
		call := exec.Command(busctl, argv...)
		call.Env = userManagerEnv(nil)
		output, err := call.CombinedOutput()
		if err != nil {
			return ScopeRef{Kind: ScopeKindNone}, MethodNone, fmt.Errorf("platform: StartTransientUnit %s: %v: %s", scope, err, strings.TrimSpace(string(output)))
		}
		deadline := time.Now().Add(scopeResolveBudget)
		for {
			if path, _ := processScope(os.Getpid()); strings.HasSuffix(path, "/"+scope+scopeSuffix) {
				return ScopeRef{Name: scope, Kind: ScopeKindCgroup, Path: path}, MethodTransientUnit, nil
			}
			if time.Now().After(deadline) {
				return ScopeRef{Kind: ScopeKindNone}, MethodNone, fmt.Errorf("platform: pid %d never landed in %s%s", os.Getpid(), scope, scopeSuffix)
			}
			time.Sleep(selfResolvePoll)
		}
	}
	dir, err := fallbackCgroupDir(c.Slice, scope)
	if err != nil {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, err
	}
	if err := writeCgroupLimits(dir, c); err != nil {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, err
	}
	if err := os.WriteFile(filepath.Join(dir, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), cgroupFileMode); err != nil {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, fmt.Errorf("platform: move self into %s: %w", dir, err)
	}
	return ScopeRef{Name: scope, Kind: ScopeKindCgroup, Path: strings.TrimPrefix(dir, cgroupMount)}, MethodCgroupWrite, nil
}

// freezableCgroup admits only agent and test scopes; the guard is the
// difference between an authority and a footgun.
func freezableCgroup(ref ScopeRef) (string, error) {
	if ref.Kind != ScopeKindCgroup || ref.Path == "" {
		return "", fmt.Errorf("platform: %s is not a cgroup scope", ref.String())
	}
	clean := filepath.Clean(ref.Path)
	if !strings.Contains(clean, "/"+agentSlice+"/") && !strings.Contains(clean, "/"+testSlicePrefix) {
		return "", fmt.Errorf("platform: refusing to freeze %s: not under %s", clean, agentSlice)
	}
	return filepath.Join(cgroupMount, clean, "cgroup.freeze"), nil
}

func freezeScope(ref ScopeRef) error { return writeFreeze(ref, "1") }

func thawScope(ref ScopeRef) error { return writeFreeze(ref, "0") }

func writeFreeze(ref ScopeRef, value string) error {
	path, err := freezableCgroup(ref)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(value), cgroupFileMode); err != nil {
		return fmt.Errorf("platform: write %s: %w", path, err)
	}
	return nil
}

func scopeFrozen(ref ScopeRef) (bool, error) {
	path, err := freezableCgroup(ref)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	// cgroup.freeze reads back the request; cgroup.events carries the
	// effective state ("frozen 1") once every task has stopped.
	events, err := os.ReadFile(filepath.Join(filepath.Dir(path), "cgroup.events"))
	if err != nil {
		return strings.TrimSpace(string(data)) == "1", nil
	}
	for _, line := range strings.Split(string(events), "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "frozen" {
			return fields[1] == "1", nil
		}
	}
	return strings.TrimSpace(string(data)) == "1", nil
}

// readableCgroup resolves the directory of any cgroup a ref names. Reading
// is not freezing: freezableCgroup guards mutation to agent and test slices,
// while occupancy must be readable for the slice itself and for a supervisor
// scope, so this admits any path under the mount.
func readableCgroup(ref ScopeRef) (string, error) {
	if ref.Kind != ScopeKindCgroup || strings.TrimSpace(ref.Path) == "" {
		return "", fmt.Errorf("platform: %s is not a cgroup scope", ref.String())
	}
	// Cleaning as an absolute path collapses every traversal segment, so a
	// ref can name a cgroup inside the mount and never a path outside it.
	clean := filepath.Clean("/" + strings.TrimPrefix(filepath.Clean(ref.Path), "/"))
	return filepath.Join(cgroupMount, clean), nil
}

func scopeProcesses(ref ScopeRef) ([]int, error) {
	dir, err := readableCgroup(ref)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "cgroup.procs"))
	if err != nil {
		return nil, fmt.Errorf("platform: read cgroup.procs of %s: %w", ref.Path, err)
	}
	pids := make([]int, 0, 16)
	for _, line := range strings.Split(string(data), "\n") {
		field := strings.TrimSpace(line)
		if field == "" {
			continue
		}
		pid, convErr := strconv.Atoi(field)
		if convErr != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

func scopeOccupancy(ref ScopeRef) (Occupancy, error) {
	dir, err := readableCgroup(ref)
	if err != nil {
		return Occupancy{}, err
	}
	if _, statErr := os.Stat(dir); statErr != nil {
		return Occupancy{}, fmt.Errorf("platform: read occupancy of %s: %w", ref.Path, statErr)
	}
	out := Occupancy{
		Path:             filepath.Clean("/" + strings.TrimPrefix(filepath.Clean(ref.Path), "/")),
		Tasks:            cgroupNumber(dir, "pids.current"),
		TasksMax:         cgroupNumber(dir, "pids.max"),
		MemoryBytes:      cgroupNumber(dir, "memory.current"),
		MemoryHighBytes:  cgroupNumber(dir, "memory.high"),
		MemoryMaxBytes:   cgroupNumber(dir, "memory.max"),
		MemoryHighEvents: cgroupEvent(dir, "memory.events", "high"),
	}
	return out, nil
}

// cgroupNumber reads a single-value cgroup file. "max" is Unlimited, an
// unreadable or unparsable file is Unknown: a ceiling that cannot be read is
// never reported as a number a bar would compare against.
func cgroupNumber(dir, name string) int64 {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return Unknown
	}
	field := strings.TrimSpace(string(data))
	if field == "max" {
		return Unlimited
	}
	value, err := strconv.ParseInt(field, 10, 64)
	if err != nil {
		return Unknown
	}
	return value
}

// cgroupEvent reads one counter from a flat-keyed cgroup file.
func cgroupEvent(dir, name, key string) int64 {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return Unknown
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == key {
			value, convErr := strconv.ParseInt(fields[1], 10, 64)
			if convErr != nil {
				return Unknown
			}
			return value
		}
	}
	return Unknown
}

func scopeChildren(ref ScopeRef) ([]ScopeRef, error) {
	dir, err := readableCgroup(ref)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("platform: list children of %s: %w", ref.Path, err)
	}
	parent := filepath.Clean("/" + strings.TrimPrefix(filepath.Clean(ref.Path), "/"))
	out := make([]ScopeRef, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		out = append(out, ScopeRef{
			Name: strings.TrimSuffix(entry.Name(), scopeSuffix),
			Kind: ScopeKindCgroup,
			Path: filepath.Join(parent, entry.Name()),
		})
	}
	return out, nil
}

// adoptIntoScope prefers the user manager: a transient scope started with
// PIDs= is a real unit an operator can see and stop. systemd-run cannot adopt
// a running process, so the manager is asked over its own bus. Without a
// reachable bus the pid is written into a cgroup made by hand under the
// slice, which the kernel enforces just the same; the method is reported so a
// caller can record which one happened.
func adoptIntoScope(spec AdoptSpec) (ScopeRef, string, error) {
	unit := spec.Scope
	if !strings.HasSuffix(unit, scopeSuffix) {
		unit += scopeSuffix
	}
	if ref, err := adoptThroughManager(spec, unit); err == nil {
		return ref, MethodSystemdRun, nil
	}
	ref, err := adoptByCgroupWrite(spec, unit)
	if err != nil {
		return ScopeRef{Kind: ScopeKindNone}, MethodNone, err
	}
	return ref, MethodCgroupWrite, nil
}

// adoptThroughManager calls StartTransientUnit with the pid. busctl is the
// systemd package's own client, so it is present wherever the manager is.
func adoptThroughManager(spec AdoptSpec, unit string) (ScopeRef, error) {
	busctl, err := exec.LookPath("busctl")
	if err != nil {
		return ScopeRef{}, err
	}
	description := spec.Description
	if strings.TrimSpace(description) == "" {
		description = unit
	}
	properties := systemdProperties(spec.Containment)
	// Slice, Description and PIDs, then one entry per ceiling.
	argv := []string{
		"--user", "call", "org.freedesktop.systemd1", "/org/freedesktop/systemd1",
		"org.freedesktop.systemd1.Manager", "StartTransientUnit", "ssa(sv)a(sa(sv))",
		unit, "fail", strconv.Itoa(3 + len(properties)),
		"Slice", "s", spec.Slice,
		"Description", "s", description,
		"PIDs", "au", "1", strconv.Itoa(spec.PID),
	}
	for _, property := range properties {
		key, value, ok := strings.Cut(property, "=")
		if !ok {
			continue
		}
		argv = append(argv, key, transientPropertyType(key), value)
	}
	argv = append(argv, "0")
	if out, runErr := exec.Command(busctl, argv...).CombinedOutput(); runErr != nil {
		return ScopeRef{}, fmt.Errorf("platform: adopt pid %d into %s: %w: %s", spec.PID, unit, runErr, strings.TrimSpace(string(out)))
	}
	path := systemdScopeCgroup(strings.TrimSuffix(unit, scopeSuffix))
	if path == "" {
		return ScopeRef{}, fmt.Errorf("platform: the user manager reported no cgroup for %s", unit)
	}
	return ScopeRef{Name: strings.TrimSuffix(unit, scopeSuffix), Kind: ScopeKindCgroup, Path: path}, nil
}

// transientPropertyType is the D-Bus signature of a unit property systemd
// accepts on a transient scope. Weights and task counts are unsigned 64-bit;
// memory ceilings are byte counts of the same width.
func transientPropertyType(key string) string {
	switch key {
	case "CPUWeight", "TasksMax", "MemoryHigh", "MemoryMax":
		return "t"
	default:
		return "s"
	}
}

// adoptByCgroupWrite places the pid without the manager.
func adoptByCgroupWrite(spec AdoptSpec, unit string) (ScopeRef, error) {
	slicePath := sliceCgroup(spec.Slice)
	if slicePath == "" {
		return ScopeRef{}, fmt.Errorf("platform: no cgroup for %s", spec.Slice)
	}
	dir := filepath.Join(cgroupMount, slicePath, unit)
	if mkErr := os.MkdirAll(dir, cgroupDirMode); mkErr != nil {
		return ScopeRef{}, fmt.Errorf("platform: create %s: %w", dir, mkErr)
	}
	if writeErr := writeCgroupLimits(dir, spec.Containment); writeErr != nil {
		return ScopeRef{}, writeErr
	}
	if writeErr := os.WriteFile(filepath.Join(dir, "cgroup.procs"), []byte(strconv.Itoa(spec.PID)), cgroupFileMode); writeErr != nil {
		return ScopeRef{}, fmt.Errorf("platform: move pid %d into %s: %w", spec.PID, dir, writeErr)
	}
	return ScopeRef{Name: strings.TrimSuffix(unit, scopeSuffix), Kind: ScopeKindCgroup, Path: filepath.Join(slicePath, unit)}, nil
}

func sliceCgroupPath(slice string) (string, error) {
	if path := sliceCgroup(slice); path != "" {
		return path, nil
	}
	return "", fmt.Errorf("platform: the user manager reported no cgroup for %s", slice)
}
