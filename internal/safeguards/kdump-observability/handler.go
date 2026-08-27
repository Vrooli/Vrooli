// Package kdumpobservability installs a root-owned collector that turns kdump
// crash dumps into small, group-readable panic summaries.
//
// kdump already writes everything needed to explain a panic, but writes it
// where no scenario can read it: /var/crash/<stamp>/dmesg.<stamp> is mode 600
// root, and the vmcore beside it is roughly the size of system RAM. On
// 2026-08-19 this host panicked in the ntfs3 write path, kdump captured a
// complete 5.8 GB dump, and nothing in Vrooli noticed — the evidence had to be
// recovered by hand with makedumpfile.
//
// This safeguard is the same setup-time privilege bridge pstore-observability
// provides for /sys/fs/pstore: `sudo vrooli setup` installs a collector that
// exports a bounded summary into the shared host-observability directory, and
// unprivileged autoheal reads it from there.
//
// Two properties are deliberate:
//
//   - The vmcore is never copied. Only the panic-relevant head of the dmesg and
//     an extracted summary cross the boundary, so the export directory stays
//     small enough to keep indefinitely.
//   - Summaries outlive the dumps they describe. Pruning removes whole vmcore
//     directories once their summary exists, so crash history survives even
//     after the raw dumps are reclaimed.
package kdumpobservability

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/daemonreload"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	observabilityGroup = "vrooli-observability"
	exportRoot         = "/var/lib/vrooli/host-observability"
	crashExportDir     = exportRoot + "/crashdumps"
	crashSourceDir     = "/var/crash"
	collectorPath      = "/usr/local/libexec/vrooli/kdump-collector"
	libexecDir         = "/usr/local/libexec/vrooli"
	servicePath        = "/etc/systemd/system/vrooli-kdump-collector.service"
	timerPath          = "/etc/systemd/system/vrooli-kdump-collector.timer"
	serviceName        = "vrooli-kdump-collector.service"
	timerName          = "vrooli-kdump-collector.timer"

	// defaultRetainVmcores matches the manifest default. It is duplicated here
	// so the handler behaves correctly with no resolved config; the invariant
	// is covered by TestDefaultsMatchManifest.
	defaultRetainVmcores = 2
)

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

// retainVmcores resolves how many vmcore directories to keep. A declared value
// below one is treated as the default rather than obeyed: retaining zero dumps
// would delete the evidence for a crash that just happened.
func retainVmcores(config map[string]any) int {
	if config == nil {
		return defaultRetainVmcores
	}
	var value int
	switch v := config["retain_vmcores"].(type) {
	case float64:
		value = int(v)
	case int:
		value = v
	default:
		return defaultRetainVmcores
	}
	if value < 1 {
		return defaultRetainVmcores
	}
	return value
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "kdump summary export is Linux-only")
		return status
	}
	if !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "kdump summary export requires systemd service/timer support")
		return status
	}

	user := targetUser()
	pending := pendingState(user, retainVmcores(requirement.Config))
	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "kdump summary export is installed for "+user)
		return status
	}

	if hostreqkit.RunningAsRootFn() {
		status.Required = true
	}
	status.Notes = append(status.Notes, "kdump summary export pending: "+strings.Join(pending, ", "))
	if user == "" {
		status.Notes = append(status.Notes, "could not determine invoking user; run setup from the intended project user")
	}
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
		return status, nil
	}
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	user := targetUser()
	if user == "" {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "cannot apply kdump summary export without a target user")
		return status, nil
	}
	retain := retainVmcores(status.Config)
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes,
			fmt.Sprintf("dry-run: would create group %s, add %s, install the collector service/timer, prepare %s, and retain the newest %d vmcore directories",
				observabilityGroup, user, crashExportDir, retain))
		return status, nil
	}

	steps := []struct {
		name string
		fn   func() error
	}{
		{"create observability group", func() error {
			if groupExists() {
				return nil
			}
			return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "groupadd", []string{"--system", observabilityGroup}, opts)
		}},
		{"add user to observability group", func() error {
			if userInGroup(user) {
				return nil
			}
			return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "usermod", []string{"-aG", observabilityGroup, user}, opts)
		}},
		{"create managed directories", func() error {
			for _, dir := range []string{libexecDir, exportRoot, crashExportDir} {
				if err := hostreqkit.EnsureManagedDir(dir, opts.SudoMode, opts); err != nil {
					return err
				}
			}
			return nil
		}},
		{"set export permissions", func() error {
			if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "chown", []string{"root:" + observabilityGroup, exportRoot, crashExportDir}, opts); err != nil {
				return err
			}
			return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "chmod", []string{"750", exportRoot, crashExportDir}, opts)
		}},
		{"install collector", func() error {
			return hostreqkit.InstallManagedExecutable(collectorPath, collectorContent(retain), opts.SudoMode, opts)
		}},
		{"install systemd units", func() error {
			if err := hostreqkit.InstallManagedContent(servicePath, serviceContent(), opts.SudoMode, opts); err != nil {
				return err
			}
			return hostreqkit.InstallManagedContent(timerPath, timerContent(), opts.SudoMode, opts)
		}},
		{"enable collector timer", func() error {
			if _, err := daemonreload.Reload(context.Background(), daemonreload.CurrentRoot(), opts); err != nil {
				return err
			}
			if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"enable", "--now", timerName}, opts); err != nil {
				return err
			}
			return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"start", serviceName}, opts)
		}},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, step.name+" failed: "+err.Error())
			return status, nil
		}
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes,
		"kdump summary export installed; log out/in or reboot if the current shell does not yet see group membership")
	return status, nil
}

