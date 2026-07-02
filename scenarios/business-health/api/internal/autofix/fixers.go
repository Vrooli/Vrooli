// Package autofix hosts business-health's deterministic fixers on the
// shared maturity-go registry — one fixer per `fix_class: auto` mapping in
// .vrooli/maturity.json. Every fixer is judgment-free: it repairs shape,
// never content (TODO markers stand in where product intent is needed).
package autofix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/maturity-go/autofix"
	intent "intent-go"
)

// NewRegistry returns the scenario's fixer registry.
func NewRegistry() *autofix.Registry {
	return autofix.NewRegistry(
		templateSectionsFixer(),
		otIDFormatFixer(),
		missingRegistryFixer(),
		readmeFixer(),
		invalidStatusFixer(),
		orphanOTFixer(),
	)
}

// ApplySequential applies rules ONE AT A TIME, re-previewing between
// writes, because several fixers here can target the same file (both PRD
// fixers edit PRD.md; the registry's bulk Apply would let a later
// same-file snapshot clobber an earlier one).
func ApplySequential(reg *autofix.Registry, root string, ruleIDs []string) ([]autofix.Candidate, error) {
	rules := ruleIDs
	if len(rules) == 0 {
		rules = RuleIDs()
	}
	var out []autofix.Candidate
	for _, rule := range rules {
		applied, err := reg.Apply(root, []string{rule})
		if err != nil {
			return out, err
		}
		out = append(out, applied...)
	}
	return out, nil
}

// RuleIDs lists the registered fixer rules in apply order.
func RuleIDs() []string {
	return []string{
		"prd_missing_requirements",
		"requirements_readme",
		"business_invalid_status",
		intent.CodeOTIDFormat,
		intent.CodeTemplateSections,
		intent.CodeOTOrphan,
	}
}

// --- prd_template_sections -------------------------------------------------

// templateSectionsFixer appends missing required sections (TODO-marked)
// to PRD.md. Section order follows the canonical template.
func templateSectionsFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		doc, err := intent.ExtractPRDDocument(root)
		if err != nil || !doc.Present {
			return nil, nil // presence is prd_missing_prd (manual/wizard)
		}
		missing := intent.CheckTemplateSections(doc, intent.DefaultPRDTemplate())
		if len(missing) == 0 {
			return nil, nil
		}
		path := filepath.Join(root, "PRD.md")
		before, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		after := strings.TrimRight(string(before), "\n") + "\n"
		count := 0
		for _, section := range intent.DefaultPRDTemplate() {
			if !section.Required {
				continue
			}
			if !hasSection(doc, section.Level, section.Title) {
				after += fmt.Sprintf("\n## %s\n\nTODO: %s.\n", section.Title, section.Description)
				count++
				for _, sub := range section.Subsections {
					if sub.Required {
						after += fmt.Sprintf("\n### %s\n\n- [ ] OT-%s-001 | TODO: outcome title | TODO: one-line description.\n", sub.Title, tierToken(sub.Title))
						count++
					}
				}
				continue
			}
			for _, sub := range section.Subsections {
				if sub.Required && !hasSection(doc, sub.Level, sub.Title) {
					after += fmt.Sprintf("\n### %s\n\n- [ ] OT-%s-001 | TODO: outcome title | TODO: one-line description.\n", sub.Title, tierToken(sub.Title))
					count++
				}
			}
		}
		if count == 0 {
			return nil, nil
		}
		return []autofix.Candidate{{
			RuleID:      intent.CodeTemplateSections,
			FilePath:    path,
			Description: fmt.Sprintf("Scaffold %d missing required section(s) with TODO-marked bodies", count),
			Before:      string(before),
			After:       after,
		}}, nil
	}
	return autofix.Fixer{
		RuleID:  intent.CodeTemplateSections,
		Preview: preview,
		CanFix:  func(root, _ string) bool { return previewable(preview, root) },
	}
}

