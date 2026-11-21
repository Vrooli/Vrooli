package sessions

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"
)

func TestValidateAgentRunIdentityCreateRequiresExactLocalReservedServiceShape(t *testing.T) {
	base := CreateInput{RequestedSessionID: uuid.NewString(), Origin: "programmatic", AgentRunIdentityToken: "signed-child-token"}
	service := identity.Principal{Kind: identity.ActorService, Subject: "agent-manager", Verified: true, Source: identity.SourceScenarioAuthenticator}
	ctx := identity.WithPrincipal(context.Background(), service)
	if err := validateAgentRunIdentityCreate(ctx, base); err != nil {
		t.Fatalf("valid service shape rejected: %v", err)
	}
	native := base
	native.AgentType = "codex"
	native.LaunchMode = "codex_app_server"
	native.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server"}`
	native.WorkingDir = "/workspace/goal"
	if err := validateAgentRunIdentityCreate(ctx, native); err != nil {
		t.Fatalf("typed reserved Codex native create rejected: %v", err)
	}
	// Existing independently verified human attribution remains a label and
	// does not replace the service identity required for the child token.
	base.HumanOwnerSubject = "operator@example.test"
	if err := validateAgentRunIdentityCreate(ctx, base); err != nil {
		t.Fatalf("verified human attribution rejected: %v", err)
	}
	for _, tc := range []struct {
		name      string
		principal identity.Principal
		edit      func(*CreateInput)
	}{
		{name: "wrong service", principal: identity.Principal{Kind: identity.ActorService, Subject: "other", Verified: true, Source: identity.SourceScenarioAuthenticator}},
		{name: "unverified service", principal: identity.Principal{Kind: identity.ActorService, Subject: "agent-manager", Source: identity.SourceScenarioAuthenticator}},
		{name: "wrong auth source", principal: identity.Principal{Kind: identity.ActorService, Subject: "agent-manager", Verified: true, Source: identity.SourcePersonalLocal}},
		{name: "human caller", principal: identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Source: identity.SourceScenarioAuthenticator}},
		{name: "unreserved", principal: service, edit: func(in *CreateInput) { in.RequestedSessionID = "" }},
		{name: "remote", principal: service, edit: func(in *CreateInput) { in.TargetID = "node" }},
		{name: "ui origin", principal: service, edit: func(in *CreateInput) { in.Origin = "ui" }},
		{name: "server executed launch", principal: service, edit: func(in *CreateInput) { in.ExecuteLaunchCommand = true }},
		{name: "managed launch without descriptor", principal: service, edit: func(in *CreateInput) { in.LaunchMode = "codex_app_server" }},
		{name: "non-Codex native launch", principal: service, edit: func(in *CreateInput) {
			in.AgentType = "claude"
			in.LaunchMode = "codex_app_server"
			in.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server"}`
		}},
		{name: "shell command with native launch", principal: service, edit: func(in *CreateInput) {
			in.AgentType = "codex"
			in.LaunchMode = "codex_app_server"
			in.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server"}`
			in.LaunchCommand = "echo unsafe"
		}},
		{name: "native launch with shell execution", principal: service, edit: func(in *CreateInput) {
			in.AgentType = "codex"
			in.LaunchMode = "codex_app_server"
			in.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server"}`
			in.ExecuteLaunchCommand = true
		}},
		{name: "native launch with ephemeral backend", principal: service, edit: func(in *CreateInput) {
			in.AgentType = "codex"
			in.LaunchMode = "codex_app_server"
			in.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server"}`
			in.Backend = "ephemeral"
		}},
		{name: "identity credential in descriptor", principal: service, edit: func(in *CreateInput) {
			in.AgentType = "codex"
			in.LaunchMode = "codex_app_server"
			in.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server","environment":{"VROOLI_AGENT_IDENTITY_TOKEN":"forged"}}`
		}},
		{name: "padded token", principal: service, edit: func(in *CreateInput) { in.AgentRunIdentityToken = " token" }},
		{name: "oversize token", principal: service, edit: func(in *CreateInput) { in.AgentRunIdentityToken = strings.Repeat("x", 16385) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			in.AgentRunIdentityToken = "signed-child-token"
			in.RequestedSessionID = uuid.NewString()
			in.Origin = "programmatic"
			in.TargetID = ""
			in.ExecuteLaunchCommand = false
			in.LaunchMode = ""
			in.AgentType = ""
			in.LaunchDescriptorJSON = ""
			in.LaunchCommand = ""
			if tc.edit != nil {
				tc.edit(&in)
			}
			if err := validateAgentRunIdentityCreate(identity.WithPrincipal(context.Background(), tc.principal), in); err == nil {
				t.Fatal("unsafe identity placement accepted")
			}
		})
	}
}
