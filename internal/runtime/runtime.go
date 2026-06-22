package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

var ErrUnsupportedPlatform = hostreqkit.ErrUnsupportedPlatform

type Host = hostreqkit.Host

type EnsureOptions = hostreqkit.EnsureOptions

func Current() Host {
	return currentHost()
}

func InspectRequirements(environment string, resolution hostreq.Resolution) (Report, error) {
	env := hostreq.NormalizeEnvironment(environment)
	return inspectResolution(Current(), env, resolution)
}

func EnsureRequirements(opts EnsureOptions, resolution hostreq.Resolution) (Report, error) {
	opts.Environment = hostreq.NormalizeEnvironment(opts.Environment)
	return ensureResolution(opts, resolution)
}

// EnsureTool inspects and (unless opts.DryRun) installs a single host tool by
// name through its registered runtime handler, returning the final status. It is
// the engine behind `vrooli host install <tool>`: it honors the capability gate
// (a not-applicable tool is returned as such, never installed) and the
// url/release fetch path (no sudo, into ~/.vrooli/bin). A tool with no
// registered handler comes back unsupported.
func EnsureTool(name string, opts EnsureOptions) (ItemStatus, error) {
	opts.Environment = hostreq.NormalizeEnvironment(opts.Environment)
	host := Current()
	requirement := hostreq.ResolvedRequirement{
		Name:     strings.TrimSpace(name),
		Kind:     hostreq.KindTool,
		Required: true,
	}
	status := inspectRequirement(host, requirement)
	if status.ExecutionState == hostreqkit.ExecutionAlreadyPresent {
		return status, nil
	}
	updated, err := applyRequirement(host, status, opts)
	if err != nil {
		return status, err
	}
	return annotateBlockingReason(updated), nil
}

func ensureResolution(opts EnsureOptions, resolution hostreq.Resolution) (Report, error) {
	// Earlier versions of vrooli ran `go install` and `npm install`
	// directly under sudo (not dropping privileges), which left root-
	// owned files in the operator's ~/go/pkg/mod and ~/.cache that broke
	// subsequent non-sudo `go build` invocations with errors like
	// "missing go.sum entry". Detect and repair before doing anything
	// else that might depend on the user's caches being writable.
	// Idempotent: a no-op when ownership is already correct.
	repairInvokingUserCacheOwnership(opts)

	report, err := inspectResolution(Current(), opts.Environment, resolution)
	if err != nil {
		return Report{}, err
	}
	if !opts.AutoInstall {
		return report, missingRequiredError(report, opts)
	}

	for index, status := range report.Tools {
		if requirementSatisfied(status) {
			continue
		}
		if !status.Required && !opts.IncludeOptional {
			if isPendingState(status.ExecutionState) {
				report.Tools[index] = markOptionalSkipped(status)
			}
			continue
		}
		updated, applyErr := applyRequirement(report.Host, status, opts)
		if applyErr != nil {
			return Report{}, applyErr
		}
		report.Tools[index] = annotateBlockingReason(updated)
	}
	for index, status := range report.Safeguards {
		if requirementSatisfied(status) {
			continue
		}
		if !status.Required && !opts.IncludeOptional {
			if isPendingState(status.ExecutionState) {
				report.Safeguards[index] = markOptionalSkipped(status)
			}
			continue
		}
		updated, applyErr := applyRequirement(report.Host, status, opts)
		if applyErr != nil {
			return Report{}, applyErr
		}
		report.Safeguards[index] = annotateBlockingReason(updated)
	}

	report = summarizeReport(report)
	return report, missingRequiredError(report, opts)
}

// markOptionalSkipped is the path for optional items that the operator did not
// opt into via --include-optional. We deliberately do not call Apply: the item
// stays in whatever Pending-flavored state Inspect returned, but we tag it so
// the renderer routes it into the "Optional — opt in with --include-optional"
// group instead of the generic Pending bucket.
func markOptionalSkipped(status ItemStatus) ItemStatus {
	status.BlockingReason = hostreqkit.BlockingOptionalSkipped
	return status
}

