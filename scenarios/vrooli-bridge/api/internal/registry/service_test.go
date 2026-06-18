package registry_test

import (
	"context"
	"testing"

	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/registry/mocks"

	"github.com/stretchr/testify/require"
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

// [REQ:BRG-P0-001] Register normalises capability/scope lists (trim + drop empty).
func TestService_Register_NormalisesLists(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := registry.NewService(repo)
	n, err := svc.Register(context.Background(), registry.RegisterInput{
		Name: "a", OS: "linux", Arch: "amd64",
		Capabilities: []string{" scenario test* ", "", "  "},
		Scopes:       []string{"registry list", " "},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"scenario test*"}, n.Capabilities)
	require.Equal(t, []string{"registry list"}, n.Scopes)
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
