package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	vroolilauncher "github.com/vrooli/vrooli/internal/safeguards/vrooli-launcher"
)

// vrooliLauncherStatFn / vrooliExecutableFn are test seams for vrooliInvocation.
// Production maps them to os.Stat(LauncherPath) / os.Executable. Tests
// override to simulate "shim present", "shim missing", or "executable
// lookup failed" without touching the real filesystem.
var (
	vrooliLauncherStatFn = func() (os.FileInfo, error) { return os.Stat(vroolilauncher.LauncherPath) }
	vrooliExecutableFn   = os.Executable
)

// vrooliInvocation returns the command string the action block should
// suggest when telling the operator how to re-run the CLI under sudo or
// in any other constrained-PATH context.
//
// Bootstrap chicken-and-egg: on a fresh install the launcher shim at
// /usr/local/bin/vrooli does not exist yet — sudo's secure_path does not
// include ~/.vrooli/bin (where `make install` writes the real binary), so
// a bare `sudo vrooli ...` fails with "command not found". The action
// block needs to emit a command that actually works, so:
//
//   - If the launcher shim exists at /usr/local/bin/vrooli, return the
//     bare name "vrooli". After the first sudo'd `--include-optional`
//     run installs the shim, this is the steady state.
//   - Otherwise, return the absolute path of the running binary
//     (os.Executable). `sudo /home/user/.vrooli/bin/vrooli ...` works
//     regardless of secure_path because sudo only consults PATH when the
//     argument is a bare name.
//
// We deliberately do NOT shell out to `sudo -n env` to discover the real
// secure_path — that would either prompt for a password or look like a
// lie when sudo is unavailable. A static check of the shim's well-known
// location is enough: the vrooli_launcher safeguard
// (internal/safeguards/vrooli-launcher) is the authoritative installer;
// if it reports AlreadyPresent the file is at LauncherPath, period.
//
// If os.Executable itself fails (vanishingly rare — unsupported platform
// or deleted binary), fall back to the bare "vrooli" name. Worst case the
// operator gets the same "command not found" they had before, which is
// no worse than the status quo and keeps the hint readable.
func vrooliInvocation() string {
	if _, err := vrooliLauncherStatFn(); err == nil {
		return "vrooli"
	}
	exe, err := vrooliExecutableFn()
	if err != nil || strings.TrimSpace(exe) == "" {
		return "vrooli"
	}
	return exe
}

// renderMode controls which printer style renderSetupRequirementResult uses.
type renderMode int

const (
	renderModeGrouped renderMode = iota
	renderModeVerbose
)

func renderSetupRequirementResult(w io.Writer, opts Options, report vrooliruntime.Report) {
	if w == nil {
		return
	}

	label := "result"
	if opts.DryRun {
		label = "dry-run result"
	}
	_, _ = fmt.Fprintf(
		w,
		"[INFO]    Host requirements %s (environment=%s)\n",
		label,
		displaySelection(report.Environment, displaySelection(opts.Environment, defaultEnvironment)),
	)
	_, _ = fmt.Fprintf(
		w,
		"[INFO]    Selection: resources=%s scenarios=%s\n",
		displaySelection(opts.Resources, "enabled"),
		displaySelection(opts.Scenarios, "none"),
	)
	mode := renderModeGrouped
	if opts.Verbose {
		mode = renderModeVerbose
	}
	renderSetupRequirementOverview(w, report, true, mode)
}

func renderSetupRequirementOverview(w io.Writer, report vrooliruntime.Report, executed bool, mode renderMode) {
	items := appendRequirementItems(nil, report.Tools)
	items = appendRequirementItems(items, report.Safeguards)
	if len(items) == 0 {
		_, _ = fmt.Fprintln(w, "[INFO]    No declared host requirements selected")
		return
	}

	if hostSummary := summarizeHost(report.Host); hostSummary != "" {
		_, _ = fmt.Fprintf(w, "[INFO]    Host: %s\n", hostSummary)
	}

	if mode == renderModeVerbose {
		_, _ = fmt.Fprintf(w, "[INFO]    Summary: %s\n", summarizeExecutionStates(items, executed))
		renderRequirementVerboseSection(w, "Tools", report.Tools, executed)
		renderRequirementVerboseSection(w, "Safeguards", report.Safeguards, executed)
		return
	}

	renderGrouped(w, report)
}

