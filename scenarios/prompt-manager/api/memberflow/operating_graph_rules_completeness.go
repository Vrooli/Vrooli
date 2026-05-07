package memberflow

import "fmt"

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

type graphDeclaredIntakeMissingRule struct{}

func (r graphDeclaredIntakeMissingRule) ID() string { return "graph_declared_intake_missing" }
func (r graphDeclaredIntakeMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredIntakeMissingRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphDeclaredIntakeMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDeclaredIntakeMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	return declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelTopicIntake, "declared intake")
}

type graphDeclaredRequiredReadMissingRule struct{}

func (r graphDeclaredRequiredReadMissingRule) ID() string {
	return "graph_declared_required_read_missing"
}

func (r graphDeclaredRequiredReadMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredRequiredReadMissingRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphDeclaredRequiredReadMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDeclaredRequiredReadMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	return declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelTopicRequiredRead, "declared required read")
}

type graphDeclaredEvidenceMissingRule struct{}

func (r graphDeclaredEvidenceMissingRule) ID() string { return "graph_declared_evidence_missing" }
func (r graphDeclaredEvidenceMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredEvidenceMissingRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphDeclaredEvidenceMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDeclaredEvidenceMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	return declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelTopicEvidenceConsumed, "declared evidence")
}

type graphDeclaredOutputMissingRule struct{}

func (r graphDeclaredOutputMissingRule) ID() string { return "graph_declared_output_missing" }
func (r graphDeclaredOutputMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredOutputMissingRule) DefaultSeverity() Severity  { return SeverityWarning }
func (r graphDeclaredOutputMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDeclaredOutputMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	return append(
		declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelTopicOutput, "declared output"),
		declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelPOROutput, "declared PoR output")...,
	)
}

type graphDeclaredDecisionOwnedMissingRule struct{}

func (r graphDeclaredDecisionOwnedMissingRule) ID() string {
	return "graph_declared_decision_owned_missing"
}

func (r graphDeclaredDecisionOwnedMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredDecisionOwnedMissingRule) DefaultSeverity() Severity  { return SeverityError }
func (r graphDeclaredDecisionOwnedMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDeclaredDecisionOwnedMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	return declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelDecisionOwned, "declared decision ownership")
}

type graphDeclaredDecisionConsumedMissingRule struct{}

func (r graphDeclaredDecisionConsumedMissingRule) ID() string {
	return "graph_declared_decision_consumed_missing"
}

func (r graphDeclaredDecisionConsumedMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}
func (r graphDeclaredDecisionConsumedMissingRule) DefaultSeverity() Severity { return SeverityError }
func (r graphDeclaredDecisionConsumedMissingRule) AppliesTo(mode string) bool {
	return mode == "contract"
}

func (r graphDeclaredDecisionConsumedMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	return declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelDecisionConsumed, "declared decision consumption")
}

type graphDeclaredCapabilityGapMissingRule struct{}

func (r graphDeclaredCapabilityGapMissingRule) ID() string {
	return "graph_declared_capability_gap_missing"
}

func (r graphDeclaredCapabilityGapMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}

func (r graphDeclaredCapabilityGapMissingRule) DefaultSeverity() Severity {
	return SeverityWarning
}
func (r graphDeclaredCapabilityGapMissingRule) AppliesTo(mode string) bool { return mode == "contract" }
func (r graphDeclaredCapabilityGapMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	return declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelCapabilityGapRaised, "declared capability-gap routing")
}

type graphDeclaredExternalProducerMissingRule struct{}

func (r graphDeclaredExternalProducerMissingRule) ID() string {
	return "graph_declared_external_producer_missing"
}

func (r graphDeclaredExternalProducerMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}

func (r graphDeclaredExternalProducerMissingRule) DefaultSeverity() Severity {
	return SeverityWarning
}

func (r graphDeclaredExternalProducerMissingRule) AppliesTo(mode string) bool {
	return mode == "contract"
}

func (r graphDeclaredExternalProducerMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	return declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelExternalProducer, "declared external producer")
}

type graphDeclaredCrossTeamOutputMissingRule struct{}

func (r graphDeclaredCrossTeamOutputMissingRule) ID() string {
	return "graph_declared_cross_team_output_missing"
}

func (r graphDeclaredCrossTeamOutputMissingRule) Group() OperatingGraphRuleGroup {
	return OperatingRuleGroupCompleteness
}

func (r graphDeclaredCrossTeamOutputMissingRule) DefaultSeverity() Severity {
	return SeverityWarning
}

func (r graphDeclaredCrossTeamOutputMissingRule) AppliesTo(mode string) bool {
	return mode == "contract"
}

func (r graphDeclaredCrossTeamOutputMissingRule) Check(ctx OperatingGraphRuleContext) []OperatingGraphFinding {
	return declaredRuntimeRelationshipMissingFindings(ctx, r, operatingRelCrossTeamOutput, "declared cross-team output")
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
