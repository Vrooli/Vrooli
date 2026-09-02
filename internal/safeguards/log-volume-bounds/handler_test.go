package logvolumebounds

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const distroStanza = `/var/log/syslog
/var/log/mail.log
/var/log/kern.log
/var/log/auth.log
/var/log/user.log
/var/log/cron.log
{
	rotate 4
	weekly
	missingok
	notifempty
	compress
	delaycompress
	sharedscripts
	postrotate
		/usr/lib/rsyslog/rsyslog-rotate
	endscript
}
`

const distroRsyslogConf = `# /etc/rsyslog.conf configuration file for rsyslog
module(load="imuxsock") # provides support for local system logging
module(load="imklog" permitnonkernelfacility="on")
$RepeatedMsgReduction on
$IncludeConfig /etc/rsyslog.d/*.conf
`

const legacyRsyslogConf = "$ModLoad imuxsock\n$ModLoad imklog\n$IncludeConfig /etc/rsyslog.d/*.conf\n"

func newTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "log_volume_bounds", Handler: "log_volume_bounds"})
}

func linuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "log_volume_bounds", Kind: hostreqspec.KindSafeguard, Required: true}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux", PackageManager: "apt-get", SupportsSysctl: true, SupportsSystemd: true}
}

// fixture wires every seam to an in-memory host so Inspect and Apply can be
// exercised end to end without touching /etc.
type fixture struct {
	files    map[string]string
	sizes    map[string]int64
	commands []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	restore := hostreqkittest.StubLookups(t)
	f := &fixture{files: map[string]string{}, sizes: map[string]int64{}}
	f.files[rsyslogConfPath] = distroRsyslogConf
	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "logrotate", "rsyslogd", "systemctl":
			return "/usr/sbin/" + name, nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if content, ok := f.files[path]; ok {
			return []byte(content), nil
		}
		return nil, os.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		f.commands = append(f.commands, name+" "+strings.Join(args, " "))
		// Mirror the effect of `install` and `cp` so later reads see the write.
		switch name {
		case "install":
			src, dst := args[len(args)-2], args[len(args)-1]
			content, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			f.files[dst] = string(content)
		case "cp":
			f.files[args[1]] = f.files[args[0]]
		case "truncate":
			f.sizes[args[len(args)-1]] = 0
		case "rm":
			delete(f.sizes, args[len(args)-1])
			delete(f.files, args[len(args)-1])
		}
		return nil
	}
	// Setup runs as root; the privileged seam refuses to elevate otherwise.
	origRoot := hostreqkit.RunningAsRootFn
	hostreqkit.RunningAsRootFn = func() bool { return true }
	origSize, origTail, origNow := fileSizeFn, readTailFn, nowFn
	fileSizeFn = func(path string) (int64, error) {
		if size, ok := f.sizes[path]; ok {
			return size, nil
		}
		return 0, os.ErrNotExist
	}
	readTailFn = func(path string, n int64) ([]byte, error) {
		return []byte("gnome-keyring-d: couldn't accept new control request: Too many open files\n"), nil
	}
	nowFn = func() time.Time { return time.Date(2026, 9, 2, 3, 0, 0, 0, time.UTC) }
	t.Cleanup(func() {
		restore()
		fileSizeFn, readTailFn, nowFn = origSize, origTail, origNow
		hostreqkit.RunningAsRootFn = origRoot
	})
	return f
}