func hasSection(doc intent.PRDDocument, level int, title string) bool {
	want := intent.NormalizeSectionTitle(title)
	for _, s := range doc.Sections {
		if s.Level == level && intent.NormalizeSectionTitle(s.Title) == want {
			return true
		}
	}
	return false
}

func tierToken(title string) string {
	for _, tier := range []string{"P0", "P1", "P2"} {
		if strings.Contains(title, tier) {
			return tier
		}
	}
	return "P0"
}

// --- prd_ot_id_format --------------------------------------------------------

var otLineIDPattern = regexp.MustCompile(`^(\s*- \[[ xX]\]\s*)(\S+)(.*)$`)

// otIDFormatFixer normalizes OT ids in PRD.md to canonical form and
// rewrites matching prd_ref values in requirements modules so the join
// stays closed.
func otIDFormatFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		doc, err := intent.ExtractPRDDocument(root)
		if err != nil || !doc.Present {
			return nil, nil
		}
		renames := map[string]string{}
		for _, ot := range doc.Targets {
			if canonical := intent.CanonicalOTID(ot.RawID); canonical != ot.RawID {
				renames[ot.RawID] = canonical
			}
		}
		if len(renames) == 0 {
			return nil, nil
		}
		var out []autofix.Candidate
		prdPath := filepath.Join(root, "PRD.md")
		before, err := os.ReadFile(prdPath)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(before), "\n")
		for i, line := range lines {
			m := otLineIDPattern.FindStringSubmatch(line)
			if len(m) != 4 {
				continue
			}
			if canonical, ok := renames[m[2]]; ok {
				lines[i] = m[1] + canonical + m[3]
			}
		}
		after := strings.Join(lines, "\n")
		if after != string(before) {
			out = append(out, autofix.Candidate{
				RuleID:      intent.CodeOTIDFormat,
				FilePath:    prdPath,
				Description: fmt.Sprintf("Normalize %d operational-target id(s) to OT-P<tier>-NNN", len(renames)),
				Before:      string(before),
				After:       after,
			})
		}
		// Rewrite matching prd_refs (string-level: prd_ref values are exact
		// id tokens, so a quoted replace is precise).
		reg, err := intent.ExtractRequirementsRegistry(root)
		if err == nil && reg.Present {
			for _, module := range reg.Modules {
				path := filepath.Join(root, filepath.FromSlash(module.Path))
				data, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				moduleAfter := string(data)
				for raw, canonical := range renames {
					moduleAfter = strings.ReplaceAll(moduleAfter, `"prd_ref": "`+raw+`"`, `"prd_ref": "`+canonical+`"`)
					moduleAfter = strings.ReplaceAll(moduleAfter, `"prd_ref":"`+raw+`"`, `"prd_ref":"`+canonical+`"`)
				}
				if moduleAfter != string(data) {
					out = append(out, autofix.Candidate{
						RuleID:      intent.CodeOTIDFormat,
						FilePath:    path,
						Description: "Repoint prd_ref values at the normalized OT ids",
						Before:      string(data),
						After:       moduleAfter,
					})
				}
			}
		}
		return out, nil
	}
	return autofix.Fixer{
		RuleID:  intent.CodeOTIDFormat,
		Preview: preview,
		CanFix:  func(root, _ string) bool { return previewable(preview, root) },
	}
}

// --- prd_missing_requirements ------------------------------------------------

const minimalIndex = `{
  "_metadata": {
    "description": "Parent registry linking operational targets to technical requirements",
    "auto_sync_enabled": true,
    "schema_version": "1.0.0"
  },
  "imports": []
}
`

