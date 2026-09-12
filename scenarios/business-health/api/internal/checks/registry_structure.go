package checks

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"business-health/internal/extraction"

	intent "intent-go"
)

// starterTemplateTag marks requirements that still carry the scaffolded
// starter registry.
const starterTemplateTag = "template-starter"

// validStatuses is the requirement status vocabulary (matches the native
// test-genie validator and the requirement schema doc).
var validStatuses = map[string]struct{}{
	"pending": {}, "planned": {}, "in_progress": {}, "complete": {}, "not_implemented": {},
}

// registryPresenceCheck emits prd_missing_requirements when the scenario
// has no parseable requirements/index.json, and surfaces module parse
// errors when the registry exists but files are broken.
type registryPresenceCheck struct{}

func (registryPresenceCheck) Name() string { return "registry-presence" }

func (registryPresenceCheck) Run(_ context.Context, c extraction.Contract) []intent.Finding {
	if !c.Registry.Present {
		return []intent.Finding{{
			Code:       "prd_missing_requirements",
			Severity:   "error",
			Message:    "requirements/index.json is missing — the scenario makes no machine-checkable requirement claims.",
			Suggestion: "Run `business-health fix " + c.Scenario + " --apply` for a minimal registry, or the wizard to derive one from the PRD.",
			Locations:  []string{"requirements/index.json"},
			Provenance: "business-health",
		}}
	}
	var out []intent.Finding
	for _, pe := range c.Registry.ParseErrors {
		out = append(out, intent.Finding{
			Code:       "business_registry_unparseable",
			Severity:   "error",
			Message:    fmt.Sprintf("Registry file %s cannot be parsed: %s", pe.Path, pe.Err),
			Suggestion: "Fix the JSON (or the dangling import in index.json) so the registry loads.",
			Locations:  []string{pe.Path},
			Provenance: "business-health",
		})
	}
	return out
}

// registryStructureCheck re-hosts the native structural-integrity rules:
// duplicate IDs, import cycles, orphaned children/depends_on refs, and
// missing IDs. Codes and `:ID` suffixing match the native business phase
// so afids stay stable through the delegation cutover.
type registryStructureCheck struct{}

func (registryStructureCheck) Name() string { return "registry-structure" }

func (registryStructureCheck) Run(_ context.Context, c extraction.Contract) []intent.Finding {
	if !c.Registry.Present {
		return nil
	}
	reqs := c.Registry.Requirements()
	var out []intent.Finding
	out = append(out, duplicateIDFindings(reqs)...)
	out = append(out, missingIDFindings(reqs)...)
	out = append(out, orphanedRefFindings(reqs)...)
	out = append(out, cycleFindings(reqs)...)
	return out
}

func duplicateIDFindings(reqs []intent.RegistryRequirement) []intent.Finding {
	byID := make(map[string][]intent.RegistryRequirement)
	for _, r := range reqs {
		if r.ID == "" {
			continue
		}
		key := strings.ToLower(r.ID)
		byID[key] = append(byID[key], r)
	}
	keys := make([]string, 0, len(byID))
	for k, group := range byID {
		if len(group) > 1 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var out []intent.Finding
	for _, k := range keys {
		group := byID[k]
		locations := make([]string, 0, len(group))
		for _, r := range group {
			locations = append(locations, r.Module)
		}
		out = append(out, intent.Finding{
			Code:       "business_duplicate_req_id:" + group[0].ID,
			Severity:   "error",
			Message:    fmt.Sprintf("Requirement ID %q is declared %d times (IDs are case-insensitive).", group[0].ID, len(group)),
			Suggestion: "Give every requirement a unique ID; merge or rename the duplicate.",
			Locations:  dedupeStrings(locations),
			ClaimID:    group[0].ID,
			Provenance: "business-health",
		})
	}
	return out
}

func missingIDFindings(reqs []intent.RegistryRequirement) []intent.Finding {
	var out []intent.Finding
	for _, r := range reqs {
		if r.ID != "" {
			continue
		}
		out = append(out, intent.Finding{
			Code:       "business_req_missing_id",
			Severity:   "error",
			Message:    fmt.Sprintf("Requirement #%d in %s has no ID.", r.Index+1, r.Module),
			Suggestion: "Assign the requirement a stable ID (e.g. REQ-AREA-001).",
			Locations:  []string{r.Module},
			Provenance: "business-health",
		})
	}
	return out
}

func orphanedRefFindings(reqs []intent.RegistryRequirement) []intent.Finding {
	ids := make(map[string]struct{}, len(reqs))
	for _, r := range reqs {
		if r.ID != "" {
			ids[strings.ToLower(r.ID)] = struct{}{}
		}
	}
	var out []intent.Finding
	emit := func(r intent.RegistryRequirement, ref, kind, severity string) {
		out = append(out, intent.Finding{
			Code:       "business_orphaned_ref:" + r.ID,
			Severity:   severity,
			Message:    fmt.Sprintf("Requirement %s has a %s reference to %q, which does not exist.", r.ID, kind, ref),
			Suggestion: "Remove the dangling reference or add the missing requirement it points at.",
			Locations:  []string{r.Module},
			ClaimID:    r.ID,
			RelatedID:  ref,
			Provenance: "business-health",
		})
	}
	for _, r := range reqs {
		if r.ID == "" {
			continue
		}
		for _, child := range r.Children {
			if _, ok := ids[strings.ToLower(strings.TrimSpace(child))]; !ok && strings.TrimSpace(child) != "" {
				emit(r, child, "children", "error")
			}
		}
		for _, dep := range r.DependsOn {
			if _, ok := ids[strings.ToLower(strings.TrimSpace(dep))]; !ok && strings.TrimSpace(dep) != "" {
				emit(r, dep, "depends_on", "warning")
			}
		}
	}
	return out
}

// cycleFindings detects cycles in the children/depends_on graph via
// iterative DFS with three-color marking.
func cycleFindings(reqs []intent.RegistryRequirement) []intent.Finding {
	adjacent := make(map[string][]string)
	module := make(map[string]string)
	for _, r := range reqs {
		if r.ID == "" {
			continue
		}
		id := strings.ToLower(r.ID)
		module[id] = r.Module
		for _, edge := range append(append([]string{}, r.Children...), r.DependsOn...) {
			edge = strings.ToLower(strings.TrimSpace(edge))
			if edge != "" {
				adjacent[id] = append(adjacent[id], edge)
			}
		}
	}

	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int)
	var cyclePath []string

	var visit func(node string, stack []string) bool
	visit = func(node string, stack []string) bool {
		color[node] = gray
		stack = append(stack, node)
		for _, next := range adjacent[node] {
			switch color[next] {
			case gray:
				// Found a cycle: slice the stack from next's position.
				for i, n := range stack {
					if n == next {
						cyclePath = append(append([]string{}, stack[i:]...), next)
						return true
					}
				}
				cyclePath = []string{next, node, next}
				return true
			case white:
				if visit(next, stack) {
					return true
				}
			}
		}
		color[node] = black
		return false
	}

	nodes := make([]string, 0, len(adjacent))
	for node := range adjacent {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if color[node] == white && visit(node, nil) {
			break
		}
	}
	if len(cyclePath) == 0 {
		return nil
	}
	locations := make([]string, 0, len(cyclePath))
	for _, id := range cyclePath {
		if m, ok := module[id]; ok {
			locations = append(locations, m)
		}
	}
	return []intent.Finding{{
		Code:       "business_import_cycle:" + strings.ToUpper(cyclePath[0]),
		Severity:   "error",
		Message:    fmt.Sprintf("Requirement references form a cycle: %s.", strings.ToUpper(strings.Join(cyclePath, " -> "))),
		Suggestion: "Break the cycle in the requirement hierarchy (children/depends_on must form a DAG).",
		Locations:  dedupeStrings(locations),
		ClaimID:    strings.ToUpper(cyclePath[0]),
		Provenance: "business-health",
	}}
}

