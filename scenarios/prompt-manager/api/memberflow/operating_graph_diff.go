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
	registry := DefaultOperatingRelationshipRegistry()
	for _, rel := range ctx.Index.GraphRelationships.All() {
		if !ctx.Matcher.GraphBackedByRuntime(rel, ctx.Index.RuntimeRelationships) {
			diffs = append(diffs, operatingGraphDiffFromGraphRel(rel, ctx.Runtime, registry))
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
			diffs = append(diffs, operatingGraphDiffFromRuntimeRel(rel, ctx.Block, registry))
		}
	}
	return diffs
}

func operatingGraphDiffFromGraphRel(rel OperatingRelationship, runtime OperatingGraphRuntime, registry OperatingRelationshipRegistry) OperatingGraphContractDiff {
	diff := OperatingGraphContractDiff{
		Kind:             "graph_relationship_missing_in_runtime",
		Relationship:     string(rel.Kind),
		Team:             rel.Team,
		Member:           rel.Member,
		Topic:            rel.Topic,
		Decision:         rel.Decision,
		Path:             rel.Path,
		External:         rel.External,
		ProducerTeam:     rel.ProducerTeam,
		TargetTeam:       rel.TargetTeam,
		SourcePath:       rel.Source.Path,
		Line:             rel.Source.Line,
		RuntimePath:      runtimePathForGraphRelationship(rel, runtime),
		AcceptableFields: registry.AcceptableRuntimeFields(rel.Kind),
	}
	if spec, ok := registry.Spec(rel.Kind); ok {
		diff.Suggestions = spec.GraphSuggestions(diff)
		diff.Detail = detailForGraphRelationshipMissingRuntime(diff, spec)
	}
	return diff
}

func operatingGraphDiffFromRuntimeRel(rel OperatingRelationship, block OperatingGraphBlock, registry OperatingRelationshipRegistry) OperatingGraphContractDiff {
	graphKind := registry.GraphKindForRuntime(rel.Kind)
	diff := OperatingGraphContractDiff{
		Kind:         "runtime_relationship_missing_in_graph",
		Relationship: string(graphKind),
		Team:         rel.Team,
		Member:       rel.Member,
		Topic:        rel.Topic,
		Decision:     rel.Decision,
		Path:         rel.Path,
		External:     rel.External,
		ProducerTeam: rel.ProducerTeam,
		TargetTeam:   rel.TargetTeam,
		SourcePath:   block.Source.Path,
		Line:         block.Source.FenceLine,
		RuntimePath:  rel.Source.Path,
	}
	if spec, ok := registry.Spec(graphKind); ok {
		diff.Suggestions = spec.RuntimeSuggestions(diff)
		diff.Detail = detailForRuntimeRelationshipMissingGraph(diff, spec)
	}
	return diff
}

func runtimeRelationshipAsGraphRelationship(rel OperatingRelationship) OperatingRelationshipKind {
	return DefaultOperatingRelationshipRegistry().GraphKindForRuntime(rel.Kind)
}

func runtimePathForGraphRelationship(rel OperatingRelationship, runtime OperatingGraphRuntime) string {
	if rel.Member != "" {
		return runtimeMemberTopicsPath(runtime, rel.Team, rel.Member)
	}
	if rel.Kind == operatingRelCrossTeamOutput || rel.Kind == operatingRelExternalProducerIntake || rel.Kind == operatingRelUniversalSourceWrite {
		if match := matchingRuntimeRelationshipPath(rel, runtime); match != "" {
			return match
		}
	}
	return runtimeTeamPath(runtime, rel.Team)
}

func matchingRuntimeRelationshipPath(rel OperatingRelationship, runtime OperatingGraphRuntime) string {
	matcher := NewOperatingRelationshipMatcher()
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
			graphRel.ProducerTeam == runtimeRel.ProducerTeam &&
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

func detailForGraphRelationshipMissingRuntime(diff OperatingGraphContractDiff, spec OperatingRelationshipSpec) string {
	return fmt.Sprintf("%s says %s. Runtime has no matching declaration.", relationshipLocation(diff), spec.Statement(diff))
}

func detailForRuntimeRelationshipMissingGraph(diff OperatingGraphContractDiff, spec OperatingRelationshipSpec) string {
	return fmt.Sprintf("%s declares %s. The contract graph does not show a matching relationship.", diff.RuntimePath, spec.Statement(diff))
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
