package devicegraph

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CommandRunner runs one host probe and returns its combined stdout. It is the
// injection point that lets every command-driven parser run against a fixture
// instead of the live host.
type CommandRunner func(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error)

// Env is the injectable environment every collector in this package reads
// through. Zero values fall back to the live host, so production callers pass
// an empty Env and tests pass fixture roots and a stub runner. There is no
// package-level global: the seam is the value, not the process.
type Env struct {
	// SysRoot is the sysfs mount point. Defaults to /sys.
	SysRoot string
	// DevRoot is the device-node directory. Defaults to /dev.
	DevRoot string
	// HardwareIDPaths are the candidate PCI/USB id database locations, tried
	// in order. Defaults to the common distribution locations.
	HardwareIDPaths []string
	// Run executes a host probe. Defaults to exec.CommandContext.
	Run CommandRunner
	// LookPath resolves a probe binary. Defaults to exec.LookPath.
	LookPath func(string) (string, error)
	// Now supplies the observation timestamp. Defaults to time.Now.
	Now func() time.Time
}

const defaultProbeTimeout = 5 * time.Second

func (e Env) normalized() Env {
	if e.SysRoot == "" {
		e.SysRoot = "/sys"
	}
	if e.DevRoot == "" {
		e.DevRoot = "/dev"
	}
	if len(e.HardwareIDPaths) == 0 {
		e.HardwareIDPaths = []string{"/usr/share/hwdata", "/usr/share/misc", "/var/lib/usbutils"}
	}
	if e.Run == nil {
		e.Run = runHostCommand
	}
	if e.LookPath == nil {
		e.LookPath = exec.LookPath
	}
	if e.Now == nil {
		e.Now = time.Now
	}
	return e
}

func runHostCommand(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	// CombinedOutput is deliberate: smartctl reports permission failures on
	// stdout as JSON while still exiting non-zero, and the diagnostic text of
	// the other probes is the reason string we must surface.
	return exec.CommandContext(probeCtx, name, args...).CombinedOutput()
}

func (e Env) sys(parts ...string) string {
	return filepath.Join(append([]string{e.SysRoot}, parts...)...)
}

func (e Env) readText(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

func (e Env) readInt(path string) (int64, bool) {
	text, ok := e.readText(path)
	if !ok {
		return 0, false
	}
	value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// listDir returns the sorted entry names of a directory so enumeration order
// is stable regardless of filesystem ordering.
func (e Env) listDir(path string) ([]string, bool) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, false
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, true
}

// linkBase returns the base name of a symlink target, which is how sysfs
// expresses "which driver/subsystem is bound here".
func (e Env) linkBase(path string) (string, bool) {
	target, err := os.Readlink(path)
	if err != nil {
		return "", false
	}
	base := filepath.Base(strings.TrimSuffix(target, "/"))
	if base == "." || base == "/" || base == "" {
		return "", false
	}
	return base, true
}

// resolve follows a sysfs class symlink to the real device path. Fixtures use
// relative symlinks exactly as the kernel does, so the same code walks both.
func (e Env) resolve(path string) (string, bool) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false
	}
	return resolved, true
}

// ancestors returns the resolved path and each of its parent directories, from
// the device itself outward, stopping at the sysfs devices root. It is the
// structural walk every "which bus is this on" question is answered with — no
// name-prefix matching anywhere.
func (e Env) ancestors(resolved string) []string {
	stops := e.devicesRoots()
	chain := make([]string, 0, 12)
	current := filepath.Clean(resolved)
	// The depth cap is a backstop against a symlink arrangement that never
	// reaches the devices root; sysfs is far shallower than this in practice.
	for depth := 0; depth < 64; depth++ {
		if current == "/" || current == "." {
			return chain
		}
		if _, stop := stops[current]; stop {
			return chain
		}
		chain = append(chain, current)
		parent := filepath.Dir(current)
		if parent == current {
			return chain
		}
		current = parent
	}
	return chain
}

// devicesRoots returns the sysfs device-tree root in both its configured and
// its fully resolved form, because a fixture root (or a host with a symlinked
// mount point) resolves to a different absolute path than the one configured.
func (e Env) devicesRoots() map[string]struct{} {
	roots := map[string]struct{}{}
	configured := filepath.Clean(e.sys("devices"))
	roots[configured] = struct{}{}
	if resolved, ok := e.resolve(configured); ok {
		roots[filepath.Clean(resolved)] = struct{}{}
	}
	return roots
}

// isVirtualDevice reports whether a resolved sysfs path lives under the kernel's
// virtual device tree. Loop, zram, device-mapper, bridge and veth nodes all
// land there, which is how they are excluded without matching any name.
func (e Env) isVirtualDevice(resolved string) bool {
	candidates := []string{filepath.Clean(e.sys("devices", "virtual"))}
	if r, ok := e.resolve(e.sys("devices", "virtual")); ok {
		candidates = append(candidates, filepath.Clean(r))
	}
	cleaned := filepath.Clean(resolved)
	for _, candidate := range candidates {
		if cleaned == candidate || strings.HasPrefix(cleaned, candidate+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// subsystemOf reports which kernel bus a sysfs node is attached to.
func (e Env) subsystemOf(path string) (string, bool) {
	return e.linkBase(filepath.Join(path, "subsystem"))
}
