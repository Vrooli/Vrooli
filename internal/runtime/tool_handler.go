package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// runtimeArch reports the host architecture (Go GOARCH) used to select a
// source target. A var so tests can simulate other arches.
var runtimeArch = func() string { return runtime.GOARCH }

// fetchBinaryFn is the seam for downloading single-file url/release tools,
// defaulting to the shared binaryfetch package. Tests stub it to avoid real
// network I/O.
var fetchBinaryFn = binaryfetch.Fetch

// fetchDirFn is the seam for downloading dir-layout tools (whole archive tree
// into ~/.vrooli/opt/<tool>). Tests stub it to avoid real network I/O.
var fetchDirFn = binaryfetch.FetchDir

// userLocalBinDir returns the no-sudo install target ~/.vrooli/bin for the
// invoking user (correct even under `sudo vrooli setup`).
var userLocalBinDir = func() (string, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolve user home for ~/.vrooli/bin: %w", err)
	}
	return filepath.Join(home, ".vrooli", "bin"), nil
}

// userLocalOptDir returns the no-sudo install target ~/.vrooli/opt/<tool> for a
// dir-layout tool's extracted archive tree.
var userLocalOptDir = func(tool string) (string, error) {
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolve user home for ~/.vrooli/opt: %w", err)
	}
	return filepath.Join(home, ".vrooli", "opt", tool), nil
}

type toolHandler struct {
	manifest hostreqkit.ToolManifest
}

type toolInstallStrategy uint8

const (
	toolInstallPackage toolInstallStrategy = iota
	toolInstallFetch
)

func newGenericToolHandler(m hostreqkit.ToolManifest) hostreqkit.Handler {
	return toolHandler{manifest: m}
}

func (h toolHandler) Name() string           { return h.manifest.Name }
func (h toolHandler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }

func (h toolHandler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)

	// Hardware-capability gate runs first: a tool whose `requires` is unmet on
	// this host is cleanly not-applicable regardless of source type, so a CPU
	// fallback can still be offered and setup never fails on it.
	if gated, isGated := h.capabilityGate(requirement, status); isGated {
		return gated
	}

	// A tool can be declared manual either by the scenario (requirement.Manual)
	// or by the platform manifest (manifest.Manual, e.g. a pip CLI / from-source
	// build). Fold them so `vrooli host install <tool>` reports manual tools as
	// manual-action-required with their installHint, not a bare "unsupported".
	requirement.Manual = requirement.Manual || h.manifest.Manual

	if h.installStrategy(host) == toolInstallFetch {
		return h.inspectFetch(host, requirement, status)
	}
	return h.inspectPackage(host, requirement, status)
}

// installStrategy resolves a mixed source/packages manifest for this host.
// A matching verified source target wins. When the source has no target for
// this OS/architecture, an explicitly declared host package remains a valid
// fallback. A source-only manifest stays on the fetch path so inspectFetch can
// report the precise missing-target reason.
func (h toolHandler) installStrategy(host hostreqkit.Host) toolInstallStrategy {
	if h.manifest.SourceType() == "package" {
		return toolInstallPackage
	}
	if _, ok := h.manifest.Source.TargetFor(host.OS, runtimeArch()); ok {
		return toolInstallFetch
	}
	if strings.TrimSpace(h.manifest.PackageNameForHost(host)) != "" {
		return toolInstallPackage
	}
	return toolInstallFetch
}

// capabilityGate evaluates the effective `requires` predicate (manifest first,
// then declaration) against the host facts. It returns a not-applicable status
// and true when the gate is unmet; otherwise the second return is false and the
// caller proceeds.
func (h toolHandler) capabilityGate(requirement hostreqspec.ResolvedRequirement, status hostreqkit.ItemStatus) (hostreqkit.ItemStatus, bool) {
	gate := effectiveCapability(h.manifest.Requires, requirement.Requires)
	if gate == nil {
		return status, false
	}
	if ok, reason := gate.Evaluate(capabilityFactsFn()); !ok {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.InstallSupported = false
		status.Command, status.Installed = resolveFetchCommand(h.manifest.Commands)
		if reason != "" {
			status.Notes = append(status.Notes, reason)
		}
		return status, true
	}
	return status, false
}

