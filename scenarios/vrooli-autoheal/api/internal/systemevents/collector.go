package systemevents

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// kernelSignalSourceKey namespaces this collector's persisted cursor and
// per-boot scan markers in the CursorStore.
const kernelSignalSourceKey = "journalctl/kernel"

// kernelGrepPattern is the big-regex used to surface hardware/driver kernel
// signals. Kept as a package var so a shared collection pass (sub-phase 2d)
// could reuse it without duplicating the literal.
const kernelGrepPattern = `Previous system reset reason|uncorrected error|machine check|MCE|hardware error|AER|NVRM|Xid|amdgpu|CMD_RUN timeout|Module nvidia|nvidia driver is not loaded|nvidia devices`

// currentBootRescanTail bounds the fallback re-scan of the current boot when
// no usable cursor exists (cold start or cursor invalidation). It matches the
// pre-incremental `-n 500` behavior so detection coverage is unchanged.
const currentBootRescanTail = 500

type HostCollector struct {
	platform *platform.Capabilities
	exec     checks.CommandExecutor
	journal  *journal.Reader
	cursors  CursorStore
	now      func() time.Time
	readFile func(string) ([]byte, error)
	glob     func(string) ([]string, error)

	// execsAvoided counts journalctl kernel-grep invocations skipped because a
	// historical boot was already scanned or the current boot was read
	// incrementally via cursor. Surfaced through ExecsAvoided() for the status
	// endpoint. Updated only on the single ingest goroutine.
	execsAvoided int64
}

func NewHostCollector(plat *platform.Capabilities, exec checks.CommandExecutor, reader *journal.Reader) *HostCollector {
	return NewHostCollectorWithCursors(plat, exec, reader, nil)
}

// NewHostCollectorWithCursors builds a HostCollector with an explicit
// CursorStore for incremental kernel-signal ingestion. A nil cursors store
// degrades gracefully to the legacy "scan every boot" behavior (still correct,
// just not incremental), which keeps tests that don't care about cursors simple.
func NewHostCollectorWithCursors(plat *platform.Capabilities, exec checks.CommandExecutor, reader *journal.Reader, cursors CursorStore) *HostCollector {
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
		cursors:  cursors,
		now:      func() time.Time { return time.Now().UTC() },
		readFile: os.ReadFile,
		glob:     filepath.Glob,
	}
}

// ExecsAvoided returns the cumulative count of kernel-grep journalctl
// invocations skipped since process start (already-scanned historical boots +
// incremental current-boot reads). Safe to call from the ingest goroutine.
func (c *HostCollector) ExecsAvoided() int64 {
	if c == nil {
		return 0
	}
	return c.execsAvoided
}

func (c *HostCollector) Collect(ctx context.Context) ([]Event, []SourceStatus) {
	switch c.platform.Platform {
	case platform.Linux:
		return c.collectLinux(ctx)
	case platform.Windows:
		return c.collectPortableHostLogs(ctx, "windows-eventlog")
	case platform.MacOS:
		return c.collectPortableHostLogs(ctx, "macos-unified-log")
	default:
		return nil, []SourceStatus{c.status("host-system-events", SourceUnsupported, "system event timeline ingestion is unsupported on this platform")}
	}
}

func (c *HostCollector) collectPortableHostLogs(ctx context.Context, source string) ([]Event, []SourceStatus) {
	if c.journal == nil {
		return nil, []SourceStatus{c.status(source, SourceUnsupported, "host-log reader unavailable")}
	}
	logs, err := c.journal.QueryLogs(ctx, journal.QueryOpts{Tail: currentBootRescanTail})
	if err != nil {
		return nil, []SourceStatus{c.status(source, SourceDegraded, err.Error())}
	}
	events := make([]Event, 0, len(logs))
	for _, entry := range logs {
		if strings.TrimSpace(entry.Message) == "" {
			continue
		}
		occurredAt := entry.Timestamp
		if occurredAt.IsZero() {
			occurredAt = c.now()
		}
		details := map[string]any{
			"provider":  entry.Identifier,
			"eventId":   entry.EventID,
			"processId": entry.PID,
			"raw":       entry.Raw,
		}
		events = append(events, Event{
			OccurredAt: occurredAt,
			Source:     source,
			Platform:   string(c.platform.Platform),
			Category:   "system",
			Severity:   SeverityInfo,
			Title:      portableLogTitle(entry),
			Summary:    entry.Message,
			BootID:     entry.BootID,
			Details:    details,
		})
	}
	return events, []SourceStatus{c.statusWithCount(source, SourceOK, len(events), "")}
}

