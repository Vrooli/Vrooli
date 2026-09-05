// Package design is the HTTP/Connect boundary for the design domain: it adapts
// the generated DesignService handler onto the transport-agnostic
// internal/design service and exports the domain's static metadata (Endpoints)
// for the modules registry.
//
// Design owns no table, so this package exports no Schema(); it composes the
// brands domain behind one adapter defined here — the composition root is the
// only place brands meets design.
package design

import (
	"context"
	"errors"
	"log"

	"brand-manager/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	designconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design/design_v1connect"

	internalbrands "brand-manager/internal/brands"
	internaldesign "brand-manager/internal/design"
)

// Module returns the design domain's contribution to the API: the generated
// Connect-RPC DesignService handler over the design service plus a brand adapter
// that reads brands through the brands domain's service.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	brandsSvc := internalbrands.NewService(
		internalbrands.NewSQLiteRepository(db, clk),
		internalbrands.NewSQLiteVersionRepository(db, clk),
		logger,
	)

	svc := internaldesign.NewService(brandStore{brands: brandsSvc}, logger)
	connectPath, connectHandler := designconnect.NewDesignServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "design",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// brandStore adapts the brands service onto the design BrandStore seam: design
// reads a brand through the normal brands read path and renders it. A brands
// not-found is translated into the design sentinel; any other error propagates
// unchanged so a lookup outage never masquerades as not-found.
type brandStore struct {
	brands internalbrands.Service
}

func (s brandStore) Get(ctx context.Context, brandID string) (internaldesign.Brand, error) {
	b, err := s.brands.Get(ctx, brandID)
	if err != nil {
		var notFound internalbrands.ErrBrandNotFound
		if errors.As(err, &notFound) {
			return internaldesign.Brand{}, internaldesign.ErrBrandNotFound{ID: brandID}
		}
		return internaldesign.Brand{}, err
	}
	return internaldesign.Brand{
		ID:          b.ID,
		Name:        b.Name,
		Description: b.Description,
		Identity: internaldesign.Identity{
			DisplayName: b.Identity.DisplayName,
			Tagline:     b.Identity.Tagline,
			LogoPath:    b.Identity.LogoPath,
			FaviconPath: b.Identity.FaviconPath,
			IconPath:    b.Identity.IconPath,
		},
		Colors: internaldesign.Colors{
			Primary:    b.Colors.Primary,
			Secondary:  b.Colors.Secondary,
			Accent:     b.Colors.Accent,
			Background: b.Colors.Background,
			Surface:    b.Colors.Surface,
			Text:       b.Colors.Text,
			Error:      b.Colors.Error,
		},
		Typography: internaldesign.Typography{
			HeadingFont:  b.Typography.HeadingFont,
			BodyFont:     b.Typography.BodyFont,
			MonoFont:     b.Typography.MonoFont,
			BaseFontSize: b.Typography.BaseFontSize,
		},
		Voice: internaldesign.Voice{
			Tone:     b.Voice.Tone,
			Style:    b.Voice.Style,
			Keywords: b.Voice.Keywords,
		},
		Notes:   b.Notes,
		Version: b.Version,
	}, nil
}
