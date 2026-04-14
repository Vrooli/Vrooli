package setup

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreq"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
)

func renderSetupRequirementPlan(w io.Writer, opts Options, report vrooliruntime.Report) {
	if w == nil {
		return
	}

	_, _ = fmt.Fprintf(
		w,
		"[INFO]    Host requirements plan (environment=%s resources=%s scenarios=%s dry_run=%t)\n",
		displaySelection(report.Environment, displaySelection(opts.Environment, defaultEnvironment)),
		displaySelection(opts.Resources, "enabled"),
		displaySelection(opts.Scenarios, "none"),
		opts.DryRun,
	)
	renderSetupRequirementOverview(w, report, false)
}

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
	renderSetupRequirementOverview(w, report, true)
}

func renderSetupRequirementOverview(w io.Writer, report vrooliruntime.Report, executed bool) {
	items := appendRequirementItems(nil, report.Tools)
	items = appendRequirementItems(items, report.Safeguards)
	if len(items) == 0 {
		_, _ = fmt.Fprintln(w, "[INFO]    No declared host requirements selected")
		return
	}

	if hostSummary := summarizeHost(report.Host); hostSummary != "" {
		_, _ = fmt.Fprintf(w, "[INFO]    Host: %s\n", hostSummary)
	}
	_, _ = fmt.Fprintf(w, "[INFO]    Summary: %s\n", summarizeExecutionStates(items, executed))
	renderRequirementSection(w, "Tools", report.Tools, executed)
	renderRequirementSection(w, "Safeguards", report.Safeguards, executed)
}

func renderRequirementSection(w io.Writer, heading string, items []vrooliruntime.ItemStatus, executed bool) {
	if len(items) == 0 {
		return
	}

	_, _ = fmt.Fprintf(w, "[INFO]    %s:\n", heading)
	for _, item := range items {
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
	}
}

func appendRequirementItems(dst []vrooliruntime.ItemStatus, items []vrooliruntime.ItemStatus) []vrooliruntime.ItemStatus {
	return append(dst, items...)
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

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
