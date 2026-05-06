package memberflow

import (
	"fmt"
	"sort"
)

func DiffOperatingGraphs(blocks []OperatingGraphBlock, runtime OperatingGraphRuntime, teamFilter, idFilter string) []OperatingGraphContractDiff {
	diffs := []OperatingGraphContractDiff{}
	for _, block := range filterOperatingGraphBlocks(blocks, teamFilter, idFilter) {
		if block.Metadata.Mode != "contract" {
			continue
		}
		ctx := NewOperatingGraphContractContext(block, runtime)
		diffs = append(diffs, DiffOperatingGraphContract(ctx)...)
	}
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Kind != diffs[j].Kind {
			return diffs[i].Kind < diffs[j].Kind
		}
		if diffs[i].Relationship != diffs[j].Relationship {
			return diffs[i].Relationship < diffs[j].Relationship
		}
		return diffs[i].Detail < diffs[j].Detail
	})
	return diffs
}

func DiffOperatingGraphContract(ctx OperatingGraphContractContext) []OperatingGraphContractDiff {
	var diffs []OperatingGraphContractDiff
	for _, rel := range ctx.Index.GraphRelationships.All() {
		if !ctx.Matcher.GraphBackedByRuntime(rel, ctx.Index.RuntimeRelationships) {
			diffs = append(diffs, operatingGraphDiffFromGraphRel(rel, ctx.Runtime))
		}
	}
	seenRuntimeDiffs := map[string]bool{}
	for _, rel := range ctx.Index.RuntimeRelationships.All() {
		if rel.Kind == operatingRelExternalProducerIntake {
			continue
		}
		diffKey := operatingRelationshipDiffDedupeKey(rel)
		if seenRuntimeDiffs[diffKey] {
			continue
		}
		seenRuntimeDiffs[diffKey] = true
		if !ctx.Matcher.RuntimeShownInGraph(rel, ctx.Index.GraphRelationships) {
			diffs = append(diffs, operatingGraphDiffFromRuntimeRel(rel, ctx.Block))
		}
	}
	return diffs
}

func operatingGraphDiffFromGraphRel(rel OperatingRelationship, runtime OperatingGraphRuntime) OperatingGraphContractDiff {
	diff := OperatingGraphContractDiff{
		Kind:             "graph_relationship_missing_in_runtime",
		Relationship:     string(rel.Kind),
		Team:             rel.Team,
		Member:           rel.Member,
		Topic:            rel.Topic,
		Decision:         rel.Decision,
		Path:             rel.Path,
		External:         rel.External,
		TargetTeam:       rel.TargetTeam,
		SourcePath:       rel.Source.Path,
		Line:             rel.Source.Line,
		RuntimePath:      runtimePathForGraphRelationship(rel, runtime),
		AcceptableFields: acceptableRuntimeFields(rel),
	}
	diff.Suggestions = suggestionsForGraphRelationship(diff)
	diff.Detail = detailForGraphRelationshipMissingRuntime(diff)
	return diff
}

func operatingGraphDiffFromRuntimeRel(rel OperatingRelationship, block OperatingGraphBlock) OperatingGraphContractDiff {
	diff := OperatingGraphContractDiff{
		Kind:         "runtime_relationship_missing_in_graph",
		Relationship: string(runtimeRelationshipAsGraphRelationship(rel)),
		Team:         rel.Team,
		Member:       rel.Member,
		Topic:        rel.Topic,
		Decision:     rel.Decision,
		Path:         rel.Path,
		External:     rel.External,
		TargetTeam:   rel.TargetTeam,
		SourcePath:   block.Source.Path,
		Line:         block.Source.FenceLine,
		RuntimePath:  rel.Source.Path,
	}
	diff.Suggestions = suggestionsForRuntimeRelationship(diff)
	diff.Detail = detailForRuntimeRelationshipMissingGraph(diff)
	return diff
}

func runtimeRelationshipAsGraphRelationship(rel OperatingRelationship) OperatingRelationshipKind {
	if isRuntimeReadRelationship(rel.Kind) {
		return operatingRelTopicRead
	}
	return rel.Kind
}

func runtimePathForGraphRelationship(rel OperatingRelationship, runtime OperatingGraphRuntime) string {
	if rel.Member != "" {
		return runtimeMemberTopicsPath(runtime, rel.Team, rel.Member)
	}
	if rel.Kind == operatingRelCrossTeamOutput || rel.Kind == operatingRelExternalProducerIntake {
		if match := matchingRuntimeRelationshipPath(rel, runtime); match != "" {
			return match
		}
	}
	return runtimeTeamPath(runtime, rel.Team)
}