// renderGrouped prints a status-grouped, scannable summary. Failures and
// pending operator-action items appear at the top with one line per item;
// already-present and not-applicable groups collapse to a single line each.
//
// Render order: action block → Failed → NeedsSudo → NeedsOperatorInput →
// Optional → Applied → AlreadyPresent → NotApplicable → Unsupported → Δ.
func renderGrouped(w io.Writer, report vrooliruntime.Report) {
	items := appendRequirementItems(nil, report.Tools)
	items = appendRequirementItems(items, report.Safeguards)
	groups := groupItemsByOutcome(items)

	renderActionBlock(w, groups)

	if len(groups.Failed) > 0 {
		_, _ = fmt.Fprintf(w, "Failed (%d):\n", len(groups.Failed))
		for _, item := range groups.Failed {
			renderGroupedItem(w, "✗", item, true)
		}
	}

	if len(groups.NeedsSudo) > 0 {
		// Use vrooliInvocation here too so the group header matches the
		// action block: bare `sudo vrooli setup` post-shim, absolute path
		// pre-shim. Keeps the suggested command consistent in two places.
		_, _ = fmt.Fprintf(w, "Needs sudo — re-run with `sudo %s setup` (%d):\n", vrooliInvocation(), len(groups.NeedsSudo))
		for _, item := range groups.NeedsSudo {
			renderGroupedItem(w, "✗", item, true)
		}
	}

	if len(groups.NeedsOperatorInput) > 0 {
		_, _ = fmt.Fprintf(w, "Needs operator input (%d):\n", len(groups.NeedsOperatorInput))
		for _, item := range groups.NeedsOperatorInput {
			renderGroupedItem(w, "•", item, false)
		}
	}

	if len(groups.Optional) > 0 {
		_, _ = fmt.Fprintf(w, "Optional — opt in with `%s setup --include-optional` (%d): %s\n",
			vrooliInvocation(), len(groups.Optional), itemNames(groups.Optional))
	}

	if len(groups.Applied) > 0 {
		_, _ = fmt.Fprintf(w, "Applied (%d):\n", len(groups.Applied))
		for _, item := range groups.Applied {
			renderGroupedItem(w, "✓", item, false)
		}
	}

	if len(groups.AlreadyPresent) > 0 {
		_, _ = fmt.Fprintf(w, "Already present (%d): %s\n",
			len(groups.AlreadyPresent), itemNames(groups.AlreadyPresent))
	}

	if len(groups.NotApplicable) > 0 {
		_, _ = fmt.Fprintf(w, "Not applicable (%d):\n", len(groups.NotApplicable))
		for _, item := range groups.NotApplicable {
			renderGroupedItem(w, "–", item, true)
		}
	}

	if len(groups.Unsupported) > 0 {
		_, _ = fmt.Fprintf(w, "Unsupported (%d): %s\n",
			len(groups.Unsupported), itemNames(groups.Unsupported))
	}

	_, _ = fmt.Fprintf(w, "Δ %s\n", deltaSummary(groups))
	_, _ = fmt.Fprintln(w, "Run 'vrooli setup explain <name>' for reasons / notes / declarer.")
}

// renderActionBlock prints a top-of-output summary of the exact commands the
// operator should run to clear the actionable groups. It is the headline UX
// win of the grouped renderer: tells the user what to do without making them
// scan the whole report. Suppressed when nothing is actionable.
func renderActionBlock(w io.Writer, groups outcomeGroups) {
	type action struct {
		hint    string
		command string
	}
	var actions []action

	// vrooliInvocation resolves to the bare name "vrooli" once the
	// launcher shim is installed; otherwise it returns the absolute path so
	// `sudo <abs-path> ...` works without the shim. See vrooliInvocation's
	// doc comment for the bootstrap rationale.
	cmd := vrooliInvocation()

	if needsSudo := len(groups.NeedsSudo); needsSudo > 0 {
		actions = append(actions, action{
			hint:    fmt.Sprintf("Re-run with sudo to install %d blocked item(s):", needsSudo),
			command: "sudo " + cmd + " setup",
		})
	}
	if optional := len(groups.Optional); optional > 0 {
		actions = append(actions, action{
			hint:    fmt.Sprintf("Apply %d optional safeguard(s):", optional),
			command: cmd + " setup --include-optional",
		})
	}
	if needsInput := len(groups.NeedsOperatorInput); needsInput > 0 {
		actions = append(actions, action{
			hint:    fmt.Sprintf("Resolve %d item(s) waiting on operator input:", needsInput),
			command: cmd + " setup explain <name>",
		})
	}

	if len(actions) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, "To finish setup:")
	for _, a := range actions {
		_, _ = fmt.Fprintf(w, "  → %-52s %s\n", a.hint, a.command)
	}
	_, _ = fmt.Fprintln(w)
}