// registryQualityCheck re-hosts the quality-level rules: missing titles,
// invalid status vocabulary, starter-template residue, and requirements
// with no validation (P0 escalates to error). Starter-tagged rows are
// excluded from per-row checks — the single starter finding already says
// "replace this registry".
type registryQualityCheck struct{}

func (registryQualityCheck) Name() string { return "registry-quality" }

func (registryQualityCheck) Run(_ context.Context, c extraction.Contract) []intent.Finding {
	if !c.Registry.Present {
		return nil
	}
	var out []intent.Finding
	var starterFiles []string
	starterSeen := make(map[string]struct{})

	for _, r := range c.Registry.Requirements() {
		if r.HasTag(starterTemplateTag) {
			if _, ok := starterSeen[r.Module]; !ok {
				starterSeen[r.Module] = struct{}{}
				starterFiles = append(starterFiles, r.Module)
			}
			continue
		}
		if r.ID == "" {
			continue // business_req_missing_id covers this
		}
		if r.Title == "" {
			out = append(out, intent.Finding{
				Code:       "business_req_missing_title:" + r.ID,
				Severity:   "warning",
				Message:    fmt.Sprintf("Requirement %s has no title.", r.ID),
				Suggestion: "Give the requirement a short, behavior-describing title.",
				Locations:  []string{r.Module},
				ClaimID:    r.ID,
				Provenance: "business-health",
			})
		}
		if r.Status != "" {
			if _, ok := validStatuses[r.Status]; !ok {
				out = append(out, intent.Finding{
					Code:       "business_invalid_status:" + r.ID,
					Severity:   "warning",
					Message:    fmt.Sprintf("Requirement %s has status %q, outside the vocabulary.", r.ID, r.Status),
					Suggestion: "Use one of the valid statuses: pending, planned, in_progress, complete, not_implemented.",
					Locations:  []string{r.Module},
					ClaimID:    r.ID,
					Provenance: "business-health",
				})
			}
		}
		if len(r.Validations) == 0 {
			severity := "warning"
			if strings.EqualFold(r.Criticality, "P0") {
				severity = "error"
			}
			out = append(out, intent.Finding{
				Code:       "business_req_no_validation:" + r.ID,
				Severity:   severity,
				Message:    fmt.Sprintf("Requirement %s (%s) declares no validation — nothing ties it to evidence.", r.ID, displayCriticality(r.Criticality)),
				Suggestion: "Add a validation entry pointing at the test that proves this requirement (and tag the test with [REQ:" + r.ID + "]).",
				Locations:  []string{r.Module},
				ClaimID:    r.ID,
				Provenance: "business-health",
			})
		}
	}

	if len(starterFiles) > 0 {
		sort.Strings(starterFiles)
		out = append(out, intent.Finding{
			Code:       "business_starter_template",
			Severity:   "warning",
			Message:    fmt.Sprintf("The requirements registry still contains %d starter-template module(s) — it describes the scaffold, not this scenario.", len(starterFiles)),
			Suggestion: "Replace the template-starter requirements with this scenario's real requirements (derive them from PRD.md operational targets).",
			Locations:  starterFiles,
			Provenance: "business-health",
		})
	}
	return out
}

func displayCriticality(c string) string {
	if s := strings.TrimSpace(c); s != "" {
		return s
	}
	return "no criticality"
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
