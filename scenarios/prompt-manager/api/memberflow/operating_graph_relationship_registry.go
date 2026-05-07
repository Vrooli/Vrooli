package memberflow

import "fmt"

type OperatingGraphEdgeShape struct {
	FromKind OperatingGraphNodeKind
	ToKind   OperatingGraphNodeKind
}

type OperatingRelationshipSpec struct {
	Kind                 OperatingRelationshipKind
	RuntimeKinds         []OperatingRelationshipKind
	GraphShape           OperatingGraphEdgeShape
	RuntimeFields        []string
	GraphSuggestions     func(OperatingGraphContractDiff) []string
	RuntimeSuggestions   func(OperatingGraphContractDiff) []string
	Statement            func(OperatingGraphContractDiff) string
	ValidationRule       string
	ValidationSeverity   Severity
	CoverageIncluded     bool
	DiffIncluded         bool
	RuntimeOnlyCompletes bool
}

type OperatingRelationshipRegistry struct {
	specs []OperatingRelationshipSpec
}

func DefaultOperatingRelationshipRegistry() OperatingRelationshipRegistry {
	return OperatingRelationshipRegistry{specs: []OperatingRelationshipSpec{
		{
			Kind:                 operatingRelTopicRead,
			RuntimeKinds:         []OperatingRelationshipKind{operatingRelTopicIntake, operatingRelTopicRequiredRead, operatingRelTopicEvidenceConsumed},
			GraphShape:           OperatingGraphEdgeShape{FromKind: OperatingGraphNodeKindTopic, ToKind: OperatingGraphNodeKindMember},
			RuntimeFields:        []string{"intake", "required_read", "evidence_consumed"},
			GraphSuggestions:     graphTopicReadSuggestions,
			RuntimeSuggestions:   runtimeTopicReadSuggestions,
			Statement:            topicReadStatement,
			ValidationRule:       "graph_declared_intake_missing,graph_declared_required_read_missing,graph_declared_evidence_missing",
			ValidationSeverity:   SeverityError,
			CoverageIncluded:     true,
			DiffIncluded:         true,
			RuntimeOnlyCompletes: true,
		},
		{
			Kind:                 operatingRelTopicOutput,
			RuntimeKinds:         []OperatingRelationshipKind{operatingRelTopicOutput},
			GraphShape:           OperatingGraphEdgeShape{FromKind: OperatingGraphNodeKindMember, ToKind: OperatingGraphNodeKindTopic},
			RuntimeFields:        []string{"output"},
			GraphSuggestions:     graphTopicOutputSuggestions,
			RuntimeSuggestions:   runtimeTopicOutputSuggestions,
			Statement:            topicOutputStatement,
			ValidationRule:       "graph_declared_output_missing",
			ValidationSeverity:   SeverityWarning,
			CoverageIncluded:     true,
			DiffIncluded:         true,
			RuntimeOnlyCompletes: true,
		},
		{
			Kind:                 operatingRelPOROutput,
			RuntimeKinds:         []OperatingRelationshipKind{operatingRelPOROutput},
			GraphShape:           OperatingGraphEdgeShape{FromKind: OperatingGraphNodeKindMember, ToKind: OperatingGraphNodeKindPOR},
			RuntimeFields:        []string{"output.destination_kind=por_file"},
			GraphSuggestions:     graphPOROutputSuggestions,
			RuntimeSuggestions:   runtimePOROutputSuggestions,
			Statement:            porOutputStatement,
			ValidationRule:       "graph_declared_output_missing",
			ValidationSeverity:   SeverityWarning,
			CoverageIncluded:     true,
			DiffIncluded:         true,
			RuntimeOnlyCompletes: true,
		},
		{
			Kind:                 operatingRelDecisionOwned,
			RuntimeKinds:         []OperatingRelationshipKind{operatingRelDecisionOwned},
			GraphShape:           OperatingGraphEdgeShape{FromKind: OperatingGraphNodeKindMember, ToKind: OperatingGraphNodeKindDecision},
			RuntimeFields:        []string{"decisions_owned"},
			GraphSuggestions:     graphDecisionOwnedSuggestions,
			RuntimeSuggestions:   runtimeDecisionOwnedSuggestions,
			Statement:            decisionOwnedStatement,
			ValidationRule:       "graph_declared_decision_owned_missing",
			ValidationSeverity:   SeverityError,
			CoverageIncluded:     true,
			DiffIncluded:         true,
			RuntimeOnlyCompletes: true,
		},
		{
			Kind:                 operatingRelDecisionConsumed,
			RuntimeKinds:         []OperatingRelationshipKind{operatingRelDecisionConsumed, operatingRelTopicEvidenceConsumed},
			GraphShape:           OperatingGraphEdgeShape{FromKind: OperatingGraphNodeKindDecision, ToKind: OperatingGraphNodeKindMember},
			RuntimeFields:        []string{"decisions_consumed", "evidence_consumed.for_decisions"},
			GraphSuggestions:     graphDecisionConsumedSuggestions,
			RuntimeSuggestions:   runtimeDecisionConsumedSuggestions,
			Statement:            decisionConsumedStatement,
			ValidationRule:       "graph_declared_decision_consumed_missing",
			ValidationSeverity:   SeverityError,
			CoverageIncluded:     true,
			DiffIncluded:         true,
			RuntimeOnlyCompletes: true,
		},
		{
			Kind:                 operatingRelCapabilityGapRaised,
			RuntimeKinds:         []OperatingRelationshipKind{operatingRelCapabilityGapRaised},
			GraphShape:           OperatingGraphEdgeShape{FromKind: OperatingGraphNodeKindMember, ToKind: OperatingGraphNodeKindDecision},
			RuntimeFields:        []string{"raises_capability_gaps"},
			GraphSuggestions:     graphCapabilityGapRaisedSuggestions,
			RuntimeSuggestions:   runtimeCapabilityGapRaisedSuggestions,
			Statement:            capabilityGapRaisedStatement,
			ValidationRule:       "graph_declared_capability_gap_missing",
			ValidationSeverity:   SeverityWarning,
			CoverageIncluded:     true,
			DiffIncluded:         true,
			RuntimeOnlyCompletes: true,
		},
		{
			Kind:                 operatingRelExternalProducer,
			RuntimeKinds:         []OperatingRelationshipKind{operatingRelExternalProducer},
			GraphShape:           OperatingGraphEdgeShape{FromKind: OperatingGraphNodeKindExternal, ToKind: OperatingGraphNodeKindMember},
			RuntimeFields:        []string{"external_producers"},
			GraphSuggestions:     graphExternalProducerSuggestions,
			RuntimeSuggestions:   runtimeExternalProducerSuggestions,
			Statement:            externalProducerStatement,
			ValidationRule:       "graph_declared_external_producer_missing",
			ValidationSeverity:   SeverityWarning,
			CoverageIncluded:     true,
			DiffIncluded:         true,
			RuntimeOnlyCompletes: true,
		},
		{
			Kind:                 operatingRelExternalProducerIntake,
			RuntimeKinds:         []OperatingRelationshipKind{operatingRelExternalProducerIntake},
			GraphShape:           OperatingGraphEdgeShape{FromKind: OperatingGraphNodeKindExternal, ToKind: OperatingGraphNodeKindTopic},
			RuntimeFields:        []string{"external_producers", "intake"},
			GraphSuggestions:     graphExternalProducerIntakeSuggestions,
			RuntimeSuggestions:   runtimeExternalProducerIntakeSuggestions,
			Statement:            externalProducerIntakeStatement,
			ValidationRule:       "graph_edge_unbacked",
			ValidationSeverity:   SeverityError,
			CoverageIncluded:     true,
			DiffIncluded:         true,
			RuntimeOnlyCompletes: false,
		},
		{
			Kind:                 operatingRelCrossTeamOutput,
			RuntimeKinds:         []OperatingRelationshipKind{operatingRelCrossTeamOutput},
			GraphShape:           OperatingGraphEdgeShape{FromKind: OperatingGraphNodeKindTopic, ToKind: OperatingGraphNodeKindTeam},
			RuntimeFields:        []string{"output.destination_team"},
			GraphSuggestions:     graphCrossTeamOutputSuggestions,
			RuntimeSuggestions:   runtimeCrossTeamOutputSuggestions,
			Statement:            crossTeamOutputStatement,
			ValidationRule:       "graph_declared_cross_team_output_missing",
			ValidationSeverity:   SeverityWarning,
			CoverageIncluded:     true,
			DiffIncluded:         true,
			RuntimeOnlyCompletes: true,
		},
	}}
}

