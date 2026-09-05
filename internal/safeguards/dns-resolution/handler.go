package dnsresolution

import (
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	resolvedConfigDir  = "/etc/systemd/resolved.conf.d"
	resolvedConfigPath = "/etc/systemd/resolved.conf.d/99-vrooli-dns.conf"
	resolvedContent    = "# Managed by Vrooli -- do not edit manually\n[Resolve]\nDNS=8.8.8.8 8.8.4.4 1.1.1.1\nFallbackDNS=9.9.9.9\n"
)

// ResolveFn is a test seam for DNS resolution checks.
var ResolveFn = func(host string) error {
	_, err := hostreqkit.CombinedOutputFn("getent", "hosts", host)
	return err
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

	if host.OS != string(hostreqspec.PlatformLinux) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "DNS resolution management is only supported on Linux")
		return status
	}

	if !host.SupportsSystemd {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "requires systemd-resolved for managed DNS configuration")
		return status
	}

	configOK := hostreqkit.FileContentMatches(resolvedConfigPath, resolvedContent)
	if configOK {
		status.Applied = true
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "reliable DNS servers configured via "+resolvedConfigPath)
		return status
	}

	status.Notes = append(status.Notes, "managed DNS configuration not applied")
	if ResolveFn("google.com") != nil {
		status.Notes = append(status.Notes, "DNS resolution is currently failing")
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
		status.Notes = append(status.Notes, "manual safeguard action required by manifest declaration")
		return status, nil
	}

	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would configure reliable DNS servers in "+resolvedConfigPath)
		return status, nil
	}

	if err := hostreqkit.EnsureManagedDir(resolvedConfigDir, opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	if err := hostreqkit.InstallManagedContent(resolvedConfigPath, resolvedContent, opts.SudoMode, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	// Restart systemd-resolved to pick up the new config.
	if _, err := hostreqkit.LookPathFn("systemctl"); err == nil {
		if restartErr := hostreqkit.RunPrivilegedCommand(opts.SudoMode, "systemctl", []string{"restart", "systemd-resolved"}, opts); restartErr != nil {
			status.Notes = append(status.Notes, "config installed but systemd-resolved restart failed: "+restartErr.Error())
		}
	}

	// Flush DNS cache if resolvectl is available.
	if _, err := hostreqkit.LookPathFn("resolvectl"); err == nil {
		_ = hostreqkit.RunPrivilegedCommand(opts.SudoMode, "resolvectl", []string{"flush-caches"}, opts)
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "reliable DNS servers configured in "+resolvedConfigPath)
	return status, nil
}