func (h toolHandler) inspectPackage(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement, status hostreqkit.ItemStatus) hostreqkit.ItemStatus {
	status.PackageName = h.manifest.PackageNameForHost(host)
	command, installed := hostreqkit.ResolveCommand(h.manifest.Commands)
	status.Command = command
	status.Installed = installed
	status.SupportClass = hostreqkit.SupportSupported
	status.InstallSupported = strings.TrimSpace(status.PackageName) != "" && !requirement.Manual
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.InstallSupported = false
	}
	if installed {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	}
	if !installed {
		if status.SupportClass == hostreqkit.SupportSupported && strings.TrimSpace(status.PackageName) == "" {
			status.SupportClass = hostreqkit.SupportUnsupported
			status.ExecutionState = hostreqkit.ExecutionUnsupported
		}
		if h.manifest.InstallHint != "" {
			status.Notes = append(status.Notes, h.manifest.InstallHint)
		}
		return status
	}

	version := hostreqkit.ReadVersion(command, h.manifest.VersionArgs)
	if version != "" {
		status.Version = version
	}
	if passed, detail := hostreqkit.RunVerificationCheck(h.manifest.VerificationCheck); !passed {
		status.Notes = append(status.Notes, detail)
	}
	return status
}

// inspectFetch reports the readiness of a url/release tool. "Installed" means the
// command is on PATH or already present in ~/.vrooli/bin.
func (h toolHandler) inspectFetch(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement, status hostreqkit.ItemStatus) hostreqkit.ItemStatus {
	command, installed := resolveFetchCommand(h.manifest.Commands)
	status.Command = command
	status.Installed = installed
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.InstallSupported = false
		if installed {
			status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		} else {
			status.ExecutionState = hostreqkit.ExecutionManualActionRequired
			if h.manifest.InstallHint != "" {
				status.Notes = append(status.Notes, h.manifest.InstallHint)
			}
		}
		return status
	}

	if installed {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.InstallSupported = false
		if version := h.readFetchVersion(command); version != "" {
			status.Version = version
		}
		return status
	}

	target, ok := h.manifest.Source.TargetFor(host.OS, runtimeArch())
	if !ok {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.InstallSupported = false
		status.Notes = append(status.Notes, fmt.Sprintf("no %s/%s release target declared for %q", host.OS, runtimeArch(), h.manifest.Name))
		return status
	}
	status.InstallSupported = true
	if target.IsDir() {
		status.Notes = append(status.Notes, fmt.Sprintf("install fetches %s and installs %s into ~/.vrooli/opt/%s (with a launcher in ~/.vrooli/bin)", target.URL, h.manifest.Name, h.manifest.Name))
	} else {
		status.Notes = append(status.Notes, fmt.Sprintf("install fetches %s into ~/.vrooli/bin", target.URL))
	}
	if h.manifest.InstallHint != "" {
		status.Notes = append(status.Notes, h.manifest.InstallHint)
	}
	return status
}

func (h toolHandler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.Installed {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	switch status.SupportClass {
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		status.Notes = append(status.Notes, "manual install required by manifest declaration")
		return status, nil
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "requirement is not applicable on this host")
		return status, nil
	}

	if h.installStrategy(host) == toolInstallFetch {
		return h.applyFetch(host, status, opts)
	}
	return h.applyPackage(host, status, opts)
}

func (h toolHandler) applyPackage(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if !status.InstallSupported || strings.TrimSpace(status.PackageName) == "" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return status, nil
	}
	command, args, err := hostreqkit.InstallCommand(host, status.PackageName, opts.SudoMode)
	if err != nil {
		status.Notes = append(status.Notes, err.Error())
		if hostreqkit.IsSudoSkipped(err) {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.BlockingReason = hostreqkit.BlockingNeedsSudo
			return status, nil
		}
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldInstall
		status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would run %s %s", command, strings.Join(args, " ")))
		return status, nil
	}
	if err := hostreqkit.RunInstallCommand(command, args, opts); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	commandName, installed := hostreqkit.ResolveCommand(h.manifest.Commands)
	status.Command = commandName
	status.Installed = installed
	if installed {
		status.ExecutionState = hostreqkit.ExecutionInstalled
		status.Version = hostreqkit.ReadVersion(commandName, h.manifest.VersionArgs)
		if passed, detail := hostreqkit.RunVerificationCheck(h.manifest.VerificationCheck); !passed {
			status.Notes = append(status.Notes, detail)
		}
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, "install command completed but the tool is still not available on PATH")
	return status, nil
}

