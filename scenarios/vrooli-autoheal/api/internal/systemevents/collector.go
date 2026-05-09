package systemevents

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/journal"
	"vrooli-autoheal/internal/platform"
)

type HostCollector struct {
	platform *platform.Capabilities
	exec     checks.CommandExecutor
	journal  *journal.Reader
	now      func() time.Time
	readFile func(string) ([]byte, error)
	glob     func(string) ([]string, error)
}

func NewHostCollector(plat *platform.Capabilities, exec checks.CommandExecutor, reader *journal.Reader) *HostCollector {
	if plat == nil {
		plat = platform.Detect()
	}
	if exec == nil {
		exec = checks.DefaultExecutor
	}
	if reader == nil {
		reader = journal.NewReader(exec)
	}
	return &HostCollector{
		platform: plat,
		exec:     exec,
		journal:  reader,
		now:      func() time.Time { return time.Now().UTC() },
		readFile: os.ReadFile,
		glob:     filepath.Glob,
	}
}

func (c *HostCollector) Collect(ctx context.Context) ([]Event, []SourceStatus) {
	switch c.platform.Platform {
	case platform.Linux:
		return c.collectLinux(ctx)
	case platform.Windows:
		return nil, []SourceStatus{c.status("windows-eventlog", SourceUnsupported, "Windows event-log timeline ingestion is not implemented in this build")}
	case platform.MacOS:
		return nil, []SourceStatus{c.status("macos-unified-log", SourceUnsupported, "macOS timeline ingestion is not implemented in this build")}
	default:
		return nil, []SourceStatus{c.status("host-system-events", SourceUnsupported, "system event timeline ingestion is unsupported on this platform")}
	}
}

func (c *HostCollector) collectLinux(ctx context.Context) ([]Event, []SourceStatus) {
	var events []Event
	var statuses []SourceStatus
	packageEvents, packageStatus := c.collectLinuxPackageLogs()
	events = append(events, packageEvents...)
	statuses = append(statuses, packageStatus...)
	journalEvents, journalStatuses := c.collectLinuxJournal(ctx)
	events = append(events, journalEvents...)
	statuses = append(statuses, journalStatuses...)
	return events, statuses
}

func (c *HostCollector) collectLinuxPackageLogs() ([]Event, []SourceStatus) {
	var events []Event
	var statuses []SourceStatus
	if aptEvents, err := c.collectAPTLogs(); err == nil {
		events = append(events, aptEvents...)
		statuses = append(statuses, c.statusWithCount("apt-history", SourceOK, len(aptEvents), ""))
	} else {
		statuses = append(statuses, c.status("apt-history", SourceDegraded, err.Error()))
	}
	if dpkgEvents, err := c.collectDPKGLogs(); err == nil {
		events = append(events, dpkgEvents...)
		statuses = append(statuses, c.statusWithCount("dpkg-log", SourceOK, len(dpkgEvents), ""))
	} else {
		statuses = append(statuses, c.status("dpkg-log", SourceDegraded, err.Error()))
	}
	return events, statuses
}

func (c *HostCollector) collectAPTLogs() ([]Event, error) {
	paths, err := c.logPaths("/var/log/apt/history.log*")
	if err != nil {
		return nil, err
	}
	var events []Event
	for _, path := range paths {
		content, err := c.readMaybeGzip(path)
		if err != nil {
			continue
		}
		events = append(events, ParseAPTHistory(content)...)
	}
	return events, nil
}

func (c *HostCollector) collectDPKGLogs() ([]Event, error) {
	paths, err := c.logPaths("/var/log/dpkg.log*")
	if err != nil {
		return nil, err
	}
	var events []Event
	for _, path := range paths {
		content, err := c.readMaybeGzip(path)
		if err != nil {
			continue
		}
		events = append(events, ParseDPKGLog(content)...)
	}
	return events, nil
}

func (c *HostCollector) logPaths(pattern string) ([]string, error) {
	paths, err := c.glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no logs matched %s", pattern)
	}
	sort.Strings(paths)
	return paths, nil
}

