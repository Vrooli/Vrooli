package bountycases

import (
	"client-rehearsal/internal/bountycases"
	"client-rehearsal/internal/module"
	"context"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/bountycases/bountycases_v1connect"
)

// Module builds the private owner endpoint. The caller wires this only after
// the Client Rehearsal API's private binding policy is registered. The
// endpoint is omitted from the general CLI manifest; its intended call path
// is the exact-digest Program Runtime private binding.
func Module(db *database.RoutedDB, trustedRoot string) (module.Module, error) {
	registrationDir := filepath.Join(trustedRoot, "registrations")
	if err := os.MkdirAll(registrationDir, 0o700); err != nil {
		return module.Module{}, err
	}
	service, err := bountycases.NewService(bountycases.NewSQLiteRepository(db), trustedRoot)
	if err != nil {
		return module.Module{}, err
	}
	if _, err := bountycases.LoadDirectory(context.Background(), service, registrationDir); err != nil {
		return module.Module{}, err
	}
	path, handler := connect.NewBountyCaseServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "bounty-cases", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}, nil
}

func Schema() string { return bountycases.Schema() }
