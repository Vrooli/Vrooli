package hostinventory

import (
	"context"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"vrooli-autoheal/internal/journal"
)

func collectLinux(parent context.Context, c *DefaultCollector, inv *HostInventory) {
	ctx, cancel := context.WithTimeout(parent, 8*time.Second)
	defer cancel()

	inv.Kernel.Release = commandString(ctx, c, "uname", "-r")
	inv.Kernel.Version = commandString(ctx, c, "uname", "-v")
	if inv.Kernel.Release != "" {
		_, err := c.fs.Stat(filepath.Join("/lib/modules", inv.Kernel.Release))
		inv.Kernel.ModuleTreePresent = err == nil
		if err != nil {
			setProbe(inv, "kernelModuleTree", ProbeDegraded, err)
		} else {
			setProbe(inv, "kernelModuleTree", ProbeOK, nil)
		}
	}
	if b, err := c.fs.ReadFile("/proc/sys/kernel/random/boot_id"); err == nil {
		inv.BootID = strings.TrimSpace(string(b))
		setProbe(inv, "bootId", ProbeOK, nil)
	} else {
		setProbe(inv, "bootId", ProbeDegraded, err)
	}
	if entries, err := c.fs.ReadDir("/lib/modules"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				inv.Kernel.InstalledModuleTrees = append(inv.Kernel.InstalledModuleTrees, entry.Name())
			}
		}
		sort.Strings(inv.Kernel.InstalledModuleTrees)
		setProbe(inv, "installedModuleTrees", ProbeOK, nil)
	} else {
		setProbe(inv, "installedModuleTrees", ProbeDegraded, err)
	}
	if b, err := c.fs.ReadFile("/proc/modules"); err == nil {
		inv.Kernel.LoadedModules = parseProcModules(string(b))
		setProbe(inv, "loadedModules", ProbeOK, nil)
	} else {
		setProbe(inv, "loadedModules", ProbeDegraded, err)
	}

	if _, err := exec.LookPath("lspci"); err == nil {
		if out, err := c.exec.CombinedOutput(ctx, "lspci", "-nnk"); err == nil {
			inv.Devices = ParseLSPCI(string(out))
			setProbe(inv, "devices", ProbeOK, nil)
		} else {
			setProbe(inv, "devices", ProbeDegraded, err)
		}
	} else {
		setProbe(inv, "devices", ProbeUnsupported, err)
	}

	inv.Runtimes = collectRuntimeTools(ctx, c)
	setProbe(inv, "runtimes", ProbeOK, nil)
	inv.Packages = collectPackageState(ctx, c, inv.Kernel.Release)
	if inv.Packages.Manager == "" {
		setProbe(inv, "packageManager", ProbeUnsupported, nil)
	} else {
		setProbe(inv, "packageManager", ProbeOK, nil)
	}
	inv.Signals = collectKernelSignals(ctx, c)
	if c.journal == nil {
		setProbe(inv, "journal", ProbeUnsupported, nil)
	} else {
		setProbe(inv, "journal", ProbeOK, nil)
	}
}

func commandString(ctx context.Context, c *DefaultCollector, name string, args ...string) string {
	out, err := c.exec.CombinedOutput(ctx, name, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parseProcModules(content string) []string {
	var modules []string
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			modules = append(modules, fields[0])
		}
	}
	sort.Strings(modules)
	return modules
}

func collectRuntimeTools(ctx context.Context, c *DefaultCollector) []RuntimeToolInfo {
	var tools []RuntimeToolInfo
	for _, spec := range []struct {
		name string
		args []string
	}{
		{name: "nvidia-smi", args: []string{"--query-gpu=name,driver_version", "--format=csv,noheader"}},
		{name: "docker", args: []string{"version", "--format", "{{.Server.Version}}"}},
		{name: "podman", args: []string{"version", "--format", "{{.Server.Version}}"}},
		{name: "rocm-smi", args: []string{"--showdriverversion"}},
	} {
		path, err := exec.LookPath(spec.name)
		info := RuntimeToolInfo{Name: spec.name, Path: path}
		if err != nil {
			info.Error = "not found"
			tools = append(tools, info)
			continue
		}
		out, err := c.exec.CombinedOutput(ctx, spec.name, spec.args...)
		if err != nil {
			info.Error = truncateEvidence(string(out)+" "+err.Error(), 300)
		} else {
			info.Callable = true
			info.Version = truncateEvidence(string(out), 300)
		}
		tools = append(tools, info)
	}
	return tools
}

func collectKernelSignals(ctx context.Context, c *DefaultCollector) []HostSignal {
	if c.journal == nil {
		return nil
	}
	entries, err := c.journal.QueryLogs(ctx, journal.QueryOpts{
		Kernel:   true,
		Since:    "24 hours ago",
		Priority: "0..4",
		Grep:     "(reset|timeout|I/O error|machine check|MCE|corrupt|GPU|drm|amdgpu|nvidia|xid|pcie|AER)",
		Tail:     50,
		Reverse:  true,
	})
	if err != nil {
		return []HostSignal{{Source: "journal", Severity: "warning", Category: "probe_error", Message: truncateEvidence(err.Error(), 300)}}
	}
	signals := make([]HostSignal, 0, len(entries))
	for _, entry := range entries {
		signals = append(signals, HostSignal{
			Source:    "journal",
			Severity:  severityFromPriority(entry.Priority),
			Timestamp: entry.Timestamp,
			BootID:    entry.BootID,
			Category:  categorizeSignal(entry.Message),
			Message:   truncateEvidence(entry.Message, 500),
		})
	}
	return signals
}

func severityFromPriority(priority int) string {
	if priority >= 0 && priority <= 2 {
		return "critical"
	}
	return "warning"
}

func categorizeSignal(message string) string {
	lower := strings.ToLower(message)
	for _, pair := range []struct{ needle, category string }{
		{"machine check", "machine_check"},
		{"mce", "machine_check"},
		{"reset", "device_reset"},
		{"timeout", "device_timeout"},
		{"i/o error", "io_error"},
		{"corrupt", "filesystem_corruption"},
		{"xid", "device_runtime"},
		{"aer", "bus_error"},
		{"pcie", "bus_error"},
		{"gpu", "display_or_accelerator"},
		{"drm", "display_or_accelerator"},
		{"amdgpu", "display_or_accelerator"},
		{"nvidia", "display_or_accelerator"},
	} {
		if strings.Contains(lower, pair.needle) {
			return pair.category
		}
	}
	return "kernel_warning"
}
