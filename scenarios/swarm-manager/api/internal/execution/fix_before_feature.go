package execution

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"swarm-manager/internal/backlogstatus"
	"swarm-manager/internal/pathutil"
)

// Fix-before-feature gate modes. Mirrors settings.FixBeforeFeature* but is
// defined here because execution must not import settings (settings imports
// execution via its adapters, so the dependency only runs one way).
const (
	FixBeforeFeatureOff     = "off"
	FixBeforeFeatureSuggest = "suggest"
	FixBeforeFeatureBlock   = "block"
)

// openRemediationItem is the minimal projection of an open fix/chore backlog
// item the gate needs: its identity and the scenarios it touches.
type openRemediationItem struct {
	kind      string
	name      string
	scenarios []string
}

// fixBeforeFeatureResult is the gate outcome. At most one of blockingReason /
// advisory is set; both are empty when the gate does not fire.
type fixBeforeFeatureResult struct {
	blockingReason string // "block" mode — forceable
	advisory       string // "suggest" mode — non-blocking
}

// evaluateFixBeforeFeature is the pure core of the gate: given the feature
// item's kind, the scenarios it targets, the set of open remediation items,
// and the configured mode, it decides whether to advise or block. It reads no
// filesystem state so it is fully table-testable.
//
// The gate fires only for feature work (kind=execute) onto a scenario that has
// open fix/chore items. "off" is silent; "suggest" returns an advisory; "block"
// returns a forceable blocking reason.
func evaluateFixBeforeFeature(itemKind string, featureScenarios []string, openItems []openRemediationItem, mode string) fixBeforeFeatureResult {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" || mode == FixBeforeFeatureOff {
		return fixBeforeFeatureResult{}
	}
	if strings.ToLower(strings.TrimSpace(itemKind)) != "execute" {
		return fixBeforeFeatureResult{}
	}
	if len(featureScenarios) == 0 {
		return fixBeforeFeatureResult{}
	}

	targets := make(map[string]struct{}, len(featureScenarios))
	for _, s := range featureScenarios {
		if t := strings.TrimSpace(s); t != "" {
			targets[t] = struct{}{}
		}
	}

	// Collect conflicting remediation items grouped by target scenario.
	conflicts := map[string][]string{} // scenario -> ["fix/foo", ...]
	for _, ri := range openItems {
		key := strings.ToLower(strings.TrimSpace(ri.kind)) + "/" + strings.TrimSpace(ri.name)
		for _, sc := range ri.scenarios {
			if _, ok := targets[sc]; ok {
				conflicts[sc] = append(conflicts[sc], key)
			}
		}
	}
	if len(conflicts) == 0 {
		return fixBeforeFeatureResult{}
	}

	msg := describeFixBeforeFeatureConflicts(conflicts)
	switch mode {
	case FixBeforeFeatureBlock:
		return fixBeforeFeatureResult{blockingReason: msg + " — resolve them first or re-queue with force."}
	case FixBeforeFeatureSuggest:
		return fixBeforeFeatureResult{advisory: msg + " — consider resolving the fix work before stacking feature work."}
	default:
		return fixBeforeFeatureResult{}
	}
}

// describeFixBeforeFeatureConflicts renders a stable, human-readable summary of
// the open remediation items blocking feature work, grouped by scenario.
func describeFixBeforeFeatureConflicts(conflicts map[string][]string) string {
	scenarios := make([]string, 0, len(conflicts))
	for sc := range conflicts {
		scenarios = append(scenarios, sc)
	}
	sort.Strings(scenarios)

	parts := make([]string, 0, len(scenarios))
	for _, sc := range scenarios {
		items := pathutil.UniqueSortedStrings(conflicts[sc])
		parts = append(parts, fmt.Sprintf("%s has %d open remediation item(s) (%s)", sc, len(items), strings.Join(items, ", ")))
	}
	return strings.Join(parts, "; ")
}

// gatherOpenRemediationItems scans the fix/ and chore/ backlog directories and
// returns the open items (status not completed/failed, not archived) projected
// to the scenarios they touch via acceptance_allow globs. Items with no
// resolvable scenario are dropped (they cannot conflict). Filesystem errors on
// individual items are skipped fail-open — the gate must never wedge queuing.
func (s *Service) gatherOpenRemediationItems() []openRemediationItem {
	var out []openRemediationItem
	for _, kind := range []string{"fix", "chore"} {
		dir := s.kindDir(kind)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			item, err := s.loadBacklogItem(kind, name)
			if err != nil {
				continue
			}
			if !isOpenRemediationStatus(item.Status) || item.ArchivedAt != nil {
				continue
			}
			scenarios := pathutil.UniqueSortedStrings(pathutil.ScenariosFromGlobs(item.AcceptanceAllow))
			if len(scenarios) == 0 {
				continue
			}
			out = append(out, openRemediationItem{kind: kind, name: name, scenarios: scenarios})
		}
	}
	return out
}

// isOpenRemediationStatus reports whether a remediation item in this status
// still counts as outstanding. Completed and failed are terminal; everything
// else (backlog, researching, ready, queued, in_progress, in_review,
// review_pending, needs_followup) is open.
func isOpenRemediationStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case backlogstatus.Completed, backlogstatus.Failed:
		return false
	default:
		return true
	}
}

// applyFixBeforeFeatureGate runs the gate for a feature item being queued and
// writes the result onto the preflight. It loads governance (for the mode) and
// scans open remediation items. Any error loading governance fails open (no
// gate) so a transient settings outage never blocks queuing.
func (s *Service) applyFixBeforeFeatureGate(item backlogItem, preflight *ProcessPreflight) {
	if strings.ToLower(strings.TrimSpace(item.Kind)) != "execute" {
		return
	}
	gov, err := s.governanceProvider.LoadGovernance()
	if err != nil {
		return
	}
	mode := strings.ToLower(strings.TrimSpace(gov.FixBeforeFeature))
	featureScenarios := pathutil.UniqueSortedStrings(pathutil.ScenariosFromGlobs(item.AcceptanceAllow))
	if len(featureScenarios) == 0 {
		return
	}

	openItems := s.gatherOpenRemediationItems()

	// Gate (Tier 1): advise or block when feature work stacks on open fixes.
	if mode != "" && mode != FixBeforeFeatureOff {
		result := evaluateFixBeforeFeature(item.Kind, featureScenarios, openItems, mode)
		if result.blockingReason != "" {
			preflight.ForceableBlockingReasons = append(preflight.ForceableBlockingReasons, result.blockingReason)
		}
		if result.advisory != "" {
			preflight.Advisories = append(preflight.Advisories, result.advisory)
		}
	}

	// Discovery (Tier 2, opt-in): async-surface latent fixes for scenarios with
	// no known open remediation work. Never blocks this queue call.
	if gov.FixBeforeFeatureDiscovery {
		s.maybeTriggerDiscovery(featureScenarios, openItems)
	}
}