func (r OperatingRelationshipRegistry) Specs() []OperatingRelationshipSpec {
	out := make([]OperatingRelationshipSpec, len(r.specs))
	copy(out, r.specs)
	return out
}

func (r OperatingRelationshipRegistry) CoverageSpecs() []OperatingRelationshipSpec {
	var out []OperatingRelationshipSpec
	for _, spec := range r.specs {
		if spec.CoverageIncluded {
			out = append(out, spec)
		}
	}
	return out
}

func (r OperatingRelationshipRegistry) Spec(kind OperatingRelationshipKind) (OperatingRelationshipSpec, bool) {
	for _, spec := range r.specs {
		if spec.Kind == kind {
			return spec, true
		}
	}
	return OperatingRelationshipSpec{}, false
}

func (r OperatingRelationshipRegistry) GraphKindForRuntime(kind OperatingRelationshipKind) OperatingRelationshipKind {
	for _, spec := range r.specs {
		if spec.AcceptsRuntimeKind(kind) {
			return spec.Kind
		}
	}
	return kind
}

func (r OperatingRelationshipRegistry) RelationshipFromEdge(team string, source OperatingSourceRef, from, to OperatingGraphNode) (OperatingRelationship, bool) {
	spec, ok := r.specForEdge(from, to)
	if !ok {
		return OperatingRelationship{}, false
	}
	rel := OperatingRelationship{Kind: spec.Kind, Team: team, Source: source}
	switch spec.Kind {
	case operatingRelTopicRead:
		rel.Topic = from.Value
		rel.Member = to.Value
	case operatingRelTopicOutput:
		rel.Member = from.Value
		rel.Topic = to.Value
	case operatingRelPOROutput:
		rel.Member = from.Value
		rel.Path = to.Value
	case operatingRelDecisionOwned:
		rel.Member = from.Value
		rel.Decision = to.Value
	case operatingRelDecisionConsumed:
		rel.Decision = from.Value
		rel.Member = to.Value
	case operatingRelCapabilityGapRaised:
		rel.Member = from.Value
		rel.Decision = to.Value
	case operatingRelExternalProducer:
		rel.External = from.Value
		rel.Member = to.Value
	case operatingRelExternalProducerIntake:
		rel.External = from.Value
		rel.Topic = to.Value
	case operatingRelCrossTeamOutput:
		rel.Topic = from.Value
		rel.TargetTeam = to.Value
	default:
		return OperatingRelationship{}, false
	}
	return rel, true
}

