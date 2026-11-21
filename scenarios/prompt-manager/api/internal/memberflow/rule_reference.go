package memberflow

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Generated rule-reference tables.
//
// docs/agent-system/OPERATING_GRAPHS.md and docs/agent-system/TOPICS_SCHEMA.md
// each carried a hand-written table of rule ids, severities, and meanings. Both
// restated the catalog, and both drifted from it silently: a rule could be
// added, deleted, or re-severitied with nothing forcing the documentation to
// follow. The tables are now rendered from the catalog and pinned by a test.

// RuleReferenceTarget is one generated block: a file, the marker pair that
// delimits the generated region, and the rule groups the block covers.
type RuleReferenceTarget struct {
	Path   string
	Marker string
	// Groups selects which catalogue entries belong in this block. An empty
	// slice means every group.
	Groups []RuleGroup
}

// RuleReferenceTargets is the full set of generated documentation blocks.
//
// PROSE_SCAN_TARGETS.md is deliberately absent: it carries a regex table, not a
// rule table, and its rows document matcher syntax the catalog does not model.
func RuleReferenceTargets(docsDir string) []RuleReferenceTarget {
	return []RuleReferenceTarget{
		{
			Path:   docsDir + "/agent-system/OPERATING_GRAPHS.md",
			Marker: "graph",
			Groups: []RuleGroup{
				OperatingRuleGroupEntity, OperatingRuleGroupEdgeTruth,
				OperatingRuleGroupCompleteness, OperatingRuleGroupDocs,
				OperatingRuleGroupCoherence, OperatingRuleGroupPrompt,
			},
		},
		{
			Path:   docsDir + "/agent-system/TOPICS_SCHEMA.md",
			Marker: "topic",
			Groups: []RuleGroup{OperatingRuleGroupTopic},
		},
	}
}

// RenderRuleReference renders the markdown table for one target's groups.
func RenderRuleReference(catalog RuleCatalog, groups []RuleGroup) string {
	allowed := make(map[RuleGroup]bool, len(groups))
	for _, group := range groups {
		allowed[group] = true
	}
	filtered := make(RuleCatalog, len(catalog))
	for id, entry := range catalog {
		if len(groups) == 0 || allowed[entry.Group] {
			filtered[id] = entry
		}
	}
	return filtered.Markdown()
}

var ruleReferenceBlock = regexp.MustCompile(
	`(?s)(<!-- BEGIN GENERATED: rule-catalog ([a-z-]+) -->\n)(.*?)(<!-- END GENERATED: rule-catalog [a-z-]+ -->)`)

// ApplyRuleReference returns the file content with its generated block replaced.
// The explanatory note above the table is regenerated too, so the instruction
// not to hand-edit cannot itself be hand-edited away.
func ApplyRuleReference(content, table string) (string, bool) {
	match := ruleReferenceBlock.FindStringSubmatchIndex(content)
	if match == nil {
		return content, false
	}
	const note = "_Generated from the validation rule catalog by `prompt-manager graph rules`. " +
		"Do not edit inside the markers; edit the catalog in " +
		"`scenarios/prompt-manager/api/memberflow` and regenerate._"
	body := note + "\n\n" + strings.TrimRight(table, "\n") + "\n"
	return content[:match[3]] + body + content[match[8]:], true
}

// GenerateRuleReference rewrites every generated block in place and reports the
// files it changed.
func GenerateRuleReference(docsDir string) ([]string, error) {
	catalog, err := DefaultRuleCatalog()
	if err != nil {
		return nil, err
	}
	var changed []string
	for _, target := range RuleReferenceTargets(docsDir) {
		raw, err := os.ReadFile(target.Path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", target.Path, err)
		}
		updated, ok := ApplyRuleReference(string(raw), RenderRuleReference(catalog, target.Groups))
		if !ok {
			return nil, fmt.Errorf("%s has no rule-catalog generated block", target.Path)
		}
		if updated != string(raw) {
			if err := os.WriteFile(target.Path, []byte(updated), 0o644); err != nil {
				return nil, fmt.Errorf("write %s: %w", target.Path, err)
			}
			changed = append(changed, target.Path)
		}
	}
	return changed, nil
}
