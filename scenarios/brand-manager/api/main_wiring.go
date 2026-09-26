package main

import (
	"context"
	"errors"
	"log"
	"strings"

	"brand-manager/internal/candidates"
	"brand-manager/internal/imagetools"
	"brand-manager/internal/render"
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

// brandContext resolves a brand by id or slug for candidates: its display name,
// its picked mark, and its container style as a render style.
type brandContext struct {
	brands internalbrands.Service
	styles *styles.Service
}

func (b brandContext) Resolve(ctx context.Context, ref string) (candidates.BrandInfo, error) {
	brand, err := b.find(ctx, strings.TrimSpace(ref))
	if err != nil {
		return candidates.BrandInfo{}, err
	}
	name := brand.Identity.DisplayName
	if strings.TrimSpace(name) == "" {
		name = brand.Name
	}
	info := candidates.BrandInfo{ID: brand.ID, Name: name, Description: brand.Description, MarkAssetID: brand.MarkAssetID}
	if brand.ContainerStyleID != "" && b.styles != nil {
		if st, serr := b.styles.GetStyle(ctx, brand.ContainerStyleID); serr == nil {
			info.Style = renderStyleOf(st)
		}
	}
	return info, nil
}

// find accepts an id, then falls back to an exact slug or name match.
func (b brandContext) find(ctx context.Context, ref string) (internalbrands.Brand, error) {
	brand, err := b.brands.Get(ctx, ref)
	if err == nil {
		return brand, nil
	}
	list, lerr := b.brands.List(ctx, internalbrands.ListFilter{Limit: 1000})
	if lerr != nil {
		return internalbrands.Brand{}, lerr
	}
	for _, c := range list {
		if strings.EqualFold(c.Slug, ref) {
			return c, nil
		}
	}
	for _, c := range list {
		if strings.EqualFold(c.Name, ref) {
			return c, nil
		}
	}
	return internalbrands.Brand{}, err
}

func renderStyleOf(st styles.ContainerStyle) render.Style {
	glow := make([]render.GlowLayer, 0, len(st.Glow))
	for _, g := range st.Glow {
		glow = append(glow, render.GlowLayer{Width: g.Width, Opacity: g.Opacity})
	}
	return render.Style{
		Shape:            st.Shape,
		CornerRatio:      st.CornerRatio,
		BackgroundTop:    st.BackgroundTop,
		BackgroundBottom: st.BackgroundBottom,
		MarkScale:        st.MarkScale,
		MaskableScale:    st.MaskableScale,
		AccentColor:      st.AccentColor,
		Glow:             glow,
	}
}

// buildCandidatesService wires candidates over assets + image-tools + brands +
// styles. It returns the assets and brands services too: the Logo page's
// read-only HTTP endpoints (candidate thumbnails and the brand mark) need the
// same instances, and rebuilding them would duplicate adapters.
func buildCandidatesService(db *database.RoutedDB, clk schedule.Clock, assetsDir string, logger *log.Logger, stylesSvc *styles.Service) (*candidates.Service, internalassets.Service, internalbrands.Service) {
	brandRepo := internalbrands.NewSQLiteRepository(db, clk)
	brandVersions := internalbrands.NewSQLiteVersionRepository(db, clk)
	brandSvc := internalbrands.NewService(brandRepo, brandVersions, logger)

	assetsRepo := internalassets.NewSQLiteRepository(db, clk)
	assetsSvc := internalassets.NewService(assetsRepo, internalassets.NewFSBlobStore(assetsDir), brandExistsResolver{repo: brandRepo}, logger)

	store := candidates.NewSQLiteStore(db, clk.Now)
	svc := candidates.NewService(store, assetsSvc, imagetools.NewClient(), brandMarkSetter{svc: brandSvc}, brandContext{brands: brandSvc, styles: stylesSvc})
	return svc, assetsSvc, brandSvc
}

// buildStylesService constructs the styles service and applies the seed.
func buildStylesService(db *database.RoutedDB, clk schedule.Clock) (*styles.Service, error) {
	svc := styles.NewService(styles.NewSQLiteStore(db, clk.Now))
	if err := svc.Seed(context.Background()); err != nil {
		return nil, err
	}
	return svc, nil
}