// missingRegistryFixer creates a minimal valid registry when
// requirements/index.json is absent.
func missingRegistryFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		indexPath := filepath.Join(root, "requirements", "index.json")
		if _, err := os.Stat(indexPath); err == nil {
			return nil, nil
		}
		out := []autofix.Candidate{{
			RuleID:      "prd_missing_requirements",
			FilePath:    indexPath,
			Description: "Create a minimal valid requirements registry",
			After:       minimalIndex,
		}}
		readmePath := filepath.Join(root, "requirements", "README.md")
		if _, err := os.Stat(readmePath); err != nil {
			out = append(out, autofix.Candidate{
				RuleID:      "prd_missing_requirements",
				FilePath:    readmePath,
				Description: "Seed the canonical requirements README",
				After:       canonicalReadme,
			})
		}
		return out, nil
	}
	return autofix.Fixer{
		RuleID:  "prd_missing_requirements",
		Preview: preview,
		CanFix:  func(root, _ string) bool { return previewable(preview, root) },
	}
}

// --- requirements_readme -------------------------------------------------------

const canonicalReadme = `# Requirements

Requirement modules live here, one folder per group of operational
targets. Every requirement links back to a PRD operational target via
` + "`prd_ref`" + ` and carries at least one validation entry pointing at its
proof.

- Statuses are earned, not asserted: auto-sync updates them from
  ` + "`[REQ:ID]`" + `-tagged test results on comprehensive suite runs.
- Replace scaffolded manual validation stubs with test-typed entries
  (a ` + "`ref`" + ` to the test file plus the ` + "`[REQ:ID]`" + ` tag) as behavior lands.
- Validate with ` + "`business-health validate scenario <scenario>`" + `; inspect
  traceability with ` + "`business-health matrix show <scenario>`" + `.
`

var readmePhrases = []string{"operational target", "auto-sync", "validation"}

// readmeFixer restores the canonical README when it is missing or has
// drifted away from explaining the registry contract. Existing content is
// preserved below the canonical block.
func readmeFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		if _, err := os.Stat(filepath.Join(root, "requirements", "index.json")); err != nil {
			return nil, nil // whole-registry creation is prd_missing_requirements
		}
		path := filepath.Join(root, "requirements", "README.md")
		before, err := os.ReadFile(path)
		if err != nil {
			return []autofix.Candidate{{
				RuleID:      "requirements_readme",
				FilePath:    path,
				Description: "Restore the canonical requirements README",
				After:       canonicalReadme,
			}}, nil
		}
		lower := strings.ToLower(string(before))
		missing := false
		for _, phrase := range readmePhrases {
			if !strings.Contains(lower, phrase) {
				missing = true
			}
		}
		if !missing {
			return nil, nil
		}
		after := canonicalReadme + "\n---\n\n<!-- Prior content preserved below by the requirements_readme fixer. -->\n\n" + string(before)
		return []autofix.Candidate{{
			RuleID:      "requirements_readme",
			FilePath:    path,
			Description: "Restore the canonical registry contract explanation (prior content preserved below)",
			Before:      string(before),
			After:       after,
		}}, nil
	}
	return autofix.Fixer{
		RuleID:  "requirements_readme",
		Preview: preview,
		CanFix:  func(root, _ string) bool { return previewable(preview, root) },
	}
}

// --- business_invalid_status ----------------------------------------------------

// statusAliases are the recognizable variants the fixer normalizes.
// Unrecognizable statuses are left alone (the finding stays, a human
// picks the honest value).
var statusAliases = map[string]string{
	"completed":       "complete",
	"done":            "complete",
	"implemented":     "complete",
	"inprogress":      "in_progress",
	"in-progress":     "in_progress",
	"wip":             "in_progress",
	"draft":           "planned",
	"notimplemented":  "not_implemented",
	"not-implemented": "not_implemented",
	"todo":            "pending",
}