// applyFetch installs a url/release tool by fetching+verifying its binary into
// ~/.vrooli/bin. No sudo: the install location is user-local.
func (h toolHandler) applyFetch(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	target, ok := h.manifest.Source.TargetFor(host.OS, runtimeArch())
	if !ok {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, fmt.Sprintf("no %s/%s release target declared for %q", host.OS, runtimeArch(), h.manifest.Name))
		return status, nil
	}
	binDir, err := userLocalBinDir()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	binName := firstNonEmpty(h.manifest.Commands)
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldInstall
		if target.IsDir() {
			status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would fetch %s and install %s into ~/.vrooli/opt/%s (launcher in %s)", target.URL, h.manifest.Name, h.manifest.Name, binDir))
		} else {
			status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would fetch %s into %s", target.URL, binDir))
		}
		return status, nil
	}

	spec := binaryfetch.Target{
		Name:    binName,
		URL:     target.URL,
		SHA256:  target.SHA256,
		Archive: target.Archive,
		Layout:  target.Layout,
		BinPath: target.BinPath,
		Mode:    target.Mode,
	}

	if target.IsDir() {
		if err := h.installDir(spec, binDir, binName, opts); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("install %s: %v", binName, err))
			return status, nil
		}
	} else if _, err := fetchBinaryFn(context.Background(), spec, binDir, fetchProgress(opts.Stdout, binName)); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, fmt.Sprintf("fetch %s: %v", binName, err))
		return status, nil
	}

	command, installed := resolveFetchCommand(h.manifest.Commands)
	status.Command = command
	status.Installed = installed
	if installed {
		status.ExecutionState = hostreqkit.ExecutionInstalled
		if version := h.readFetchVersion(command); version != "" {
			status.Version = version
		}
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionFailed
	status.Notes = append(status.Notes, fmt.Sprintf("fetch completed but %q is not present in ~/.vrooli/bin", binName))
	return status, nil
}

// installDir fetches a dir-layout tool's whole archive tree into
// ~/.vrooli/opt/<tool> and writes a launcher at ~/.vrooli/bin/<command> that
// runs the entry binary with the opt dir on LD_LIBRARY_PATH.
func (h toolHandler) installDir(spec binaryfetch.Target, binDir, command string, opts hostreqkit.EnsureOptions) error {
	optDir, err := userLocalOptDir(h.manifest.Name)
	if err != nil {
		return err
	}
	entry, err := fetchDirFn(context.Background(), spec, optDir, fetchProgress(opts.Stdout, command))
	if err != nil {
		return err
	}
	return writeLauncher(binDir, command, entry)
}

// writeLauncher writes an executable script at <binDir>/<command> that prepends
// the entry binary's directory to LD_LIBRARY_PATH (so sibling shared libraries
// resolve from any cwd) and execs the real binary, forwarding all args. On
// Windows it writes a <command>.bat that prepends the dir to PATH instead.
func writeLauncher(binDir, command, optBinPath string) error {
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create bin dir: %w", err)
	}
	dir := filepath.Dir(optBinPath)
	if runtime.GOOS == "windows" {
		path := filepath.Join(binDir, command+".bat")
		script := fmt.Sprintf("@echo off\r\nset \"PATH=%s;%%PATH%%\"\r\n\"%s\" %%*\r\n", dir, optBinPath)
		return os.WriteFile(path, []byte(script), 0o755) //nolint:gosec // launcher must be executable
	}
	path := filepath.Join(binDir, command)
	script := fmt.Sprintf("#!/bin/sh\nDIR=%s\nexport LD_LIBRARY_PATH=\"$DIR:$LD_LIBRARY_PATH\"\nexec %s \"$@\"\n",
		shellSingleQuote(dir), shellSingleQuote(optBinPath))
	return os.WriteFile(path, []byte(script), 0o755) //nolint:gosec // launcher must be executable
}

// shellSingleQuote wraps s in single quotes, safely escaping embedded quotes, so
// the launcher script handles paths with spaces or shell metacharacters.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// readFetchVersion reads a fetched tool's version, trying the bare command (when
// ~/.vrooli/bin is on PATH) then the explicit ~/.vrooli/bin path.
func (h toolHandler) readFetchVersion(command string) string {
	if version := hostreqkit.ReadVersion(command, h.manifest.VersionArgs); version != "" {
		return version
	}
	binDir, err := userLocalBinDir()
	if err != nil {
		return ""
	}
	return hostreqkit.ReadVersion(filepath.Join(binDir, command), h.manifest.VersionArgs)
}

// resolveFetchCommand reports whether any candidate command is available, on
// PATH or in ~/.vrooli/bin. When none is found it returns the canonical command
// name (the first candidate) so the installer knows the target filename.
func resolveFetchCommand(candidates []string) (string, bool) {
	if cmd, ok := hostreqkit.ResolveCommand(candidates); ok {
		return cmd, true
	}
	if binDir, err := userLocalBinDir(); err == nil {
		for _, candidate := range candidates {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}
			if info, statErr := os.Stat(filepath.Join(binDir, candidate)); statErr == nil && !info.IsDir() {
				return candidate, true
			}
		}
	}
	return firstNonEmpty(candidates), false
}

func firstNonEmpty(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// fetchProgress adapts download progress to human-readable stage lines on w,
// emitting once per stage transition (not per byte). Returns nil when w is nil.
func fetchProgress(w io.Writer, name string) binaryfetch.ProgressFunc {
	if w == nil {
		return nil
	}
	var lastStage binaryfetch.Stage
	return func(p binaryfetch.Progress) {
		if p.Stage == lastStage {
			return
		}
		lastStage = p.Stage
		fmt.Fprintf(w, "%s %s\n", p.Stage, name)
	}
}
