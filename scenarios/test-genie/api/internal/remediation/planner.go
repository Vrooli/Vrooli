package remediation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Evidence is the normalized input to the pure planner. Adapters that read
// current or legacy run artifacts must make absence explicit through Degraded.
type Evidence struct {
	SourceExecutionID string
	SourceRunID       string
	Scenario          string
	TargetKind        string
	TargetID          string
	CompletedAt       time.Time
	Phases            []Phase
	Findings          []Finding
	DegradedReasons   []string
}

// BuildPlan normalizes and ranks evidence without performing I/O. It refuses
// to manufacture actionable work when the identity-bearing evidence is absent.
func BuildPlan(e Evidence) Plan {
	plan := Plan{
		SourceExecutionID: strings.TrimSpace(e.SourceExecutionID), SourceRunID: strings.TrimSpace(e.SourceRunID),
		Scenario: strings.TrimSpace(e.Scenario), TargetKind: strings.TrimSpace(e.TargetKind), TargetID: strings.TrimSpace(e.TargetID), CreatedAt: e.CompletedAt.UTC(), Phases: append([]Phase(nil), e.Phases...),
		DegradedReasons: normalizedIDs(e.DegradedReasons),
	}
	plan.TargetKind, plan.TargetID = targetIdentity(plan.TargetKind, plan.TargetID, plan.Scenario)
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = time.Now().UTC()
	}
	if plan.SourceExecutionID == "" {
		plan.DegradedReasons = append(plan.DegradedReasons, "source execution id is unavailable")
	}
	if plan.SourceRunID == "" {
		plan.DegradedReasons = append(plan.DegradedReasons, "source run artifact is unavailable")
	}
	if plan.Scenario == "" {
		plan.DegradedReasons = append(plan.DegradedReasons, "source scenario is unavailable")
	}
	phaseByName := make(map[string]Phase, len(plan.Phases))
	for _, phase := range plan.Phases {
		phaseByName[phase.Name] = phase
	}

	byID := make(map[string]*Finding, len(e.Findings))
	for _, raw := range e.Findings {
		raw.StableID = strings.TrimSpace(raw.StableID)
		if raw.StableID == "" {
			plan.DegradedReasons = append(plan.DegradedReasons, "finding without stable id")
			continue
		}
		raw.Locations = normalizedIDs(raw.Locations)
		raw.Domains = normalizedIDs(raw.Domains)
		if len(raw.Occurrences) == 0 {
			raw.Occurrences = []Occurrence{{Phase: raw.Phase, Locations: append([]string(nil), raw.Locations...)}}
		}
		if phase, ok := phaseByName[raw.Phase]; ok {
			raw.Gating = strings.EqualFold(raw.Class, "deterministic") && !strings.EqualFold(phase.ResultGating, "advisory")
		}
		if existing, ok := byID[raw.StableID]; ok {
			existing.Occurrences = append(existing.Occurrences, raw.Occurrences...)
			existing.Locations = normalizedIDs(append(existing.Locations, raw.Locations...))
			if severityRank(raw.Severity) > severityRank(existing.Severity) {
				existing.Severity = raw.Severity
			}
			existing.Gating = existing.Gating || raw.Gating
			continue
		}
		copy := raw
		byID[copy.StableID] = &copy
	}
	plan.DegradedReasons = normalizedIDs(plan.DegradedReasons)
	plan.Degraded = len(plan.DegradedReasons) > 0
	if plan.Degraded {
		return plan
	}
	for _, finding := range byID {
		plan.Findings = append(plan.Findings, *finding)
	}
	sort.Slice(plan.Findings, func(i, j int) bool { return findingLess(plan.Findings[i], plan.Findings[j]) })
	plan.Bundles = buildBundles(plan.Findings, phaseByName)
	return plan
}

func findingLess(a, b Finding) bool {
	if a.Gating != b.Gating {
		return a.Gating
	}
	if severityRank(a.Severity) != severityRank(b.Severity) {
		return severityRank(a.Severity) > severityRank(b.Severity)
	}
	if a.Class != b.Class {
		return a.Class == "deterministic"
	}
	return a.StableID < b.StableID
}

func severityRank(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "blocker":
		return 4
	case "error":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

func buildBundles(findings []Finding, phases map[string]Phase) []Bundle {
	groups := make(map[string][]Finding)
	for _, finding := range findings {
		key, reason := bundleKey(finding, phases[finding.Phase])
		groups[key+"\x1f"+reason] = append(groups[key+"\x1f"+reason], finding)
	}
	bundles := make([]Bundle, 0, len(groups))
	for compound, group := range groups {
		parts := strings.SplitN(compound, "\x1f", 2)
		ids, phaseSet := make([]string, 0, len(group)), map[string]struct{}{}
		gating := false
		for _, finding := range group {
			ids = append(ids, finding.StableID)
			phaseSet[finding.Phase] = struct{}{}
			gating = gating || finding.Gating
		}
		sort.Strings(ids)
		phaseNames := make([]string, 0, len(phaseSet))
		for name := range phaseSet {
			phaseNames = append(phaseNames, name)
		}
		sort.Strings(phaseNames)
		sum := sha256.Sum256([]byte(strings.Join(ids, "\x1f")))
		bundles = append(bundles, Bundle{ID: "bundle:" + hex.EncodeToString(sum[:8]), Reason: parts[1], FindingIDs: ids, PhaseNames: phaseNames, Gating: gating})
	}
	sort.Slice(bundles, func(i, j int) bool {
		if bundles[i].Gating != bundles[j].Gating {
			return bundles[i].Gating
		}
		return bundles[i].ID < bundles[j].ID
	})
	for i := range bundles {
		bundles[i].Rank = i + 1
	}
	return bundles
}

func bundleKey(f Finding, phase Phase) (string, string) {
	if len(f.Locations) > 0 {
		return "location:" + f.Locations[0], "shared location"
	}
	if phase.Provider != "" {
		return "provider:" + phase.Provider, "shared provider"
	}
	if f.Source != "" {
		return "source:" + f.Source, "shared finding source"
	}
	return "finding:" + f.StableID, "individual finding"
}

// ValidateSelection proves selectors refer only to the immutable source plan.
func ValidateSelection(plan Plan, ids []string) ([]string, error) {
	if plan.Degraded {
		return nil, fmt.Errorf("%w: source evidence is degraded", ErrInvalidSelector)
	}
	selected := normalizedIDs(ids)
	known := make(map[string]struct{}, len(plan.Findings))
	for _, finding := range plan.Findings {
		known[finding.StableID] = struct{}{}
	}
	for _, id := range selected {
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("%w: finding %q is not in source run", ErrInvalidSelector, id)
		}
	}
	return selected, nil
}

