package memberflow

import (
	"fmt"
	"sort"
	"strings"
)

// RulePass names the validation entry point a rule executes in. Every rule
// belongs to exactly one pass, and the pass is derived from the rule's group
// by rulePassForGroup rather than declared per rule, so the three entry points
// cannot disagree about which rules belong to them.
type RulePass string

const (
	RulePassTopic RulePass = "topic"
	RulePassGraph RulePass = "graph"
	RulePassModel RulePass = "model"
)

// rulePassForGroup is the single mapping from rule group to validation pass.
// It is deliberately exhaustive with no permissive default: a group that has
// not been assigned a pass fails registration loudly instead of silently
// executing against a context its rules were never written to read.
func rulePassForGroup(group RuleGroup) (RulePass, error) {
	switch group {
	case OperatingRuleGroupTopic:
		return RulePassTopic, nil
	case OperatingRuleGroupEntity, OperatingRuleGroupEdgeTruth,
		OperatingRuleGroupCompleteness, OperatingRuleGroupDocs,
		OperatingRuleGroupCoherence, OperatingRuleGroupPrompt:
		return RulePassGraph, nil
	case OperatingRuleGroupPlanOfRecord,
		OperatingModelRuleGroupStructure, OperatingModelRuleGroupDecision,
		OperatingModelRuleGroupExternalInput, OperatingModelRuleGroupOutput,
		OperatingModelRuleGroupFeedback, OperatingModelRuleGroupGap,
		OperatingModelRuleGroupAdoption, OperatingModelRuleGroupDiscoverability:
		return RulePassModel, nil
	default:
		return "", fmt.Errorf("rule group %q is not assigned to a validation pass", group)
	}
}

// RuleRegistry is the authoritative registration surface for validation rules.
// Registration validates identity uniqueness and pass assignment up front so
// rule execution never depends on an accidental duplicate identifier or on a
// per-call-site filter.
type RuleRegistry struct {
	rules  []Rule
	byID   map[string]Rule
	byPass map[RulePass][]Rule
}

// RuleCatalogEntry is the operator-facing identity for a validation rule.
// A rule without this metadata is not actionable enough to be registered by
// catalog-enforced construction.
type RuleCatalogEntry struct {
	ID          string
	Group       RuleGroup
	Severity    Severity
	Description string
	Actuator    string
}

type RuleCatalog map[string]RuleCatalogEntry

func NewRuleCatalog(entries ...RuleCatalogEntry) (RuleCatalog, error) {
	catalog := make(RuleCatalog, len(entries))
	for _, entry := range entries {
		if entry.ID == "" || entry.Group == "" || entry.Severity == "" || entry.Description == "" || entry.Actuator == "" {
			return nil, fmt.Errorf("rule catalog entry must include id, group, severity, description, and actuator")
		}
		if _, exists := catalog[entry.ID]; exists {
			return nil, fmt.Errorf("rule catalog contains duplicate identifier %q", entry.ID)
		}
		catalog[entry.ID] = entry
	}
	return catalog, nil
}

func NewRuleRegistry(rules ...Rule) (*RuleRegistry, error) {
	return newRuleRegistry(nil, rules...)
}

// NewRuleRegistryWithCatalog requires every rule to have exactly one
// operator-facing catalog entry and keeps metadata aligned with execution.
func NewRuleRegistryWithCatalog(catalog RuleCatalog, rules ...Rule) (*RuleRegistry, error) {
	return newRuleRegistry(catalog, rules...)
}

