package logvolumebounds

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// The safeguard bounds the flat-file system log store so that no writer, however
// broken, can fill the disk through it. It exists because on 2026-09-01 a wedged
// gnome-keyring-daemon logged one error line ~50,000 times a second into
// /var/log/syslog and /var/log/auth.log; the two files reached 320 GB in 70
// hours while journald, which has a size cap, stayed at 4 GB. See doc.go.
const (
	// stanzaPath is the distribution's logrotate stanza for the rsyslog files.
	// It is the only file that knows which flat logs exist, so the safeguard
	// edits it in place rather than declaring a second stanza for the same
	// files (logrotate rejects duplicate log entries).
	stanzaPath = "/etc/logrotate.d/rsyslog"
	// stanzaBackupPath preserves the distribution stanza before the first
	// edit. It lives outside /etc/logrotate.d because logrotate parses every
	// file in that directory.
	stanzaBackupDir  = "/var/lib/vrooli/log-volume-bounds"
	stanzaBackupPath = stanzaBackupDir + "/logrotate-rsyslog.orig"

	timerDropInDir  = "/etc/systemd/system/logrotate.timer.d"
	timerDropInPath = timerDropInDir + "/99-vrooli-hourly.conf"

	// rsyslogConfPath is the main rsyslog configuration. Distributions load
	// imuxsock there with RainerScript syntax (`module(load="imuxsock")`), and
	// rsyslog refuses the legacy `$SystemLogRateLimit*` directives once a module
	// has been loaded that way, so the rate limit has to be set on the module
	// line itself. The original is preserved once, like the logrotate stanza.
	rsyslogConfPath       = "/etc/rsyslog.conf"
	rsyslogConfBackupPath = stanzaBackupDir + "/rsyslog.conf.orig"
	// rsyslogDropInPath is used only when rsyslog.conf loads imuxsock with the
	// legacy `$ModLoad` directive, where the legacy rate-limit directives are
	// still accepted from an included file.
	rsyslogDropInPath = "/etc/rsyslog.d/05-vrooli-ratelimit.conf"
	rateLimitMarker   = managedMarker + " imuxsock rate limit"

	// tailDir receives the last bytes of any log the safeguard truncates, so
	// an emergency reclaim never destroys the only evidence of what flooded.
	tailDir = "/var/log/vrooli/log-volume-bounds"

	managedMarker = "# vrooli:log_volume_bounds"
	headerLine    = managedMarker + " managed -- do not edit manually; the distribution stanza is preserved at " + stanzaBackupPath
	boundMarker   = managedMarker + " maxsize"

	defaultMaxSize             = "1G"
	defaultEmergencyMultiplier = 8
	defaultRateLimitInterval   = 5
	defaultRateLimitBurst      = 1000
	tailBytes                  = mebibyte
	// insertedLinesPerBlock is the header plus the marker and maxsize pair.
	insertedLinesPerBlock = 3

	kibibyte        = int64(1) << 10
	mebibyte        = int64(1) << 20
	gibibyte        = int64(1) << 30
	decimalGigabyte = int64(1_000_000_000)
)

var (
	maxSizePattern = regexp.MustCompile(`^[0-9]+[kMG]?$`)
	// imuxsockModuleLine matches a RainerScript imuxsock load, capturing the
	// parameter list so the rate-limit parameters can be added or refreshed.
	imuxsockModuleLine = regexp.MustCompile(`^(\s*module\(\s*load="imuxsock")([^)]*)(\).*)$`)
	rateLimitParams    = regexp.MustCompile(`\s*SysSock\.RateLimit\.(Interval|Burst)="[0-9]+"`)
	legacyImuxsockLoad = regexp.MustCompile(`^\s*\$ModLoad\s+imuxsock\b`)
)

// Seams for the two filesystem reads the handler needs beyond ReadFileFn. Both
// are package variables so tests can drive the emergency path without a
// hundred-gigabyte fixture.
var (
	fileSizeFn = func(path string) (int64, error) {
		info, err := os.Stat(path)
		if err != nil {
			return 0, err
		}
		return info.Size(), nil
	}
	readTailFn = func(path string, n int64) ([]byte, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}
		if info.Size() > n {
			if _, err := f.Seek(info.Size()-n, io.SeekStart); err != nil {
				return nil, err
			}
		}
		return io.ReadAll(f)
	}
	nowFn = time.Now
)

