package memberflow

import (
	"testing"

	"prompt-manager/teamcontract"
)

func TestOperatingActorResolverExpandsDeclaredMarketingAliases(t *testing.T) {
	resolver := NewOperatingActorResolver(OperatingGraphMetadata{Extra: map[string]string{
		"actor_alias.advertisers": "group:advertisers",
		"actor_group.advertisers": "member:oss-advertiser, member:subscription-advertiser",
	}})

	refs := resolver.Resolve("marketing-crew", OperatingGraphRuntime{}, "advertisers, operator")
	if len(refs) != 2 {
		t.Fatalf("refs=%+v", refs)
	}
	advertisers := findOperatingActorReference(refs, OperatingActorKindGroup, "advertisers")
	if advertisers == nil {
		t.Fatalf("advertisers alias not resolved as group: %+v", refs)
	}
	if findOperatingActorReference(refs, OperatingActorKindExternal, "operator") == nil {
		t.Fatalf("operator alias not resolved as external: %+v", refs)
	}

	expanded := resolver.Expand("marketing-crew", OperatingGraphRuntime{}, *advertisers)
	if len(expanded) != 2 || expanded[0].Value != "oss-advertiser" || expanded[1].Value != "subscription-advertiser" {
		t.Fatalf("advertisers expansion=%+v", expanded)
	}
}

func TestOperatingActorResolverInfersAliasesFromGraphActorLabels(t *testing.T) {
	graph := OperatingGraph{Nodes: []OperatingGraphNode{
		{Kind: OperatingGraphNodeKindMember, Value: "brand-manager", Display: "Brand Manager"},
		{Kind: OperatingGraphNodeKindExternal, Value: "vision-walk", Display: "Vision walk"},
		{Kind: OperatingGraphNodeKindTeam, Value: "monetization", Display: "Monetization team"},
		{Kind: OperatingGraphNodeKindProcess, Value: "learning-synthesis", Display: "Learning synthesis"},
		{Kind: OperatingGraphNodeKindTopic, Value: "research-inbox/*", Display: "research-inbox/*"},
	}}
	resolver := NewOperatingActorResolver(OperatingGraphMetadata{}, graph)

	refs := resolver.Resolve("marketing-crew", OperatingGraphRuntime{}, "Brand Manager, brand-manager, Vision walk, monetization team, Learning synthesis")
	if findOperatingActorReference(refs, OperatingActorKindMember, "brand-manager") == nil {
		t.Fatalf("brand manager label/value aliases not resolved: %+v", refs)
	}
	if findOperatingActorReference(refs, OperatingActorKindExternal, "vision-walk") == nil {
		t.Fatalf("vision walk label alias not resolved: %+v", refs)
	}
	if findOperatingActorReference(refs, OperatingActorKindTeam, "monetization") == nil {
		t.Fatalf("monetization team label alias not resolved: %+v", refs)
	}
	if findOperatingActorReference(refs, OperatingActorKindProcess, "learning-synthesis") == nil {
		t.Fatalf("process label alias not resolved: %+v", refs)
	}
	if findOperatingActorReference(refs, OperatingActorKindUnknown, "research-inbox/*") != nil {
		t.Fatalf("topic node should not become an actor alias: %+v", refs)
	}
}

func TestOperatingActorResolverExplicitAliasOverridesGraphLabel(t *testing.T) {
	graph := OperatingGraph{Nodes: []OperatingGraphNode{
		{Kind: OperatingGraphNodeKindMember, Value: "brand-manager", Display: "Advertisers"},
	}}
	resolver := NewOperatingActorResolver(OperatingGraphMetadata{Extra: map[string]string{
		"actor_alias.advertisers": "group:advertisers",
		"actor_group.advertisers": "member:oss-advertiser, member:subscription-advertiser",
	}}, graph)

	refs := resolver.Resolve("marketing-crew", OperatingGraphRuntime{}, "Advertisers")
	if findOperatingActorReference(refs, OperatingActorKindGroup, "advertisers") == nil {
		t.Fatalf("explicit alias should override inferred graph label alias: %+v", refs)
	}
}

func findOperatingActorReference(refs []OperatingActorReference, kind OperatingActorKind, value string) *OperatingActorReference {
	for i := range refs {
		if refs[i].Kind == kind && refs[i].Value == value {
			return &refs[i]
		}
	}
	return nil
}

func TestDefaultOperatingActorResolverExpandsAnyMarketingMemberFromTeamContract(t *testing.T) {
	resolver := NewOperatingActorResolver(OperatingGraphMetadata{Extra: map[string]string{
		"actor_group.marketing-members": "team-members",
	}})
	ref := OperatingActorReference{Kind: OperatingActorKindGroup, Value: "marketing-members", Raw: "any marketing member"}
	runtime := OperatingGraphRuntime{
		Contracts: TeamContractRegistry{
			"team-a": {TeamID: "team-a", Contract: &teamcontract.OperatingContract{
				Members: map[string]teamcontract.MemberContract{
					"alpha": {},
					"beta":  {},
				},
			}},
		},
	}

	expanded := resolver.Expand("team-a", runtime, ref)
	if len(expanded) != 2 || expanded[0].Value != "alpha" || expanded[1].Value != "beta" {
		t.Fatalf("team-aware marketing member expansion=%+v", expanded)
	}
}

func TestDefaultOperatingActorResolverDoesNotKnowMarketingAliases(t *testing.T) {
	resolver := DefaultOperatingActorResolver{}
	refs := resolver.Resolve("team-a", OperatingGraphRuntime{}, "advertisers")
	if len(refs) != 1 || refs[0].Kind != OperatingActorKindUnknown {
		t.Fatalf("default resolver should not know marketing aliases: %+v", refs)
	}
}