func targetUser() string {
	return strings.TrimSpace(hostreqkit.InvokingUser())
}

func pendingState(user string, retain int) []string {
	var pending []string
	if !groupExists() {
		pending = append(pending, "group "+observabilityGroup+" missing")
	}
	if user == "" {
		pending = append(pending, "target user unknown")
	} else if !userInGroup(user) {
		pending = append(pending, user+" not in "+observabilityGroup)
	}
	if !dirModeGroupOK(exportRoot) {
		pending = append(pending, exportRoot+" permissions need update")
	}
	if !dirModeGroupOK(crashExportDir) {
		pending = append(pending, crashExportDir+" permissions need update")
	}
	if !hostreqkit.FileContentMatches(collectorPath, collectorContent(retain)) {
		pending = append(pending, collectorPath+" missing or stale")
	}
	if !hostreqkit.FileContentMatches(servicePath, serviceContent()) {
		pending = append(pending, servicePath+" missing or stale")
	}
	if !hostreqkit.FileContentMatches(timerPath, timerContent()) {
		pending = append(pending, timerPath+" missing or stale")
	}
	if !timerActive() {
		pending = append(pending, timerName+" not active")
	}
	return pending
}

func groupExists() bool {
	out, err := hostreqkit.CombinedOutputFn("getent", "group", observabilityGroup)
	return err == nil && strings.TrimSpace(string(out)) != ""
}

func userInGroup(user string) bool {
	if user == "" {
		return false
	}
	out, err := hostreqkit.CombinedOutputFn("id", "-nG", user)
	if err != nil {
		return false
	}
	for _, group := range strings.Fields(string(out)) {
		if group == observabilityGroup {
			return true
		}
	}
	return false
}