func (f *fixture) ran(prefix string) bool {
	for _, c := range f.commands {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func TestRenderStanzaInsertsBoundOnce(t *testing.T) {
	rendered := renderStanza(distroStanza, "1G")
	if strings.Count(rendered, "\tmaxsize 1G\n") != 1 {
		t.Fatalf("expected exactly one maxsize line, got:\n%s", rendered)
	}
	if !strings.HasPrefix(rendered, headerLine) {
		t.Fatalf("rendered stanza lacks the managed header")
	}
	// The bound sits inside the block, immediately after the opening brace.
	if !strings.Contains(rendered, "{\n\t"+boundMarker+"\n\tmaxsize 1G\n\trotate 4\n") {
		t.Fatalf("maxsize not placed after the opening brace:\n%s", rendered)
	}
	// Rendering the rendered output again is a fixed point.
	if again := renderStanza(rendered, "1G"); again != rendered {
		t.Fatalf("renderStanza is not idempotent:\n%s\n---\n%s", rendered, again)
	}
	// Stripping recovers the distribution content byte for byte.
	if got := stripManaged(rendered); strings.TrimRight(got, "\n") != strings.TrimRight(distroStanza, "\n") {
		t.Fatalf("stripManaged did not recover the original:\n%s", got)
	}
}

func TestRenderStanzaRespectsExistingSizeDirective(t *testing.T) {
	self := "/var/log/foo.log\n{\n\tsize 100M\n\trotate 2\n}\n"
	rendered := renderStanza(self, "1G")
	if strings.Contains(rendered, "maxsize") {
		t.Fatalf("a block that already bounds itself must be left alone:\n%s", rendered)
	}
}

func TestRenderStanzaChangesBoundWhenSettingChanges(t *testing.T) {
	first := renderStanza(distroStanza, "1G")
	second := renderStanza(first, "512M")
	if strings.Contains(second, "maxsize 1G") || !strings.Contains(second, "maxsize 512M") {
		t.Fatalf("bound did not follow the setting:\n%s", second)
	}
}

func TestRenderRsyslogConfEditsTheModuleLineOnce(t *testing.T) {
	s := resolveSettings(nil)
	rendered := renderRsyslogConf(distroRsyslogConf, s)
	want := rateLimitMarker + "\nmodule(load=\"imuxsock\" SysSock.RateLimit.Interval=\"5\" SysSock.RateLimit.Burst=\"1000\") # provides support for local system logging\n"
	if !strings.Contains(rendered, want) {
		t.Fatalf("module line not rewritten as expected:\n%s", rendered)
	}
	if strings.Count(rendered, "SysSock.RateLimit.Interval") != 1 {
		t.Fatalf("rate limit inserted more than once:\n%s", rendered)
	}
	if again := renderRsyslogConf(rendered, s); again != rendered {
		t.Fatalf("renderRsyslogConf is not idempotent:\n%s\n---\n%s", rendered, again)
	}
	if got := stripRsyslogManaged(rendered); got != distroRsyslogConf {
		t.Fatalf("stripRsyslogManaged did not recover the original:\n%s", got)
	}
	// A changed setting replaces the parameters instead of stacking them.
	changed := renderRsyslogConf(rendered, resolveSettings(map[string]any{"rate_limit_burst": 2000}))
	if !strings.Contains(changed, `Burst="2000"`) || strings.Contains(changed, `Burst="1000"`) {
		t.Fatalf("setting change not applied:\n%s", changed)
	}
}

func TestRenderRsyslogConfRespectsDistributionRateLimit(t *testing.T) {
	own := "module(load=\"imuxsock\" SysSock.RateLimit.Interval=\"1\" SysSock.RateLimit.Burst=\"50\")\n"
	if got := renderRsyslogConf(own, resolveSettings(nil)); got != own {
		t.Fatalf("a module line with its own rate limit must be left alone:\n%s", got)
	}
}

func TestDetectRsyslogMode(t *testing.T) {
	if detectRsyslogMode(distroRsyslogConf) != rsyslogModeModule {
		t.Fatal("RainerScript load not detected")
	}
	if detectRsyslogMode(legacyRsyslogConf) != rsyslogModeLegacy {
		t.Fatal("legacy load not detected")
	}
	if detectRsyslogMode("module(load=\"imklog\")\n") != rsyslogModeNone {
		t.Fatal("absent imuxsock not detected")
	}
}

func TestBoundedFilesParsesEveryPath(t *testing.T) {
	got := boundedFiles(renderStanza(distroStanza, "1G"))
	want := []string{"/var/log/syslog", "/var/log/mail.log", "/var/log/kern.log", "/var/log/auth.log", "/var/log/user.log", "/var/log/cron.log"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("boundedFiles = %v, want %v", got, want)
	}
}

func TestResolveSettingsDefaultsAndOverrides(t *testing.T) {
	s := resolveSettings(nil)
	if s.MaxSize != "1G" || s.EmergencyMultiplier != 8 || s.RateLimitInterval != 5 || s.RateLimitBurst != 1000 {
		t.Fatalf("defaults = %+v", s)
	}
	if s.emergencyBytes() != 8<<30 {
		t.Fatalf("emergencyBytes = %d, want 8 GiB", s.emergencyBytes())
	}
	s = resolveSettings(map[string]any{"max_size": "512M", "emergency_multiplier": float64(4), "rate_limit_burst": "2000", "rate_limit_interval_seconds": 10})
	if s.MaxSize != "512M" || s.EmergencyMultiplier != 4 || s.RateLimitBurst != 2000 || s.RateLimitInterval != 10 {
		t.Fatalf("overrides = %+v", s)
	}
	// Garbage is ignored, never applied.
	s = resolveSettings(map[string]any{"max_size": "lots", "emergency_multiplier": 1, "rate_limit_burst": 5})
	if s.MaxSize != "1G" || s.EmergencyMultiplier != 8 || s.RateLimitBurst != 1000 {
		t.Fatalf("invalid overrides leaked: %+v", s)
	}
}

func TestInspectPendingOnUnboundedHost(t *testing.T) {
	f := newFixture(t)
	f.files[stanzaPath] = distroStanza
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatalf("unbounded host reported Applied; notes %v", status.Notes)
	}
	joined := strings.Join(status.Notes, "\n")
	for _, want := range []string{"maxsize 1G", "hourly", "rate limit"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("notes missing %q: %v", want, status.Notes)
		}
	}
}

