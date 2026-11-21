package inference

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"ai-gateway/internal/privacyconsent"
	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/rolecatalog"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	_ "modernc.org/sqlite"
)

func TestConfidentialRemoteConsentConformanceBeforeProviderDispatch(t *testing.T) {
	db, err := sql.Open("sqlite", "file:typed-consent?mode=memory&cache=shared")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	defer db.Close()
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	store, err := privacyconsent.New(db, func() time.Time { return now })
	require.NoError(t, err)
	principalCtx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "subject-1", Verified: true})

	valid, _, err := store.Grant(context.Background(), "subject-1", "own-key", 24*time.Hour)
	require.NoError(t, err)
	revoked, revokedGrant, err := store.Grant(context.Background(), "subject-1", "own-key", 24*time.Hour)
	require.NoError(t, err)
	require.NoError(t, store.Revoke(context.Background(), "subject-1", revokedGrant.ID))
	expired, _, err := store.Grant(context.Background(), "subject-1", "own-key", time.Minute)
	require.NoError(t, err)
	now = now.Add(time.Hour)

	cases := []struct {
		name    string
		receipt string
		source  string
		subject string
		valid   bool
	}{
		{name: "missing", source: "own-key"},
		{name: "forged", receipt: strings.Repeat("x", 48), source: "own-key"},
		{name: "wrong source", receipt: valid, source: "vrooli-credits"},
		{name: "wrong user", receipt: valid, source: "own-key", subject: "subject-2"},
		{name: "revoked", receipt: revoked, source: "own-key"},
		{name: "expired", receipt: expired, source: "own-key"},
		{name: "valid selected source", receipt: valid, source: "own-key", valid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &providermocks.FakeRunner{
				Results: map[string]providers.Result{
					"resource-openrouter policy resolve --role extract.structured --json": {Stdout: `{"role":"extract.structured","model":"hosted-fixture","locality":"remote"}`},
				},
				DefaultOK: true,
				Default:   providers.Result{Stdout: `{"choices":[{"message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":2,"completion_tokens":1}}`},
			}
			catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
				"extract.structured": {Candidates: []rolecatalog.Candidate{{Provider: providers.ProviderOpenRouter, ResourceRole: "extract.structured", Reason: "hosted"}}},
			}}
			repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner}})
			require.NoError(t, err)
			repository.SetRemoteConsentVerifier(store)
			subject := tc.subject
			if subject == "" {
				subject = "subject-1"
			}
			ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: subject, Verified: true})
			ctx = providers.WithSource(ctx, tc.source)
			ctx = providers.WithPrivacyConsent(ctx, tc.receipt)
			result, runErr := repository.Run(ctx, ProviderRequest{
				PrivacyProvenance: "tests.confidential_user_input", Source: "fixture input", SchemaJSON: `{"type":"object"}`,
				Role: "extract.structured", Profile: sharedv1.Profile_PROFILE_REMOTE_ONLY,
				PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
			})
			dispatches := 0
			for _, command := range runner.Commands {
				if strings.Contains(command.String(), " generate ") {
					dispatches++
				}
			}
			if tc.valid {
				require.NoError(t, runErr)
				require.Equal(t, "openrouter", result.Provider)
				require.Equal(t, 1, dispatches)
			} else {
				require.ErrorIs(t, runErr, ErrUnavailable)
				require.Zero(t, dispatches, "invalid consent must be refused before fake provider dispatch")
			}
		})
	}

	// SECRET remains local-only even with an otherwise valid, source-bound grant.
	runner := &providermocks.FakeRunner{DefaultOK: true}
	catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
		"extract.structured": {Candidates: []rolecatalog.Candidate{{Provider: providers.ProviderOpenRouter, ResourceRole: "extract.structured", Reason: "hosted"}}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner}})
	require.NoError(t, err)
	repository.SetRemoteConsentVerifier(store)
	ctx := providers.WithPrivacyConsent(providers.WithSource(principalCtx, "own-key"), valid)
	_, err = repository.Run(ctx, ProviderRequest{PrivacyProvenance: "tests.secret", Source: "secret fixture", SchemaJSON: `{"type":"object"}`, Role: "extract.structured", Profile: sharedv1.Profile_PROFILE_REMOTE_ONLY, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET})
	require.ErrorIs(t, err, ErrUnavailable)
	for _, command := range runner.Commands {
		require.NotContains(t, command.String(), " generate ")
	}
}
