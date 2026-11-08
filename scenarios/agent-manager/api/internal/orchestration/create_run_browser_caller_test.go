package orchestration

import (
	"context"
	"testing"
	"time"

	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
)

func TestCreateRunAcceptsVerifiedBrowserPrincipalWithoutBearer(t *testing.T) {
	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "true")
	for _, source := range []coreidentity.AuthSource{coreidentity.SourceCloudflareAccess, coreidentity.SourcePersonalLocal} {
		t.Run(string(source), func(t *testing.T) {
			o, _, effects, req, _, owner := auth01CallerFixture(t)
			req.OwnerToken = ""
			owner.Source = source
			ctx := coreidentity.WithPrincipal(t.Context(), owner)
			if err := o.authenticateCreateRunCaller(ctx, &req); err != nil {
				t.Fatal(err)
			}
			if req.caller == nil || req.caller.Kind != "human" || req.OwnerSubject != owner.Subject || req.OwnerExpiresAt == nil || !req.OwnerExpiresAt.Equal(owner.ExpiresAt) || !sameCreateRunScopeCeiling(req.OwnerScopes, owner.Scopes) {
				t.Fatalf("browser authority not retained: %+v", req.caller)
			}
			if err := o.revalidateCreateRunCaller(ctx, req); err != nil {
				t.Fatal(err)
			}
			assertAuth01NoEffects(t, effects)
		})
	}
}

func TestCreateRunRejectsInvalidBrowserAuthority(t *testing.T) {
	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "true")
	for _, name := range []string{"unverified", "expired", "no-expiry", "read-only", "blank-owner", "blank-bearer", "invalid-bearer", "blank-run", "invalid-run", "changed-owner"} {
		t.Run(name, func(t *testing.T) {
			o, _, effects, req, _, owner := auth01CallerFixture(t)
			req.OwnerToken = ""
			switch name {
			case "unverified":
				owner.Verified = false
			case "expired":
				owner.ExpiresAt = o.now().Add(-time.Second)
			case "no-expiry":
				owner.ExpiresAt = time.Time{}
			case "read-only":
				owner.Scopes = []string{"agent-manager:read"}
			case "blank-owner":
				owner.Subject = " "
			case "blank-bearer":
				req.OwnerToken = " "
			case "invalid-bearer":
				req.OwnerToken = "invalid-fixture-owner"
			case "blank-run":
				req.RunIdentityToken = " "
			case "invalid-run":
				req.RunIdentityToken = "invalid"
			case "changed-owner":
				req.ExpectedOwnerSubject = "another-owner"
			}
			if err := o.authenticateCreateRunCaller(coreidentity.WithPrincipal(t.Context(), owner), &req); err == nil {
				t.Fatal("invalid browser authority accepted")
			}
			assertAuth01NoEffects(t, effects)
		})
	}
}

func TestCreateRunBrowserAndBearerRetainSharedAuthority(t *testing.T) {
	for _, name := range []string{"same-owner", "foreign-owner", "empty-scopes", "expired", "unavailable-verifier"} {
		t.Run(name, func(t *testing.T) {
			o, _, effects, req, _, owner := auth01CallerFixture(t)
			browser := owner
			browser.ExpiresAt = owner.ExpiresAt.Add(time.Hour)
			offered := owner
			offered.Scopes = []string{"agent-manager:write"}
			switch name {
			case "foreign-owner":
				offered.Subject = "another-owner"
			case "empty-scopes":
				offered.Scopes = nil
			case "expired":
				offered.ExpiresAt = o.now().Add(-time.Second)
			}
			o.ownerIdentity = createOwnerVerifier(func(_ context.Context, _ string) (coreidentity.Principal, error) { return offered, nil })
			if name == "unavailable-verifier" {
				o.ownerIdentity = nil
			}
			err := o.authenticateCreateRunCaller(coreidentity.WithPrincipal(t.Context(), browser), &req)
			if name == "same-owner" {
				if err != nil {
					t.Fatal(err)
				}
				if !sameCreateRunScopeCeiling(req.OwnerScopes, offered.Scopes) || req.OwnerExpiresAt == nil || !req.OwnerExpiresAt.Equal(offered.ExpiresAt) {
					t.Fatal("dual proofs widened scope or expiry")
				}
			} else if err == nil {
				t.Fatal("conflicting or unavailable second proof accepted")
			}
			assertAuth01NoEffects(t, effects)
		})
	}
}
