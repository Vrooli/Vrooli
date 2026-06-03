// Package registry is the search-hub registry domain's API surface: the
// generated RegistryService Connect-RPC handler that persists and lists
// provider descriptors. It is the first real domain replacing the notes
// worked example.
//
// Adding a real domain to a scenario means copying this file into
// handlers/<dom>/module.go and pointing it at <dom>'s proto-generated handler
// and service. The center (server.New) does not change.
package registry

import (
	"log"

	"search-hub/internal/clock"
	"search-hub/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"

	internalregistry "search-hub/internal/registry"
)

// Module returns the registry domain's contribution to the API: the generated
// RegistryService Connect handler backed by the SQLite store.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	store := internalregistry.NewSQLiteStore(db, clk)
	connectPath, connectHandler := registryconnect.NewRegistryServiceHandler(NewConnectHandler(Deps{
		Store:  store,
		Logger: logger,
	}))
	return module.Module{
		Name: "registry",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalregistry.Schema so the modules registry can collect
// both endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalregistry.Schema() }