func (r OperatingRelationshipRegistry) Match(graphRel, runtimeRel OperatingRelationship) bool {
	if graphRel.Team != "" && runtimeRel.Team != "" && graphRel.Team != runtimeRel.Team {
		return false
	}
	spec, ok := r.Spec(graphRel.Kind)
	if !ok || !spec.AcceptsRuntimeKind(runtimeRel.Kind) {
		return false
	}
	switch graphRel.Kind {
	case operatingRelTopicRead:
		return graphRel.Member == runtimeRel.Member &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	case operatingRelTopicOutput:
		return graphRel.Member == runtimeRel.Member &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	case operatingRelPOROutput:
		return graphRel.Member == runtimeRel.Member &&
			pathsEqual(graphRel.Path, runtimeRel.Path)
	case operatingRelDecisionOwned:
		return graphRel.Member == runtimeRel.Member &&
			graphRel.Decision == runtimeRel.Decision
	case operatingRelDecisionConsumed:
		return graphRel.Member == runtimeRel.Member &&
			graphRel.Decision == runtimeRel.Decision
	case operatingRelCapabilityGapRaised:
		return graphRel.Member == runtimeRel.Member
	case operatingRelExternalProducer:
		return graphRel.Member == runtimeRel.Member &&
			graphRel.External == runtimeRel.External
	case operatingRelExternalProducerIntake:
		return graphRel.External == runtimeRel.External &&
			(graphRel.Member == "" || runtimeRel.Member == "" || graphRel.Member == runtimeRel.Member) &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	case operatingRelCrossTeamOutput:
		return graphRel.TargetTeam == runtimeRel.TargetTeam &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	default:
		return false
	}
}

func (r OperatingRelationshipRegistry) AcceptableRuntimeFields(kind OperatingRelationshipKind) []string {
	spec, ok := r.Spec(kind)
	if !ok {
		return nil
	}
	out := make([]string, len(spec.RuntimeFields))
	copy(out, spec.RuntimeFields)
	return out
}

func (r OperatingRelationshipRegistry) Validate() error {
	seen := map[OperatingRelationshipKind]bool{}
	for _, spec := range r.specs {
		if spec.Kind == "" {
			return fmt.Errorf("relationship spec has empty kind")
		}
		if seen[spec.Kind] {
			return fmt.Errorf("duplicate relationship spec %q", spec.Kind)
		}
		seen[spec.Kind] = true
		if spec.CoverageIncluded && spec.ValidationRule == "" {
			return fmt.Errorf("relationship spec %q is covered but has no validation rule", spec.Kind)
		}
		if spec.DiffIncluded && (spec.GraphSuggestions == nil || spec.RuntimeSuggestions == nil || spec.Statement == nil) {
			return fmt.Errorf("relationship spec %q is diffed but has incomplete diff metadata", spec.Kind)
		}
	}
	return nil
}