func matchingRuntimeRelationshipPath(rel OperatingRelationship, runtime OperatingGraphRuntime) string {
	matcher := OperatingRelationshipMatcher{}
	runtimeRels := NewOperatingRelationshipSet(BuildRuntimeOperatingRelationships(runtime, rel.Team))
	for _, candidate := range runtimeRels.All() {
		if matcher.GraphBackedByRuntime(rel, NewOperatingRelationshipSet([]OperatingRelationship{candidate})) || relationshipSharesGraphlessAnchor(rel, candidate) {
			return candidate.Source.Path
		}
	}
	return ""
}

func relationshipSharesGraphlessAnchor(graphRel, runtimeRel OperatingRelationship) bool {
	if graphRel.Team != runtimeRel.Team {
		return false
	}
	switch graphRel.Kind {
	case operatingRelCrossTeamOutput:
		return runtimeRel.Kind == operatingRelCrossTeamOutput &&
			graphRel.TargetTeam == runtimeRel.TargetTeam &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	case operatingRelExternalProducerIntake:
		return runtimeRel.Kind == operatingRelExternalProducerIntake &&
			graphRel.External == runtimeRel.External &&
			topicsOverlap(graphRel.Topic, runtimeRel.Topic)
	default:
		return false
	}
}

func acceptableRuntimeFields(rel OperatingRelationship) []string {
	switch rel.Kind {
	case operatingRelTopicRead:
		return []string{"intake", "required_read", "evidence_consumed"}
	case operatingRelTopicOutput:
		return []string{"output"}
	case operatingRelPOROutput:
		return []string{"output.destination_kind=por_file"}
	case operatingRelDecisionOwned:
		return []string{"decisions_owned"}
	case operatingRelDecisionConsumed:
		return []string{"decisions_consumed", "evidence_consumed.for_decisions"}
	case operatingRelCapabilityGapRaised:
		return []string{"raises_capability_gaps"}
	case operatingRelExternalProducer, operatingRelExternalProducerIntake:
		return []string{"external_producers", "intake"}
	case operatingRelCrossTeamOutput:
		return []string{"output.destination_team"}
	default:
		return nil
	}
}

func suggestionsForGraphRelationship(diff OperatingGraphContractDiff) []string {
	switch diff.Relationship {
	case string(operatingRelTopicRead):
		return []string{
			fmt.Sprintf("add required_read %q to %s/topics.json", diff.Topic, diff.Member),
			"or remove the topic -> member edge from the operating graph",
		}
	case string(operatingRelTopicOutput):
		return []string{
			fmt.Sprintf("add output %q to %s/topics.json", diff.Topic, diff.Member),
			"or remove the member -> topic edge from the operating graph",
		}
	case string(operatingRelPOROutput):
		return []string{
			fmt.Sprintf("add a por_file output to %q in %s/topics.json", diff.Path, diff.Member),
			"or remove the member -> PoR edge from the operating graph",
		}
	case string(operatingRelDecisionOwned):
		return []string{
			fmt.Sprintf("add decisions_owned %q to %s/topics.json", diff.Decision, diff.Member),
			"or remove the member -> decision edge from the operating graph",
		}
	case string(operatingRelDecisionConsumed):
		return []string{
			fmt.Sprintf("add decisions_consumed %q to %s/topics.json", diff.Decision, diff.Member),
			"or remove the decision -> member edge from the operating graph",
		}
	case string(operatingRelCapabilityGapRaised):
		return []string{
			fmt.Sprintf("set raises_capability_gaps to true in %s/topics.json", diff.Member),
			"or remove the member -> capability-gap edge from the operating graph",
		}
	case string(operatingRelExternalProducer):
		return []string{
			fmt.Sprintf("add external_producers %q to %s/topics.json", diff.External, diff.Member),
			"or remove the external -> member edge from the operating graph",
		}
	case string(operatingRelExternalProducerIntake):
		return []string{
			fmt.Sprintf("declare an intake for %q and external_producers %q on the receiving member topics.json", diff.Topic, diff.External),
			"or remove the external -> topic edge from the operating graph",
		}
	case string(operatingRelCrossTeamOutput):
		return []string{
			fmt.Sprintf("add destination_team %q to an output for %q", diff.TargetTeam, diff.Topic),
			"or remove the topic -> team edge from the operating graph",
		}
	default:
		return nil
	}
}

