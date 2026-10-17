// Package modules manages persistent kernel-module loading on Linux hosts.
// It owns two files per managed module:
//
//   - /etc/modules-load.d/99-vrooli-<name>.conf — boot-time autoload.
//   - /etc/modprobe.d/99-vrooli-<name>.conf      — module options (only when
//     options are non-empty; the file is omitted otherwise).
//
// Both filenames carry the 99- prefix so distribution defaults under the
// same directories take precedence and Vrooli only acts as a last-priority
// override. Content is rendered deterministically (sorted option keys) so
// FileContentMatches gives a real idempotency signal.
//
// EnsureLoadAtBoot guarantees the at-boot configuration; Modprobe activates
// the module live in the current kernel. Safeguards typically call both: the
// at-boot side survives reboots, the live side avoids needing a reboot to
// take effect.
package modules

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

const (
	// LoadConfDir is the canonical Linux path for boot-time autoload files.
	LoadConfDir = "/etc/modules-load.d"
	// ModprobeConfDir is the canonical Linux path for module-option files.
	ModprobeConfDir = "/etc/modprobe.d"
	// SysModulePrefix is the kernel-exposed view of currently-loaded modules.
	// /sys/module/<name> exists exactly when the module is live.
	SysModulePrefix = "/sys/module"
	// ManagedFilePrefix marks files this package writes so an operator can
	// grep for "Vrooli" and locate everything we own.
	ManagedFilePrefix = "99-vrooli-"
	// ManagedHeader is the first line of every managed file. The leading
	// comment marker stays stable across kernels so operators can audit.
	ManagedHeader = "# Managed by Vrooli — do not edit by hand."
)

// StatFn is the seam for /sys/module/<name> existence checks. Tests override
// it; production calls os.Stat.
var StatFn = os.Stat

// Outcome reports which of the two managed files were written. Both fields
// can be true (first run with options), one (subsequent run that only
// changes one side), or none (idempotent re-run).
type Outcome struct {
	LoadFileChanged    bool
	OptionsFileChanged bool
}

// IsLoaded reports whether the named module is currently live in the kernel
// by checking for /sys/module/<name>. False on any error (module absent, /sys
// unreadable on a non-Linux host, etc.) — callers that need to distinguish
// errors from absence should use StatFn directly.
func IsLoaded(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	_, err := StatFn(filepath.Join(SysModulePrefix, name))
	return err == nil
}

// EnsureLoadAtBoot ensures the at-boot autoload + options files exist with
// the desired content. Idempotent: when both files already match, no writes
// occur and Outcome is the zero value. On change, files are written via
// hostreqkit.InstallManagedContent (atomic temp + sudo install).
//
// options is the map of module-options (e.g. for netconsole, the kernel-format
// target string under key "netconsole"). When options is nil or empty, the
// modprobe.d file is not created and OptionsFileChanged stays false. We never
// delete an existing options file; a future "remove" helper can handle that
// when we need it.
func EnsureLoadAtBoot(name string, options map[string]string, sudoMode string, opts hostreqkit.EnsureOptions) (Outcome, error) {
	if strings.TrimSpace(name) == "" {
		return Outcome{}, fmt.Errorf("modules: empty module name")
	}

	loadPath := LoadFilePath(name)
	loadContent := renderLoadFile(name)

	var outcome Outcome
	if !hostreqkit.FileContentMatches(loadPath, loadContent) {
		outcome.LoadFileChanged = true
		if !opts.DryRun {
			if err := hostreqkit.EnsureManagedDir(LoadConfDir, sudoMode, opts); err != nil {
				return Outcome{}, fmt.Errorf("ensure %s: %w", LoadConfDir, err)
			}
			if err := hostreqkit.InstallManagedContent(loadPath, loadContent, sudoMode, opts); err != nil {
				return Outcome{}, fmt.Errorf("install %s: %w", loadPath, err)
			}
		}
	}

	if len(options) > 0 {
		optPath := OptionsFilePath(name)
		optContent := renderOptionsFile(name, options)
		if !hostreqkit.FileContentMatches(optPath, optContent) {
			outcome.OptionsFileChanged = true
			if !opts.DryRun {
				if err := hostreqkit.EnsureManagedDir(ModprobeConfDir, sudoMode, opts); err != nil {
					return Outcome{}, fmt.Errorf("ensure %s: %w", ModprobeConfDir, err)
				}
				if err := hostreqkit.InstallManagedContent(optPath, optContent, sudoMode, opts); err != nil {
					return Outcome{}, fmt.Errorf("install %s: %w", optPath, err)
				}
			}
		}
	}

	return outcome, nil
}

// Modprobe loads the named module with the given options into the live
// kernel. The options-as-args form (`modprobe <name> k=v k=v`) is required
// because some modules (notably netconsole) only honor options at load time.
// No-op under DryRun.
func Modprobe(name string, options map[string]string, sudoMode string, opts hostreqkit.EnsureOptions) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("modules: empty module name")
	}
	if opts.DryRun {
		return nil
	}
	args := []string{name}
	for _, k := range slices.Sorted(maps.Keys(options)) {
		args = append(args, fmt.Sprintf("%s=%s", k, options[k]))
	}
	return hostreqkit.RunPrivilegedCommand(sudoMode, "modprobe", args, opts)
}

// LoadFilePath returns the canonical path of the at-boot autoload file for
// the named module. Exposed so safeguards can quote the path in operator-
// facing notes without re-deriving the format.
func LoadFilePath(name string) string {
	return filepath.Join(LoadConfDir, ManagedFilePrefix+name+".conf")
}

// OptionsFilePath returns the canonical path of the module-options file.
func OptionsFilePath(name string) string {
	return filepath.Join(ModprobeConfDir, ManagedFilePrefix+name+".conf")
}

func renderLoadFile(name string) string {
	return ManagedHeader + "\n" + name + "\n"
}

func renderOptionsFile(name string, options map[string]string) string {
	var b strings.Builder
	b.WriteString(ManagedHeader)
	b.WriteString("\noptions ")
	b.WriteString(name)
	for _, k := range slices.Sorted(maps.Keys(options)) {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(options[k])
	}
	b.WriteByte('\n')
	return b.String()
}
