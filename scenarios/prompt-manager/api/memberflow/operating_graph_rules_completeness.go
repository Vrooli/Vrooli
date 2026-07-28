package memberflow

import (
	"fmt"
)

type graphDeclaredMemberMissingRule struct{}

func (r graphDeclaredMemberMissingRule) ID() string { return "graph_declared_member_missing" }
func (r graphDeclaredMemberMissingRule) Group() RuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredMemberMissingRule) DefaultSeverity() Severity { return SeverityError }
func (r graphDeclaredMemberMissingRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) == "contract"
}

func (r graphDeclaredMemberMissingRule) Check(ctx RuleContext) []OperatingGraphFinding {
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
func (r graphDeclaredRuntimeRelationshipMissingRule) Group() RuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredRuntimeRelationshipMissingRule) DefaultSeverity() Severity { return r.severity }
func (r graphDeclaredRuntimeRelationshipMissingRule) AppliesTo(ctx RuleContext) bool {
	return string(ctx.Block.Metadata.Mode) == string(OperatingGraphModeContract)
}

func (r graphDeclaredRuntimeRelationshipMissingRule) Check(ctx RuleContext) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	for _, target := range r.targets {
		findings = append(findings, declaredRuntimeRelationshipMissingFindings(ctx, r, target.kind, target.label)...)
	}
	return findings
}

func graphDeclaredRuntimeRelationshipMissingRules(registry OperatingRelationshipRegistry) []Rule {
	rulesByID := map[string]*graphDeclaredRuntimeRelationshipMissingRule{}
	var order []string
	for _, spec := range registry.Specs() {
		if !spec.RuntimeOnlyCompletes {
			continue
		}
		for _, target := range spec.CompletenessTargets {
			ruleID := target.RuleID
			rule, ok := rulesByID[ruleID]
			if !ok {
				rule = &graphDeclaredRuntimeRelationshipMissingRule{
					id:       ruleID,
					severity: spec.ValidationSeverity,
				}
				rulesByID[ruleID] = rule
				order = append(order, ruleID)
			}
			rule.targets = append(rule.targets, declaredRuntimeRelationshipTarget{
				kind:  target.Kind,
				label: target.Label,
			})
		}
	}
	out := make([]Rule, 0, len(order))
	for _, id := range order {
		out = append(out, *rulesByID[id])
	}
	return out
}

func declaredRuntimeRelationshipMissingFindings(ctx RuleContext, rule Rule, kind OperatingRelationshipKind, label string) []OperatingGraphFinding {
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
		if rel.ProducerTeam != "" {
			if _, ok := ctx.Index.Node("team", rel.ProducerTeam); !ok {
				continue
			}
		}
		if rel.External != "" && rel.Member == "" && rel.Kind != operatingRelUniversalSourceWrite {
			if _, ok := ctx.Index.Node("external", rel.External); !ok {
				continue
			}
		}
		if rel.Member == "" && rel.ProducerTeam == "" && rel.TargetTeam == "" && rel.External == "" {
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
	case operatingRelUniversalSourceWrite:
		return fmt.Sprintf("%s peer team %q topic %q is missing from the contract graph", label, rel.ProducerTeam, rel.Topic)
	default:
		return fmt.Sprintf("%s %s/%s topic %q is missing from the contract graph", label, rel.Team, rel.Member, rel.Topic)
	}
}
