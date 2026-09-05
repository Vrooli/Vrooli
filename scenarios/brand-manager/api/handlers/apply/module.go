// Package apply is the HTTP/Connect boundary for the apply domain: it adapts the
// generated ApplyService handler onto the transport-agnostic internal/apply
// service and exports the domain's static metadata (Endpoints) for the modules
// registry.
//
// Apply owns no table, so this package exports no Schema(); it composes the
// brands, assets, and assignments domains behind three adapters defined here —
// the composition root is the only place those domains meet apply.
package apply

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"strings"

	"brand-manager/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply/apply_v1connect"

	internalapply "brand-manager/internal/apply"
	internalassets "brand-manager/internal/assets"
	internalassignments "brand-manager/internal/assignments"
	internalbrands "brand-manager/internal/brands"
)

// Module returns the apply domain's contribution to the API: the generated
// Connect-RPC ApplyService handler over the apply service plus brand/asset/
// assignment adapters and a filesystem workspace. scenariosRoot is the directory
// that contains scenario source trees apply writes into; assetsBlobDir is the
// assets storage root (shared with the assets module) so apply reads the same
// uploaded/generated image bytes.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger, scenariosRoot, assetsBlobDir string) module.Module {
	brandsRepo := internalbrands.NewSQLiteRepository(db, clk)
	brandsSvc := internalbrands.NewService(
		brandsRepo,
		internalbrands.NewSQLiteVersionRepository(db, clk),
		logger,
	)
	assetsSvc := internalassets.NewService(
		internalassets.NewSQLiteRepository(db, clk),
		internalassets.NewFSBlobStore(assetsBlobDir),
		assetBrandResolver{brands: brandsRepo},
		logger,
	)
	assignmentsSvc := internalassignments.NewService(
		internalassignments.NewSQLiteRepository(db, clk),
		assignmentBrandResolver{brands: brandsRepo},
		logger,
	)

	svc := internalapply.NewService(
		brandStore{brands: brandsSvc},
		assetStore{assets: assetsSvc},
		assignmentRecorder{assignments: assignmentsSvc},
		internalapply.NewFSWorkspace(scenariosRoot),
		logger,
	)
	connectPath, connectHandler := applyconnect.NewApplyServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "apply",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// brandStore adapts the brands service onto the apply BrandStore seam.
type brandStore struct {
	brands internalbrands.Service
}

func (s brandStore) Get(ctx context.Context, brandID string) (internalapply.BrandView, error) {
	b, err := s.brands.Get(ctx, brandID)
	if err != nil {
		var notFound internalbrands.ErrBrandNotFound
		if errors.As(err, &notFound) {
			return internalapply.BrandView{}, internalapply.ErrBrandNotFound{ID: brandID}
		}
		return internalapply.BrandView{}, err
	}
	return internalapply.BrandView{
		ID:          b.ID,
		Version:     b.Version,
		DisplayName: b.Identity.DisplayName,
		Tagline:     b.Identity.Tagline,
		Colors: internalapply.Colors{
			Primary:    b.Colors.Primary,
			Secondary:  b.Colors.Secondary,
			Accent:     b.Colors.Accent,
			Background: b.Colors.Background,
			Surface:    b.Colors.Surface,
			Text:       b.Colors.Text,
			Error:      b.Colors.Error,
		},
		Typography: internalapply.Typography{
			HeadingFont:  b.Typography.HeadingFont,
			BodyFont:     b.Typography.BodyFont,
			MonoFont:     b.Typography.MonoFont,
			BaseFontSize: b.Typography.BaseFontSize,
		},
	}, nil
}

// assetStore adapts the assets service onto the apply AssetStore seam. It finds
// the brand's logo/favicon by filename stem (e.g. "logo.png" → kind "logo") and
// returns its bytes. A brand without that asset yields found=false (a skip, not
// an error).
type assetStore struct {
	assets internalassets.Service
}

func (s assetStore) Read(ctx context.Context, brandID, kind string) (internalapply.AssetContent, bool, error) {
	list, err := s.assets.List(ctx, brandID)
	if err != nil {
		return internalapply.AssetContent{}, false, err
	}
	var match *internalassets.Asset
	for i := range list {
		stem := strings.TrimSuffix(list[i].Filename, filepath.Ext(list[i].Filename))
		if strings.EqualFold(stem, kind) {
			match = &list[i]
			break
		}
	}
	if match == nil {
		return internalapply.AssetContent{}, false, nil
	}
	content, err := s.assets.Download(ctx, match.ID)
	if err != nil {
		return internalapply.AssetContent{}, false, err
	}
	return internalapply.AssetContent{Filename: content.Filename, Bytes: content.Bytes}, true, nil
}

// assignmentRecorder adapts the assignments service onto the apply
// AssignmentRecorder seam, so a real apply records the link exactly like a
// normal `assignments assign` (the version is re-pinned from the brand).
type assignmentRecorder struct {
	assignments internalassignments.Service
}

func (r assignmentRecorder) Record(ctx context.Context, brandID, scenario string, elements []string) error {
	_, err := r.assignments.Assign(ctx, internalassignments.AssignInput{
		BrandID:      brandID,
		ScenarioName: scenario,
		Elements:     elements,
	})
	return err
}

// assetBrandResolver adapts the brands repository onto the assets BrandResolver
// seam (the assets service confirms a brand exists before storing/reading).
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

// assignmentBrandResolver adapts the brands repository onto the assignments
// BrandResolver seam (the assignments service pins the brand's current version).
type assignmentBrandResolver struct {
	brands internalbrands.Repository
}

func (r assignmentBrandResolver) BrandVersion(ctx context.Context, brandID string) (int, bool, error) {
	b, err := r.brands.Get(ctx, brandID)
	if err != nil {
		var notFound internalbrands.ErrBrandNotFound
		if errors.As(err, &notFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return b.Version, true, nil
}
