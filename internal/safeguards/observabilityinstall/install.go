// Package observabilityinstall owns fixed destinations and the shared install
// sequence for pstore and kdump. Callers provide the invoking user, rendered
// files, and setup options; target paths and unit names stay local here.
package observabilityinstall

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/daemonreload"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

type Target uint8

const (
	Pstore Target = iota + 1
	Kdump
)

var reloadFn = daemonreload.Reload

const (
	Group           = "vrooli-observability"
	ExportRoot      = "/var/lib/vrooli/host-observability"
	LibexecDir      = "/usr/local/libexec/vrooli"
	PstoreExportDir = ExportRoot + "/pstore"
	KdumpExportDir  = ExportRoot + "/crashdumps"

	PstoreCollectorPath = LibexecDir + "/pstore-collector"
	PstoreServicePath   = "/etc/systemd/system/vrooli-pstore-collector.service"
	PstoreTimerPath     = "/etc/systemd/system/vrooli-pstore-collector.timer"
	PstoreServiceName   = "vrooli-pstore-collector.service"
	PstoreTimerName     = "vrooli-pstore-collector.timer"

	KdumpCollectorPath = LibexecDir + "/kdump-collector"
	KdumpServicePath   = "/etc/systemd/system/vrooli-kdump-collector.service"
	KdumpTimerPath     = "/etc/systemd/system/vrooli-kdump-collector.timer"
	KdumpServiceName   = "vrooli-kdump-collector.service"
	KdumpTimerName     = "vrooli-kdump-collector.timer"
)

type Content struct {
	Collector string
	Service   string
	Timer     string
}

type setupPrivilegeSession interface {
	InstallObservability(Target, int) error
}

type targetFiles struct {
	collector, export string
	service, timer    string
	serviceName       string
	timerName         string
}

func files(target Target) (targetFiles, error) {
	switch target {
	case Pstore:
		return targetFiles{
			collector: PstoreCollectorPath, export: PstoreExportDir,
			service: PstoreServicePath, timer: PstoreTimerPath,
			serviceName: PstoreServiceName, timerName: PstoreTimerName,
		}, nil
	case Kdump:
		return targetFiles{
			collector: KdumpCollectorPath, export: KdumpExportDir,
			service: KdumpServicePath, timer: KdumpTimerPath,
			serviceName: KdumpServiceName, timerName: KdumpTimerName,
		}, nil
	default:
		return targetFiles{}, fmt.Errorf("unknown observability installer target %d", target)
	}
}

// ApplyForSetup uses the setup-owned typed worker when available, otherwise it
// preserves the standalone per-operation installer path.
func ApplyForSetup(target Target, user string, content Content, retention int, opts hostreqkit.EnsureOptions) error {
	if session, ok := opts.SetupPrivilegeSession.(setupPrivilegeSession); ok {
		return session.InstallObservability(target, retention)
	}
	return Apply(target, user, content, opts)
}

// ValidateSetupRequest keeps the privileged protocol closed to registered
// targets and target-specific parameters.
func ValidateSetupRequest(target Target, retention int) error {
	switch target {
	case Pstore:
		if retention != 0 {
			return fmt.Errorf("pstore observability does not accept vmcore retention")
		}
	case Kdump:
		if err := ValidateRetention(retention); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown observability installer target %d", target)
	}
	return nil
}

// ValidateRetention rejects values that could overflow the collector's
// retention arithmetic.
func ValidateRetention(retention int) error {
	if retention < 1 || retention >= int(^uint(0)>>1) {
		return fmt.Errorf("kdump vmcore retention is outside the supported range")
	}
	return nil
}

func Apply(target Target, user string, content Content, opts hostreqkit.EnsureOptions) error {
	f, err := files(target)
	if err != nil {
		return err
	}
	steps := []struct {
		name string
		run  func() error
	}{
		{"create observability group", func() error {
			if GroupExists() {
				return nil
			}
			return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "groupadd", []string{"--system", Group}, opts)
		}},
		{"add user to observability group", func() error {
			if UserInGroup(user) {
				return nil
			}
			return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "usermod", []string{"-aG", Group, user}, opts)
		}},
		{"create managed directories", func() error {
			for _, dir := range []string{LibexecDir, ExportRoot, f.export} {
				if err := hostreqkit.EnsureManagedDir(dir, opts); err != nil {
					return err
				}
			}
			return nil
		}},
		{"set export permissions", func() error {
			if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "chown", []string{"root:" + Group, ExportRoot, f.export}, opts); err != nil {
				return err
			}
			return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "chmod", []string{"750", ExportRoot, f.export}, opts)
		}},
		{"install collector", func() error {
			return hostreqkit.InstallManagedExecutable(f.collector, content.Collector, opts)
		}},
		{"install systemd units", func() error {
			if err := hostreqkit.InstallManagedContent(f.service, content.Service, opts); err != nil {
				return err
			}
			return hostreqkit.InstallManagedContent(f.timer, content.Timer, opts)
		}},
		{"enable collector timer", func() error {
			ctx := opts.Context
			if ctx == nil {
				ctx = context.Background()
			}
			if _, err := reloadFn(ctx, daemonreload.CurrentRoot(), opts); err != nil {
				return err
			}
			if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"enable", "--now", f.timerName}, opts); err != nil {
				return err
			}
			return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"start", f.serviceName}, opts)
		}},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			return fmt.Errorf("%s failed: %w", step.name, err)
		}
	}
	return nil
}

func GroupExists() bool {
	out, err := hostreqkit.CombinedOutputFn("getent", "group", Group)
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func UserInGroup(user string) bool {
	if user == "" {
		return false
	}
	out, err := hostreqkit.CombinedOutputFn("id", "-nG", user)
	if err != nil {
		return false
	}
	for _, name := range strings.Fields(string(out)) {
		if name == Group {
			return true
		}
	}
	return false
}
