package runtime

import (
	"fmt"
	"strings"
	"sync"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

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
			if updated, skip := skipOptional(item, includeOptional); skip {
				items[index] = updated
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
		if notesContainElevationRequired(status.Notes) {
			status.ExecutionState = hostreqkit.ExecutionManualActionRequired
			status.BlockingReason = hostreqkit.BlockingManual
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

func notesContainElevationRequired(notes []string) bool {
	for _, note := range notes {
		if strings.Contains(strings.ToLower(note), "elevation required") {
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

func inspectRequirement(host Host, requirement hostreqspec.ResolvedRequirement) ItemStatus {
	if len(requirement.Platforms) > 0 && !hostreqspec.ContainsPlatform(requirement.Platforms, host.OS) {
		return hostreqkit.NotApplicableRequirementStatus(requirement, fmt.Sprintf("declared for %s; current host is %s", strings.Join(requirement.Platforms, ", "), host.OS))
	}
	if strings.TrimSpace(requirement.ConfigError) != "" {
		return hostreqkit.InvalidConfigStatus(requirement, requirement.ConfigError)
	}
	if strings.TrimSpace(requirement.ConfigUnconfigured) != "" {
		return hostreqkit.NotApplicableRequirementStatus(requirement, requirement.ConfigUnconfigured)
	}
	h, err := lookupHandler(requirement.Kind, requirement.Name)
	if err != nil {
		return hostreqkit.UnsupportedRequirementStatus(requirement, fmt.Sprintf("runtime registry unavailable: %v", err))
	}
	if h == nil {
		return hostreqkit.UnsupportedRequirementStatus(requirement, "no native runtime handler registered")
	}
	return honorDeclaredNotApplicable(host, requirement, h.Inspect(host, requirement))
}

// honorDeclaredNotApplicable keeps a safeguard handler from contradicting its
// own manifest. A platform the manifest declares not_applicable is satisfied by
// definition; a handler that reports unsupported there makes a required
// safeguard fail `vrooli setup` on that platform. On 2026-09-15 two handlers
// did exactly that and blocked every Mac from updating, and thirteen more
// reported unsupported where their manifests said not_applicable.
func honorDeclaredNotApplicable(host Host, requirement hostreqspec.ResolvedRequirement, status ItemStatus) ItemStatus {
	if requirement.Kind != hostreqspec.KindSafeguard || status.ExecutionState != hostreqkit.ExecutionUnsupported {
		return status
	}
	platform := string(hostreqspec.PlatformFromGOOS(host.OS))
	if platform == "" || declaredSafeguardPlatformStatus(requirement.Name, platform) != "not_applicable" {
		return status
	}
	status.SupportClass = hostreqkit.SupportNotApplicable
	status.ExecutionState = hostreqkit.ExecutionNotApplicable
	status.Notes = append(status.Notes, "the safeguard manifest declares "+platform+" not applicable")
	return status
}

var (
	declaredPlatformStatusOnce sync.Once
	declaredPlatformStatus     map[string]map[string]string
)

// declaredSafeguardPlatformStatus returns the manifest's platform_status for one
// safeguard ("" when undeclared or the catalog cannot be read).
func declaredSafeguardPlatformStatus(name, platform string) string {
	declaredPlatformStatusOnce.Do(func() {
		declaredPlatformStatus = map[string]map[string]string{}
		manifests, err := safeguardManifests()
		if err != nil {
			return
		}
		for _, manifest := range manifests {
			byPlatform := make(map[string]string, len(manifest.PlatformStatus))
			for key, declared := range manifest.PlatformStatus {
				byPlatform[key] = declared.Status
			}
			declaredPlatformStatus[manifest.Name] = byPlatform
		}
	})
	return declaredPlatformStatus[name][platform]
}

func applyRequirement(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error) {
	// A malformed or incomplete assembled configuration is an admission
	// failure, not a handler input. Keep this guard at the common owner apply
	// boundary so focused CLI calls and full setup cannot run an effect after
	// inspection rejected the configuration.
	if status.BlockingReason == hostreqkit.BlockingInvalidParameter {
		return status, nil
	}
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
	case hostreqspec.KindSafeguard:
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
	message := fmt.Sprintf("missing required host requirements for %s: %s", hostreqspec.NormalizeEnvironment(report.Environment), strings.Join(missingRequiredDetails(report), ", "))
	if commands := missingRequiredToolInstallCommands(report); len(commands) > 0 {
		message += "; install missing tools with: " + strings.Join(commands, "; ")
	}
	return fmt.Errorf("%s", message)
}

func missingRequiredDetails(report Report) []string {
	statuses := make(map[string]ItemStatus, len(report.Tools)+len(report.Safeguards))
	for _, item := range report.Tools {
		statuses[item.Name] = item
	}
	for _, item := range report.Safeguards {
		statuses[item.Name] = item
	}
	details := make([]string, 0, len(report.MissingRequired))
	for _, name := range report.MissingRequired {
		status, ok := statuses[name]
		if !ok {
			details = append(details, name)
			continue
		}
		attributes := []string{"state=" + string(status.ExecutionState)}
		if status.Command != "" {
			attributes = append(attributes, "command="+status.Command)
		}
		if status.Version != "" {
			attributes = append(attributes, "version="+status.Version)
		}
		if len(status.Notes) > 0 {
			attributes = append(attributes, "detail="+compactRequirementNote(status.Notes[len(status.Notes)-1]))
		}
		details = append(details, name+" ("+strings.Join(attributes, ", ")+")")
	}
	return details
}

func compactRequirementNote(note string) string {
	const maxNoteLength = 240
	compact := strings.Join(strings.Fields(note), " ")
	if len(compact) > maxNoteLength {
		return compact[:maxNoteLength-3] + "..."
	}
	return compact
}

func missingRequiredToolInstallCommands(report Report) []string {
	commands := make([]string, 0, len(report.Tools))
	for _, tool := range report.Tools {
		if !tool.Required || requirementSatisfied(tool) {
			continue
		}
		// A tool whose version probe could not execute is present and blocked by
		// the environment, not by a bad install. Offering the install command
		// here sends the operator into a re-download loop that ends in exactly
		// the same failure. Every other blocker keeps the remediation, including
		// a broken payload, which an install does repair.
		if tool.BlockingReason == hostreqkit.BlockingProbeFailed {
			continue
		}
		commands = append(commands, "vrooli host install "+tool.Name)
	}
	return commands
}