func TestInspectPendingWhenOnlyAnOversizeLogRemains(t *testing.T) {
	f := newFixture(t)
	s := resolveSettings(nil)
	f.files[stanzaPath] = renderStanza(distroStanza, "1G")
	f.files[timerDropInPath] = timerDropInContent()
	f.files[rsyslogConfPath] = renderRsyslogConf(distroRsyslogConf, s)
	f.sizes["/var/log/syslog"] = 158_000_000_000
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatalf("a 158 GB log must keep the safeguard pending; notes %v", status.Notes)
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), "/var/log/syslog is 158.0 GB") {
		t.Fatalf("notes did not name the oversize log: %v", status.Notes)
	}
}

func TestInspectAppliedWhenEverythingBounded(t *testing.T) {
	f := newFixture(t)
	s := resolveSettings(nil)
	f.files[stanzaPath] = renderStanza(distroStanza, "1G")
	f.files[timerDropInPath] = timerDropInContent()
	f.files[rsyslogConfPath] = renderRsyslogConf(distroRsyslogConf, s)
	f.sizes["/var/log/syslog"] = 700_000_000
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("expected Applied, got %q with notes %v", status.ExecutionState, status.Notes)
	}
}

func TestInspectNotApplicableWithoutFlatSyslog(t *testing.T) {
	newFixture(t) // no stanza file present
	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("journald-only host should be not_applicable, got %q", status.ExecutionState)
	}
}

func TestApplyBoundsTheHostAndTruncatesTheFlood(t *testing.T) {
	f := newFixture(t)
	f.files[stanzaPath] = distroStanza
	f.sizes["/var/log/syslog"] = 158_000_000_000
	f.sizes["/var/log/auth.log"] = 164_000_000_000
	f.sizes["/var/log/kern.log"] = 20_000_000
	// An earlier hourly rotation already moved one flood into its `.1` copy.
	f.sizes["/var/log/syslog.1"] = 169_000_000_000
	f.sizes["/var/log/kern.log.1"] = 900_000

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	status, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{SudoMode: "error"})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if status.ExecutionState != hostreqkit.ExecutionApplied || !status.Applied {
		t.Fatalf("ExecutionState = %q, notes %v", status.ExecutionState, status.Notes)
	}

	// The distribution stanza was preserved before being edited.
	if f.files[stanzaBackupPath] != distroStanza {
		t.Fatalf("backup not taken: %q", f.files[stanzaBackupPath])
	}
	if f.files[stanzaPath] != renderStanza(distroStanza, "1G") {
		t.Fatalf("stanza not bounded:\n%s", f.files[stanzaPath])
	}
	if f.files[timerDropInPath] != timerDropInContent() {
		t.Fatalf("timer drop-in not installed")
	}
	if f.files[rsyslogConfPath] != renderRsyslogConf(distroRsyslogConf, resolveSettings(nil)) || f.files[rsyslogConfBackupPath] != distroRsyslogConf {
		t.Fatalf("rsyslog.conf not rewritten with its original preserved:\n%s\n---\n%s", f.files[rsyslogConfPath], f.files[rsyslogConfBackupPath])
	}
	if _, wroteDropIn := f.files[rsyslogDropInPath]; wroteDropIn {
		t.Fatalf("legacy drop-in must not be written when imuxsock is loaded with module()")
	}
	for _, want := range []string{"systemctl daemon-reload", "systemctl restart logrotate.timer", "rsyslogd -N1", "systemctl restart rsyslog"} {
		if !f.ran(want) {
			t.Fatalf("expected command %q; ran %v", want, f.commands)
		}
	}

	// Only the two logs past the 8 GiB threshold were truncated, the largest first,
	// and each left its tail behind as evidence.
	if !f.ran("truncate -s 0 /var/log/auth.log") || !f.ran("truncate -s 0 /var/log/syslog") {
		t.Fatalf("oversize logs not truncated; ran %v", f.commands)
	}
	if f.ran("truncate -s 0 /var/log/kern.log") {
		t.Fatalf("a 20 MB log must never be truncated")
	}
	// The rotated flood copy is removed, never truncated; a small rotated copy is untouched.
	if !f.ran("rm -f /var/log/syslog.1") || f.ran("truncate -s 0 /var/log/syslog.1") {
		t.Fatalf("rotated flood copy not removed: %v", f.commands)
	}
	if f.ran("rm -f /var/log/kern.log.1") {
		t.Fatalf("a small rotated copy must not be removed")
	}
	evidence := 0
	for path, content := range f.files {
		if strings.HasPrefix(path, tailDir+"/") {
			evidence++
			if !strings.Contains(content, "Too many open files") || !strings.Contains(content, "# log_volume_bounds") {
				t.Fatalf("evidence file %s lacks tail or header:\n%s", path, content)
			}
		}
	}
	if evidence != 3 {
		t.Fatalf("expected 3 evidence files, found %d in %v", evidence, f.files)
	}
	if f.sizes["/var/log/syslog"] != 0 || f.sizes["/var/log/auth.log"] != 0 {
		t.Fatalf("truncation did not take effect")
	}

	// A second Inspect on the repaired host is clean, and Apply is a no-op.
	status = h.Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("host still pending after Apply: %v", status.Notes)
	}
	before := len(f.commands)
	if _, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{}); err != nil || len(f.commands) != before {
		t.Fatalf("Apply on an applied host ran commands: %v", f.commands[before:])
	}
}