type settings struct {
	MaxSize             string
	EmergencyMultiplier int
	RateLimitInterval   int
	RateLimitBurst      int
}

func resolveSettings(config map[string]any) settings {
	s := settings{
		MaxSize:             defaultMaxSize,
		EmergencyMultiplier: defaultEmergencyMultiplier,
		RateLimitInterval:   defaultRateLimitInterval,
		RateLimitBurst:      defaultRateLimitBurst,
	}
	if v, ok := config["max_size"].(string); ok && maxSizePattern.MatchString(strings.TrimSpace(v)) {
		s.MaxSize = strings.TrimSpace(v)
	}
	if v, ok := intFromConfig(config["emergency_multiplier"]); ok && v >= 2 {
		s.EmergencyMultiplier = v
	}
	if v, ok := intFromConfig(config["rate_limit_interval_seconds"]); ok && v >= 1 {
		s.RateLimitInterval = v
	}
	if v, ok := intFromConfig(config["rate_limit_burst"]); ok && v >= 100 {
		s.RateLimitBurst = v
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

// maxSizeBytes converts a logrotate size token (1G, 512M, 100k, 4096) to bytes.
func maxSizeBytes(token string) int64 {
	if token == "" {
		return 0
	}
	unit := int64(1)
	switch token[len(token)-1] {
	case 'k':
		unit = kibibyte
	case 'M':
		unit = mebibyte
	case 'G':
		unit = gibibyte
	}
	digits := strings.TrimRight(token, "kMG")
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0
	}
	return n * unit
}

func (s settings) emergencyBytes() int64 {
	return maxSizeBytes(s.MaxSize) * int64(s.EmergencyMultiplier)
}

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

// stripManaged returns the stanza with every line this safeguard added removed,
// so the distribution content can be recovered even when the backup is gone.
func stripManaged(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	skipNext := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if skipNext {
			skipNext = false
			if strings.HasPrefix(trimmed, "maxsize ") {
				continue
			}
		}
		if trimmed == boundMarker {
			skipNext = true
			continue
		}
		if strings.HasPrefix(trimmed, managedMarker) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// renderStanza inserts a maxsize bound into every block of the distribution
// stanza that does not already carry a size directive. Nothing else changes:
// the file list, schedule, rotate count and scripts stay the distribution's.
func renderStanza(baseline, maxSize string) string {
	baseline = strings.TrimRight(stripManaged(baseline), "\n")
	lines := strings.Split(baseline, "\n")
	out := make([]string, 0, len(lines)+insertedLinesPerBlock)
	out = append(out, headerLine)
	depth := 0
	blockStart := -1
	for _, line := range lines {
		out = append(out, line)
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, "{") {
			depth++
			if depth == 1 {
				blockStart = len(out)
				out = append(out, "\t"+boundMarker, "\tmaxsize "+maxSize)
			}
			continue
		}
		if trimmed == "}" && depth > 0 {
			depth--
			if depth == 0 && blockStart >= 0 {
				// If the distribution block already bounds itself, withdraw ours.
				if blockHasOwnSize(out[blockStart+2 : len(out)-1]) {
					out = append(out[:blockStart], out[blockStart+2:]...)
				}
				blockStart = -1
			}
		}
	}
	return strings.Join(out, "\n") + "\n"
}

func blockHasOwnSize(lines []string) bool {
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "maxsize ") || strings.HasPrefix(t, "size ") || strings.HasPrefix(t, "minsize ") {
			return true
		}
	}
	return false
}