// AnnotateInspectOnly post-processes a Report from inspect-only flows
// (`vrooli setup status`) so the renderer's grouped output matches what the
// apply path would produce: optional pending items get tagged
// BlockingOptionalSkipped, reboot/manual states get their reasons. Without
// this, status would show every pending item in the generic operator-input
// bucket regardless of whether it's optional or actionable.
func AnnotateInspectOnly(report Report, includeOptional bool) Report {
	annotate := func(items []ItemStatus) []ItemStatus {
		for index, item := range items {
			if item.BlockingReason != hostreqkit.BlockingNone {
				continue
			}
			if !item.Required && !includeOptional && isPendingState(item.ExecutionState) {
				items[index] = markOptionalSkipped(item)
				continue
			}
			items[index] = annotateBlockingReason(item)
		}
		return items
	}
	report.Tools = annotate(report.Tools)
	report.Safeguards = annotate(report.Safeguards)
	return report
}

// isPendingState reports whether a state is "operator could still take an
// action that changes this." NotApplicable / Unsupported / AlreadyPresent /
// Failed don't qualify — those are settled outcomes.
func isPendingState(state ExecutionState) bool {
	switch state {
	case hostreqkit.ExecutionPending,
		hostreqkit.ExecutionWouldInstall,
		hostreqkit.ExecutionWouldApply:
		return true
	}
	return false
}

// annotateBlockingReason inspects the post-Apply status and infers a
// BlockingReason when the handler didn't set one explicitly. Handlers
// currently swallow privileged-command errors into their Notes, so we
// fall back to scanning Notes for the typed-sentinel strings emitted by
// hostreqkit.WithSudo. (Handlers can also set BlockingReason directly; that
// value is preserved.)
func annotateBlockingReason(status ItemStatus) ItemStatus {
	if status.BlockingReason != hostreqkit.BlockingNone {
		return status
	}
	switch status.ExecutionState {
	case hostreqkit.ExecutionFailed:
		if notesContainSudoSentinel(status.Notes) {
			status.BlockingReason = hostreqkit.BlockingNeedsSudo
		}
	case hostreqkit.ExecutionRebootRequired:
		status.BlockingReason = hostreqkit.BlockingNeedsReboot
	case hostreqkit.ExecutionManualActionRequired:
		status.BlockingReason = hostreqkit.BlockingManual
	}
	return status
}

// notesContainSudoSentinel matches the literal substrings emitted by the
// typed sudo errors in internal/hostreqkit/install.go. We intentionally use
// substring matching here (rather than errors.Is on a returned error) because
// handlers fold errors into status.Notes via fmt.Sprintf — the typed error
// chain is lost by the time the runtime sees the status. The sentinel
// strings are stable: change them here if you change them there.
func notesContainSudoSentinel(notes []string) bool {
	for _, note := range notes {
		if strings.Contains(note, "sudo skipped") || strings.Contains(note, "sudo unavailable") {
			return true
		}
	}
	return false
}

func inspectResolution(host Host, environment string, resolution hostreq.Resolution) (Report, error) {
	report := Report{
		Environment: environment,
		Host:        host,
		Tools:       make([]ToolStatus, 0, len(resolution.Tools)),
		Safeguards:  make([]SafeguardStatus, 0, len(resolution.Safeguards)),
	}
	for _, requirement := range resolution.Tools {
		report.Tools = append(report.Tools, inspectRequirement(host, requirement))
	}
	for _, requirement := range resolution.Safeguards {
		report.Safeguards = append(report.Safeguards, inspectRequirement(host, requirement))
	}
	return summarizeReport(report), nil
}

func inspectRequirement(host Host, requirement hostreq.ResolvedRequirement) ItemStatus {
	h, err := lookupHandler(requirement.Kind, requirement.Name)
	if err != nil {
		return hostreqkit.UnsupportedRequirementStatus(requirement, fmt.Sprintf("runtime registry unavailable: %v", err))
	}
	if h == nil {
		return hostreqkit.UnsupportedRequirementStatus(requirement, "no native runtime handler registered")
	}
	return h.Inspect(host, requirement)
}

