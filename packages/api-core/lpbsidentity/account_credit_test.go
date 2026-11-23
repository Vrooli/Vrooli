package lpbsidentity

import (
	"errors"
	"testing"

	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/principalrole"
)

func TestAccountCreditLifecycleRequiresExactVerifiedServiceRole(t *testing.T) {
	role := AccountCreditLifecycle
	valid := apiidentity.Principal{Kind: apiidentity.ActorService, Subject: "audio-tools-lpbs-account-credit-lifecycle", Audience: role.Audience, Scopes: []string{role.Scope}, Verified: true}
	if source, err := role.Source(valid); err != nil || source != "audio-tools" {
		t.Fatalf("Source(valid) = %q, %v", source, err)
	}
	for name, principal := range map[string]apiidentity.Principal{
		"unverified":     {Kind: apiidentity.ActorService, Subject: valid.Subject, Audience: role.Audience, Scopes: []string{role.Scope}},
		"human":          {Kind: apiidentity.ActorHuman, Subject: valid.Subject, Audience: role.Audience, Scopes: []string{role.Scope}, Verified: true},
		"wrong audience": {Kind: apiidentity.ActorService, Subject: valid.Subject, Audience: "scenario:other", Scopes: []string{role.Scope}, Verified: true},
		"wrong scope":    {Kind: apiidentity.ActorService, Subject: valid.Subject, Audience: role.Audience, Scopes: []string{GatewayCreditInference.Scope}, Verified: true},
		"extra scope":    {Kind: apiidentity.ActorService, Subject: valid.Subject, Audience: role.Audience, Scopes: []string{role.Scope, GatewayCreditInference.Scope}, Verified: true},
		"wrong suffix":   {Kind: apiidentity.ActorService, Subject: "audio-tools-lpbs-ai-inference", Audience: role.Audience, Scopes: []string{role.Scope}, Verified: true},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := role.Source(principal); !errors.Is(err, principalrole.ErrRefused) {
				t.Fatalf("Source(%#v) error = %v, want ErrRefused", principal, err)
			}
		})
	}
}