func (r OperatingRelationshipRegistry) specForEdge(from, to OperatingGraphNode) (OperatingRelationshipSpec, bool) {
	for _, spec := range r.specs {
		if spec.GraphShape.FromKind != from.Kind || spec.GraphShape.ToKind != to.Kind {
			continue
		}
		if spec.Kind == operatingRelCapabilityGapRaised && to.Value != "capability-gap" {
			continue
		}
		if spec.Kind == operatingRelDecisionOwned && to.Value == "capability-gap" {
			continue
		}
		return spec, true
	}
	return OperatingRelationshipSpec{}, false
}

func (s OperatingRelationshipSpec) AcceptsRuntimeKind(kind OperatingRelationshipKind) bool {
	for _, runtimeKind := range s.RuntimeKinds {
		if runtimeKind == kind {
			return true
		}
	}
	return false
}

func graphTopicReadSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add required_read %q to %s/topics.json", diff.Topic, diff.Member), "or remove the topic -> member edge from the operating graph"}
}

func runtimeTopicReadSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add topic:%s -> member:%s to the operating graph", diff.Topic, diff.Member), "or remove the runtime read declaration if it is no longer part of the operating contract"}
}

func topicReadStatement(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("topic:%s -> member:%s", diff.Topic, diff.Member)
}

func graphTopicOutputSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add output %q to %s/topics.json", diff.Topic, diff.Member), "or remove the member -> topic edge from the operating graph"}
}

func runtimeTopicOutputSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add member:%s -> topic:%s to the operating graph", diff.Member, diff.Topic), "or remove the runtime output declaration if it is obsolete"}
}

func topicOutputStatement(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("member:%s -> topic:%s", diff.Member, diff.Topic)
}

func graphPOROutputSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add a por_file output to %q in %s/topics.json", diff.Path, diff.Member), "or remove the member -> PoR edge from the operating graph"}
}

func runtimePOROutputSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add member:%s -> por:%s to the operating graph", diff.Member, diff.Path), "or remove the runtime por_file output if it is obsolete"}
}

func porOutputStatement(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("member:%s -> por:%s", diff.Member, diff.Path)
}

func graphDecisionOwnedSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add decisions_owned %q to %s/topics.json", diff.Decision, diff.Member), "or remove the member -> decision edge from the operating graph"}
}

func runtimeDecisionOwnedSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add member:%s -> decision:%s to the operating graph", diff.Member, diff.Decision), "or remove the runtime decision ownership if it is obsolete"}
}

func decisionOwnedStatement(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("member:%s -> decision:%s", diff.Member, diff.Decision)
}

func graphDecisionConsumedSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add decisions_consumed %q to %s/topics.json", diff.Decision, diff.Member), "or remove the decision -> member edge from the operating graph"}
}

func runtimeDecisionConsumedSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add decision:%s -> member:%s to the operating graph", diff.Decision, diff.Member), "or remove the runtime decision consumption if it is obsolete"}
}

func decisionConsumedStatement(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("decision:%s -> member:%s", diff.Decision, diff.Member)
}

func graphCapabilityGapRaisedSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("set raises_capability_gaps to true in %s/topics.json", diff.Member), "or remove the member -> capability-gap edge from the operating graph"}
}

func runtimeCapabilityGapRaisedSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add member:%s -> decision:capability-gap to the operating graph", diff.Member), "or unset raises_capability_gaps if this member should not raise gaps"}
}

func capabilityGapRaisedStatement(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("member:%s -> decision:capability-gap", diff.Member)
}

func graphExternalProducerSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add external_producers %q to %s/topics.json", diff.External, diff.Member), "or remove the external -> member edge from the operating graph"}
}

func runtimeExternalProducerSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add external:%s -> member:%s to the operating graph", diff.External, diff.Member), "or remove the runtime external producer declaration if it is obsolete"}
}

func externalProducerStatement(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("external:%s -> member:%s", diff.External, diff.Member)
}

func graphExternalProducerIntakeSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("declare an intake for %q and external_producers %q on the receiving member topics.json", diff.Topic, diff.External), "or remove the external -> topic edge from the operating graph"}
}

func runtimeExternalProducerIntakeSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add external:%s -> topic:%s to the operating graph", diff.External, diff.Topic), "or remove the runtime external producer/intake relationship if it is obsolete"}
}

func externalProducerIntakeStatement(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("external:%s -> topic:%s", diff.External, diff.Topic)
}

func graphCrossTeamOutputSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add destination_team %q to an output for %q", diff.TargetTeam, diff.Topic), "or remove the topic -> team edge from the operating graph"}
}

func runtimeCrossTeamOutputSuggestions(diff OperatingGraphContractDiff) []string {
	return []string{fmt.Sprintf("add topic:%s -> team:%s to the operating graph", diff.Topic, diff.TargetTeam), "or remove destination_team if this is not a cross-team output"}
}

func crossTeamOutputStatement(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("topic:%s -> team:%s", diff.Topic, diff.TargetTeam)
}
