package memberflow

import "testing"

func TestDefaultOperatingRelationshipRegistryIsInternallyConsistent(t *testing.T) {
	registry := DefaultOperatingRelationshipRegistry()
	if err := registry.Validate(); err != nil {
		t.Fatalf("registry should validate: %v", err)
	}

	for _, spec := range registry.Specs() {
		if spec.DiffIncluded && !spec.CoverageIncluded {
			t.Fatalf("relationship %q participates in diff but not coverage", spec.Kind)
		}
		if spec.CoverageIncluded && len(spec.ValidationRules) == 0 {
			t.Fatalf("relationship %q participates in coverage without validation metadata", spec.Kind)
		}
		if spec.CoverageIncluded && spec.ValidationSeverity == "" {
			t.Fatalf("relationship %q participates in coverage without validation severity", spec.Kind)
		}
		if spec.CoverageIncluded && spec.RuntimeCoverageMode == "" {
			t.Fatalf("relationship %q participates in coverage without runtime coverage mode", spec.Kind)
		}
		if spec.RuntimeOnlyCompletes && len(spec.RuntimeFields) == 0 {
			t.Fatalf("relationship %q requires runtime completeness without acceptable runtime fields", spec.Kind)
		}
		if spec.RuntimeOnlyCompletes && len(spec.CompletenessTargets) == 0 {
			t.Fatalf("relationship %q requires runtime completeness without registry targets", spec.Kind)
		}
	}
}

func TestDefaultOperatingGraphRulesIncludeRegistryBackedCompleteness(t *testing.T) {
	registry := DefaultOperatingRelationshipRegistry()
	rules := DefaultOperatingGraphRules()
	rulesByID := map[string]OperatingGraphRule{}
	for _, rule := range rules {
		if _, ok := rulesByID[rule.ID()]; ok {
			t.Fatalf("duplicate operating graph rule id %q", rule.ID())
		}
		rulesByID[rule.ID()] = rule
	}

	for _, spec := range registry.Specs() {
		if !spec.RuntimeOnlyCompletes {
			continue
		}
		for _, target := range spec.CompletenessTargets {
			if _, ok := rulesByID[target.RuleID]; !ok {
				t.Fatalf("registry relationship %q expects completeness rule %q", spec.Kind, target.RuleID)
			}
		}
	}
}

func TestOperatingRelationshipRegistryOwnsCoveragePolicies(t *testing.T) {
	registry := DefaultOperatingRelationshipRegistry()
	topicRead, ok := registry.Spec(operatingRelTopicRead)
	if !ok {
		t.Fatalf("topic read relationship spec missing")
	}
	if !topicRead.SubtypeCoverage {
		t.Fatalf("topic read should declare runtime subtype coverage in the registry")
	}
	if len(topicRead.CompletenessTargets) != 3 {
		t.Fatalf("topic read completeness targets = %d, want 3", len(topicRead.CompletenessTargets))
	}

	externalIntake, ok := registry.Spec(operatingRelExternalProducerIntake)
	if !ok {
		t.Fatalf("external producer intake relationship spec missing")
	}
	if externalIntake.RuntimeCoverageMode != OperatingRelationshipRuntimeCoverageGraphShown {
		t.Fatalf("external producer intake runtime coverage mode = %q, want %q", externalIntake.RuntimeCoverageMode, OperatingRelationshipRuntimeCoverageGraphShown)
	}
}

