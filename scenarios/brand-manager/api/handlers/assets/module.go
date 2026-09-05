// Package assets is the HTTP/Connect boundary for the assets domain: it adapts
// the generated AssetsService handler onto the transport-agnostic
// internal/assets service and exports the domain's static metadata (Endpoints,
// Schema) for the modules registry.
package assets

import (
	"context"
	"errors"
	"log"

	"brand-manager/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	assetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assets/assets_v1connect"

	internalassets "brand-manager/internal/assets"
	internalbrands "brand-manager/internal/brands"
)

// Module returns the assets domain's contribution to the API: the generated
// Connect-RPC AssetsService handler over the sqlite-backed catalog and a
// filesystem blob store rooted at blobBaseDir. The service confirms a brand
// exists via a thin adapter over the brands domain (the composition root is the
// only place the two domains meet).
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger, blobBaseDir string) module.Module {
	repo := internalassets.NewSQLiteRepository(db, clk)
	blobs := internalassets.NewFSBlobStore(blobBaseDir)
	resolver := brandResolver{brands: internalbrands.NewSQLiteRepository(db, clk)}
	svc := internalassets.NewService(repo, blobs, resolver, logger)
	connectPath, connectHandler := assetsconnect.NewAssetsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "assets",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalassets.Schema so the modules registry collects both
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalassets.Schema() }

// brandResolver adapts the brands repository onto the assets BrandResolver seam.
// Defined here, the composition root, so the two internal domains never import
// each other.
type brandResolver struct {
	brands internalbrands.Repository
}

func (r brandResolver) BrandExists(ctx context.Context, brandID string) (bool, error) {
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