func suggestionsForRuntimeRelationship(diff OperatingGraphContractDiff) []string {
	switch diff.Relationship {
	case string(operatingRelTopicRead):
		return []string{
			fmt.Sprintf("add topic:%s -> member:%s to the operating graph", diff.Topic, diff.Member),
			"or remove the runtime read declaration if it is no longer part of the operating contract",
		}
	case string(operatingRelTopicOutput):
		return []string{
			fmt.Sprintf("add member:%s -> topic:%s to the operating graph", diff.Member, diff.Topic),
			"or remove the runtime output declaration if it is obsolete",
		}
	case string(operatingRelPOROutput):
		return []string{
			fmt.Sprintf("add member:%s -> por:%s to the operating graph", diff.Member, diff.Path),
			"or remove the runtime por_file output if it is obsolete",
		}
	case string(operatingRelDecisionOwned):
		return []string{
			fmt.Sprintf("add member:%s -> decision:%s to the operating graph", diff.Member, diff.Decision),
			"or remove the runtime decision ownership if it is obsolete",
		}
	case string(operatingRelDecisionConsumed):
		return []string{
			fmt.Sprintf("add decision:%s -> member:%s to the operating graph", diff.Decision, diff.Member),
			"or remove the runtime decision consumption if it is obsolete",
		}
	case string(operatingRelCapabilityGapRaised):
		return []string{
			fmt.Sprintf("add member:%s -> decision:capability-gap to the operating graph", diff.Member),
			"or unset raises_capability_gaps if this member should not raise gaps",
		}
	case string(operatingRelExternalProducer):
		return []string{
			fmt.Sprintf("add external:%s -> member:%s to the operating graph", diff.External, diff.Member),
			"or remove the runtime external producer declaration if it is obsolete",
		}
	case string(operatingRelExternalProducerIntake):
		return []string{
			fmt.Sprintf("add external:%s -> topic:%s to the operating graph", diff.External, diff.Topic),
			"or remove the runtime external producer/intake relationship if it is obsolete",
		}
	case string(operatingRelCrossTeamOutput):
		return []string{
			fmt.Sprintf("add topic:%s -> team:%s to the operating graph", diff.Topic, diff.TargetTeam),
			"or remove destination_team if this is not a cross-team output",
		}
	default:
		return nil
	}
}

func detailForGraphRelationshipMissingRuntime(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("%s says %s. Runtime has no matching declaration.", relationshipLocation(diff), relationshipStatement(diff))
}

func detailForRuntimeRelationshipMissingGraph(diff OperatingGraphContractDiff) string {
	return fmt.Sprintf("%s declares %s. The contract graph does not show a matching relationship.", diff.RuntimePath, relationshipStatement(diff))
}

func relationshipLocation(diff OperatingGraphContractDiff) string {
	if diff.SourcePath == "" {
		return "contract graph"
	}
	if diff.Line > 0 {
		return fmt.Sprintf("%s:%d", diff.SourcePath, diff.Line)
	}
	return diff.SourcePath
}

func relationshipStatement(diff OperatingGraphContractDiff) string {
	switch diff.Relationship {
	case string(operatingRelTopicRead):
		return fmt.Sprintf("topic:%s -> member:%s", diff.Topic, diff.Member)
	case string(operatingRelTopicOutput):
		return fmt.Sprintf("member:%s -> topic:%s", diff.Member, diff.Topic)
	case string(operatingRelPOROutput):
		return fmt.Sprintf("member:%s -> por:%s", diff.Member, diff.Path)
	case string(operatingRelDecisionOwned):
		return fmt.Sprintf("member:%s -> decision:%s", diff.Member, diff.Decision)
	case string(operatingRelDecisionConsumed):
		return fmt.Sprintf("decision:%s -> member:%s", diff.Decision, diff.Member)
	case string(operatingRelCapabilityGapRaised):
		return fmt.Sprintf("member:%s -> decision:capability-gap", diff.Member)
	case string(operatingRelExternalProducer):
		return fmt.Sprintf("external:%s -> member:%s", diff.External, diff.Member)
	case string(operatingRelExternalProducerIntake):
		return fmt.Sprintf("external:%s -> topic:%s", diff.External, diff.Topic)
	case string(operatingRelCrossTeamOutput):
		return fmt.Sprintf("topic:%s -> team:%s", diff.Topic, diff.TargetTeam)
	default:
		return diff.Relationship
	}
}
