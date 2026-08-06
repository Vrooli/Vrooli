package memberflow

import (
	"strings"
	"testing"
)

func TestDefaultRuleRegistryRegistersEveryRegisteredRuleFamily(t *testing.T) {
	registry, err := DefaultRuleRegistry()
	if err != nil {
		t.Fatalf("DefaultRuleRegistry() error = %v", err)
	}
	want := len(DefaultOperatingGraphRules()) + len(DefaultOperatingModelRules()) + len(DefaultTopicRules()) + len(DefaultPlanOfRecordRules()) + len(DefaultObjectiveRules())
	if got := len(registry.Rules()); got != want {
		t.Fatalf("registered rules = %d, want %d", got, want)
	}
}

func TestDefaultRuleCatalogExactlyMatchesDefaultRegistry(t *testing.T) {
	catalog, err := DefaultRuleCatalog()
	if err != nil {
		t.Fatalf("DefaultRuleCatalog() error = %v", err)
	}
	rules := append(DefaultOperatingGraphRules(), DefaultOperatingModelRules()...)
	rules = append(rules, DefaultTopicRules()...)
	rules = append(rules, DefaultPlanOfRecordRules()...)
	rules = append(rules, DefaultObjectiveRules()...)
	if _, err := NewRuleRegistryWithCatalog(catalog, rules...); err != nil {
		t.Fatalf("default catalog and registry drifted: %v", err)
	}
	for _, entry := range catalog {
		if entry.Description == "" || entry.Actuator == "" {
			t.Errorf("catalog entry %q lacks operator metadata: %+v", entry.ID, entry)
		}
	}
}

func TestDefaultRuleRegistryIncludesTopicRuleAdapters(t *testing.T) {
	registry, err := DefaultRuleRegistry()
	if err != nil {
		t.Fatalf("DefaultRuleRegistry() error = %v", err)
	}
	registered := map[string]Rule{}
	for _, rule := range registry.Rules() {
		registered[rule.ID()] = rule
	}
	for _, rule := range DefaultTopicRules() {
		registeredRule, ok := registered[rule.ID()]
		if !ok {
			t.Errorf("topic rule %q is not registered", rule.ID())
			continue
		}
		if registeredRule.Group() != OperatingRuleGroupTopic {
			t.Errorf("topic rule %q group = %q, want %q", rule.ID(), registeredRule.Group(), OperatingRuleGroupTopic)
		}
	}
}

func TestTopicRuleCatalogExactlyMatchesTopicRegistry(t *testing.T) {
	catalog, err := TopicRuleCatalog()
	if err != nil {
		t.Fatalf("TopicRuleCatalog() error = %v", err)
	}
	if _, err := NewRuleRegistryWithCatalog(catalog, DefaultTopicRules()...); err != nil {
		t.Fatalf("catalog-enforced topic registry: %v", err)
	}
}

func TestTopicRegistryDefaultSeveritiesMatchRuleContracts(t *testing.T) {
	want := map[string]Severity{
		"unread_required":                SeverityError,
		"wildcard_source_misuse":         SeverityWarning,
		"missing_destination_schema":     SeverityWarning,
		"team_role_member_drift":         SeverityError,
		"member_doc_section_recommended": SeverityWarning,
	}
	for _, rule := range DefaultTopicRules() {
		severity, tracked := want[rule.ID()]
		if !tracked {
			// A tracked id may be one of several a check emits; the check's own
			// registration id is then a different id in the same family.
			for _, id := range ruleEmits(rule) {
				delete(want, id)
			}
			continue
		}
		// A multi-id check spans severities by design; the per-finding
		// severity is set by the check, so only a single-id registration
		// pins DefaultSeverity.
		emitted := ruleEmits(rule)
		if len(emitted) == 1 {
			if got := rule.DefaultSeverity(); got != severity {
				t.Errorf("%s default severity = %s, want %s", rule.ID(), got, severity)
			}
		}
		for _, id := range emitted {
			delete(want, id)
		}
	}
	for id := range want {
		t.Errorf("tracked topic rule %q was not registered", id)
	}
}

func TestDefaultRuleRegistryIncludesPlanOfRecordRuleAdapters(t *testing.T) {
	registry, err := DefaultRuleRegistry()
	if err != nil {
		t.Fatalf("DefaultRuleRegistry() error = %v", err)
	}
	registered := map[string]Rule{}
	for _, rule := range registry.Rules() {
		registered[rule.ID()] = rule
	}
	for _, rule := range DefaultPlanOfRecordRules() {
		registeredRule, ok := registered[rule.ID()]
		if !ok {
			t.Errorf("plan-of-record rule %q is not registered", rule.ID())
			continue
		}
		if registeredRule.Group() != OperatingRuleGroupPlanOfRecord {
			t.Errorf("plan-of-record rule %q group = %q, want %q", rule.ID(), registeredRule.Group(), OperatingRuleGroupPlanOfRecord)
		}
	}
}

func TestPlanOfRecordRuleCatalogExactlyMatchesRegistry(t *testing.T) {
	catalog, err := PlanOfRecordRuleCatalog()
	if err != nil {
		t.Fatalf("PlanOfRecordRuleCatalog() error = %v", err)
	}
	if _, err := NewRuleRegistryWithCatalog(catalog, DefaultPlanOfRecordRules()...); err != nil {
		t.Fatalf("catalog-enforced plan-of-record registry: %v", err)
	}
}