func portableLogTitle(entry journal.LogEntry) string {
	provider := strings.TrimSpace(entry.Identifier)
	if provider == "" {
		provider = "Host log"
	}
	if eventID := strings.TrimSpace(entry.EventID); eventID != "" {
		return fmt.Sprintf("%s event %s", provider, eventID)
	}
	return provider
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
	manager := detectedPackageManager()
	switch manager {
	case "dpkg":
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
	case "dnf", "rpm":
		adapter := "rpm"
		if manager == "dnf" {
			adapter = "dnf"
		}
		adapterEvents, err := c.collectRPMFamilyLogs(adapter)
		if err != nil {
			statuses = append(statuses, c.status(adapter+"-log", SourceUnsupported, err.Error()))
		} else {
			events = append(events, adapterEvents...)
			statuses = append(statuses, c.statusWithCount(adapter+"-log", SourceOK, len(adapterEvents), ""))
		}
	case "pacman":
		adapterEvents, err := c.collectPacmanLogs()
		if err != nil {
			statuses = append(statuses, c.status("pacman-log", SourceUnsupported, err.Error()))
		} else {
			events = append(events, adapterEvents...)
			statuses = append(statuses, c.statusWithCount("pacman-log", SourceOK, len(adapterEvents), ""))
		}
	default:
		statuses = append(statuses, c.status("package-manager", SourceUnsupported, "no supported Linux package manager was detected"))
	}
	return events, statuses
}

func detectedPackageManager() string {
	for _, candidate := range []string{"apt-get", "dnf", "pacman", "rpm"} {
		if _, err := exec.LookPath(candidate); err != nil {
			continue
		}
		switch candidate {
		case "apt-get":
			return "dpkg"
		default:
			return candidate
		}
	}
	return ""
}

func (c *HostCollector) collectRPMFamilyLogs(manager string) ([]Event, error) {
	patterns := []string{"/var/log/dnf.log*", "/var/log/yum.log*"}
	if manager == "rpm" {
		patterns = []string{"/var/log/yum.log*"}
	}
	return c.collectGenericPackageLogs(manager, patterns)
}

func (c *HostCollector) collectPacmanLogs() ([]Event, error) {
	return c.collectGenericPackageLogs("pacman", []string{"/var/log/pacman.log*"})
}

func (c *HostCollector) collectGenericPackageLogs(source string, patterns []string) ([]Event, error) {
	var events []Event
	var matched bool
	for _, pattern := range patterns {
		paths, err := c.logPaths(pattern)
		if err != nil {
			continue
		}
		matched = true
		for _, path := range paths {
			content, err := c.readMaybeGzip(path)
			if err != nil {
				continue
			}
			events = append(events, parseGenericPackageLog(source, content)...)
		}
	}
	if !matched {
		return nil, fmt.Errorf("no %s package log matched", source)
	}
	return events, nil
}

func parseGenericPackageLog(source string, content []byte) []Event {
	var events []Event
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if line == "" || !(strings.Contains(lower, "install") || strings.Contains(lower, "upgrade") || strings.Contains(lower, "remove")) {
			continue
		}
		events = append(events, Event{Fingerprint: packageEventFingerprint(source + "|" + line), OccurredAt: time.Now().UTC(), Source: source + "-log", Platform: string(platform.Linux), Category: "package", Severity: SeverityInfo, Title: "Package manager change", Summary: line, Details: map[string]any{"manager": source}})
	}
	return events
}

func packageEventFingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
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

// collectKernelSignals greps the journal for hardware/driver kernel signals.
//
// Incremental strategy (gap-free):
//   - Immutable historical boots (Index < 0) are scanned AT MOST ONCE. A
//     persisted per-boot marker lets subsequent ingests skip them entirely
//     instead of re-grepping every minute.
//   - The current boot (Index == 0) is read incrementally via a persisted
//     journald cursor: only entries newer than the last successfully-ingested
//     one are fetched. On cold start or cursor invalidation (journal
//     vacuum/rotation, reboot) we fall back to a bounded `-b 0 -n N` re-scan so
//     no event is silently skipped.
//
// The cursor advances ONLY after a successful read, so a failed ingest never
// moves past unread events.
//
// With a nil CursorStore the method degrades to scanning every boot each call
// (legacy behavior) — still correct, just not incremental.
func (c *HostCollector) collectKernelSignals(ctx context.Context, boots []journal.BootRecord) []Event {
	var events []Event
	for _, boot := range boots {
		if boot.Index == 0 {
			events = append(events, c.collectCurrentBootKernelSignals(ctx, boot)...)
			continue
		}
		events = append(events, c.collectHistoricalBootKernelSignals(ctx, boot)...)
	}
	return events
}