func renderGroupedItem(w io.Writer, marker string, item vrooliruntime.ItemStatus, attachDetail bool) {
	headline := primaryNote(item)
	if headline == "" {
		headline = operatorChoiceNote(item)
	}
	if headline == "" {
		headline = strings.TrimSpace(strings.Join(uniqueNonEmpty(item.Reasons), "; "))
	}
	if headline != "" {
		_, _ = fmt.Fprintf(w, "  %s %-28s %s\n", marker, item.Name, truncateLine(headline, 200))
	} else {
		_, _ = fmt.Fprintf(w, "  %s %s\n", marker, item.Name)
	}
	if !attachDetail {
		if choice := operatorChoiceLabel(item.OperatorChoice); choice != "" {
			_, _ = fmt.Fprintf(w, "       operator choice: %s\n", choice)
		}
		return
	}
	if detail := failureDetail(item); detail != "" {
		for _, line := range strings.Split(detail, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			_, _ = fmt.Fprintf(w, "       %s\n", line)
		}
	}
	if choice := operatorChoiceLabel(item.OperatorChoice); choice != "" {
		_, _ = fmt.Fprintf(w, "       operator choice: %s\n", choice)
	}
}

// outcomeGroups partitions items into the buckets the grouped printer renders.
//
// The renderer's design assumes the operator's first question is "what
// should I do next?", so we split the formerly-monolithic Pending bucket
// along its action axis: NeedsSudo (re-run with sudo), Optional (opt in
// with --include-optional), NeedsOperatorInput (configure something or
// reboot). Each carries a different action-block command in
// renderActionBlock.
type outcomeGroups struct {
	Failed             []vrooliruntime.ItemStatus
	NeedsSudo          []vrooliruntime.ItemStatus
	NeedsOperatorInput []vrooliruntime.ItemStatus
	Optional           []vrooliruntime.ItemStatus
	Applied            []vrooliruntime.ItemStatus
	AlreadyPresent     []vrooliruntime.ItemStatus
	NotApplicable      []vrooliruntime.ItemStatus
	Unsupported        []vrooliruntime.ItemStatus
}

func groupItemsByOutcome(items []vrooliruntime.ItemStatus) outcomeGroups {
	var groups outcomeGroups
	for _, item := range items {
		// BlockingReason takes precedence over ExecutionState for routing —
		// a Failed item with reason=needs_sudo belongs in NeedsSudo, not in
		// the generic Failed bucket. This is what lets the action block
		// generate accurate copy-pasteable next-step commands.
		switch item.BlockingReason {
		case hostreqkit.BlockingNeedsSudo:
			groups.NeedsSudo = append(groups.NeedsSudo, item)
			continue
		case hostreqkit.BlockingOptionalSkipped:
			groups.Optional = append(groups.Optional, item)
			continue
		case hostreqkit.BlockingNeedsReboot, hostreqkit.BlockingManual, hostreqkit.BlockingNeedsEnv,
			hostreqkit.BlockingOperatorChoiceMissing, hostreqkit.BlockingOperatorDeclined,
			hostreqkit.BlockingInvalidParameter, hostreqkit.BlockingCredentialStoreLocked,
			hostreqkit.BlockingCredentialStoreUnresponsive, hostreqkit.BlockingCredentialStoreUnavailable,
			hostreqkit.BlockingCredentialStoreEmpty, hostreqkit.BlockingPrerequisiteMissing:
			groups.NeedsOperatorInput = append(groups.NeedsOperatorInput, item)
			continue
		}

		switch item.ExecutionState {
		case vrooliruntime.ExecutionFailed:
			groups.Failed = append(groups.Failed, item)
		case vrooliruntime.ExecutionRebootRequired,
			vrooliruntime.ExecutionManualActionRequired:
			groups.NeedsOperatorInput = append(groups.NeedsOperatorInput, item)
		case vrooliruntime.ExecutionPending,
			vrooliruntime.ExecutionWouldInstall,
			vrooliruntime.ExecutionWouldApply:
			groups.NeedsOperatorInput = append(groups.NeedsOperatorInput, item)
		case vrooliruntime.ExecutionInstalled, vrooliruntime.ExecutionApplied:
			groups.Applied = append(groups.Applied, item)
		case vrooliruntime.ExecutionAlreadyPresent:
			groups.AlreadyPresent = append(groups.AlreadyPresent, item)
		case vrooliruntime.ExecutionNotApplicable:
			groups.NotApplicable = append(groups.NotApplicable, item)
		case vrooliruntime.ExecutionUnsupported:
			groups.Unsupported = append(groups.Unsupported, item)
		default:
			groups.NeedsOperatorInput = append(groups.NeedsOperatorInput, item)
		}
	}
	sortByName(groups.Failed)
	sortByName(groups.NeedsSudo)
	sortByName(groups.NeedsOperatorInput)
	sortByName(groups.Optional)
	sortByName(groups.Applied)
	sortByName(groups.AlreadyPresent)
	sortByName(groups.NotApplicable)
	sortByName(groups.Unsupported)
	return groups
}