func TestApplySkipsAlreadyBoundedPiecesAndKeepsExistingBackup(t *testing.T) {
	f := newFixture(t)
	f.files[stanzaBackupPath] = distroStanza
	f.files[stanzaPath] = renderStanza(distroStanza, "1G")
	f.files[timerDropInPath] = timerDropInContent()
	// Only the rsyslog rate limit is missing.
	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	status, _ = h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q, notes %v", status.ExecutionState, status.Notes)
	}
	if f.ran("cp "+stanzaPath) || f.ran("systemctl restart logrotate.timer") {
		t.Fatalf("already-bounded pieces were redone: %v", f.commands)
	}
	if !f.ran("systemctl restart rsyslog") {
		t.Fatalf("rsyslog rate limit not applied: %v", f.commands)
	}
}

func TestApplyRestoresRsyslogConfWhenValidationFails(t *testing.T) {
	f := newFixture(t)
	f.files[stanzaPath] = renderStanza(distroStanza, "1G")
	f.files[timerDropInPath] = timerDropInContent()
	base := hostreqkit.RunCommandFn
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		if name == "rsyslogd" {
			f.commands = append(f.commands, "rsyslogd -N1")
			return os.ErrInvalid
		}
		return base(name, args, opts)
	}
	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	status, _ = h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("expected failed, got %q", status.ExecutionState)
	}
	if f.files[rsyslogConfPath] != distroRsyslogConf {
		t.Fatalf("rsyslog.conf was not restored after rejection:\n%s", f.files[rsyslogConfPath])
	}
	if f.ran("systemctl restart rsyslog") {
		t.Fatalf("rsyslog must not be restarted onto a rejected config")
	}
	if !strings.Contains(strings.Join(status.Notes, "\n"), "previous content restored") {
		t.Fatalf("operator not told the file was restored: %v", status.Notes)
	}
}

func TestApplyUsesLegacyDropInWhenImuxsockIsModLoaded(t *testing.T) {
	f := newFixture(t)
	f.files[rsyslogConfPath] = legacyRsyslogConf
	f.files[stanzaPath] = renderStanza(distroStanza, "1G")
	f.files[timerDropInPath] = timerDropInContent()
	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	status, _ = h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("state %q notes %v", status.ExecutionState, status.Notes)
	}
	if f.files[rsyslogDropInPath] != rsyslogDropInContent(resolveSettings(nil)) {
		t.Fatalf("legacy drop-in not written")
	}
	if f.files[rsyslogConfPath] != legacyRsyslogConf {
		t.Fatalf("legacy rsyslog.conf must not be edited")
	}
	if !h.Inspect(linuxHost(), linuxReq()).Applied {
		t.Fatalf("legacy host still pending after apply")
	}
}

func TestApplyDryRunTouchesNothing(t *testing.T) {
	f := newFixture(t)
	f.files[stanzaPath] = distroStanza
	f.sizes["/var/log/syslog"] = 158_000_000_000
	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	status, _ = h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{DryRun: true})
	if status.ExecutionState != hostreqkit.ExecutionWouldApply || status.Applied || len(f.commands) != 0 {
		t.Fatalf("dry-run state %q, applied %v, commands %v", status.ExecutionState, status.Applied, f.commands)
	}
}