// collectHistoricalBootKernelSignals scans an immutable past boot once and
// records a marker so future ingests skip it.
func (c *HostCollector) collectHistoricalBootKernelSignals(ctx context.Context, boot journal.BootRecord) []Event {
	if c.cursors != nil && boot.BootID != "" {
		if scanned, err := c.cursors.IsBootScanned(ctx, kernelSignalSourceKey, boot.BootID); err == nil && scanned {
			c.execsAvoided++
			return nil
		}
	}
	logs, err := c.journal.QueryLogs(ctx, journal.QueryOpts{
		Kernel: true,
		Boot:   boot.BootID,
		Grep:   kernelGrepPattern,
		Tail:   currentBootRescanTail,
	})
	if err != nil {
		// Leave the boot unmarked so the next ingest retries it — never mark a
		// boot scanned on a failed read.
		return nil
	}
	events := kernelEventsFrom(logs, boot.BootID)
	if c.cursors != nil && boot.BootID != "" {
		// Best-effort: a marker write failure just means we re-scan next time.
		_ = c.cursors.MarkBootScanned(ctx, kernelSignalSourceKey, boot.BootID)
	}
	return events
}

// collectCurrentBootKernelSignals reads the live boot incrementally via a
// persisted cursor, falling back to a bounded re-scan on cold start or cursor
// invalidation.
func (c *HostCollector) collectCurrentBootKernelSignals(ctx context.Context, boot journal.BootRecord) []Event {
	if c.cursors == nil {
		// Legacy path: bounded scan of the current boot every time.
		logs, err := c.journal.QueryLogs(ctx, journal.QueryOpts{
			Kernel: true,
			Boot:   "0",
			Grep:   kernelGrepPattern,
			Tail:   currentBootRescanTail,
		})
		if err != nil {
			return nil
		}
		return kernelEventsFrom(logs, boot.BootID)
	}

	prev, err := c.cursors.GetJournalCursor(ctx, kernelSignalSourceKey)
	if err != nil {
		prev = CursorState{}
	}

	useCursor := prev.Cursor != "" && prev.BootID == boot.BootID
	if useCursor {
		logs, qerr := c.journal.QueryLogs(ctx, journal.QueryOpts{
			Kernel:      true,
			Boot:        "0",
			Grep:        kernelGrepPattern,
			ShowCursor:  true,
			AfterCursor: prev.Cursor,
		})
		if qerr == nil {
			// Incremental read succeeded: we read only the delta instead of a
			// full bounded re-scan.
			c.execsAvoided++
			c.advanceCursor(ctx, prev, logs, boot.BootID)
			return kernelEventsFrom(logs, boot.BootID)
		}
		// Cursor invalidated (vacuum/rotation) or transient failure: fall
		// through to a bounded re-scan. Do NOT advance the cursor here.
	}

	// Cold start, boot changed, or cursor invalidation: bounded re-scan.
	logs, err := c.journal.QueryLogs(ctx, journal.QueryOpts{
		Kernel:     true,
		Boot:       "0",
		Grep:       kernelGrepPattern,
		ShowCursor: true,
		Tail:       currentBootRescanTail,
	})
	if err != nil {
		// Failed re-scan: leave the cursor untouched so the next ingest retries
		// the same window — no silent event loss.
		return nil
	}
	c.advanceCursor(ctx, prev, logs, boot.BootID)
	return kernelEventsFrom(logs, boot.BootID)
}

// advanceCursor persists the cursor of the newest entry that carries one,
// pinned to the current boot. If no entry carries a cursor (e.g. the delta was
// empty), the prior cursor is retained so the next read resumes from the same
// place.
func (c *HostCollector) advanceCursor(ctx context.Context, prev CursorState, logs []journal.LogEntry, bootID string) {
	if c.cursors == nil {
		return
	}
	next := prev
	next.BootID = bootID
	for i := len(logs) - 1; i >= 0; i-- {
		if logs[i].Cursor != "" {
			next.Cursor = logs[i].Cursor
			break
		}
	}
	if next.Cursor == "" && next.BootID == prev.BootID {
		// Nothing new to advance to and the boot is unchanged: skip the write.
		return
	}
	next.UpdatedAt = c.now()
	_ = c.cursors.SetJournalCursor(ctx, kernelSignalSourceKey, next)
}

func kernelEventsFrom(logs []journal.LogEntry, bootID string) []Event {
	var events []Event
	for _, entry := range logs {
		if event, ok := kernelLogEvent(entry, bootID); ok {
			events = append(events, event)
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
