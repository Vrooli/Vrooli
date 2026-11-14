package brands_test

import (
	"context"
	"testing"

	"brand-manager/handlers/brands"
	internalbrands "brand-manager/internal/brands"
	mocks "brand-manager/internal/brands/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
	brandsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands/brands_v1connect"
)

// newClient wires the real internal service over in-memory fake repos behind the
// generated Connect handler, exercising handler + adapter + service together.
func newClient(t *testing.T) (brandsconnect.BrandsServiceClient, *mocks.FakeRepository) {
	t.Helper()
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}
	logger, _ := connectxtest.NewLogger(t)
	svc := internalbrands.NewService(repo, versions, logger)
	path, handler := brandsconnect.NewBrandsServiceHandler(brands.NewConnectHandler(brands.Deps{Service: svc, Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return brandsconnect.NewBrandsServiceClient(server.Client(), server.URL), repo
}

func TestConnect_CreateThenGet(t *testing.T) {
	client, _ := newClient(t)
	ctx := context.Background()

	created, err := client.CreateBrand(ctx, connect.NewRequest(&brandsv1.CreateBrandRequest{
		Name:     "Acme",
		Identity: &brandsv1.Identity{DisplayName: "Acme Inc"},
		Colors:   &brandsv1.Colors{Primary: "#fff"},
	}))
	require.NoError(t, err)
	require.NotEmpty(t, created.Msg.Brand.Id)
	require.Equal(t, int32(1), created.Msg.Brand.Version)
	require.Equal(t, "Acme Inc", created.Msg.Brand.Identity.DisplayName)

	got, err := client.GetBrand(ctx, connect.NewRequest(&brandsv1.GetBrandRequest{Id: created.Msg.Brand.Id}))
	require.NoError(t, err)
	require.Equal(t, "Acme", got.Msg.Brand.Name)
	require.Equal(t, "#fff", got.Msg.Brand.Colors.Primary)
}

func TestConnect_GetTokensProjectsBrandColors(t *testing.T) {
	client, _ := newClient(t)
	ctx := context.Background()

	created, err := client.CreateBrand(ctx, connect.NewRequest(&brandsv1.CreateBrandRequest{
		Name: "Acme",
		Colors: &brandsv1.Colors{
			Primary: "#112233", Secondary: "#445566", Accent: "#778899",
			Background: "#000000", Surface: "#ffffff", Text: "#fefefe", Error: "#ff0000",
		},
	}))
	require.NoError(t, err)

	got, err := client.GetTokens(ctx, connect.NewRequest(&brandsv1.GetTokensRequest{BrandId: created.Msg.Brand.Id}))
	require.NoError(t, err)
	require.Equal(t, []string{"$brand.primary", "$brand.secondary", "$brand.accent", "$brand.background", "$brand.surface", "$brand.text", "$brand.error"}, tokenNames(got.Msg.Tokens))
	require.Equal(t, []string{"#112233", "#445566", "#778899", "#000000", "#ffffff", "#fefefe", "#ff0000"}, tokenValues(got.Msg.Tokens))
}

func tokenNames(tokens []*brandsv1.Token) []string {
	values := make([]string, 0, len(tokens))
	for _, token := range tokens {
		values = append(values, token.Name)
	}
	return values
}

func tokenValues(tokens []*brandsv1.Token) []string {
	values := make([]string, 0, len(tokens))
	for _, token := range tokens {
		values = append(values, token.Value)
	}
	return values
}

func TestConnect_CreateRejectsEmptyName(t *testing.T) {
	client, _ := newClient(t)
	_, err := client.CreateBrand(context.Background(), connect.NewRequest(&brandsv1.CreateBrandRequest{Name: "  "}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnect_GetNotFound(t *testing.T) {
	client, _ := newClient(t)
	_, err := client.GetBrand(context.Background(), connect.NewRequest(&brandsv1.GetBrandRequest{Id: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestConnect_UpdateVersionConflict(t *testing.T) {
	client, repo := newClient(t)
	repo.Seed(internalbrands.Brand{ID: "b1", Name: "Acme", Version: 3})

	_, err := client.UpdateBrand(context.Background(), connect.NewRequest(&brandsv1.UpdateBrandRequest{
		Id:              "b1",
		Description:     "x",
		ExpectedVersion: 2,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestConnect_DeleteIsIdempotent(t *testing.T) {
	client, _ := newClient(t)
	// Deleting a brand that never existed still succeeds.
	_, err := client.DeleteBrand(context.Background(), connect.NewRequest(&brandsv1.DeleteBrandRequest{Id: "ghost"}))
	require.NoError(t, err)
}

func TestConnect_UpdatePartialMergeAndVersions(t *testing.T) {
	client, _ := newClient(t)
	ctx := context.Background()

	created, err := client.CreateBrand(ctx, connect.NewRequest(&brandsv1.CreateBrandRequest{
		Name:     "Acme",
		Identity: &brandsv1.Identity{DisplayName: "Acme Inc", Tagline: "old"},
	}))
	require.NoError(t, err)
	id := created.Msg.Brand.Id

	updated, err := client.UpdateBrand(ctx, connect.NewRequest(&brandsv1.UpdateBrandRequest{
		Id:       id,
		Identity: &brandsv1.Identity{Tagline: "new"},
	}))
	require.NoError(t, err)
	require.Equal(t, int32(2), updated.Msg.Brand.Version)
	require.Equal(t, "new", updated.Msg.Brand.Identity.Tagline)
	require.Equal(t, "Acme Inc", updated.Msg.Brand.Identity.DisplayName, "display name survives a tagline-only update")

	versions, err := client.ListBrandVersions(ctx, connect.NewRequest(&brandsv1.ListBrandVersionsRequest{BrandId: id}))
	require.NoError(t, err)
	require.Len(t, versions.Msg.Versions, 2, "create + update each snapshot a version")
	require.Equal(t, int32(2), versions.Msg.Versions[0].Version, "newest-first")
}
