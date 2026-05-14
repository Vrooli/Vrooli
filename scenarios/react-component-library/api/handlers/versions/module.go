// Package versions is the HTTP-handler home for the versions domain
// (req 11). Exposes VersionsService (proto:
// packages/proto/schemas/react-component-library/v1/versions).
package versions

import (
	"context"
	"database/sql"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	versionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions/versions_v1connect"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/clock"
	"react-component-library/internal/components"
	"react-component-library/internal/module"
	"react-component-library/internal/versions"
)

// Module wires the versions domain with an optional AdoptionResolver
// so the diff endpoint can compare against adopted-copy bytes. Pass
// nil resolver to disable adoption-side diffs.
func Module(db *sql.DB, clk clock.Clock, resolver versions.AdoptionResolver, logger *log.Logger) module.Module {
	repo := versions.NewSQLiteRepository(db, clk)
	svc := versions.NewService(repo, resolver)
	connectPath, connectHandler := versionsconnect.NewVersionsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "versions",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internal/versions.Schema for the modules registry.
func Schema() string { return versions.Schema() }

// BuildService is the seam main.go calls to construct one
// versions.Service that other layers (the components content-change
// listener, the CLI handlers test) can share.
func BuildService(db *sql.DB, clk clock.Clock, resolver versions.AdoptionResolver) versions.Service {
	return versions.NewService(versions.NewSQLiteRepository(db, clk), resolver)
}

// ListenerAdapter adapts the versions.Service into the
// components.ContentChangeListener seam — wires post-save recording
// without making internal/components import internal/versions.
type ListenerAdapter struct {
	Service versions.Service
	Logger  *log.Logger
}

var _ components.ContentChangeListener = (*ListenerAdapter)(nil)

func (l *ListenerAdapter) OnContentSaved(ctx context.Context, c components.Component, content components.Content) error {
	_, _, err := l.Service.Record(ctx, versions.RecordInput{
		ComponentID: c.ID,
		Content:     content.Body,
	})
	if err != nil && l.Logger != nil {
		l.Logger.Printf("versions: record on save for component %q failed: %v", c.ID, err)
	}
	return err
}

// AdoptionResolverFromService adapts adoptions.Service into the
// versions.AdoptionResolver seam. The resolver looks up the adoption
// by id and reads the adopted file content via the same scenario-file
// reader the adoptions service uses for drift refresh.
func AdoptionResolverFromService(svc adoptions.Service, reader adoptions.ScenarioFileReader) versions.AdoptionResolver {
	return &adoptionResolver{svc: svc, reader: reader}
}

type adoptionResolver struct {
	svc    adoptions.Service
	reader adoptions.ScenarioFileReader
}

func (r *adoptionResolver) ResolveAdoption(ctx context.Context, adoptionID string) (string, error) {
	a, err := r.svc.Get(ctx, adoptionID)
	if err != nil {
		return "", err
	}
	bytes, err := r.reader.Read(ctx, a.Scenario, a.AdoptedPath)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