// boundedFiles lists the log paths the stanza rotates, in order.
func boundedFiles(stanza string) []string {
	var files []string
	seen := map[string]bool{}
	depth := 0
	for _, line := range strings.Split(stanza, "\n") {
		t := strings.TrimSpace(line)
		if t == "}" {
			depth--
			continue
		}
		if strings.HasSuffix(t, "{") {
			depth++
		}
		// Paths inside a block (postrotate scripts, olddir targets) are not logs.
		if depth > 1 || (depth == 1 && !strings.HasSuffix(t, "{")) || !strings.HasPrefix(t, "/") {
			continue
		}
		for _, field := range strings.Fields(strings.TrimSuffix(t, "{")) {
			if strings.HasPrefix(field, "/") && !seen[field] {
				seen[field] = true
				files = append(files, field)
			}
		}
	}
	return files
}

func timerDropInContent() string {
	return strings.Join([]string{
		"# Managed by Vrooli (log_volume_bounds) -- do not edit manually",
		"# logrotate runs hourly so the maxsize bound in /etc/logrotate.d/rsyslog can",
		"# catch a runaway writer within the hour instead of at the weekly rotation.",
		"[Timer]",
		"OnCalendar=",
		"OnCalendar=hourly",
		"AccuracySec=5m",
		"",
	}, "\n")
}

// rsyslogMode says how the host loads imuxsock, which decides where the rate
// limit can legally be expressed.
type rsyslogMode int

const (
	rsyslogModeNone   rsyslogMode = iota // imuxsock not loaded; nothing to limit
	rsyslogModeModule                    // module(load="imuxsock" ...) — edit the line
	rsyslogModeLegacy                    // $ModLoad imuxsock — legacy drop-in works
)

func detectRsyslogMode(conf string) rsyslogMode {
	for _, line := range strings.Split(conf, "\n") {
		if imuxsockModuleLine.MatchString(line) {
			return rsyslogModeModule
		}
		if legacyImuxsockLoad.MatchString(line) {
			return rsyslogModeLegacy
		}
	}
	return rsyslogModeNone
}

