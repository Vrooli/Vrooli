package devices

import (
	"log"

	"device-sync-hub/internal/auth"
	"device-sync-hub/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices/devices_v1connect"

	internaldevices "device-sync-hub/internal/devices"
	internalrealtime "device-sync-hub/internal/realtime"
)

// Module returns the devices domain's contribution to the API: the generated
// Connect-RPC DevicesService handler. authClient is the validator the service
// uses to revoke a device's authenticator session on un-pair (and the same one
// main.go threads into the owner-auth middleware). hub (optional) backs the
// pairing-request realtime push and the live online-presence overlay; nil
// disables both without changing the stored device data.
func Module(db *database.RoutedDB, clk schedule.Clock, authClient auth.Validator, hub *internalrealtime.Hub, logger *log.Logger) module.Module {
	repo := internaldevices.NewSQLiteRepository(db, clk)
	cfg := internaldevices.Config{
		Repo:    repo,
		Clock:   clk,
		Secrets: internaldevices.CryptoSecrets{},
		Auth:    authClient,
		Logger:  logger,
	}
	if hub != nil {
		cfg.PairNotifier = hubPairingNotifier{hub: hub}
	}
	svc := internaldevices.NewService(cfg)
	path, handler := devicesconnect.NewDevicesServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Hub:     hub,
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