func sortByName(items []vrooliruntime.ItemStatus) {
	sort.SliceStable(items, func(i, j int) bool { return items[i].Name < items[j].Name })
}

func itemNames(items []vrooliruntime.ItemStatus) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name)
	}
	return strings.Join(names, ", ")
}

func deltaSummary(groups outcomeGroups) string {
	installed := 0
	applied := 0
	for _, item := range groups.Applied {
		switch item.ExecutionState {
		case vrooliruntime.ExecutionInstalled:
			installed++
		case vrooliruntime.ExecutionApplied:
			applied++
		default:
			// would_install / would_apply / pending counted as planned.
		}
	}
	pending := len(groups.NeedsSudo) + len(groups.NeedsOperatorInput) + len(groups.Optional)
	return fmt.Sprintf(
		"installed=%d  applied=%d  failed=%d  pending=%d  unchanged=%d",
		installed,
		applied,
		len(groups.Failed),
		pending,
		len(groups.AlreadyPresent)+len(groups.NotApplicable)+len(groups.Unsupported),
	)
}

// primaryNote picks the most operator-relevant single line from item.Notes.
// Notes are accumulated across Inspect/Apply, so the last entry is usually
// the most recent and most actionable.
func primaryNote(item vrooliruntime.ItemStatus) string {
	notes := uniqueNonEmpty(item.Notes)
	if len(notes) == 0 {
		return ""
	}
	return notes[len(notes)-1]
}

// failureDetail returns multi-line failure context for the indented detail
// block under a Failed item. It surfaces all distinct notes since they often
// include the captured stderr tail and the install-command resolution error.
func failureDetail(item vrooliruntime.ItemStatus) string {
	notes := uniqueNonEmpty(item.Notes)
	if len(notes) <= 1 {
		return ""
	}
	return strings.Join(notes[:len(notes)-1], "\n")
}

func truncateLine(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max-1] + "…"
}

// renderRequirementVerboseSection prints the pre-existing per-item block
// (one block per tool/safeguard with reasons, notes, declarer). Used by
// --verbose and by `vrooli setup explain`.
func renderRequirementVerboseSection(w io.Writer, heading string, items []vrooliruntime.ItemStatus, executed bool) {
	if len(items) == 0 {
		return
	}

	_, _ = fmt.Fprintf(w, "[INFO]    %s:\n", heading)
	for _, item := range items {
		renderRequirementVerboseItem(w, item, executed)
	}
}

// renderRequirementVerboseItem prints a single item's full block (no heading).
func renderRequirementVerboseItem(w io.Writer, item vrooliruntime.ItemStatus, executed bool) {
	_, _ = fmt.Fprintf(
		w,
		"  - %s [%s] %s | declared by %s\n",
		item.Name,
		requirementScope(item.Required),
		describeExecutionState(item, executed),
		describeProvenance(item),
	)
	if reasons := strings.Join(uniqueNonEmpty(item.Reasons), "; "); reasons != "" {
		_, _ = fmt.Fprintf(w, "    reasons: %s\n", reasons)
	}
	if notes := strings.Join(uniqueNonEmpty(item.Notes), "; "); notes != "" {
		_, _ = fmt.Fprintf(w, "    notes: %s\n", notes)
	}
	if choice := operatorChoiceLabel(item.OperatorChoice); choice != "" {
		_, _ = fmt.Fprintf(w, "    operator choice: %s\n", choice)
	}
	if len(item.Config) > 0 {
		if data, err := json.Marshal(item.Config); err == nil {
			_, _ = fmt.Fprintf(w, "    config: %s\n", data)
		}
	}
}