// stripRsyslogManaged removes the marker line and the rate-limit parameters
// this safeguard added to the imuxsock module line, recovering the baseline.
func stripRsyslogManaged(conf string) string {
	lines := strings.Split(conf, "\n")
	out := make([]string, 0, len(lines))
	fixNext := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == rateLimitMarker {
			fixNext = true
			continue
		}
		if strings.HasPrefix(trimmed, managedMarker) {
			continue
		}
		if fixNext {
			fixNext = false
			if m := imuxsockModuleLine.FindStringSubmatch(line); m != nil {
				line = m[1] + rateLimitParams.ReplaceAllString(m[2], "") + m[3]
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// renderRsyslogConf adds SysSock rate-limit parameters to the imuxsock module
// line. A module line that already carries its own rate-limit parameters is the
// distribution's decision and is left alone. Everything else in the file is
// kept byte for byte.
func renderRsyslogConf(baseline string, s settings) string {
	baseline = stripRsyslogManaged(baseline)
	lines := strings.Split(baseline, "\n")
	out := make([]string, 0, len(lines)+insertedLinesPerBlock)
	done := false
	for _, line := range lines {
		if !done {
			if m := imuxsockModuleLine.FindStringSubmatch(line); m != nil {
				done = true
				if !rateLimitParams.MatchString(m[2]) {
					params := fmt.Sprintf(`%s SysSock.RateLimit.Interval="%d" SysSock.RateLimit.Burst="%d"`, m[2], s.RateLimitInterval, s.RateLimitBurst)
					out = append(out, rateLimitMarker, m[1]+params+m[3])
					continue
				}
			}
		}
		out = append(out, line)
	}
	rendered := strings.Join(out, "\n")
	if !strings.HasSuffix(rendered, "\n") {
		rendered += "\n"
	}
	return rendered
}

func baselineRsyslogConf(current string) string {
	if backup, err := hostreqkit.ReadFileFn(rsyslogConfBackupPath); err == nil && strings.TrimSpace(string(backup)) != "" {
		return string(backup)
	}
	return stripRsyslogManaged(current)
}

func rsyslogDropInContent(s settings) string {
	return strings.Join([]string{
		"# Managed by Vrooli (log_volume_bounds) -- do not edit manually",
		"# Per-process ceiling on the system log socket. One process may emit",
		fmt.Sprintf("# %d messages per %d seconds; the excess is dropped and rsyslog keeps one", s.RateLimitBurst, s.RateLimitInterval),
		"# summary line. Ordinary services never approach this. A daemon in a tight",
		"# error loop (50,000 lines/s observed 2026-09-01) is cut to the ceiling.",
		fmt.Sprintf("$SystemLogRateLimitInterval %d", s.RateLimitInterval),
		fmt.Sprintf("$SystemLogRateLimitBurst %d", s.RateLimitBurst),
		"",
	}, "\n")
}

// baselineStanza returns the distribution content the rendered stanza is
// derived from: the preserved backup when one exists, otherwise the current
// file with any managed lines removed.
func baselineStanza(current string) string {
	if backup, err := hostreqkit.ReadFileFn(stanzaBackupPath); err == nil && strings.TrimSpace(string(backup)) != "" {
		return string(backup)
	}
	return stripManaged(current)
}

type oversize struct {
	Path  string
	Bytes int64
	// Rotated marks the `.1` copy logrotate leaves behind (uncompressed under
	// `delaycompress`). A rotated copy of a flood is removed rather than
	// truncated: nothing writes to it, and it would otherwise hold its bytes
	// until enough further rotations age it out.
	Rotated bool
}

// oversizeFiles reports every bounded log, and its most recent rotated copy,
// whose size is past the emergency threshold, largest first.
func oversizeFiles(files []string, threshold int64) []oversize {
	var out []oversize
	for _, path := range files {
		for _, candidate := range []oversize{{Path: path}, {Path: path + ".1", Rotated: true}} {
			size, err := fileSizeFn(candidate.Path)
			if err != nil || size <= threshold {
				continue
			}
			candidate.Bytes = size
			out = append(out, candidate)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	return out
}

func gigabytes(n int64) string {
	return fmt.Sprintf("%.1f GB", float64(n)/float64(decimalGigabyte))
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "not applicable: macOS unified logs/newsyslog or Windows event logs provide their own bounded log store; Linux flat-file rsyslog bounds do not apply")
		return status
	}
	if !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "requires systemd to run logrotate hourly")
		return status
	}
	if _, err := hostreqkit.LookPathFn("logrotate"); err != nil {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "logrotate is not installed")
		return status
	}
	currentBytes, err := hostreqkit.ReadFileFn(stanzaPath)
	if err != nil {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "no rsyslog logrotate stanza at "+stanzaPath+"; the host does not keep flat syslog files")
		return status
	}
	current := string(currentBytes)
	s := resolveSettings(requirement.Config)
	desired := renderStanza(baselineStanza(current), s.MaxSize)

	var pending []string
	if current != desired {
		pending = append(pending, "logrotate stanza does not bound each file at maxsize "+s.MaxSize)
	}
	if !hostreqkit.FileContentMatches(timerDropInPath, timerDropInContent()) {
		pending = append(pending, "logrotate.timer is not overridden to run hourly")
	}
	if !rateLimitApplied(s) {
		pending = append(pending, fmt.Sprintf("rsyslog has no per-process rate limit (%d msgs / %d s)", s.RateLimitBurst, s.RateLimitInterval))
	}
	over := oversizeFiles(boundedFiles(desired), s.emergencyBytes())
	for _, o := range over {
		kind := ""
		if o.Rotated {
			kind = " (rotated copy)"
		}
		pending = append(pending, fmt.Sprintf("%s%s is %s, over the emergency threshold of %s", o.Path, kind, gigabytes(o.Bytes), gigabytes(s.emergencyBytes())))
	}

	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, fmt.Sprintf("flat logs bounded at %s each, rotated hourly, rsyslog rate-limited", s.MaxSize))
		return status
	}
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

	s := resolveSettings(status.Config)
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes,
			fmt.Sprintf("dry-run: would bound %s at maxsize %s, run logrotate hourly, rate-limit rsyslog, and truncate any log over %s", stanzaPath, s.MaxSize, gigabytes(s.emergencyBytes())))
		return status, nil
	}
	currentBytes, err := hostreqkit.ReadFileFn(stanzaPath)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "no rsyslog logrotate stanza at "+stanzaPath)
		return status, nil
	}
	current := string(currentBytes)
	desired := renderStanza(baselineStanza(current), s.MaxSize)
	over := oversizeFiles(boundedFiles(desired), s.emergencyBytes())

	fail := func(step string, err error) (hostreqkit.ItemStatus, error) {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, step+": "+err.Error())
		return status, nil
	}

	// 1. Preserve the distribution stanza once, then install the bounded one.
	if _, err := hostreqkit.ReadFileFn(stanzaBackupPath); err != nil && !strings.Contains(current, managedMarker) {
		if err := hostreqkit.EnsureManagedDir(stanzaBackupDir, opts.SudoMode, opts); err != nil {
			return fail("create backup directory", err)
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "cp", []string{stanzaPath, stanzaBackupPath}, opts); err != nil {
			return fail("preserve distribution stanza", err)
		}
	}
	if current != desired {
		if err := hostreqkit.InstallManagedContent(stanzaPath, desired, opts.SudoMode, opts); err != nil {
			return fail("install bounded logrotate stanza", err)
		}
		status.Notes = append(status.Notes, fmt.Sprintf("logrotate stanza bounded at maxsize %s", s.MaxSize))
	}

	// 2. Run logrotate hourly so the bound is enforced within the hour.
	if !hostreqkit.FileContentMatches(timerDropInPath, timerDropInContent()) {
		if err := hostreqkit.EnsureManagedDir(timerDropInDir, opts.SudoMode, opts); err != nil {
			return fail("create timer drop-in directory", err)
		}
		if err := hostreqkit.InstallManagedContent(timerDropInPath, timerDropInContent(), opts.SudoMode, opts); err != nil {
			return fail("install hourly logrotate timer", err)
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"daemon-reload"}, opts); err != nil {
			return fail("systemctl daemon-reload", err)
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"restart", "logrotate.timer"}, opts); err != nil {
			return fail("restart logrotate.timer", err)
		}
		status.Notes = append(status.Notes, "logrotate.timer now runs hourly")
	}

	// 3. Cap what any single process can push into the flat logs.
	if !rateLimitApplied(s) {
		if err := h.applyRateLimit(s, opts); err != nil {
			return fail(err.Error(), errors.Unwrap(err))
		}
		status.Notes = append(status.Notes, fmt.Sprintf("rsyslog rate limit %d msgs / %d s per process", s.RateLimitBurst, s.RateLimitInterval))
	}

	// 4. Emergency reclaim: a log already far past the bound is a disk-full
	// incident in progress. Keep its last megabyte as evidence, then truncate.
	// rsyslog opens its files O_APPEND, so writes resume at the new end.
	for _, o := range over {
		if err := hostreqkit.EnsureManagedDir(tailDir, opts.SudoMode, opts); err != nil {
			return fail("create evidence directory", err)
		}
		tail, err := readTailFn(o.Path, tailBytes)
		if err != nil {
			return fail("read tail of "+o.Path, err)
		}
		stamp := nowFn().UTC().Format("20060102T150405Z")
		evidence := filepath.Join(tailDir, fmt.Sprintf("%s.%s.%s.tail", filepath.Base(o.Path), stamp, gigabytesSlug(o.Bytes)))
		verb := "truncated"
		if o.Rotated {
			verb = "removed"
		}
		header := fmt.Sprintf("# log_volume_bounds %s %s at %s; it was %d bytes. Last %d bytes follow.\n", verb, o.Path, stamp, o.Bytes, len(tail))
		if err := hostreqkit.InstallManagedContent(evidence, header+string(tail), opts.SudoMode, opts); err != nil {
			return fail("preserve tail of "+o.Path, err)
		}
		if o.Rotated {
			if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "rm", []string{"-f", o.Path}, opts); err != nil {
				return fail("remove rotated "+o.Path, err)
			}
			status.Notes = append(status.Notes, fmt.Sprintf("removed rotated %s (%s); last %d KB preserved at %s", o.Path, gigabytes(o.Bytes), int64(len(tail))/kibibyte, evidence))
			continue
		}
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "truncate", []string{"-s", "0", o.Path}, opts); err != nil {
			return fail("truncate "+o.Path, err)
		}
		status.Notes = append(status.Notes, fmt.Sprintf("truncated %s (%s); last %d KB preserved at %s", o.Path, gigabytes(o.Bytes), int64(len(tail))/kibibyte, evidence))
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, fmt.Sprintf("flat logs bounded at %s each, rotated hourly, rsyslog rate-limited", s.MaxSize))
	return status, nil
}

