package runtime

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/safeguards"
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
		if reconciler, ok := lookupHandler(requirement.Kind, requirement.Name); ok == nil && reconciler != nil {
			if present, supported := reconciler.(interface {
				ReconcilePresent(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error)
			}); supported {
				updated, reconcileErr := present.ReconcilePresent(host, status, opts)
				if reconcileErr != nil {
					return status, reconcileErr
				}
				return annotateBlockingReason(updated), nil
			}
		}
		return status, nil
	}
	updated, err := applyRequirement(host, status, opts)
	if err != nil {
		return status, err
	}
	return annotateBlockingReason(updated), nil
}

// EnsureSafeguard applies one named host safeguard without running the wider
// project setup lifecycle. It is the narrow, auditable path for high-risk host
// state (kernel drivers, firewall rules, and similar) when an operator needs
// to repair one capability rather than re-run every setup requirement.
func EnsureSafeguard(name string, opts EnsureOptions) (ItemStatus, error) {
	name = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "-", "_")
	opts.Environment = hostreq.NormalizeEnvironment(opts.Environment)
	host := Current()
	root, err := os.Getwd()
	if err != nil {
		return ItemStatus{}, fmt.Errorf("resolve project root for safeguard %q: %w", name, err)
	}
	requirement, err := hostreq.ResolveSafeguard(root, name, host.OS)
	if err != nil {
		return ItemStatus{}, err
	}
	status := inspectRequirement(host, requirement)
	if requirementSatisfied(status) {
		return status, nil
	}
	updated, err := applyRequirement(host, status, opts)
	if err != nil {
		return status, err
	}
	return annotateBlockingReason(updated), nil
}

// InspectSafeguard performs the unprivileged read half of focused safeguard
// handling. It never calls Apply and is the control-plane boundary for
// observed host state consumers.
func InspectSafeguard(name string) (ItemStatus, error) {
	root, err := os.Getwd()
	if err != nil {
		return ItemStatus{}, fmt.Errorf("resolve project root for safeguard %q: %w", name, err)
	}
	return InspectSafeguardAt(root, name)
}

// InspectSafeguardAt performs the unprivileged read half of focused safeguard
// handling against an explicit repository root. It never calls Apply and is
// safe for typed consumers running from a scenario module directory.
func InspectSafeguardAt(root, name string) (ItemStatus, error) {
	name = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "-", "_")
	if name == "" {
		return ItemStatus{}, fmt.Errorf("safeguard name is required")
	}
	host := Current()
	requirement, err := hostreq.ResolveSafeguard(root, name, host.OS)
	if err != nil {
		return ItemStatus{}, err
	}
	return inspectRequirement(host, requirement), nil
}

// ListObservedSafeguardsAt returns the complete, read-only host safeguard
// observation surface owned by the control plane. Errors for an individual
// safeguard remain in that safeguard's notes and failed state; one broken
// probe cannot make the rest of the roster disappear.
func ListObservedSafeguardsAt(root string, now func() time.Time) ([]hostreqkit.ObservedSafeguard, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("repository root is required")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	dirs, err := fs.ReadDir(safeguards.Manifests, ".")
	if err != nil {
		return nil, fmt.Errorf("read safeguard catalog: %w", err)
	}
	result := make([]hostreqkit.ObservedSafeguard, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		data, readErr := fs.ReadFile(safeguards.Manifests, dir.Name()+"/safeguard.json")
		if readErr != nil {
			return nil, fmt.Errorf("read safeguard %s: %w", dir.Name(), readErr)
		}
		var manifest hostreqkit.SafeguardManifest
		if unmarshalErr := json.Unmarshal(data, &manifest); unmarshalErr != nil {
			return nil, fmt.Errorf("parse safeguard %s: %w", dir.Name(), unmarshalErr)
		}
		observedAt := now().UTC()
		item := hostreqkit.ObservedSafeguard{
			Name: manifest.Name, Capability: manifest.Capability, CapabilityRole: manifest.CapabilityRole,
			Platforms: append([]string(nil), manifest.Platforms...), ObservedAt: observedAt,
		}
		status, inspectErr := InspectSafeguardAt(root, manifest.Name)
		if inspectErr != nil {
			item.SupportClass = hostreqkit.SupportUnsupported
			item.ExecutionState = hostreqkit.ExecutionFailed
			item.Notes = []string{inspectErr.Error()}
		} else {
			item.SupportClass = status.SupportClass
			item.ExecutionState = status.ExecutionState
			item.Notes = append([]string(nil), status.Notes...)
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func ensureResolution(opts EnsureOptions, resolution hostreq.Resolution) (Report, error) {
	report, err := inspectResolution(Current(), opts.Environment, resolution)
	if err != nil {
		return Report{}, err
	}
	if !opts.AutoInstall {
		return report, missingRequiredError(report, opts)
	}

	for index, status := range report.Tools {
		if opts.OnOperation != nil {
			opts.OnOperation("Applying tool " + status.Name)
		}
		if requirementSatisfied(status) {
			continue
		}
		if updated, skip := skipOptional(status, opts.IncludeOptional); skip {
			report.Tools[index] = updated
			continue
		}
		updated, applyErr := applyRequirement(report.Host, status, opts)
		if applyErr != nil {
			return Report{}, applyErr
		}
		report.Tools[index] = annotateBlockingReason(updated)
	}
	for index, status := range report.Safeguards {
		if opts.OnOperation != nil {
			opts.OnOperation("Applying safeguard " + status.Name)
		}
		if requirementSatisfied(status) {
			continue
		}
		if updated, skip := skipOptional(status, opts.IncludeOptional); skip {
			report.Safeguards[index] = updated
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

func skipOptional(status ItemStatus, includeOptional bool) (ItemStatus, bool) {
	if status.Required || !isPendingState(status.ExecutionState) {
		return status, false
	}
	switch status.OperatorChoice {
	case hostreqspec.OperatorChoiceDeclined:
		status.BlockingReason = hostreqkit.BlockingOperatorDeclined
		return status, true
	case "":
		if !includeOptional {
			return markOptionalSkipped(status), true
		}
	case hostreqspec.OperatorChoiceNotRecorded:
		if !includeOptional {
			status.BlockingReason = hostreqkit.BlockingOperatorChoiceMissing
			return status, true
		}
	case hostreqspec.OperatorChoiceOptedIn:
		if !includeOptional && !status.ConfigNonDefault {
			status.BlockingReason = hostreqkit.BlockingOptionalSkipped
			status.Notes = append(status.Notes, "optional safeguard is opted in but uses manifest defaults; supply non-default configuration or rerun with --include-optional")
			return status, true
		}
	}
	return status, false
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

func inspectRequirement(host Host, requirement hostreq.ResolvedRequirement) ItemStatus {
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
	message := fmt.Sprintf("missing required host requirements for %s: %s", hostreq.NormalizeEnvironment(report.Environment), strings.Join(missingRequiredDetails(report), ", "))
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
