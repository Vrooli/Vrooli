package memberflow

import (
	"fmt"
	"sort"
)

// RulePass names the validation entry point a rule executes in. Every rule
// belongs to exactly one pass, and the pass is derived from the rule's group
// by rulePassForGroup rather than declared per rule, so the three entry points
// cannot disagree about which rules belong to them.
type RulePass string

const (
	RulePassTopic     RulePass = "topic"
	RulePassGraph     RulePass = "graph"
	RulePassModel     RulePass = "model"
	RulePassObjective RulePass = "objective"
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
	case OperatingRuleGroupObjective:
		return RulePassObjective, nil
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

func NewRuleRegistry(rules ...Rule) (*RuleRegistry, error) {
	return newRuleRegistry(nil, rules...)
}

// NewRuleRegistryWithCatalog requires every rule to have exactly one
// operator-facing catalog entry and keeps metadata aligned with execution.
func NewRuleRegistryWithCatalog(catalog RuleCatalog, rules ...Rule) (*RuleRegistry, error) {
	return newRuleRegistry(catalog, rules...)
}

// multiEmitter is implemented by a check that produces more than one rule id.
//
// Emits is an optional interface rather than a method on Rule because the
// overwhelming majority of rules emit exactly their own id, and requiring all
// ninety of them to restate that would be ceremony with a drift risk attached.
// A check that emits several ids must declare them, and registration verifies
// the declaration against the catalog.
type multiEmitter interface {
	Emits() []string
}

// ruleEmits is the single answer to "which ids can this rule produce".
func ruleEmits(rule Rule) []string {
	if emitter, ok := rule.(multiEmitter); ok {
		if ids := emitter.Emits(); len(ids) > 0 {
			return ids
		}
	}
	return []string{rule.ID()}
}

func newRuleRegistry(catalog RuleCatalog, rules ...Rule) (*RuleRegistry, error) {
	registry := &RuleRegistry{
		byID:   make(map[string]Rule, len(rules)),
		byPass: make(map[RulePass][]Rule),
	}
	// claimed maps every catalogued id to the rule that emits it, so a catalog
	// entry cannot be claimed twice and cannot go unclaimed.
	claimed := make(map[string]string, len(rules))
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
		emitted := ruleEmits(rule)
		if catalog != nil {
			for _, emittedID := range emitted {
				entry, ok := catalog[emittedID]
				if !ok {
					return nil, fmt.Errorf("validation rule %q emits %q, which is missing from the rule catalog", id, emittedID)
				}
				if claimant, taken := claimed[emittedID]; taken {
					return nil, fmt.Errorf("rule catalog entry %q is claimed by both %q and %q", emittedID, claimant, id)
				}
				claimed[emittedID] = id
				// Group and severity are pinned only for a rule that emits
				// exactly its own id. A multi-emit check spans several
				// severities by design — ruleMemberDocSections emits five
				// errors and one warning — and the per-finding severity is set
				// by the check, not by the registration.
				if len(emitted) == 1 && (entry.Group != rule.Group() || entry.Severity != rule.DefaultSeverity()) {
					return nil, fmt.Errorf("rule catalog metadata for %q disagrees with its registration", id)
				}
			}
		}
		registry.rules = append(registry.rules, rule)
		registry.byID[id] = rule
	}
	// Ranging a nil catalog is a no-op, which is exactly the unenforced case.
	for id := range catalog {
		if _, ok := claimed[id]; !ok {
			return nil, fmt.Errorf("rule catalog entry %q has no registered rule", id)
		}
	}
	return registry, nil
}

// EmittedIDs returns every rule id the registry can produce, across all
// registrations. It is the registration-model-independent answer to "which
// rules exist": one registration may claim several ids.
func (r *RuleRegistry) EmittedIDs() []string {
	ids := make([]string, 0, len(r.rules))
	for _, rule := range r.rules {
		ids = append(ids, ruleEmits(rule)...)
	}
	sort.Strings(ids)
	return ids
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
	rules = append(rules, DefaultObjectiveRules()...)
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
	objectiveCatalog, err := ObjectiveRuleCatalog()
	if err != nil {
		return nil, err
	}
	entries = append(entries, catalogEntries(objectiveCatalog)...)
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