// rateLimitApplied reports whether the host's imuxsock input already carries
// the wanted per-process ceiling, in whichever form its configuration allows.
func rateLimitApplied(s settings) bool {
	confBytes, err := hostreqkit.ReadFileFn(rsyslogConfPath)
	if err != nil {
		return true // no rsyslog.conf: nothing to limit, nothing pending
	}
	conf := string(confBytes)
	switch detectRsyslogMode(conf) {
	case rsyslogModeModule:
		return conf == renderRsyslogConf(baselineRsyslogConf(conf), s)
	case rsyslogModeLegacy:
		return hostreqkit.FileContentMatches(rsyslogDropInPath, rsyslogDropInContent(s))
	default:
		return true
	}
}

// stepError carries the step name for the operator note and the cause.
type stepError struct {
	step  string
	cause error
}

func (e *stepError) Error() string { return e.step }
func (e *stepError) Unwrap() error { return e.cause }

// applyRateLimit writes the rate limit where rsyslog will accept it, validates
// the whole configuration with rsyslogd before restarting, and puts the
// previous content back if validation fails so a typo can never take the log
// service down.
func (h handler) applyRateLimit(s settings, opts hostreqkit.EnsureOptions) error {
	confBytes, err := hostreqkit.ReadFileFn(rsyslogConfPath)
	if err != nil {
		return nil
	}
	current := string(confBytes)
	var (
		target, desired, previous string
		hadPrevious               bool
	)
	switch detectRsyslogMode(current) {
	case rsyslogModeModule:
		target, desired, previous, hadPrevious = rsyslogConfPath, renderRsyslogConf(baselineRsyslogConf(current), s), current, true
		if _, err := hostreqkit.ReadFileFn(rsyslogConfBackupPath); err != nil && !strings.Contains(current, managedMarker) {
			if err := hostreqkit.EnsureManagedDir(stanzaBackupDir, opts.SudoMode, opts); err != nil {
				return &stepError{"create backup directory", err}
			}
			if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "cp", []string{rsyslogConfPath, rsyslogConfBackupPath}, opts); err != nil {
				return &stepError{"preserve distribution rsyslog.conf", err}
			}
		}
	case rsyslogModeLegacy:
		target, desired = rsyslogDropInPath, rsyslogDropInContent(s)
		if prev, err := hostreqkit.ReadFileFn(rsyslogDropInPath); err == nil {
			previous, hadPrevious = string(prev), true
		}
	default:
		return nil
	}

	if err := hostreqkit.InstallManagedContent(target, desired, opts.SudoMode, opts); err != nil {
		return &stepError{"install rsyslog rate limit in " + target, err}
	}
	if _, err := hostreqkit.LookPathFn("rsyslogd"); err == nil {
		if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "rsyslogd", []string{"-N1"}, opts); err != nil {
			if hadPrevious {
				_ = hostreqkit.InstallManagedContent(target, previous, opts.SudoMode, opts)
			} else {
				_ = hostreqkit.RunPrivilegedCommand(opts.SudoMode, "rm", []string{"-f", target}, opts)
			}
			return &stepError{"rsyslog rejected the rate-limit change to " + target + " (previous content restored)", err}
		}
	}
	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"restart", "rsyslog"}, opts); err != nil {
		return &stepError{"restart rsyslog", err}
	}
	return nil
}

func gigabytesSlug(n int64) string {
	return fmt.Sprintf("%dGB", n/decimalGigabyte)
}
