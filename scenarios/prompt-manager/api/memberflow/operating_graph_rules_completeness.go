package memberflow

import (
	"fmt"
	"strings"
)

type graphDeclaredMemberMissingRule struct{}

func (r graphDeclaredMemberMissingRule) ID() string { return "graph_declared_member_missing" }
func (r graphDeclaredMemberMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredMemberMissingRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphDeclaredMemberMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDeclaredMemberMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, r)
	teamContract := ctx.Runtime.Contracts[ctx.Block.Metadata.Team]
	if teamContract == nil || teamContract.Contract == nil {
		return nil
	}
	var findings []OperatingGraphFinding
	for member := range teamContract.Contract.Members {
		if _, ok := ctx.Index.Node("member", member); ok {
			continue
		}
		f := builder.base(ctx.Block.Source.Path, ctx.Block.Source.FenceLine, fmt.Sprintf("team contract member %q is missing from the contract graph", member))
		f.Member = member
		findings = append(findings, f)
	}
	return findings
}

type graphDeclaredRuntimeRelationshipMissingRule struct {
	id       string
	targets  []declaredRuntimeRelationshipTarget
	severity Severity
}

type declaredRuntimeRelationshipTarget struct {
	kind  OperatingRelationshipKind
	label string
}

func (r graphDeclaredRuntimeRelationshipMissingRule) ID() string { return r.id }
func (r graphDeclaredRuntimeRelationshipMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredRuntimeRelationshipMissingRule) DefaultSeverity() Severity { return r.severity }
func (r graphDeclaredRuntimeRelationshipMissingRule) AppliesTo(mode string) bool {
	return mode == string(OperatingGraphModeContract)
}

func (r graphDeclaredRuntimeRelationshipMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	for _, target := range r.targets {
		findings = append(findings, declaredRuntimeRelationshipMissingFindings(ctx, r, target.kind, target.label)...)
	}
	return findings
}

func graphDeclaredRuntimeRelationshipMissingRules(registry OperatingRelationshipRegistry) []OperatingGraphRule {
	rulesByID := map[string]*graphDeclaredRuntimeRelationshipMissingRule{}
	var order []string
	for _, spec := range registry.Specs() {
		if !spec.RuntimeOnlyCompletes {
			continue
		}
		ruleIDs := splitRelationshipValidationRules(spec.ValidationRule)
		runtimeKinds := spec.RuntimeKinds
		if spec.Kind == operatingRelTopicRead {
			runtimeKinds = []OperatingRelationshipKind{operatingRelTopicIntake, operatingRelTopicRequiredRead, operatingRelTopicEvidenceConsumed}
		}
		for i, ruleID := range ruleIDs {
			if ruleID == "" {
				continue
			}
			rule, ok := rulesByID[ruleID]
			if !ok {
				rule = &graphDeclaredRuntimeRelationshipMissingRule{
					id:       ruleID,
					severity: spec.ValidationSeverity,
				}
				rulesByID[ruleID] = rule
				order = append(order, ruleID)
			}
			var kind OperatingRelationshipKind
			if spec.Kind == operatingRelTopicRead && i < len(runtimeKinds) {
				kind = runtimeKinds[i]
			} else {
				kind = spec.Kind
			}
			rule.targets = append(rule.targets, declaredRuntimeRelationshipTarget{
				kind:  kind,
				label: declaredRuntimeRelationshipLabel(kind),
			})
		}
	}
	out := make([]OperatingGraphRule, 0, len(order))
	for _, id := range order {
		out = append(out, *rulesByID[id])
	}
	return out
}

func splitRelationshipValidationRules(raw string) []string {
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func declaredRuntimeRelationshipLabel(kind OperatingRelationshipKind) string {
	switch kind {
	case operatingRelTopicIntake:
		return "declared intake"
	case operatingRelTopicRequiredRead:
		return "declared required read"
	case operatingRelTopicEvidenceConsumed:
		return "declared evidence"
	case operatingRelTopicOutput:
		return "declared output"
	case operatingRelPOROutput:
		return "declared PoR output"
	case operatingRelDecisionOwned:
		return "declared decision ownership"
	case operatingRelDecisionConsumed:
		return "declared decision consumption"
	case operatingRelCapabilityGapRaised:
		return "declared capability-gap routing"
	case operatingRelExternalProducer:
		return "declared external producer"
	case operatingRelCrossTeamOutput:
		return "declared cross-team output"
	default:
		return "declared relationship"
	}
}

func declaredRuntimeRelationshipMissingFindings(ctx OperatingGraphRuleContext, rule OperatingGraphRule, kind OperatingRelationshipKind, label string) []OperatingGraphFinding {
	builder := NewOperatingFindingBuilder(ctx, rule)
	var findings []OperatingGraphFinding
	for _, rel := range ctx.Index.RuntimeRelationshipsByKind(kind) {
		if rel.Member != "" {
			if _, ok := ctx.Index.Node("member", rel.Member); !ok {
				continue
			}
		}
		if rel.TargetTeam != "" {
			if _, ok := ctx.Index.Node("team", rel.TargetTeam); !ok {
				continue
			}
		}
		if rel.External != "" && rel.Member == "" {
			if _, ok := ctx.Index.Node("external", rel.External); !ok {
				continue
			}
		}
		if rel.Member == "" && rel.TargetTeam == "" && rel.External == "" {
			continue
		}
		if ctx.Matcher.RuntimeShownInGraph(rel, ctx.Index.GraphRelationships) {
			continue
		}
		f := builder.WithRelationship(rel, declaredRuntimeRelationshipMissingDetail(label, rel))
		f.SourcePath = ctx.Block.Source.Path
		f.Line = 0
		findings = append(findings, f)
	}
	return findings
}

func declaredRuntimeRelationshipMissingDetail(label string, rel OperatingRelationship) string {
	switch rel.Kind {
	case operatingRelPOROutput:
		return fmt.Sprintf("%s %s/%s path %q is missing from the contract graph", label, rel.Team, rel.Member, rel.Path)
	case operatingRelDecisionOwned, operatingRelDecisionConsumed, operatingRelCapabilityGapRaised:
		return fmt.Sprintf("%s %s/%s decision %q is missing from the contract graph", label, rel.Team, rel.Member, rel.Decision)
	case operatingRelExternalProducer:
		return fmt.Sprintf("%s %s/%s external %q is missing from the contract graph", label, rel.Team, rel.Member, rel.External)
	case operatingRelCrossTeamOutput:
		return fmt.Sprintf("%s %s/%s topic %q to team %q is missing from the contract graph", label, rel.Team, rel.Member, rel.Topic, rel.TargetTeam)
	default:
		return fmt.Sprintf("%s %s/%s topic %q is missing from the contract graph", label, rel.Team, rel.Member, rel.Topic)
	}
}
