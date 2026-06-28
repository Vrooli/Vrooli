// Package assignments is the HTTP/Connect boundary for the assignments domain:
// it adapts the generated AssignmentsService handler onto the
// transport-agnostic internal/assignments service and exports the domain's
// static metadata (Endpoints, Schema) for the modules registry.
package assignments

import (
	"context"
	"errors"
	"log"

	"brand-manager/internal/clock"
	"brand-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	assignmentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments/assignments_v1connect"

	internalassignments "brand-manager/internal/assignments"
	internalbrands "brand-manager/internal/brands"
)

// Module returns the assignments domain's contribution to the API: the
// generated Connect-RPC AssignmentsService handler over the sqlite-backed
// service. The service resolves brand versions via a thin adapter over the
// brands domain (the composition root is the only place the two domains meet).
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	repo := internalassignments.NewSQLiteRepository(db, clk)
	resolver := brandResolver{brands: internalbrands.NewSQLiteRepository(db, clk)}
	svc := internalassignments.NewService(repo, resolver, logger)
	connectPath, connectHandler := assignmentsconnect.NewAssignmentsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "assignments",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalassignments.Schema so the modules registry collects
// both endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalassignments.Schema() }

// brandResolver adapts the brands repository onto the assignments
// BrandResolver seam. Defined here, the composition root, so the two internal
// domains never import each other.
type brandResolver struct {
	brands internalbrands.Repository
}

func (r brandResolver) BrandVersion(ctx context.Context, brandID string) (int, bool, error) {
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
