package protoconv

import (
	"reflect"
	"testing"

	"agent-manager/internal/domain"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type policyLists struct {
	allowedTools, deniedTools, allowedPaths, deniedPaths   []string
	declaredScopes, skillPack                              []string
	allowGlobs, allowExtensions, denyGlobs, denyExtensions []string
}

func domainProfilePolicyLists(p *domain.AgentProfile) policyLists {
	return policyLists{p.AllowedTools, p.DeniedTools, p.AllowedPaths, p.DeniedPaths, p.DeclaredScopes, p.SkillPack,
		p.SandboxConfig.Acceptance.Allow.PathGlobs, p.SandboxConfig.Acceptance.Allow.Extensions,
		p.SandboxConfig.Acceptance.Deny.PathGlobs, p.SandboxConfig.Acceptance.Deny.Extensions}
}

func protoProfilePolicyLists(p *pb.AgentProfile) policyLists {
	return policyLists{p.AllowedTools, p.DeniedTools, p.AllowedPaths, p.DeniedPaths, p.DeclaredScopes, p.SkillPack,
		p.SandboxConfig.Acceptance.Allow.PathGlobs, p.SandboxConfig.Acceptance.Allow.Extensions,
		p.SandboxConfig.Acceptance.Deny.PathGlobs, p.SandboxConfig.Acceptance.Deny.Extensions}
}

func clonePolicyLists(lists policyLists) policyLists {
	return policyLists{
		allowedTools: append([]string(nil), lists.allowedTools...), deniedTools: append([]string(nil), lists.deniedTools...),
		allowedPaths: append([]string(nil), lists.allowedPaths...), deniedPaths: append([]string(nil), lists.deniedPaths...),
		declaredScopes: append([]string(nil), lists.declaredScopes...), skillPack: append([]string(nil), lists.skillPack...),
		allowGlobs: append([]string(nil), lists.allowGlobs...), allowExtensions: append([]string(nil), lists.allowExtensions...),
		denyGlobs: append([]string(nil), lists.denyGlobs...), denyExtensions: append([]string(nil), lists.denyExtensions...),
	}
}

func mutatePolicyLists(lists ...[]string) {
	for _, list := range lists {
		if len(list) > 0 {
			list[0] = "mutated"
		}
	}
}

func TestAgentProfileConversionsDoNotAliasMutablePolicyLists(t *testing.T) {
	newDomainProfile := func() *domain.AgentProfile {
		return &domain.AgentProfile{
			AllowedTools: []string{"read", "write"}, DeniedTools: []string{"shell"},
			AllowedPaths: []string{"src/**"}, DeniedPaths: []string{"secrets/**"},
			DeclaredScopes: []string{"task:read"}, SkillPack: []string{"review"},
			SandboxConfig: &domain.SandboxConfig{Acceptance: domain.SandboxAcceptanceConfig{
				Allow: domain.SandboxFileCriteria{PathGlobs: []string{"src/**"}, Extensions: []string{".go"}},
				Deny:  domain.SandboxFileCriteria{PathGlobs: []string{"secrets/**"}, Extensions: []string{".key"}},
			}},
		}
	}
	newProtoProfile := func() *pb.AgentProfile {
		return &pb.AgentProfile{
			AllowedTools: []string{"read", "write"}, DeniedTools: []string{"shell"},
			AllowedPaths: []string{"src/**"}, DeniedPaths: []string{"secrets/**"},
			DeclaredScopes: []string{"task:read"}, SkillPack: []string{"review"},
			SandboxConfig: &pb.SandboxConfig{Acceptance: &pb.SandboxAcceptanceConfig{
				Allow: &pb.SandboxFileCriteria{PathGlobs: []string{"src/**"}, Extensions: []string{".go"}},
				Deny:  &pb.SandboxFileCriteria{PathGlobs: []string{"secrets/**"}, Extensions: []string{".key"}},
			}},
		}
	}

	t.Run("domain to proto", func(t *testing.T) {
		source := newDomainProfile()
		converted := AgentProfileToProto(source)
		want := clonePolicyLists(protoProfilePolicyLists(converted))
		before := domainProfilePolicyLists(source)
		if !reflect.DeepEqual(want, before) {
			t.Fatalf("proto conversion lost or changed policy lists: got %+v, want %+v", want, before)
		}
		mutatePolicyLists(before.allowedTools, before.deniedTools, before.allowedPaths, before.deniedPaths,
			before.declaredScopes, before.skillPack, before.allowGlobs, before.allowExtensions, before.denyGlobs, before.denyExtensions)
		if got := protoProfilePolicyLists(converted); !reflect.DeepEqual(got, want) {
			t.Fatalf("converted policy lists alias domain input: got %+v, want %+v", got, want)
		}
	})

	t.Run("proto to domain", func(t *testing.T) {
		source := newProtoProfile()
		converted := AgentProfileFromProto(source)
		want := clonePolicyLists(domainProfilePolicyLists(converted))
		before := protoProfilePolicyLists(source)
		if !reflect.DeepEqual(want, before) {
			t.Fatalf("domain conversion lost or changed policy lists: got %+v, want %+v", want, before)
		}
		mutatePolicyLists(before.allowedTools, before.deniedTools, before.allowedPaths, before.deniedPaths,
			before.declaredScopes, before.skillPack, before.allowGlobs, before.allowExtensions, before.denyGlobs, before.denyExtensions)
		if got := domainProfilePolicyLists(converted); !reflect.DeepEqual(got, want) {
			t.Fatalf("converted policy lists alias proto input: got %+v, want %+v", got, want)
		}
	})
}

type runConfigLists struct {
	allowedTools, deniedTools, allowedPaths, deniedPaths []string
}

func TestRunConfigConversionsDoNotAliasMutablePolicyLists(t *testing.T) {
	newDomainConfig := func() *domain.RunConfig {
		return &domain.RunConfig{AllowedTools: []string{"read", "write"}, DeniedTools: []string{"shell"}, AllowedPaths: []string{"src/**"}, DeniedPaths: []string{"secrets/**"}}
	}
	newProtoConfig := func() *pb.RunConfig {
		return &pb.RunConfig{AllowedTools: []string{"read", "write"}, DeniedTools: []string{"shell"}, AllowedPaths: []string{"src/**"}, DeniedPaths: []string{"secrets/**"}}
	}
	t.Run("domain to proto", func(t *testing.T) {
		source := newDomainConfig()
		converted := RunConfigToProto(source)
		want := runConfigLists{[]string{"read", "write"}, []string{"shell"}, []string{"src/**"}, []string{"secrets/**"}}
		source.AllowedTools[0], source.DeniedTools[0], source.AllowedPaths[0], source.DeniedPaths[0] = "mutated", "mutated", "mutated", "mutated"
		if got := (runConfigLists{converted.AllowedTools, converted.DeniedTools, converted.AllowedPaths, converted.DeniedPaths}); !reflect.DeepEqual(got, want) {
			t.Fatalf("proto policy lists changed with domain input: got %+v, want %+v", got, want)
		}
	})
	t.Run("proto to domain", func(t *testing.T) {
		source := newProtoConfig()
		converted := RunConfigFromProto(source)
		want := runConfigLists{[]string{"read", "write"}, []string{"shell"}, []string{"src/**"}, []string{"secrets/**"}}
		source.AllowedTools[0], source.DeniedTools[0], source.AllowedPaths[0], source.DeniedPaths[0] = "mutated", "mutated", "mutated", "mutated"
		if got := (runConfigLists{converted.AllowedTools, converted.DeniedTools, converted.AllowedPaths, converted.DeniedPaths}); !reflect.DeepEqual(got, want) {
			t.Fatalf("domain policy lists changed with proto input: got %+v, want %+v", got, want)
		}
	})
}
