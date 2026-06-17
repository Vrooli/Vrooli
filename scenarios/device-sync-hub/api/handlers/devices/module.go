package devices

import (
	"log"

	"device-sync-hub/internal/auth"
	"device-sync-hub/internal/clock"
	"device-sync-hub/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices/devices_v1connect"

	internaldevices "device-sync-hub/internal/devices"
)

// Module returns the devices domain's contribution to the API: the generated
// Connect-RPC DevicesService handler. authClient is the validator the service
// uses to revoke a device's authenticator session on un-pair (and the same one
// main.go threads into the owner-auth middleware).
func Module(db *database.RoutedDB, clk clock.Clock, authClient auth.Validator, logger *log.Logger) module.Module {
	repo := internaldevices.NewSQLiteRepository(db, clk)
	svc := internaldevices.NewService(internaldevices.Config{
		Repo:    repo,
		Clock:   clk,
		Secrets: internaldevices.CryptoSecrets{},
		Auth:    authClient,
		Logger:  logger,
	})
	path, handler := devicesconnect.NewDevicesServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "devices",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the devices domain schema so the modules registry collects
// endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internaldevices.Schema() }