func dirModeGroupOK(dir string) bool {
	out, err := hostreqkit.CombinedOutputFn("stat", "-c", "%a %G", dir)
	if err != nil {
		return false
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	return len(fields) == 2 && fields[0] == "750" && fields[1] == observabilityGroup
}

func timerActive() bool {
	out, err := hostreqkit.CombinedOutputFn("systemctl", "is-enabled", timerName)
	return err == nil && strings.TrimSpace(string(out)) == "enabled"
}

// collectorContent renders the export script.
//
// The panic-relevant part of a kdump dmesg is its tail — the oops banner, the
// register dump and the call trace are the last thing the dying kernel wrote —
// so the export keeps a bounded tail rather than the whole file.
//
// Pruning is deliberately narrow: it only ever removes a direct child of
// /var/crash whose name matches kdump's timestamp form, only beyond the newest
// N, and only once that directory's summary has been exported. Anything else in
// /var/crash — including the .crash reports apport leaves there — is untouched.
func collectorContent(retain int) string {
	return `#!/bin/sh
set -eu

src="` + crashSourceDir + `"
dst="` + crashExportDir + `"
group="` + observabilityGroup + `"
retain=` + strconv.Itoa(retain) + `
manifest="$dst/manifest.json"
tail_lines=2000

install -d -o root -g "$group" -m 750 "$dst"

publish() {
  # publish <source-file> <destination-name>
  tmp="$dst/.$2.tmp"
  if cp -- "$1" "$tmp" 2>/dev/null; then
    chown root:"$group" "$tmp"
    chmod 640 "$tmp"
    mv -f -- "$tmp" "$dst/$2"
    return 0
  fi
  rm -f -- "$tmp"
  return 1
}

count=0
dumps_json=""

for dir in "$src"/*; do
  [ -d "$dir" ] || continue
  stamp=$(basename "$dir")
  # kdump names its directories with a YYYYMMDDHHMM timestamp. Anything else in
  # /var/crash belongs to another tool and is not ours to read or remove.
  case "$stamp" in
    ''|*[!0-9]*) continue ;;
  esac

  dmesg_file=""
  for candidate in "$dir"/dmesg.*; do
    [ -f "$candidate" ] || continue
    dmesg_file="$candidate"
    break
  done
  [ -n "$dmesg_file" ] || continue

  summary="$dst/$stamp.dmesg"
  if [ ! -f "$summary" ]; then
    tmp="$dst/.$stamp.dmesg.tmp"
    if tail -n "$tail_lines" -- "$dmesg_file" > "$tmp" 2>/dev/null; then
      chown root:"$group" "$tmp"
    chmod 640 "$tmp"
      mv -f -- "$tmp" "$summary"
    else
      rm -f -- "$tmp"
      continue
    fi
  fi

  # Extract the one line an incident report leads with, plus the faulting
  # command, so a consumer does not have to parse the whole tail.
  reason=$(grep -m1 -E 'kernel BUG at|Oops:|general protection fault|Unable to handle kernel|Kernel panic' "$summary" 2>/dev/null | sed 's/"/\\"/g' | tr -d '\n' || true)
  comm=$(grep -m1 -oE 'Comm: [^ ]+' "$summary" 2>/dev/null | cut -d' ' -f2 | tr -d '\n' || true)
  vmcore_bytes=$(du -sb -- "$dir" 2>/dev/null | cut -f1 || echo 0)

  if [ -n "$dumps_json" ]; then dumps_json="$dumps_json,"; fi
  dumps_json="$dumps_json{\"stamp\":\"$stamp\",\"summary\":\"$stamp.dmesg\",\"reason\":\"$reason\",\"comm\":\"$comm\",\"bytes\":${vmcore_bytes:-0}}"
  count=$((count + 1))
done

now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
tmp_manifest="$dst/.manifest.tmp"
printf '{"collectedAt":"%s","sourcePath":"%s","retainVmcores":%s,"dumpCount":%s,"dumps":[%s]}\n' \
  "$now" "$src" "$retain" "$count" "$dumps_json" > "$tmp_manifest"
chown root:"$group" "$tmp_manifest"
chmod 640 "$tmp_manifest"
mv -f -- "$tmp_manifest" "$manifest"

# Prune oldest vmcore directories beyond the retention count. Summaries stay.
if [ "$count" -gt "$retain" ]; then
  ls -1 "$src" 2>/dev/null | grep -E '^[0-9]+$' | sort -r | tail -n +$((retain + 1)) | while read -r old; do
    [ -n "$old" ] || continue
    [ -d "$src/$old" ] || continue
    # Never remove a dump whose summary was not exported: that would discard
    # the only remaining record of the crash.
    [ -f "$dst/$old.dmesg" ] || continue
    rm -rf -- "$src/$old"
  done
fi
`
}

func serviceContent() string {
	return `[Unit]
Description=Vrooli kdump panic summary collector
Documentation=internal/safeguards/kdump-observability/handler.go
ConditionPathExists=` + crashSourceDir + `

[Service]
Type=oneshot
ExecStart=` + collectorPath + `
`
}

func timerContent() string {
	return `[Unit]
Description=Run Vrooli kdump panic summary collector

[Timer]
OnBootSec=2min
OnUnitActiveSec=1h
Persistent=true

[Install]
WantedBy=timers.target
`
}