func TestRuleRegistryRejectsDuplicateIdentifiers(t *testing.T) {
	rules := DefaultOperatingModelRules()
	_, err := NewRuleRegistry(rules[0], rules[0])
	if err == nil {
		t.Fatal("NewRuleRegistry() error = nil, want duplicate identifier rejection")
	}
}

func TestCatalogEnforcedRegistryRejectsMissingAndOrphanedEntries(t *testing.T) {
	rule := DefaultOperatingModelRules()[0]
	if _, err := NewRuleRegistryWithCatalog(RuleCatalog{}, rule); err == nil {
		t.Fatal("catalog-enforced registry accepted a missing entry")
	}

	entry := RuleCatalogEntry{
		ID:          rule.ID(),
		Group:       rule.Group(),
		Severity:    rule.DefaultSeverity(),
		Description: "test rule",
		Actuator:    "test actuator",
	}
	catalog, err := NewRuleCatalog(entry)
	if err != nil {
		t.Fatalf("NewRuleCatalog() error = %v", err)
	}
	if _, err := NewRuleRegistryWithCatalog(catalog, rule); err != nil {
		t.Fatalf("catalog-enforced registry rejected aligned entry: %v", err)
	}

	orphan, err := NewRuleCatalog(entry, RuleCatalogEntry{ID: "orphan", Group: rule.Group(), Severity: rule.DefaultSeverity(), Description: "orphan", Actuator: "test actuator"})
	if err != nil {
		t.Fatalf("NewRuleCatalog(orphan) error = %v", err)
	}
	if _, err := NewRuleRegistryWithCatalog(orphan, rule); err == nil {
		t.Fatal("catalog-enforced registry accepted an orphan entry")
	}
}

// Every registered rule must land in exactly one pass, and the three passes
// together must account for the whole registry. Before pass assignment existed
// each entry point re-derived its own membership, and the graph entry point did
// so with a denylist that ran any unrecognised group by default.
func TestEveryRegisteredRuleBelongsToExactlyOnePass(t *testing.T) {
	registry, err := DefaultRuleRegistry()
	if err != nil {
		t.Fatalf("DefaultRuleRegistry() error = %v", err)
	}

	seen := make(map[string]RulePass)
	total := 0
	for _, pass := range []RulePass{RulePassTopic, RulePassGraph, RulePassModel, RulePassObjective} {
		for _, rule := range registry.RulesForPass(pass) {
			if prior, duplicated := seen[rule.ID()]; duplicated {
				t.Errorf("rule %q is in pass %q and pass %q", rule.ID(), prior, pass)
				continue
			}
			seen[rule.ID()] = pass
			total++
		}
	}

	if got := len(registry.Rules()); total != got {
		t.Fatalf("rules across passes = %d, want %d registered", total, got)
	}
	_ = registry.EmittedIDs()
}

func TestRuleRegistryRejectsGroupWithNoPassAssignment(t *testing.T) {
	rule := unassignedGroupRule{}
	if _, err := NewRuleRegistry(rule); err == nil {
		t.Fatal("NewRuleRegistry() accepted a rule whose group has no validation pass")
	}
}

// A topic-pass rule reports through the Finding shape. Registration must reject
// one that cannot, rather than letting the topic pass skip it at runtime.
func TestRuleRegistryRejectsTopicRuleWithoutFindingSupport(t *testing.T) {
	rule := topicGroupRuleWithoutFindings{}
	if _, err := NewRuleRegistry(rule); err == nil {
		t.Fatal("NewRuleRegistry() accepted a topic-pass rule that cannot produce findings")
	}
}

type unassignedGroupRule struct{}

func (unassignedGroupRule) ID() string                                { return "unassigned_group_rule" }
func (unassignedGroupRule) Group() RuleGroup                          { return RuleGroup("not_a_real_group") }
func (unassignedGroupRule) DefaultSeverity() Severity                 { return SeverityWarning }
func (unassignedGroupRule) AppliesTo(RuleContext) bool                { return true }
func (unassignedGroupRule) Check(RuleContext) []OperatingGraphFinding { return nil }

type topicGroupRuleWithoutFindings struct{}

func (topicGroupRuleWithoutFindings) ID() string       { return "topic_rule_without_findings" }
func (topicGroupRuleWithoutFindings) Group() RuleGroup { return OperatingRuleGroupTopic }
func (topicGroupRuleWithoutFindings) DefaultSeverity() Severity {
	return SeverityWarning
}
func (topicGroupRuleWithoutFindings) AppliesTo(RuleContext) bool { return true }
func (topicGroupRuleWithoutFindings) Check(RuleContext) []OperatingGraphFinding {
	return nil
}

func TestRuleCatalogMarkdownIsStableAndComplete(t *testing.T) {
	catalog, err := DefaultRuleCatalog()
	if err != nil {
		t.Fatalf("DefaultRuleCatalog() error = %v", err)
	}
	markdown := catalog.Markdown()
	if got := strings.Count(markdown, "\n") - 2; got != len(catalog) {
		t.Fatalf("catalog markdown rows = %d, want %d:\n%s", got, len(catalog), markdown)
	}
	if !strings.Contains(markdown, "| `actual_writer_undeclared` |") {
		t.Fatalf("catalog markdown omits registered rule:\n%s", markdown)
	}
}
