package phases

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"test-genie/internal/business"

	reqparsing "test-genie/internal/requirements/parsing"
	reqtypes "test-genie/internal/requirements/types"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// starterTemplateTag marks requirements that still carry the scaffolded
// starter registry (the scenario creator seeds requirements tagged this way;
// they are placeholders, not real requirements).
const starterTemplateTag = "template-starter"

// businessRuleFindingSpec maps a structural-validation rule name to its
// finding code and suggestion. Severity comes from the issue itself (the
// rules already classify error vs warning); codes are namespaced
// `business_*` so afid stable IDs never collide with other sources.
type businessRuleFindingSpec struct {
	Code       string
	Suggestion string
}

var businessRuleFindings = map[string]businessRuleFindingSpec{
	"duplicate_id": {
		Code:       "business_duplicate_req_id",
		Suggestion: "Give every requirement a unique ID; merge or rename the duplicate.",
	},
	"cycle_detection": {
		Code:       "business_import_cycle",
		Suggestion: "Break the cycle in the requirement hierarchy (children/depends_on must form a DAG).",
	},
	"orphaned_child": {
		Code:       "business_orphaned_ref",
		Suggestion: "Remove the dangling reference or add the missing requirement it points at.",
	},
	"invalid_reference": {
		Code:       "business_validation_ref_missing",
		Suggestion: "Point validation.ref at an existing test file (path relative to the scenario root), or remove the stale entry.",
	},
	"missing_id": {
		Code:       "business_req_missing_id",
		Suggestion: "Assign the requirement a stable ID (e.g. REQ-AREA-001).",
	},
	"missing_title": {
		Code:       "business_req_missing_title",
		Suggestion: "Give the requirement a short, behavior-describing title.",
	},
	"invalid_status": {
		Code:       "business_invalid_status",
		Suggestion: "Use one of the valid statuses: pending, planned, in_progress, complete, not_implemented.",
	},
}

// businessFindings converts the business phase's structured validation output
// into normalized ArchitectureFindings (source=BUSINESS). Two inputs:
// structural-validation Issues (typed via their Rule name) and registry-drift
// checks computed from the parsed module index (starter-template registries,
// requirements with no validation, prd_refs that match no PRD operational
// target). Severities are capped at ERROR — the business dimension is
// advisory in v1 and must never hard-stop a suite (no BLOCKER).
func businessFindings(scenario, scenarioDir string, r *business.RunResult) []*architecturev1.ArchitectureFinding {
	if r == nil {
		return nil
	}
	out := businessIssueFindings(scenario, scenarioDir, r.Issues)
	out = append(out, businessRegistryFindings(scenario, scenarioDir, r.Index)...)
	return out
}

// businessIssueFindings types the structural-validation issues. The
// requirement ID joins the code so each defect gets a distinct afid (the
// stable-ID hash is scenario+source+code+locations, and several issues can
// share one module file).
func businessIssueFindings(scenario, scenarioDir string, issues []reqtypes.ValidationIssue) []*architecturev1.ArchitectureFinding {
	out := make([]*architecturev1.ArchitectureFinding, 0, len(issues))
	for _, issue := range issues {
		spec, ok := businessRuleFindings[issue.Rule]
		if !ok {
			// Unmapped rule (a future addition): still emit, keyed by rule
			// name, so new rules are visible without a producer change.
			spec = businessRuleFindingSpec{
				Code:       "business_" + strings.TrimSpace(issue.Rule),
				Suggestion: "Fix the structural issue reported by the requirements validator.",
			}
		}
		code := spec.Code
		if id := strings.TrimSpace(issue.RequirementID); id != "" {
			code += ":" + id
		}
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_BUSINESS,
			code,
			string(issue.Severity),
			issue.Message,
			spec.Suggestion,
			nonEmptyLocations(relTo(scenarioDir, issue.FilePath)),
			nil,
		))
	}
	return out
}