// invalidStatusFixer normalizes requirement status aliases at the JSON
// string level, scoped to requirement `status` values only — a targeted
// replace per offending record keeps validation-entry statuses (a
// different vocabulary where e.g. "implemented" is valid) untouched.
func invalidStatusFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		reg, err := intent.ExtractRequirementsRegistry(root)
		if err != nil || !reg.Present {
			return nil, nil
		}
		var out []autofix.Candidate
		for _, module := range reg.Modules {
			var renames []string
			for _, r := range module.Requirements {
				lower := strings.ToLower(strings.TrimSpace(r.Status))
				if canonical, ok := statusAliases[lower]; ok && r.Status != canonical {
					renames = append(renames, r.Status, canonical)
				}
			}
			if len(renames) == 0 {
				continue
			}
			path := filepath.Join(root, filepath.FromSlash(module.Path))
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			after := string(data)
			for i := 0; i < len(renames); i += 2 {
				after = replaceRequirementStatus(after, renames[i], renames[i+1])
			}
			if after != string(data) {
				out = append(out, autofix.Candidate{
					RuleID:      "business_invalid_status",
					FilePath:    path,
					Description: fmt.Sprintf("Normalize %d recognizable status alias(es) to the canonical vocabulary", len(renames)/2),
					Before:      string(data),
					After:       after,
				})
			}
		}
		return out, nil
	}
	return autofix.Fixer{
		RuleID:  "business_invalid_status",
		Preview: preview,
		CanFix:  func(root, _ string) bool { return previewable(preview, root) },
	}
}

// replaceRequirementStatus rewrites `"status": "<alias>"` occurrences that
// belong to requirement records. Validation entries live inside a
// "validation" array whose entries also carry "type" — the requirement
// status line is the one NOT preceded by a validation-context key on the
// same object. A precise structural pass would need order-preserving JSON;
// this targeted regex requires the alias to be an invalid REQUIREMENT
// status that is also not a valid VALIDATION status, or replaces only when
// unambiguous.
func replaceRequirementStatus(content, alias, canonical string) string {
	// "implemented" is valid for validations — never blanket-replace it.
	validValidationStatuses := map[string]struct{}{"not_implemented": {}, "planned": {}, "implemented": {}, "failing": {}}
	if _, validForValidation := validValidationStatuses[strings.ToLower(alias)]; validForValidation {
		// Scoped pass: only replace status lines NOT inside validation
		// entries. Cheap structural heuristic: validation entries are
		// indented deeper (they sit inside the validation array). Compare
		// indentation of the status line against the record's "id" lines.
		return replaceStatusOutsideValidation(content, alias, canonical)
	}
	for _, spacing := range []string{`"status": "`, `"status":"`} {
		content = strings.ReplaceAll(content, spacing+alias+`"`, spacing+canonical+`"`)
	}
	return content
}

// replaceStatusOutsideValidation replaces status values only on lines whose
// indentation matches requirement-level keys (shallower than validation
// entries). JSON written by this repo's tooling is consistently indented,
// which makes the depth heuristic reliable; hand-mangled files simply get
// no replacement (the finding remains for a human).
func replaceStatusOutsideValidation(content, alias, canonical string) string {
	lines := strings.Split(content, "\n")
	// Find the indentation of requirement-level "id" keys.
	idIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, `"id"`) {
			idIndent = len(line) - len(trimmed)
			break
		}
	}
	if idIndent < 0 {
		return content
	}
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		indent := len(line) - len(trimmed)
		if indent != idIndent || !strings.HasPrefix(trimmed, `"status"`) {
			continue
		}
		for _, spacing := range []string{`"status": "`, `"status":"`} {
			lines[i] = strings.Replace(lines[i], spacing+alias+`"`, spacing+canonical+`"`, 1)
		}
	}
	return strings.Join(lines, "\n")
}

// --- intent.ot_orphan --------------------------------------------------------------

const orphanModuleSlug = "90-uncovered-targets"

