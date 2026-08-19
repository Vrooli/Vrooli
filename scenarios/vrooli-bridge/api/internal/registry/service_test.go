package registry_test

import (
	"context"
	"strings"
	"testing"

	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/registry/mocks"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scopecatalog"
)

// [REQ:BRG-P0-001] Register validates the required identity fields and trims.
func TestService_Register_Validation(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		in      registry.RegisterInput
		wantErr bool
		field   string
	}{
		{"missing name", registry.RegisterInput{OS: "linux", Arch: "amd64"}, true, "name"},
		{"whitespace name", registry.RegisterInput{Name: "  ", OS: "linux", Arch: "amd64"}, true, "name"},
		{"missing os", registry.RegisterInput{Name: "a", Arch: "amd64"}, true, "os"},
		{"missing arch", registry.RegisterInput{Name: "a", OS: "linux"}, true, "arch"},
		{"valid", registry.RegisterInput{Name: " a ", OS: "linux", Arch: "amd64"}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFakeRepository()
			svc := registry.NewService(repo)
			n, err := svc.Register(ctx, tc.in)
			if tc.wantErr {
				var invalid registry.ErrInvalidNode
				require.ErrorAs(t, err, &invalid)
				require.Equal(t, tc.field, invalid.Field)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "a", n.Name, "name trimmed")
			require.Equal(t, int64(1), repo.CreateCalls.Load())
		})
	}
}

// [REQ:BRG-P0-001] Register normalises self-reported capabilities while owner
// grants retain a strict, non-normalizing grammar.
func TestService_Register_NormalisesCapabilities(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := registry.NewService(repo, registry.WithGrantValidator(registry.NewCatalogGrantValidator(scopecatalog.Catalog{
		Scopes: []scopecatalog.Scope{{Scenario: "demo", Value: "demo:read"}},
	})))
	n, err := svc.Register(context.Background(), registry.RegisterInput{
		Name: "a", OS: "linux", Arch: "amd64",
		Capabilities: []string{" scenario test* ", "", "  "},
		Scopes:       []string{"demo:read"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"scenario test*"}, n.Capabilities)
	require.Equal(t, []string{"demo:read"}, n.Scopes)
}

func TestServiceRejectsCommandNamedAndUnknownGrantsAtRegisterAndUpdate(t *testing.T) {
	catalog := scopecatalog.Catalog{Scopes: []scopecatalog.Scope{
		{Scenario: "web-console", Value: "web-console:read"},
		{Scenario: "web-console", Value: "web-console:write"},
	}}
	valid := registry.NewCatalogGrantValidator(catalog)
	tests := []struct {
		name  string
		scope string
		ok    bool
	}{
		{name: "exact", scope: "web-console:read", ok: true},
		{name: "namespace wildcard", scope: "web-console:*", ok: true},
		{name: "effect wildcard", scope: "*:read", ok: true},
		{name: "universal", scope: "*", ok: true},
		{name: "session transport", scope: "vrooli-bridge:session", ok: true},
		{name: "command named", scope: "scenario status*"},
		{name: "unknown exact", scope: "unknown:read"},
		{name: "unknown namespace wildcard", scope: "unknown:*"},
		{name: "verb suffix wildcard", scope: "web-console:read*"},
		{name: "leading whitespace", scope: " web-console:read"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := mocks.NewFakeRepository()
			svc := registry.NewService(repo, registry.WithGrantValidator(valid))
			created, registerErr := svc.Register(context.Background(), registry.RegisterInput{
				Name: "node", OS: "linux", Arch: "amd64", Scopes: []string{test.scope},
			})
			if test.ok {
				require.NoError(t, registerErr)
				_, updateErr := svc.Update(context.Background(), registry.UpdateInput{ID: created.ID, Name: "node", Scopes: []string{test.scope}})
				require.NoError(t, updateErr)
				return
			}
			var invalid registry.ErrInvalidGrant
			require.ErrorAs(t, registerErr, &invalid)
			require.Equal(t, test.scope, invalid.Scope)
			require.NotEmpty(t, invalid.Reason)

			repo.Seed(registry.Node{ID: "existing", Name: "node", OS: "linux", Arch: "amd64"})
			_, updateErr := svc.Update(context.Background(), registry.UpdateInput{ID: "existing", Name: "node", Scopes: []string{test.scope}})
			require.ErrorAs(t, updateErr, &invalid)
			require.True(t, strings.Contains(updateErr.Error(), test.scope), "typed refusal must name rejected scope: %v", updateErr)
		})
	}
}

func TestService_Update_RequiresIDAndName(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := registry.NewService(repo)

	_, err := svc.Update(context.Background(), registry.UpdateInput{Name: "x"})
	var invalid registry.ErrInvalidNode
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "id", invalid.Field)

	_, err = svc.Update(context.Background(), registry.UpdateInput{ID: "abc"})
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "name", invalid.Field)
}

func TestService_Revoke_RequiresID(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := registry.NewService(repo)
	_, err := svc.Revoke(context.Background(), "   ")
	var invalid registry.ErrInvalidNode
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "id", invalid.Field)
}

// [REQ:BRG-P0-001] Revoke delegates to the repository and propagates not-found.
func TestService_Revoke_NotFound(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := registry.NewService(repo)
	_, err := svc.Revoke(context.Background(), "ghost")
	require.ErrorAs(t, err, &registry.ErrNodeNotFound{})
	require.Equal(t, int64(1), repo.RevokeCalls.Load())
}

func TestService_Remove_RequiresRevocation(t *testing.T) {
	repo := mocks.NewFakeRepository()
	repo.Seed(registry.Node{ID: "active"})
	repo.Seed(registry.Node{ID: "revoked", RevokedAt: repo.Now.Add(1)})
	svc := registry.NewService(repo)
	err := svc.Remove(context.Background(), "active")
	var active registry.ErrNodeActive
	require.ErrorAs(t, err, &active)
	require.NoError(t, svc.Remove(context.Background(), "revoked"))
	_, err = repo.Get(context.Background(), "revoked")
	require.ErrorAs(t, err, &registry.ErrNodeNotFound{})
}
