package main

import (
	"context"
	"errors"
	"log"

	"brand-manager/internal/candidates"
	"brand-manager/internal/imagetools"
	"brand-manager/internal/styles"

	internalassets "brand-manager/internal/assets"
	internalbrands "brand-manager/internal/brands"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
)

// brandExistsResolver satisfies assets.BrandResolver over the brands repository.
type brandExistsResolver struct{ repo internalbrands.Repository }

func (r brandExistsResolver) BrandExists(ctx context.Context, brandID string) (bool, error) {
	_, err := r.repo.Get(ctx, brandID)
	if err != nil {
		var notFound internalbrands.ErrBrandNotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// brandMarkSetter records a picked mark on the brand.
type brandMarkSetter struct{ svc internalbrands.Service }

func (s brandMarkSetter) SetMarkAsset(ctx context.Context, brandID, assetID string) error {
	_, err := s.svc.Update(ctx, internalbrands.UpdateInput{ID: brandID, MarkAssetID: assetID})
	return err
}

// buildCandidatesService wires candidates over assets + image-tools + brands.
// It returns the assets and brands services too: the Logo page's read-only HTTP
// endpoints (candidate thumbnails and the brand mark) need the same instances,
// and rebuilding them would duplicate adapters.
func buildCandidatesService(db *database.RoutedDB, clk schedule.Clock, assetsDir string, logger *log.Logger) (*candidates.Service, internalassets.Service, internalbrands.Service) {
	brandRepo := internalbrands.NewSQLiteRepository(db, clk)
	brandVersions := internalbrands.NewSQLiteVersionRepository(db, clk)
	brandSvc := internalbrands.NewService(brandRepo, brandVersions, logger)

	assetsRepo := internalassets.NewSQLiteRepository(db, clk)
	assetsSvc := internalassets.NewService(assetsRepo, internalassets.NewFSBlobStore(assetsDir), brandExistsResolver{repo: brandRepo}, logger)

	store := candidates.NewSQLiteStore(db, clk.Now)
	return candidates.NewService(store, assetsSvc, imagetools.NewClient(), brandMarkSetter{svc: brandSvc}), assetsSvc, brandSvc
}

// buildStylesService constructs the styles service and applies the seed.
func buildStylesService(db *database.RoutedDB, clk schedule.Clock) (*styles.Service, error) {
	svc := styles.NewService(styles.NewSQLiteStore(db, clk.Now))
	if err := svc.Seed(context.Background()); err != nil {
		return nil, err
	}
	return svc, nil
}
