package preview

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	domain "react-component-library/internal/preview"
)

type compositionContractService struct {
	domain.Service
	input domain.Composition
	err   error
}

func (s *compositionContractService) GetCompositionBundle(_ context.Context, input domain.Composition) (domain.CompositionBundle, error) {
	s.input = input
	if s.err != nil {
		return domain.CompositionBundle{}, s.err
	}
	return domain.CompositionBundle{
		Bundle:   domain.Bundle{JS: "export const Card = () => null", SHA256: "bundle-digest", Warnings: []string{"optional dependency omitted"}},
		Revision: input.Revision,
		Gaps:     []domain.CompositionGap{{Region: "content", Code: "asset_missing", Message: "Choose a content component", Required: true}},
	}, nil
}

func TestGetCompositionBundleContract(t *testing.T) {
	t.Run("service unavailable", func(t *testing.T) {
		h := NewConnectHandler(Deps{})
		_, err := h.GetCompositionBundle(context.Background(), connect.NewRequest(&previewv1.GetCompositionBundleRequest{}))
		require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
		require.Contains(t, err.Error(), "composition bundling is unavailable")
	})

	t.Run("service errors map to invalid argument", func(t *testing.T) {
		service := &compositionContractService{err: errors.New("template export is invalid")}
		h := NewConnectHandler(Deps{Service: service})
		_, err := h.GetCompositionBundle(context.Background(), connect.NewRequest(&previewv1.GetCompositionBundleRequest{Revision: "draft"}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.Contains(t, err.Error(), "template export is invalid")
	})

	t.Run("projects request and response", func(t *testing.T) {
		service := &compositionContractService{}
		h := NewConnectHandler(Deps{Service: service})
		request := &previewv1.GetCompositionBundleRequest{
			Revision: "draft-7",
			Template: &previewv1.CompositionAsset{CatalogId: "templates.page", Version: "2.1.0", Export: "Page"},
			Regions: []*previewv1.CompositionRegion{
				{Id: "header", Parent: "root", Slot: []string{"header"}, Required: true, Asset: &previewv1.CompositionAsset{CatalogId: "navigation.menu", Version: "1.0.2", Export: "Menu"}},
				{Id: "content", Parent: "root", Slot: []string{"main"}, Required: false},
			},
		}
		response, err := h.GetCompositionBundle(context.Background(), connect.NewRequest(request))
		require.NoError(t, err)
		require.Equal(t, domain.Composition{Revision: "draft-7", Template: domain.CompositionAsset{CatalogID: "templates.page", Version: "2.1.0", Export: "Page"}, Regions: []domain.CompositionRegion{
			{ID: "header", Parent: "root", Slot: []string{"header"}, Required: true, Asset: &domain.CompositionAsset{CatalogID: "navigation.menu", Version: "1.0.2", Export: "Menu"}},
			{ID: "content", Parent: "root", Slot: []string{"main"}},
		}}, service.input)
		require.Equal(t, "draft-7", response.Msg.Revision)
		require.Equal(t, "export const Card = () => null", response.Msg.Js)
		require.Equal(t, "bundle-digest", response.Msg.Sha256)
		require.Equal(t, []string{"optional dependency omitted"}, response.Msg.Warnings)
		require.Equal(t, "content", response.Msg.Gaps[0].Region)
		require.Equal(t, "asset_missing", response.Msg.Gaps[0].Code)
		require.True(t, response.Msg.Gaps[0].Required)
	})
}
