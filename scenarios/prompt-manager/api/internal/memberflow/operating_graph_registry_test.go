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
		if spec.CoverageIncluded && spec.RuntimeCoverageMode == "" {
			t.Fatalf("relationship %q participates in coverage without runtime coverage mode", spec.Kind)
		}
		// A covered relationship has to name the declaration fields its
		// runtime side is read from, or coverage counts nothing.
		if spec.CoverageIncluded && len(spec.RuntimeFields) == 0 {
			t.Fatalf("relationship %q participates in coverage without acceptable runtime fields", spec.Kind)
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
	// The coarse topic->member read edge stands for three declaration fields.
	// OPERATING_GRAPHS.md records that coarseness as a deliberate readability
	// decision, so the registry must keep all three rather than splitting them.
	if len(topicRead.RuntimeFields) != 3 {
		t.Fatalf("topic read runtime fields = %d, want 3 (intake, required_read, evidence_consumed)", len(topicRead.RuntimeFields))
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

	graphExternalTopic := OperatingRelationship{Kind: operatingRelExternalProducerIntake, Team: "team-a", External: "operator", Topic: "research-inbox/*"}
	runtimeExternalMember := OperatingRelationship{Kind: operatingRelExternalProducer, Team: "team-a", External: "operator", Member: "researcher"}
	if registry.Match(graphExternalTopic, runtimeExternalMember) {
		t.Fatalf("external -> topic must not be satisfied by external -> member alone")
	}
}
