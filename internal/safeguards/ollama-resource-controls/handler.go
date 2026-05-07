// Package ollamaresourcecontrols installs a systemd drop-in that caps
// ollama.service memory usage and biases the system-wide OOM killer toward
// it. Motivation: on 2026-05-07 the dev workstation hard-reset four times
// in 24 h with no kernel-side panic logs. The active ollama.service was the
// stock installer's unit with no cgroup directives; agents that bypass the
// resource-ollama wrapper hit localhost:11434 directly and could drive an
// embeddings burst large enough to be a candidate cause. Whatever the root
// cause turns out to be, an unbounded ollama is a known host-stability risk
// and the cheapest mitigation is enforced cgroup limits.
//
// We use a high-priority drop-in (99-vrooli-...) rather than rewriting
// ollama.service so we layer cleanly on top of whatever Ollama's upstream
// installer or any pre-existing wrapper has already written.
package ollamaresourcecontrols

import (
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	dropinDir  = "/etc/systemd/system/ollama.service.d"
	dropinPath = dropinDir + "/99-vrooli-resource-controls.conf"
)

// Resource control values. Memory caps use systemd percentage syntax
// (% of physical RAM) so they scale across hosts. CPUQuota is intentionally
// omitted — memory exhaustion hangs kernels; CPU saturation is recoverable.
var managedDirectives = []struct {
	Key   string
	Value string
}{
	{"MemoryHigh", "60%"},
	{"MemoryMax", "70%"},
	{"TasksMax", "4096"},
	{"OOMScoreAdjust", "500"},
}

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
		status.Notes = append(status.Notes, "ollama resource controls require systemd (Linux only)")
		return status
	}

	if !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "host does not support systemd")
		return status
	}

	// Only applicable when ollama.service is actually present. Hosts without
	// ollama (CI runners, fresh installs) shouldn't see this as a missing
	// safeguard.
	if !ollamaUnitPresent() {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "ollama.service not present on this host")
		return status
	}

	if hostreqkit.FileContentMatches(dropinPath, buildDropinContent()) {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "ollama resource-control drop-in already in place")
		return status
	}

	status.Notes = append(status.Notes, "ollama resource-control drop-in missing or stale at "+dropinPath)
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

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would install "+dropinPath)
		return status, nil
	}

	if err := hostreqkit.EnsureManagedDir(dropinDir, opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	if err := hostreqkit.InstallManagedContent(dropinPath, buildDropinContent(), opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	if err := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"daemon-reload"}, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "drop-in written but systemctl daemon-reload failed: "+err.Error())
		return status, nil
	}

	// Don't auto-restart ollama. The drop-in is read on next start; a forced
	// restart here would interrupt running embeddings/inference for any
	// caller currently using ollama. Surface the note instead so the operator
	// chooses when to apply.
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes,
		"drop-in installed; run `systemctl restart ollama` to apply cgroup limits to the running process")
	return status, nil
}

// ollamaUnitPresent returns true when systemd knows about ollama.service.
// Uses `systemctl list-unit-files` so we catch both /etc/systemd/system and
// /lib/systemd/system without hardcoding paths.
func ollamaUnitPresent() bool {
	out, err := hostreqkit.CombinedOutputFn("systemctl", "list-unit-files", "--no-pager", "--no-legend", "ollama.service")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "ollama.service")
}

func buildDropinContent() string {
	var b strings.Builder
	b.WriteString("# Managed by Vrooli -- do not edit manually\n")
	b.WriteString("# Caps ollama.service so an embeddings burst cannot exhaust host RAM.\n")
	b.WriteString("# See internal/safeguards/ollama-resource-controls/handler.go for rationale.\n")
	b.WriteString("[Service]\n")
	for _, d := range managedDirectives {
		b.WriteString(d.Key)
		b.WriteString("=")
		b.WriteString(d.Value)
		b.WriteString("\n")
	}
	return b.String()
}