func (c *HostCollector) readMaybeGzip(path string) ([]byte, error) {
	raw, err := c.readFile(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(path, ".gz") {
		return raw, nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (c *HostCollector) collectLinuxJournal(ctx context.Context) ([]Event, []SourceStatus) {
	if c.journal == nil || !c.journal.Available(ctx) {
		return nil, []SourceStatus{c.status("journalctl", SourceUnsupported, "journalctl unavailable")}
	}
	var events []Event
	boots, err := c.journal.ListBoots(ctx)
	if err != nil {
		return nil, []SourceStatus{c.status("journalctl", SourceDegraded, err.Error())}
	}
	for _, boot := range boots {
		events = append(events, Event{
			OccurredAt: boot.FirstEntry,
			Source:     "journalctl",
			Platform:   string(platform.Linux),
			Category:   "boot",
			Severity:   SeverityInfo,
			Title:      "Boot started",
			Summary:    fmt.Sprintf("Boot %s started", shortID(boot.BootID)),
			BootID:     boot.BootID,
			Details: map[string]any{
				"bootIndex":  boot.Index,
				"firstEntry": boot.FirstEntry.UTC().Format(time.RFC3339Nano),
				"lastEntry":  boot.LastEntry.UTC().Format(time.RFC3339Nano),
			},
		})
		if boot.Index < 0 && !boot.LastEntry.IsZero() {
			events = append(events, Event{
				OccurredAt: boot.LastEntry,
				Source:     "journalctl",
				Platform:   string(platform.Linux),
				Category:   "boot",
				Severity:   SeverityInfo,
				Title:      "Boot ended",
				Summary:    fmt.Sprintf("Boot %s last recorded log entry", shortID(boot.BootID)),
				BootID:     boot.BootID,
				Details:    map[string]any{"bootIndex": boot.Index},
			})
		}
	}
	kernelEvents := c.collectKernelSignals(ctx, boots)
	events = append(events, kernelEvents...)
	return events, []SourceStatus{c.statusWithCount("journalctl", SourceOK, len(events), "")}
}

func (c *HostCollector) collectKernelSignals(ctx context.Context, boots []journal.BootRecord) []Event {
	var events []Event
	for _, boot := range boots {
		bootRef := boot.BootID
		if boot.Index == 0 {
			bootRef = "0"
		}
		logs, err := c.journal.QueryLogs(ctx, journal.QueryOpts{
			Kernel: true,
			Boot:   bootRef,
			Grep:   `Previous system reset reason|uncorrected error|machine check|MCE|hardware error|AER|NVRM|Xid|amdgpu|CMD_RUN timeout|Module nvidia|nvidia driver is not loaded|nvidia devices`,
			Tail:   500,
		})
		if err != nil {
			continue
		}
		for _, entry := range logs {
			if event, ok := kernelLogEvent(entry, boot.BootID); ok {
				events = append(events, event)
			}
		}
	}
	return events
}

func (c *HostCollector) status(source string, state SourceStatusState, err string) SourceStatus {
	platformName := string(c.platform.Platform)
	if platformName == "" {
		platformName = runtime.GOOS
	}
	return SourceStatus{Source: source, Platform: platformName, Status: state, LastIngestedAt: c.now(), LastError: err}
}

func (c *HostCollector) statusWithCount(source string, state SourceStatusState, count int, err string) SourceStatus {
	status := c.status(source, state, err)
	status.Capabilities = map[string]any{"eventCount": count}
	return status
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

var dpkgLineRE = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+(\d{2}:\d{2}:\d{2})\s+(\S+)\s+(\S+)(?:\s+(.*))?$`)

func ParseDPKGLog(content []byte) []Event {
	var events []Event
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		match := dpkgLineRE.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		action, pkg := match[3], match[4]
		category, ok := packageCategory(pkg)
		if !ok || !interestingPackageAction(action) {
			continue
		}
		ts, err := time.ParseInLocation("2006-01-02 15:04:05", match[1]+" "+match[2], time.Local)
		if err != nil {
			continue
		}
		events = append(events, Event{
			OccurredAt: ts.UTC(),
			Source:     "dpkg-log",
			Platform:   string(platform.Linux),
			Category:   category,
			Severity:   SeverityInfo,
			Title:      fmt.Sprintf("Package %s: %s", action, pkg),
			Summary:    fmt.Sprintf("%s %s %s", action, pkg, strings.TrimSpace(match[5])),
			Details:    map[string]any{"action": action, "package": pkg, "raw": line},
		})
	}
	return events
}

func ParseAPTHistory(content []byte) []Event {
	var events []Event
	var start time.Time
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Start-Date:") {
			parsed, err := time.ParseInLocation("2006-01-02  15:04:05", strings.TrimSpace(strings.TrimPrefix(line, "Start-Date:")), time.Local)
			if err == nil {
				start = parsed.UTC()
			}
			continue
		}
		action, payload, ok := aptActionLine(line)
		if !ok || start.IsZero() {
			continue
		}
		packages := extractAptPackages(payload)
		byCategory := map[string][]string{}
		for _, pkg := range packages {
			if category, ok := packageCategory(pkg); ok {
				byCategory[category] = append(byCategory[category], pkg)
			}
		}
		for category, pkgs := range byCategory {
			sort.Strings(pkgs)
			events = append(events, Event{
				OccurredAt: start,
				Source:     "apt-history",
				Platform:   string(platform.Linux),
				Category:   category,
				Severity:   SeverityInfo,
				Title:      fmt.Sprintf("APT %s: %s", strings.ToLower(action), category),
				Summary:    fmt.Sprintf("APT %s touched %d %s package(s): %s", strings.ToLower(action), len(pkgs), category, strings.Join(pkgs, ", ")),
				Details:    map[string]any{"action": action, "packages": pkgs, "raw": line},
			})
		}
	}
	return events
}

func aptActionLine(line string) (string, string, bool) {
	for _, action := range []string{"Install", "Upgrade", "Remove", "Purge"} {
		prefix := action + ":"
		if strings.HasPrefix(line, prefix) {
			return action, strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
		}
	}
	return "", "", false
}

var aptPkgRE = regexp.MustCompile(`(^|,\s*)([a-zA-Z0-9.+:_-]+)(?:\s|\(|,|$)`)

func extractAptPackages(payload string) []string {
	seen := map[string]struct{}{}
	var packages []string
	for _, match := range aptPkgRE.FindAllStringSubmatch(payload, -1) {
		pkg := strings.TrimSpace(match[2])
		pkg = strings.Split(pkg, ":")[0]
		if pkg == "" {
			continue
		}
		if _, ok := seen[pkg]; ok {
			continue
		}
		seen[pkg] = struct{}{}
		packages = append(packages, pkg)
	}
	return packages
}

func packageCategory(pkg string) (string, bool) {
	lower := strings.ToLower(pkg)
	switch {
	case strings.Contains(lower, "firmware") || strings.Contains(lower, "microcode"):
		return "firmware", true
	case strings.HasPrefix(lower, "linux-image") || strings.HasPrefix(lower, "linux-modules") || strings.HasPrefix(lower, "linux-headers") || lower == "linux-generic-hwe-24.04":
		return "kernel", true
	case strings.Contains(lower, "nvidia") || strings.Contains(lower, "amdgpu") || strings.Contains(lower, "mesa") || strings.Contains(lower, "vulkan") || strings.Contains(lower, "libdrm"):
		return "driver", true
	case strings.Contains(lower, "gnome-shell") || strings.Contains(lower, "mutter") || strings.Contains(lower, "xserver-xorg-video"):
		return "display", true
	default:
		return "", false
	}
}

func interestingPackageAction(action string) bool {
	switch action {
	case "install", "upgrade", "remove", "purge", "configure", "Install", "Upgrade", "Remove", "Purge":
		return true
	default:
		return false
	}
}

func kernelLogEvent(entry journal.LogEntry, fallbackBootID string) (Event, bool) {
	message := strings.TrimSpace(entry.Message)
	if message == "" {
		message = strings.TrimSpace(entry.Raw)
	}
	if message == "" {
		return Event{}, false
	}
	lower := strings.ToLower(message)
	event := Event{
		OccurredAt: entry.Timestamp,
		Source:     "journalctl",
		Platform:   string(platform.Linux),
		Severity:   SeverityWarning,
		Title:      "Kernel signal",
		Summary:    message,
		BootID:     entry.BootID,
		Details:    map[string]any{"priority": entry.Priority},
	}
	if event.BootID == "" {
		event.BootID = fallbackBootID
	}
	switch {
	case strings.Contains(lower, "previous system reset reason") || strings.Contains(lower, "uncorrected error") || strings.Contains(lower, "machine check") || strings.Contains(lower, "hardware error"):
		event.Category = "crash"
		event.Severity = SeverityCritical
		event.Title = "Hardware/reset signal"
	case strings.Contains(lower, "nvrm") || strings.Contains(lower, "xid") || strings.Contains(lower, "module nvidia") || strings.Contains(lower, "nvidia driver"):
		event.Category = "driver"
		event.Title = "NVIDIA driver signal"
		if strings.Contains(lower, "not found") || strings.Contains(lower, "not loaded") || strings.Contains(lower, "missing") {
			event.Summary = "NVIDIA kernel module missing or not loaded: " + message
		}
	case strings.Contains(lower, "amdgpu"):
		event.Category = "driver"
		event.Title = "AMDGPU driver signal"
	case strings.Contains(lower, "aer") || strings.Contains(lower, "cmd_run timeout") || strings.Contains(lower, "xhci"):
		event.Category = "hardware"
		event.Title = "PCIe/USB controller signal"
	default:
		return Event{}, false
	}
	return event, true
}
