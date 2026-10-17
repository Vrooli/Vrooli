// Package discovery is the HTTP/Connect boundary for the discovery domain: it
// adapts the generated DiscoveryService handler onto the transport-agnostic
// internal/discovery service and exports the domain's static metadata
// (Endpoints) for the modules registry.
//
// Discovery owns no table, so this package exports no Schema(); it composes the
// brands domain behind one adapter defined here — the composition root is the
// only place brands meets discovery.
package discovery

import (
	"context"
	"log"

	"brand-manager/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery/discovery_v1connect"

	internalbrands "brand-manager/internal/brands"
	internaldiscovery "brand-manager/internal/discovery"
)

// Module returns the discovery domain's contribution to the API: the generated
// Connect-RPC DiscoveryService handler over the discovery service plus a brand
// adapter and a filesystem scanner. scenariosRoot is the directory that contains
// scenario source trees discovery scans for branding state.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger, scenariosRoot string) module.Module {
	brandsSvc := internalbrands.NewService(
		internalbrands.NewSQLiteRepository(db, clk),
		internalbrands.NewSQLiteVersionRepository(db, clk),
		logger,
	)

	svc := internaldiscovery.NewService(
		internaldiscovery.NewFSScanner(scenariosRoot),
		brandStore{brands: brandsSvc},
		logger,
	)
	connectPath, connectHandler := discoveryconnect.NewDiscoveryServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "discovery",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// brandStore adapts the brands service onto the discovery BrandStore seam: an
// import persists the discovered draft through the normal brands create +
// version-snapshot path.
type brandStore struct {
	brands internalbrands.Service
}

func (s brandStore) Create(ctx context.Context, draft internaldiscovery.DraftBrand) (internaldiscovery.Created, error) {
	created, err := s.brands.Create(ctx, internalbrands.CreateInput{
		Name:        draft.Name,
		Description: draft.Description,
		Identity: internalbrands.Identity{
			DisplayName: draft.Identity.DisplayName,
			Tagline:     draft.Identity.Tagline,
			LogoPath:    draft.Identity.LogoPath,
			FaviconPath: draft.Identity.FaviconPath,
			IconPath:    draft.Identity.IconPath,
		},
		Colors: internalbrands.Colors{
			Primary:    draft.Colors.Primary,
			Secondary:  draft.Colors.Secondary,
			Accent:     draft.Colors.Accent,
			Background: draft.Colors.Background,
			Surface:    draft.Colors.Surface,
			Text:       draft.Colors.Text,
			Error:      draft.Colors.Error,
		},
	})
	if err != nil {
		return internaldiscovery.Created{}, err
	}
	return internaldiscovery.Created{
		ID:      created.ID,
		Name:    created.Name,
		Version: created.Version,
	}, nil
}
