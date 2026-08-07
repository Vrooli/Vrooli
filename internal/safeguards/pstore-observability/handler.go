// Package pstoreobservability installs a root-owned pstore collector that
// mirrors crash artifacts into a group-readable export directory.
//
// Scenarios must not run as root. This safeguard is the setup-time privilege
// bridge: `sudo vrooli setup` creates a narrow read-only channel for crash
// forensics, then autoheal can read exported files as the normal project user.
package pstoreobservability

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/daemonreload"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	observabilityGroup = "vrooli-observability"
	exportRoot         = "/var/lib/vrooli/host-observability"
	pstoreExportDir    = exportRoot + "/pstore"
	pstoreSourceDir    = "/sys/fs/pstore"
	manifestFilename   = "manifest.json"
	collectorPath      = "/usr/local/libexec/vrooli/pstore-collector"
	libexecDir         = "/usr/local/libexec/vrooli"
	servicePath        = "/etc/systemd/system/vrooli-pstore-collector.service"
	timerPath          = "/etc/systemd/system/vrooli-pstore-collector.timer"
	serviceName        = "vrooli-pstore-collector.service"
	timerName          = "vrooli-pstore-collector.timer"
)

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

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
		status.Notes = append(status.Notes, "pstore observability export is Linux-only")
		return status
	}
	if !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "pstore observability export requires systemd service/timer support")
		return status
	}

	user := targetUser()
	pending := pendingState(user)
	if len(pending) == 0 {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "pstore observability export is installed for "+user)
		if fresh := manifestFreshness(); fresh != "" {
			status.Notes = append(status.Notes, fresh)
		}
		return status
	}

	if hostreqkit.RunningAsRootFn() {
		status.Required = true
	}
	status.Notes = append(status.Notes, "pstore observability pending: "+strings.Join(pending, ", "))
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
		status.Notes = append(status.Notes, "cannot apply pstore observability without a target user")
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes,
			fmt.Sprintf("dry-run: would create group %s, add %s, install collector service/timer, and prepare %s",
				observabilityGroup, user, pstoreExportDir))
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
			for _, dir := range []string{libexecDir, exportRoot, pstoreExportDir} {
				if err := hostreqkit.EnsureManagedDir(dir, opts.SudoMode, opts); err != nil {
					return err
				}
			}
			return nil
		}},
		{"set export permissions", func() error {
			if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "chown", []string{"root:" + observabilityGroup, exportRoot, pstoreExportDir}, opts); err != nil {
				return err
			}
			return hostreqkit.RunPrivilegedCommand(opts.SudoMode, "chmod", []string{"0750", exportRoot, pstoreExportDir}, opts)
		}},
		{"install collector", func() error {
			return hostreqkit.InstallManagedExecutable(collectorPath, collectorContent(), opts.SudoMode, opts)
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
		"pstore observability export installed; log out/in or reboot if the current shell does not yet see group membership")
	return status, nil
}

func targetUser() string {
	return strings.TrimSpace(hostreqkit.InvokingUser())
}

func pendingState(user string) []string {
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
	if !dirModeGroupOK(pstoreExportDir) {
		pending = append(pending, pstoreExportDir+" permissions need update")
	}
	if !hostreqkit.FileContentMatches(collectorPath, collectorContent()) {
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

func dirModeGroupOK(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	if info.Mode().Perm() != 0o750 {
		return false
	}
	gid, ok := groupGID()
	return ok && fileInfoGroupMatches(info, gid)
}

func groupGID() (uint32, bool) {
	out, err := hostreqkit.CombinedOutputFn("getent", "group", observabilityGroup)
	if err != nil {
		return 0, false
	}
	fields := strings.Split(strings.TrimSpace(string(out)), ":")
	if len(fields) < 3 {
		return 0, false
	}
	var gid uint64
	if _, err := fmt.Sscanf(fields[2], "%d", &gid); err != nil {
		return 0, false
	}
	return uint32(gid), true
}

func timerActive() bool {
	out, err := hostreqkit.CombinedOutputFn("systemctl", "is-enabled", timerName)
	if err != nil || strings.TrimSpace(string(out)) != "enabled" {
		return false
	}
	out, err = hostreqkit.CombinedOutputFn("systemctl", "is-active", timerName)
	return err == nil && strings.TrimSpace(string(out)) == "active"
}

func manifestFreshness() string {
	info, err := os.Stat(pstoreExportDir + "/" + manifestFilename)
	if err != nil {
		return "collector manifest not present yet"
	}
	age := time.Since(info.ModTime()).Round(time.Second)
	return fmt.Sprintf("collector manifest age: %s", age)
}

func collectorContent() string {
	return `#!/bin/sh
set -eu

src="` + pstoreSourceDir + `"
dst="` + pstoreExportDir + `"
group="` + observabilityGroup + `"
manifest="$dst/` + manifestFilename + `"

install -d -o root -g "$group" -m 0750 "$dst"
count=0
files_json=""

if [ -d "$src" ]; then
  for file in "$src"/*; do
    [ -f "$file" ] || continue
    name=$(basename "$file")
    case "$name" in
      */*|.*) continue ;;
    esac
    tmp="$dst/.$name.tmp"
    if cp -- "$file" "$tmp"; then
      chown root:"$group" "$tmp"
      chmod 0640 "$tmp"
      mv -f -- "$tmp" "$dst/$name"
      size=$(wc -c < "$dst/$name" | tr -d ' ')
      if [ -n "$files_json" ]; then files_json="$files_json,"; fi
      files_json="$files_json{\"name\":\"$name\",\"size\":$size}"
      count=$((count + 1))
    else
      rm -f -- "$tmp"
    fi
  done
fi

now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
tmp_manifest="$dst/.manifest.tmp"
printf '{"collectedAt":"%s","sourcePath":"%s","artifactCount":%s,"files":[%s]}\n' "$now" "$src" "$count" "$files_json" > "$tmp_manifest"
chown root:"$group" "$tmp_manifest"
chmod 0640 "$tmp_manifest"
mv -f -- "$tmp_manifest" "$manifest"
`
}

func serviceContent() string {
	return `[Unit]
Description=Vrooli pstore crash artifact collector
Documentation=internal/safeguards/pstore-observability/handler.go
ConditionPathExists=` + pstoreSourceDir + `

[Service]
Type=oneshot
ExecStart=` + collectorPath + `
`
}

func timerContent() string {
	return `[Unit]
Description=Run Vrooli pstore crash artifact collector

[Timer]
OnBootSec=30s
OnUnitActiveSec=5min
AccuracySec=30s
Persistent=true
Unit=` + serviceName + `

[Install]
WantedBy=timers.target
`
}
