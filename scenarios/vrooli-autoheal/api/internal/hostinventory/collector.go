package hostinventory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/journal"
	"vrooli-autoheal/internal/platform"
)

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]os.DirEntry, error)
	Stat(path string) (os.FileInfo, error)
}

type OSFileSystem struct{}

func (OSFileSystem) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (OSFileSystem) ReadDir(path string) ([]os.DirEntry, error) { return os.ReadDir(path) }
func (OSFileSystem) Stat(path string) (os.FileInfo, error)      { return os.Stat(path) }

type Collector interface {
	Collect(ctx context.Context) (HostInventory, error)
}

type CollectorOptions struct {
	Platform   *platform.Capabilities
	Executor   checks.CommandExecutor
	FileSystem FileSystem
	Journal    *journal.Reader
	Now        func() time.Time
}

type DefaultCollector struct {
	platform *platform.Capabilities
	exec     checks.CommandExecutor
	fs       FileSystem
	journal  *journal.Reader
	now      func() time.Time
}

func NewCollector(opts CollectorOptions) *DefaultCollector {
	exec := opts.Executor
	if exec == nil {
		exec = checks.DefaultExecutor
	}
	fs := opts.FileSystem
	if fs == nil {
		fs = OSFileSystem{}
	}
	plat := opts.Platform
	if plat == nil {
		plat = platform.Detect()
	}
	reader := opts.Journal
	if reader == nil {
		reader = journal.NewReader(exec)
	}
	now := opts.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &DefaultCollector{platform: plat, exec: exec, fs: fs, journal: reader, now: now}
}

func (c *DefaultCollector) Collect(ctx context.Context) (HostInventory, error) {
	inv := HostInventory{
		CollectedAt: c.now().UTC(),
		Platform:    string(c.platform.Platform),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		ProbeStatus: map[string]ProbeState{},
		ProbeErrors: map[string]string{},
	}
	if c.platform.Platform != platform.Linux {
		inv.ProbeStatus["host"] = ProbeUnsupported
		inv.Unsupported = append(inv.Unsupported, "host inventory is currently implemented for Linux hosts")
		inv.Fingerprint = Fingerprint(inv)
		return inv, nil
	}
	collectLinux(ctx, c, &inv)
	inv.Fingerprint = Fingerprint(inv)
	if len(inv.ProbeErrors) == 0 {
		inv.ProbeErrors = nil
	}
	return inv, nil
}

type CachedCollector struct {
	inner   Collector
	ttl     time.Duration
	mu      sync.Mutex
	last    HostInventory
	lastErr error
}

func NewCachedCollector(inner Collector, ttl time.Duration) *CachedCollector {
	return &CachedCollector{inner: inner, ttl: ttl}
}

func (c *CachedCollector) Collect(ctx context.Context) (HostInventory, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.last.CollectedAt.IsZero() && time.Since(c.last.CollectedAt) < c.ttl {
		return c.last, c.lastErr
	}
	c.last, c.lastErr = c.inner.Collect(ctx)
	return c.last, c.lastErr
}

func Fingerprint(inv HostInventory) string {
	stable := struct {
		Platform string            `json:"platform"`
		OS       string            `json:"os"`
		Arch     string            `json:"arch"`
		BootID   string            `json:"bootId"`
		Kernel   KernelInfo        `json:"kernel"`
		Devices  []DeviceInfo      `json:"devices"`
		Runtimes []RuntimeToolInfo `json:"runtimes"`
		Packages PackageState      `json:"packages"`
	}{
		Platform: inv.Platform,
		OS:       inv.OS,
		Arch:     inv.Arch,
		BootID:   inv.BootID,
		Kernel:   inv.Kernel,
		Devices:  inv.Devices,
		Runtimes: inv.Runtimes,
		Packages: inv.Packages,
	}
	sort.Strings(stable.Kernel.LoadedModules)
	sort.Slice(stable.Devices, func(i, j int) bool { return stable.Devices[i].Address < stable.Devices[j].Address })
	sort.Slice(stable.Runtimes, func(i, j int) bool { return stable.Runtimes[i].Name < stable.Runtimes[j].Name })
	b, _ := json.Marshal(stable)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:16])
}

func setProbe(inv *HostInventory, name string, state ProbeState, err error) {
	inv.ProbeStatus[name] = state
	if err != nil {
		inv.ProbeErrors[name] = err.Error()
	}
}

func truncateEvidence(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}