func applyRequirement(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	h, err := lookupHandler(status.Kind, status.Name)
	if err != nil {
		return status, fmt.Errorf("runtime registry unavailable: %w", err)
	}
	if h == nil {
		status.Notes = append(status.Notes, "no native runtime handler registered")
		status.SupportClass = SupportUnsupported
		return status, nil
	}
	return h.Apply(host, status, opts)
}

func summarizeReport(report Report) Report {
	report.MissingRequired = report.MissingRequired[:0]
	report.MissingOptional = report.MissingOptional[:0]
	for _, tool := range report.Tools {
		appendMissingRequirement(&report, tool)
	}
	for _, safeguard := range report.Safeguards {
		appendMissingRequirement(&report, safeguard)
	}
	return report
}

func appendMissingRequirement(report *Report, status ItemStatus) {
	if requirementSatisfied(status) {
		return
	}
	if status.Required {
		report.MissingRequired = append(report.MissingRequired, status.Name)
		return
	}
	report.MissingOptional = append(report.MissingOptional, status.Name)
}

func requirementSatisfied(status ItemStatus) bool {
	// ExecutionAlreadyPresent is the canonical "the requirement is met by some
	// other path" signal (e.g. mcelog superseded by rasdaemon). Honor it for
	// both tools and safeguards regardless of Installed/Applied bookkeeping —
	// otherwise the supersede branch would still surface in MissingRequired.
	if status.ExecutionState == hostreqkit.ExecutionAlreadyPresent {
		return true
	}
	// A capability-gated tool that is not applicable on this host is not
	// "missing" — it was intentionally skipped (e.g. a GPU-only backend on a
	// CPU-only host), so it must not surface in MissingRequired/Optional.
	if status.ExecutionState == hostreqkit.ExecutionNotApplicable {
		return true
	}
	switch status.Kind {
	case hostreq.KindSafeguard:
		return status.Applied
	default:
		return status.Installed
	}
}

func missingRequiredError(report Report, opts EnsureOptions) error {
	if opts.DryRun {
		return nil
	}
	if len(report.MissingRequired) == 0 {
		return nil
	}
	return fmt.Errorf("missing required host requirements for %s: %s", hostreq.NormalizeEnvironment(report.Environment), strings.Join(report.MissingRequired, ", "))
}

// repairInvokingUserCacheOwnership chowns the standard per-user vrooli-
// touched cache directories back to $SUDO_USER when we're running as
// root. This is purely a legacy-damage repair: earlier versions of
// vrooli ran `go install` and `npm install` directly under sudo (without
// dropping privileges), which deposited root-owned files into ~/go and
// ~/.cache. Subsequent non-sudo `go build` invocations then failed with
// "missing go.sum entry" because the operator could not write to those
// paths. This pass corrects the ownership once and is a no-op for
// already-correctly-owned files.
//
// Best-effort: chown failures are non-fatal — we don't block setup over
// a stale cache repair. Only runs when sudo'd with $SUDO_USER set; on
// non-sudo invocations there's nothing to fix.
//
// We intentionally limit the targets to dirs vrooli either creates or
// writes to. ~/.cache as a whole contains a lot of unrelated state we
// must not touch.
func repairInvokingUserCacheOwnership(opts EnsureOptions) {
	if !hostreqkit.RunningAsRootFn() {
		return
	}
	user := strings.TrimSpace(os.Getenv("SUDO_USER"))
	if user == "" || user == "root" {
		return
	}
	home, err := hostreqkit.InvokingUserHomeDir()
	if err != nil || home == "" {
		return
	}
	targets := []string{
		filepath.Join(home, "go"),
		filepath.Join(home, ".cache", "go-build"),
		filepath.Join(home, ".cache", "vrooli"),
		filepath.Join(home, ".local", "bin"),
	}
	for _, target := range targets {
		if _, statErr := os.Stat(target); statErr != nil {
			continue
		}
		spec := user + ":" + user
		// chown -R runs as the current process (root) — we need root to
		// change ownership of root-owned files back to $SUDO_USER. This
		// is the one place in setup where running directly as root is
		// the correct thing.
		_ = hostreqkit.RunCommandFn("chown", []string{"-R", spec, target}, opts)
	}
}