func ValidateRequirementSelection(plan Plan, ids []string) ([]string, error) {
	if plan.Degraded {
		return nil, fmt.Errorf("%w: source evidence is degraded", ErrInvalidSelector)
	}
	selected := normalizedIDs(ids)
	known := make(map[string]struct{}, len(plan.Requirements))
	for _, requirement := range plan.Requirements {
		known[requirement.ID] = struct{}{}
	}
	for _, id := range selected {
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("%w: requirement %q is not in source scenario evidence", ErrInvalidSelector, id)
		}
	}
	return selected, nil
}

// Compare computes a stable-ID delta for the job's selected source set. It is
// retained for callers that only have selectors; verification should prefer
// ComparePlan so changed severity and skipped applicability are recorded.
func Compare(selected []string, verified []Finding, verificationAvailable bool) FindingDelta {
	return ComparePlan(Plan{}, selected, verified, verificationAvailable, nil)
}

// ComparePlan compares a verification result with immutable source evidence.
// Selected findings from phases absent from the current planned run are skipped,
// not misreported as resolved. A stable ID that remains but changes severity is
// kept in remaining and separately attributed as a changed-severity finding.
func ComparePlan(source Plan, selected []string, verified []Finding, verificationAvailable bool, plannedPhases []string) FindingDelta {
	selected = normalizedIDs(selected)
	if !verificationAvailable {
		return FindingDelta{Unverifiable: selected}
	}
	planned := make(map[string]struct{}, len(plannedPhases))
	for _, phase := range plannedPhases {
		if phase = strings.TrimSpace(phase); phase != "" {
			planned[phase] = struct{}{}
		}
	}
	sourceByID := make(map[string]Finding, len(source.Findings))
	for _, finding := range source.Findings {
		if finding.StableID != "" {
			sourceByID[finding.StableID] = finding
		}
	}
	current := make(map[string]Finding, len(verified))
	for _, finding := range verified {
		if finding.StableID != "" {
			current[finding.StableID] = finding
		}
	}
	delta := FindingDelta{}
	selectedSet := make(map[string]struct{}, len(selected))
	for _, id := range selected {
		selectedSet[id] = struct{}{}
		if sourceFinding, ok := sourceByID[id]; ok && len(planned) > 0 {
			if _, applicable := planned[sourceFinding.Phase]; !applicable {
				delta.Skipped = append(delta.Skipped, id)
				continue
			}
		}
		if verificationFinding, ok := current[id]; ok {
			delta.Remaining = append(delta.Remaining, id)
			if sourceFinding, sourceKnown := sourceByID[id]; sourceKnown && severityRank(sourceFinding.Severity) != severityRank(verificationFinding.Severity) {
				delta.ChangedSeverity = append(delta.ChangedSeverity, id)
			}
		} else {
			delta.Resolved = append(delta.Resolved, id)
		}
	}
	for id := range current {
		if _, ok := selectedSet[id]; !ok {
			delta.New = append(delta.New, id)
		}
	}
	sort.Strings(delta.Resolved)
	sort.Strings(delta.Remaining)
	sort.Strings(delta.New)
	sort.Strings(delta.ChangedSeverity)
	sort.Strings(delta.Skipped)
	return delta
}

// CompareRequirements compares selected immutable requirement evidence with a
// fresh requirement snapshot. Missing or unavailable evidence remains
// unverifiable rather than silently resolving a requirement.
func CompareRequirements(source Plan, selected []string, verified []RequirementEvidence, verificationAvailable bool) RequirementDelta {
	selected = normalizedIDs(selected)
	if !verificationAvailable {
		return RequirementDelta{Unverifiable: selected}
	}
	current := make(map[string]RequirementEvidence, len(verified))
	for _, requirement := range verified {
		if requirement.ID != "" {
			current[requirement.ID] = requirement
		}
	}
	delta := RequirementDelta{}
	for _, id := range selected {
		requirement, ok := current[id]
		if !ok {
			delta.Unverifiable = append(delta.Unverifiable, id)
			continue
		}
		switch strings.ToLower(strings.TrimSpace(requirement.LiveStatus)) {
		case "passed", "pass", "satisfied":
			delta.Resolved = append(delta.Resolved, id)
		case "skipped", "not_applicable", "not-applicable":
			delta.Skipped = append(delta.Skipped, id)
		case "", "unknown", "unavailable", "not_run", "not-run":
			delta.Unverifiable = append(delta.Unverifiable, id)
		default:
			delta.Remaining = append(delta.Remaining, id)
		}
	}
	sort.Strings(delta.Resolved)
	sort.Strings(delta.Remaining)
	sort.Strings(delta.Skipped)
	sort.Strings(delta.Unverifiable)
	return delta
}