func TestOperatingRelationshipRegistryMapsSupportedGraphEdges(t *testing.T) {
	registry := DefaultOperatingRelationshipRegistry()
	tests := []struct {
		name string
		from OperatingGraphNode
		to   OperatingGraphNode
		want OperatingRelationshipKind
	}{
		{
			name: "topic read",
			from: OperatingGraphNode{Kind: OperatingGraphNodeKindTopic, Value: "research-inbox/*"},
			to:   OperatingGraphNode{Kind: OperatingGraphNodeKindMember, Value: "researcher"},
			want: operatingRelTopicRead,
		},
		{
			name: "topic output",
			from: OperatingGraphNode{Kind: OperatingGraphNodeKindMember, Value: "researcher"},
			to:   OperatingGraphNode{Kind: OperatingGraphNodeKindTopic, Value: "hook-record/*"},
			want: operatingRelTopicOutput,
		},
		{
			name: "por output",
			from: OperatingGraphNode{Kind: OperatingGraphNodeKindMember, Value: "brand-manager"},
			to:   OperatingGraphNode{Kind: OperatingGraphNodeKindPOR, Value: "docs/marketing/STRATEGY.md"},
			want: operatingRelPOROutput,
		},
		{
			name: "decision owned",
			from: OperatingGraphNode{Kind: OperatingGraphNodeKindMember, Value: "researcher"},
			to:   OperatingGraphNode{Kind: OperatingGraphNodeKindDecision, Value: "audience-update"},
			want: operatingRelDecisionOwned,
		},
		{
			name: "decision consumed",
			from: OperatingGraphNode{Kind: OperatingGraphNodeKindDecision, Value: "audience-update"},
			to:   OperatingGraphNode{Kind: OperatingGraphNodeKindMember, Value: "marketing-contrarian"},
			want: operatingRelDecisionConsumed,
		},
		{
			name: "capability gap raised",
			from: OperatingGraphNode{Kind: OperatingGraphNodeKindMember, Value: "researcher"},
			to:   OperatingGraphNode{Kind: OperatingGraphNodeKindDecision, Value: "capability-gap"},
			want: operatingRelCapabilityGapRaised,
		},
		{
			name: "external producer",
			from: OperatingGraphNode{Kind: OperatingGraphNodeKindExternal, Value: "operator"},
			to:   OperatingGraphNode{Kind: OperatingGraphNodeKindMember, Value: "researcher"},
			want: operatingRelExternalProducer,
		},
		{
			name: "external producer intake",
			from: OperatingGraphNode{Kind: OperatingGraphNodeKindExternal, Value: "operator"},
			to:   OperatingGraphNode{Kind: OperatingGraphNodeKindTopic, Value: "research-inbox/*"},
			want: operatingRelExternalProducerIntake,
		},
		{
			name: "cross team output",
			from: OperatingGraphNode{Kind: OperatingGraphNodeKindTopic, Value: "monetization-benchmark-adjacent-record/*"},
			to:   OperatingGraphNode{Kind: OperatingGraphNodeKindTeam, Value: "monetization"},
			want: operatingRelCrossTeamOutput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rel, ok := registry.RelationshipFromEdge("marketing-crew", OperatingSourceRef{}, tc.from, tc.to)
			if !ok {
				t.Fatalf("expected edge to map to %q", tc.want)
			}
			if rel.Kind != tc.want {
				t.Fatalf("relationship kind = %q, want %q", rel.Kind, tc.want)
			}
		})
	}
}

func TestOperatingRelationshipRegistryMatchesRuntimeKinds(t *testing.T) {
	registry := DefaultOperatingRelationshipRegistry()
	read := OperatingRelationship{Kind: operatingRelTopicRead, Team: "team-a", Member: "researcher", Topic: "research-inbox/*"}
	for _, runtimeKind := range []OperatingRelationshipKind{
		operatingRelTopicIntake,
		operatingRelTopicRequiredRead,
		operatingRelTopicEvidenceConsumed,
	} {
		runtime := OperatingRelationship{Kind: runtimeKind, Team: "team-a", Member: "researcher", Topic: "research-inbox/item"}
		if !registry.Match(read, runtime) {
			t.Fatalf("topic read should match runtime kind %q", runtimeKind)
		}
	}

	decisionRead := OperatingRelationship{Kind: operatingRelDecisionConsumed, Team: "team-a", Member: "contrarian", Decision: "audience-update"}
	evidence := OperatingRelationship{Kind: operatingRelTopicEvidenceConsumed, Team: "team-a", Member: "contrarian", Topic: "audience-scan/*", Decision: "audience-update"}
	if !registry.Match(decisionRead, evidence) {
		t.Fatalf("decision consumption should be satisfiable by evidence consumed for the decision")
	}

	graphExternalTopic := OperatingRelationship{Kind: operatingRelExternalProducerIntake, Team: "team-a", External: "operator", Topic: "research-inbox/*"}
	runtimeExternalMember := OperatingRelationship{Kind: operatingRelExternalProducer, Team: "team-a", External: "operator", Member: "researcher"}
	if registry.Match(graphExternalTopic, runtimeExternalMember) {
		t.Fatalf("external -> topic must not be satisfied by external -> member alone")
	}
}