// orphanOTFixer scaffolds one stub requirement (status planned, no fake
// validation evidence — a manual TODO stub) per orphaned operational
// target, in a dedicated module, and declares it in index.json imports.
func orphanOTFixer() autofix.Fixer {
	preview := func(root string) ([]autofix.Candidate, error) {
		doc, err := intent.ExtractPRDDocument(root)
		if err != nil || !doc.Present {
			return nil, nil
		}
		reg, err := intent.ExtractRequirementsRegistry(root)
		if err != nil || !reg.Present {
			return nil, nil
		}
		covered := map[string]struct{}{}
		for _, r := range reg.Requirements() {
			covered[intent.CanonicalOTID(r.PRDRef)] = struct{}{}
		}
		var orphans []intent.OperationalTarget
		for _, ot := range doc.Targets {
			if _, ok := covered[ot.ID]; !ok {
				orphans = append(orphans, ot)
			}
		}
		if len(orphans) == 0 {
			return nil, nil
		}
		sort.Slice(orphans, func(i, j int) bool { return orphans[i].ID < orphans[j].ID })

		prefix := derivePrefix(filepath.Base(root))
		reqs := make([]map[string]any, 0, len(orphans))
		for _, ot := range orphans {
			reqs = append(reqs, map[string]any{
				"id":          prefix + "-" + strings.TrimPrefix(ot.ID, "OT-"),
				"title":       stubTitle(ot),
				"status":      "planned",
				"prd_ref":     ot.ID,
				"criticality": ot.Tier,
				"description": "Scaffolded stub for an operational target that had no requirement; replace with real falsifiable claims.",
				"validation": []map[string]any{{
					"type":   "manual",
					"status": "planned",
					"notes":  "Scaffolded stub: define the real validation for " + ot.ID + ".",
				}},
			})
		}
		modulePath := filepath.Join(root, "requirements", orphanModuleSlug, "module.json")
		moduleBefore := ""
		if data, err := os.ReadFile(modulePath); err == nil {
			moduleBefore = string(data)
		}
		module := map[string]any{
			"_metadata": map[string]any{
				"module":         "uncovered-targets",
				"description":    "Stub requirements scaffolded for operational targets that had no coverage (intent.ot_orphan fixer).",
				"schema_version": "1.0.0",
			},
			"requirements": reqs,
		}
		moduleJSON, err := json.MarshalIndent(module, "", "  ")
		if err != nil {
			return nil, err
		}
		out := []autofix.Candidate{{
			RuleID:      intent.CodeOTOrphan,
			FilePath:    modulePath,
			Description: fmt.Sprintf("Scaffold %d stub requirement(s) for uncovered operational targets", len(orphans)),
			Before:      moduleBefore,
			After:       string(moduleJSON) + "\n",
		}}

		indexPath := filepath.Join(root, "requirements", "index.json")
		indexData, err := os.ReadFile(indexPath)
		if err == nil && !strings.Contains(string(indexData), orphanModuleSlug+"/module.json") {
			var index map[string]any
			if err := json.Unmarshal(indexData, &index); err == nil {
				imports, _ := index["imports"].([]any)
				imports = append(imports, orphanModuleSlug+"/module.json")
				index["imports"] = imports
				indexJSON, err := json.MarshalIndent(index, "", "  ")
				if err == nil {
					out = append(out, autofix.Candidate{
						RuleID:      intent.CodeOTOrphan,
						FilePath:    indexPath,
						Description: "Declare the uncovered-targets module in index imports",
						Before:      string(indexData),
						After:       string(indexJSON) + "\n",
					})
				}
			}
		}
		return out, nil
	}
	return autofix.Fixer{
		RuleID:  intent.CodeOTOrphan,
		Preview: preview,
		CanFix:  func(root, _ string) bool { return previewable(preview, root) },
	}
}

func stubTitle(ot intent.OperationalTarget) string {
	if strings.TrimSpace(ot.Title) != "" {
		return strings.TrimSpace(ot.Title)
	}
	return "Cover " + ot.ID
}

func derivePrefix(slug string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(slug) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9' && b.Len() > 0) {
			b.WriteRune(r)
		}
		if b.Len() == 8 {
			break
		}
	}
	if b.Len() < 2 {
		return "REQ"
	}
	return b.String()
}

func previewable(preview func(string) ([]autofix.Candidate, error), root string) bool {
	candidates, err := preview(root)
	return err == nil && len(candidates) > 0
}
