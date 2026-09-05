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

const (
	cgroupMount     = "/sys/fs/cgroup"
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