func newRuleRegistry(catalog RuleCatalog, rules ...Rule) (*RuleRegistry, error) {
	registry := &RuleRegistry{
		byID:   make(map[string]Rule, len(rules)),
		byPass: make(map[RulePass][]Rule),
	}
	for _, rule := range rules {
		if rule == nil {
			return nil, fmt.Errorf("validation rule registry contains nil rule")
		}
		id := rule.ID()
		if id == "" {
			return nil, fmt.Errorf("validation rule registry contains unnamed rule")
		}
		if _, exists := registry.byID[id]; exists {
			return nil, fmt.Errorf("validation rule registry contains duplicate identifier %q", id)
		}
		pass, err := rulePassForGroup(rule.Group())
		if err != nil {
			return nil, fmt.Errorf("validation rule %q: %w", id, err)
		}
		// The topic pass reports through the richer Finding shape, so a rule
		// routed there must be able to produce one. Catching this at
		// registration keeps the pass from silently skipping the rule.
		if pass == RulePassTopic {
			if _, ok := rule.(findingRule); !ok {
				return nil, fmt.Errorf("validation rule %q is in the topic pass but does not implement findingRule", id)
			}
		}
		registry.byPass[pass] = append(registry.byPass[pass], rule)
		if catalog != nil {
			entry, ok := catalog[id]
			if !ok {
				return nil, fmt.Errorf("validation rule %q is missing from the rule catalog", id)
			}
			if entry.Group != rule.Group() || entry.Severity != rule.DefaultSeverity() {
				return nil, fmt.Errorf("rule catalog metadata for %q disagrees with its registration", id)
			}
		}
		registry.rules = append(registry.rules, rule)
		registry.byID[id] = rule
	}
	// Ranging a nil catalog is a no-op, which is exactly the unenforced case.
	for id := range catalog {
		if _, ok := registry.byID[id]; !ok {
			return nil, fmt.Errorf("rule catalog entry %q has no registered rule", id)
		}
	}
	return registry, nil
}

func (r *RuleRegistry) Rules() []Rule {
	return append([]Rule(nil), r.rules...)
}

// RulesForPass returns the rules registered to one validation pass, in
// registration order. Each entry point asks for its own pass instead of
// re-deriving membership from the rule's group.
func (r *RuleRegistry) RulesForPass(pass RulePass) []Rule {
	return append([]Rule(nil), r.byPass[pass]...)
}

func DefaultRuleRegistry() (*RuleRegistry, error) {
	rules := append(DefaultOperatingGraphRules(), DefaultOperatingModelRules()...)
	rules = append(rules, DefaultTopicRules()...)
	rules = append(rules, DefaultPlanOfRecordRules()...)
	catalog, err := DefaultRuleCatalog()
	if err != nil {
		return nil, err
	}
	return NewRuleRegistryWithCatalog(catalog, rules...)
}

// DefaultRuleCatalog is the single catalog used by every shipped validation
// family. Rule metadata is intentionally assembled before registry creation,
// then verified bidirectionally by NewRuleRegistryWithCatalog. This keeps the
// execution and operator-facing surfaces from drifting apart.
func DefaultRuleCatalog() (RuleCatalog, error) {
	graphCatalog, err := OperatingGraphRuleCatalog()
	if err != nil {
		return nil, err
	}
	entries := catalogEntries(graphCatalog)
	modelCatalog, err := OperatingModelRuleCatalog()
	if err != nil {
		return nil, err
	}
	topicCatalog, err := TopicRuleCatalog()
	if err != nil {
		return nil, err
	}
	porCatalog, err := PlanOfRecordRuleCatalog()
	if err != nil {
		return nil, err
	}
	entries = append(entries, catalogEntries(topicCatalog)...)
	entries = append(entries, catalogEntries(porCatalog)...)
	entries = append(entries, catalogEntries(modelCatalog)...)
	return NewRuleCatalog(entries...)
}

func catalogEntries(catalog RuleCatalog) []RuleCatalogEntry {
	ids := make([]string, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	entries := make([]RuleCatalogEntry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, catalog[id])
	}
	return entries
}

// Markdown renders the operator-facing rule reference from the catalog. It
// is deterministic so documentation generation can fail on semantic catalog
// drift rather than producing noisy reorder-only diffs.
func (c RuleCatalog) Markdown() string {
	ids := make([]string, 0, len(c))
	for id := range c {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var b strings.Builder
	b.WriteString("| Rule | Group | Default severity | Description | Actuator |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, id := range ids {
		entry := c[id]
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | %s | %s |\n", entry.ID, entry.Group, entry.Severity, entry.Description, entry.Actuator)
	}
	return b.String()
}