// businessRegistryFindings runs the registry-drift checks over the parsed
// index: starter-template registries, requirements with empty validation[],
// and prd_refs that match no operational target in PRD.md. Requirements still
// tagged template-starter are excluded from the per-requirement checks — the
// single starter-template finding already says "replace this registry", and
// per-row findings on placeholder rows would be noise.
func businessRegistryFindings(scenario, scenarioDir string, index *reqparsing.ModuleIndex) []*architecturev1.ArchitectureFinding {
	if index == nil {
		return nil
	}

	var out []*architecturev1.ArchitectureFinding
	prdTargets, prdExists := prdOperationalTargets(scenarioDir)

	var starterFiles []string
	starterSeen := make(map[string]struct{})

	for _, module := range index.Modules {
		moduleLoc := relTo(scenarioDir, module.FilePath)
		for i := range module.Requirements {
			req := &module.Requirements[i]

			if hasTag(req.Tags, starterTemplateTag) {
				if _, ok := starterSeen[moduleLoc]; !ok {
					starterSeen[moduleLoc] = struct{}{}
					starterFiles = append(starterFiles, moduleLoc)
				}
				continue
			}

			id := strings.TrimSpace(req.ID)
			if id == "" {
				continue // missing_id rule already covers this
			}

			if len(req.Validations) == 0 {
				severity := "warning"
				if req.Criticality == reqtypes.CriticalityP0 {
					severity = "error"
				}
				out = append(out, newFinding(
					scenario,
					architecturev1.FindingSource_FINDING_SOURCE_BUSINESS,
					"business_req_no_validation:"+id,
					severity,
					fmt.Sprintf("Requirement %s (%s) declares no validation — nothing ties it to evidence.", id, displayCriticality(req.Criticality)),
					"Add a validation entry pointing at the test that proves this requirement (and tag the test with [REQ:"+id+"]).",
					nonEmptyLocations(moduleLoc),
					nil,
				))
			}

			prdRef := strings.TrimSpace(req.PRDRef)
			if prdExists && strings.HasPrefix(prdRef, "OT-") {
				if _, ok := prdTargets[prdRef]; !ok {
					out = append(out, newFinding(
						scenario,
						architecturev1.FindingSource_FINDING_SOURCE_BUSINESS,
						"business_prd_ref_unmatched:"+id,
						"warning",
						fmt.Sprintf("Requirement %s references %s, which matches no operational target in PRD.md.", id, prdRef),
						"Fix the prd_ref to an existing operational target, or add the missing target to PRD.md (Operational Targets section).",
						nonEmptyLocations(moduleLoc, "PRD.md"),
						nil,
					))
				}
			}
		}
	}

	if len(starterFiles) > 0 {
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_BUSINESS,
			"business_starter_template",
			"warning",
			fmt.Sprintf("The requirements registry still contains %d starter-template module(s) — it describes the scaffold, not this scenario.", len(starterFiles)),
			"Replace the template-starter requirements with this scenario's real requirements (derive them from PRD.md operational targets).",
			starterFiles,
			nil,
		))
	}

	return out
}

// otTokenPattern matches operational-target IDs as they appear in the
// canonical PRD template's checklists (`- [ ] OT-P0-001 | Title | ...`).
var otTokenPattern = regexp.MustCompile(`\bOT-[A-Za-z0-9]+-[A-Za-z0-9]+\b`)

// prdOperationalTargets extracts the set of operational-target IDs from the
// scenario's PRD.md. The second return is false when no PRD.md exists, in
// which case the prd_ref check is skipped entirely (PRD presence is the docs
// phase's concern, not this producer's).
func prdOperationalTargets(scenarioDir string) (map[string]struct{}, bool) {
	data, err := os.ReadFile(filepath.Join(scenarioDir, "PRD.md"))
	if err != nil {
		return nil, false
	}
	targets := make(map[string]struct{})
	for _, token := range otTokenPattern.FindAllString(string(data), -1) {
		targets[token] = struct{}{}
	}
	return targets, true
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if strings.EqualFold(strings.TrimSpace(t), want) {
			return true
		}
	}
	return false
}

func displayCriticality(c reqtypes.Criticality) string {
	if s := strings.TrimSpace(string(c)); s != "" {
		return s
	}
	return "no criticality"
}
