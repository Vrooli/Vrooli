// Package generation is the HTTP/Connect boundary for the generation domain: it
// adapts the generated GenerationService handler onto the transport-agnostic
// internal/generation service and exports the domain's static metadata
// (Endpoints) for the modules registry.
//
// Generation owns no table, so this package exports no Schema(); it composes the
// brands and assets domains behind two adapters defined here — the composition
// root is the only place those domains meet generation.
package generation

import (
	"context"
	"errors"
	"log"

	"brand-manager/internal/clock"
	"brand-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	generationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation/generation_v1connect"

	internalassets "brand-manager/internal/assets"
	internalbrands "brand-manager/internal/brands"
	internalgeneration "brand-manager/internal/generation"
)

// Module returns the generation domain's contribution to the API: the generated
// Connect-RPC GenerationService handler over the provider chain (built from the
// environment) plus brand/asset adapters. blobBaseDir is the assets storage root
// (shared with the assets module) so generated images land beside uploaded ones.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger, blobBaseDir string) module.Module {
	brandsSvc := internalbrands.NewService(
		internalbrands.NewSQLiteRepository(db, clk),
		internalbrands.NewSQLiteVersionRepository(db, clk),
		logger,
	)
	assetsSvc := internalassets.NewService(
		internalassets.NewSQLiteRepository(db, clk),
		internalassets.NewFSBlobStore(blobBaseDir),
		assetBrandResolver{brands: internalbrands.NewSQLiteRepository(db, clk)},
		logger,
	)

	svc := internalgeneration.NewService(
		internalgeneration.NewChainFromEnv(),
		brandStore{brands: brandsSvc},
		assetStore{assets: assetsSvc},
		logger,
	)
	connectPath, connectHandler := generationconnect.NewGenerationServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "generation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// brandStore adapts the brands service onto the generation BrandStore seam.
// Reading and applying go through the brands service so partial-merge and
// version snapshots happen exactly as a normal UpdateBrand would.
type brandStore struct {
	brands internalbrands.Service
}

func (s brandStore) Get(ctx context.Context, brandID string) (internalgeneration.BrandView, error) {
	b, err := s.brands.Get(ctx, brandID)
	if err != nil {
		var notFound internalbrands.ErrBrandNotFound
		if errors.As(err, &notFound) {
			return internalgeneration.BrandView{}, internalgeneration.ErrBrandNotFound{ID: brandID}
		}
		return internalgeneration.BrandView{}, err
	}
	return internalgeneration.BrandView{
		ID:           b.ID,
		Name:         b.Name,
		Description:  b.Description,
		Notes:        b.Notes,
		PrimaryColor: b.Colors.Primary,
		Version:      b.Version,
	}, nil
}

func (s brandStore) ApplyElements(ctx context.Context, in internalgeneration.ApplyElementsInput) (int, error) {
	update := internalbrands.UpdateInput{ID: in.BrandID}
	if in.Colors != nil {
		update.Colors = internalbrands.Colors{
			Primary:    in.Colors.Primary,
			Secondary:  in.Colors.Secondary,
			Accent:     in.Colors.Accent,
			Background: in.Colors.Background,
			Surface:    in.Colors.Surface,
			Text:       in.Colors.Text,
			Error:      in.Colors.Error,
		}
	}
	if in.Typography != nil {
		update.Typography = internalbrands.Typography{
			HeadingFont:  in.Typography.HeadingFont,
			BodyFont:     in.Typography.BodyFont,
			MonoFont:     in.Typography.MonoFont,
			BaseFontSize: in.Typography.BaseFontSize,
		}
	}
	if in.Voice != nil {
		update.Voice = internalbrands.Voice{
			Tone:     in.Voice.Tone,
			Style:    in.Voice.Style,
			Keywords: in.Voice.Keywords,
		}
	}
	updated, err := s.brands.Update(ctx, update)
	if err != nil {
		var notFound internalbrands.ErrBrandNotFound
		if errors.As(err, &notFound) {
			return 0, internalgeneration.ErrBrandNotFound{ID: in.BrandID}
		}
		return 0, err
	}
	return updated.Version, nil
}

// assetStore adapts the assets service onto the generation AssetStore seam.
type assetStore struct {
	assets internalassets.Service
}

func (s assetStore) Store(ctx context.Context, in internalgeneration.AssetUpload) (internalgeneration.StoredAsset, error) {
	a, err := s.assets.Upload(ctx, internalassets.UploadInput{
		BrandID:  in.BrandID,
		Filename: in.Filename,
		MimeType: in.MimeType,
		Content:  in.Content,
	})
	if err != nil {
		return internalgeneration.StoredAsset{}, err
	}
	return internalgeneration.StoredAsset{
		ID:       a.ID,
		Filename: a.Filename,
		MimeType: a.MimeType,
		Size:     a.Size,
	}, nil
}

// assetBrandResolver adapts the brands repository onto the assets BrandResolver
// seam (the assets service confirms a brand exists before storing). Defined here
// so the generation module can build a self-contained assets service.
type assetBrandResolver struct {
	brands internalbrands.Repository
}

func (r assetBrandResolver) BrandExists(ctx context.Context, brandID string) (bool, error) {
	_, err := r.brands.Get(ctx, brandID)
	if err != nil {
		var notFound internalbrands.ErrBrandNotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