func operatorChoiceNote(item vrooliruntime.ItemStatus) string {
	switch item.BlockingReason {
	case hostreqkit.BlockingOperatorChoiceMissing:
		return "no operator choice recorded"
	case hostreqkit.BlockingOperatorDeclined:
		return "operator declined"
	}
	return ""
}

func operatorChoiceLabel(choice hostreqspec.OperatorChoice) string {
	switch choice {
	case hostreqspec.OperatorChoiceOptedIn:
		return "opted_in"
	case hostreqspec.OperatorChoiceDeclined:
		return "declined"
	case hostreqspec.OperatorChoiceNotRecorded, "":
		return "not_recorded"
	default:
		return string(choice)
	}
}

func appendRequirementItems(dst []vrooliruntime.ItemStatus, items []vrooliruntime.ItemStatus) []vrooliruntime.ItemStatus {
	dst = append(dst, items...)
	for _, item := range items {
		if item.Name != "remote_desktop_access" || item.SelectedProvider != "gnome-user-shared" || item.CredentialStoreState == "ready" {
			continue
		}
		dependency := item
		dependency.Name = "credential_store"
		dependency.Command = ""
		dependency.Notes = []string{"dependency of remote_desktop_access; credential-store state: " + item.CredentialStoreState}
		dependency.Reasons = []string{"remote_desktop_access requires credential-store readiness"}
		dst = append(dst, dependency)
	}
	return dst
}

func summarizeHost(host vrooliruntime.Host) string {
	parts := []string{host.OS}
	if strings.TrimSpace(host.PackageManager) != "" {
		parts = append(parts, "pkg="+host.PackageManager)
	}
	return strings.Join(parts, " ")
}

func summarizeExecutionStates(items []vrooliruntime.ItemStatus, executed bool) string {
	counts := map[string]int{}
	for _, item := range items {
		counts[describeExecutionState(item, executed)]++
	}

	labels := make([]string, 0, len(counts))
	for label := range counts {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		parts = append(parts, fmt.Sprintf("%d %s", counts[label], label))
	}
	return strings.Join(parts, ", ")
}

func describeExecutionState(item vrooliruntime.ItemStatus, executed bool) string {
	switch item.ExecutionState {
	case vrooliruntime.ExecutionAlreadyPresent:
		return "already_present"
	case vrooliruntime.ExecutionWouldInstall:
		return "would_install"
	case vrooliruntime.ExecutionWouldApply:
		return "would_apply"
	case vrooliruntime.ExecutionInstalled:
		return "installed"
	case vrooliruntime.ExecutionApplied:
		return "applied"
	case vrooliruntime.ExecutionRebootRequired:
		return "reboot_required"
	case vrooliruntime.ExecutionManualActionRequired:
		return "manual_action_required"
	case vrooliruntime.ExecutionUnsupported:
		return "unsupported"
	case vrooliruntime.ExecutionNotApplicable:
		return "not_applicable"
	case vrooliruntime.ExecutionFailed:
		return "failed"
	case vrooliruntime.ExecutionPending:
		if executed {
			return "pending"
		}
		if item.Kind == hostreq.KindSafeguard {
			return "planned_apply"
		}
		return "planned_install"
	default:
		if executed {
			return string(item.ExecutionState)
		}
		return "planned"
	}
}

func describeProvenance(item vrooliruntime.ItemStatus) string {
	if len(item.Provenance) == 0 {
		return "unknown"
	}

	parts := make([]string, 0, len(item.Provenance))
	for _, provenance := range item.Provenance {
		parts = append(parts, fmt.Sprintf("%s:%s (%s)", provenance.Kind, provenance.Name, provenance.Source))
	}
	return strings.Join(parts, ", ")
}

func displaySelection(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func requirementScope(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}

// findItemByName returns the matching ItemStatus from the report (case-
// insensitive), searching tools first then safeguards. Used by `setup explain`.
