package pathhygiene

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// startupFiles are the shell startup files the safeguard manages, in the
// order a login shell reads them. Only files that already exist are
// touched, except that a host with none of them gets ~/.profile — the
// POSIX-standard location, read by sh, dash, and bash login shells alike.
var startupFiles = []string{".profile", ".bashrc", ".zshrc"}

// Test seams. Production wiring is the real host.
var (
	homeDirFn   = hostreqkit.InvokingUserHomeDir
	pathEnvFn   = func() string { return os.Getenv("PATH") }
	readFileFn  = func(path string) ([]byte, error) { return os.ReadFile(path) }
	statFn      = func(path string) (os.FileInfo, error) { return os.Stat(path) }
	writeFileFn = writeFileAtomic
)

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler is the constructor wired into the runtime registry under the
// handler name "path_hygiene" (see internal/runtime/registry.go).
func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

// target is one startup file and the rewrite it needs.
type target struct {
	path      string
	current   string
	rewritten string
	findings  Findings
	exists    bool
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	// Windows sets user PATH through the registry (setx / Environment key),
	// not shell startup files, so this safeguard has nothing to manage
	// there. Report Unsupported so it disappears cleanly rather than
	// failing.
	if host.OS != "linux" && host.OS != "darwin" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes,
			"PATH hygiene is POSIX-only; Windows user PATH lives in the registry and needs a separate design")
		return status
	}

	home, err := homeDirFn()
	if err != nil {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "cannot resolve the invoking user's home directory: "+err.Error())
		return status
	}

	targets, err := h.plan(home)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status
	}

	pending := pendingTargets(targets)
	status.Applied = len(pending) == 0
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		status.Notes = append(status.Notes, "managed PATH block current in "+describeFiles(targets))
	} else {
		for _, t := range pending {
			status.Notes = append(status.Notes, describePending(home, t))
		}
	}

	// Observations the safeguard reports but never auto-fixes.
	status.Notes = append(status.Notes, h.observePath(home)...)
	return status
}

// plan computes the rewrite for every managed startup file.
func (h handler) plan(home string) ([]target, error) {
	targets := make([]target, 0, len(startupFiles))
	anyExists := false
	for _, name := range startupFiles {
		path := filepath.Join(home, name)
		if _, err := statFn(path); err != nil {
			targets = append(targets, target{path: path})
			continue
		}
		anyExists = true
		data, err := readFileFn(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		content := string(data)
		rewritten, findings := Rewrite(content)
		targets = append(targets, target{
			path: path, current: content, rewritten: rewritten,
			findings: findings, exists: true,
		})
	}
	if anyExists {
		return targets, nil
	}
	// Fresh host with no startup files at all: create ~/.profile.
	rewritten, findings := Rewrite("")
	targets[0] = target{path: targets[0].path, rewritten: rewritten, findings: findings}
	return targets, nil
}

func pendingTargets(targets []target) []target {
	pending := make([]target, 0, len(targets))
	for _, t := range targets {
		if t.rewritten != "" && t.findings.Changed() {
			pending = append(pending, t)
		}
	}
	return pending
}

func describeFiles(targets []target) string {
	names := make([]string, 0, len(targets))
	for _, t := range targets {
		if t.exists {
			names = append(names, filepath.Base(t.path))
		}
	}
	if len(names) == 0 {
		return "no shell startup files"
	}
	return strings.Join(names, ", ")
}

func describePending(home string, t target) string {
	rel := strings.TrimPrefix(t.path, home+string(filepath.Separator))
	switch {
	case t.findings.LegacyLines > 0 && t.findings.ReplacedBlock:
		return fmt.Sprintf("~/%s: %d unguarded Vrooli PATH line(s) alongside the managed block", rel, t.findings.LegacyLines)
	case t.findings.LegacyLines > 0:
		return fmt.Sprintf("~/%s: %d unguarded Vrooli PATH line(s) will be replaced by one managed block", rel, t.findings.LegacyLines)
	case t.findings.ReplacedBlock:
		return fmt.Sprintf("~/%s: managed block is out of date", rel)
	default:
		return fmt.Sprintf("~/%s: managed block missing", rel)
	}
}

// observePath reports live-PATH problems the safeguard does not own.
func (h handler) observePath(home string) []string {
	pathEnv := pathEnvFn()
	if strings.TrimSpace(pathEnv) == "" {
		return nil
	}
	notes := make([]string, 0, 2)

	if total, unique := EntryCount(pathEnv), UniqueEntryCount(pathEnv); total > unique {
		dups := DuplicateEntries(pathEnv)
		worst := make([]string, 0, 3)
		for i, d := range dups {
			if i == 3 {
				break
			}
			worst = append(worst, fmt.Sprintf("%s x%d", d.Dir, d.Count))
		}
		notes = append(notes, fmt.Sprintf(
			"live PATH has %d entries, %d unique (%s) — duplicates make every lookup miss scan farther; not auto-removed because they are not all Vrooli's",
			total, unique, strings.Join(worst, ", ")))
	}

	canonical := filepath.Join(home, filepath.FromSlash(CanonicalBinSuffix))
	if shadows := ShadowingBinaries(pathEnv, canonical, "vrooli"); len(shadows) > 0 {
		sort.Strings(shadows)
		notes = append(notes, fmt.Sprintf(
			"another vrooli binary precedes %s on PATH: %s — a bare `vrooli` may run a stale build; remove it or let the managed block take the front",
			canonical, strings.Join(shadows, ", ")))
	}
	return notes
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

	home, err := homeDirFn()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "cannot resolve the invoking user's home directory: "+err.Error())
		return status, nil
	}
	targets, err := h.plan(home)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	pending := pendingTargets(targets)

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		for _, t := range pending {
			status.Notes = append(status.Notes, "dry-run: would rewrite "+t.path)
		}
		return status, nil
	}

	// No elevation anywhere in this path: these are the invoking user's own
	// dotfiles, and `vrooli setup` is the only place Vrooli may ask for
	// privilege at all.
	for _, t := range pending {
		if err := writeFileFn(t.path, t.rewritten); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, fmt.Sprintf("rewrite %s failed: %v", t.path, err))
			return status, nil
		}
		status.Notes = append(status.Notes, fmt.Sprintf(
			"%s: removed %d unguarded PATH line(s), installed the managed block",
			t.path, t.findings.LegacyLines))
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes,
		"open a new shell (or `exec $SHELL -l`) for the corrected PATH to take effect")
	return status, nil
}

// writeFileAtomic replaces a file's contents without ever leaving a
// truncated startup file behind — a half-written ~/.profile would break
// every new login shell on the host.
func writeFileAtomic(path, content string) error {
	mode := os.FileMode(0o644)
	if info, err := statFn(path); err == nil {
		mode = info.Mode().Perm()
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".vrooli-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
