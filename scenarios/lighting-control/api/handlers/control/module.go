package control

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control/controlconnect"
	"lighting-control/internal/control"
	"lighting-control/internal/module"
	internalpolicy "lighting-control/internal/policy"
)

// Module wires the lighting control service to its durable store and the
// explicit, loopback-only OpenRGB SDK client. An absent SDK port means the
// resource is disabled; startup never starts or probes OpenRGB.
func Module(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger) module.Module {
	return ModuleWithDomainService(NewDomainService(db, clock, BackendFromEnvironment()), logger)
}

func ModuleWithService(db *database.RoutedDB, clock schedule.Clock, logger *log.Logger, backend control.Backend) module.Module {
	return ModuleWithDomainService(NewDomainService(db, clock, backend), logger)
}

// BackendFromEnvironment configures only the explicitly exported loopback SDK
// port for an enabled OpenRGB resource. No default port or lifecycle action is
// inferred when the resource did not provide that export.
func BackendFromEnvironment() control.Backend {
	port := os.Getenv("RESOURCE_PORT_SDK")
	if port == "" {
		return nil
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return invalidSDKConfiguration{}
	}
	return control.SDKClient{Address: net.JoinHostPort("127.0.0.1", strconv.Itoa(n))}
}

type invalidSDKConfiguration struct{}

func (invalidSDKConfiguration) ProtocolVersion(context.Context) (uint32, error) {
	return 0, errors.New("RESOURCE_PORT_SDK must be a valid TCP port")
}

func (invalidSDKConfiguration) Discover(context.Context) ([]control.SDKController, error) {
	return nil, errors.New("RESOURCE_PORT_SDK must be a valid TCP port")
}

func (invalidSDKConfiguration) SendZoneColors(context.Context, uint32, uint32, []control.RGB) (control.SDKCommandResult, error) {
	return control.SDKCommandResult{}, errors.New("RESOURCE_PORT_SDK must be a valid TCP port")
}

func NewDomainService(db *database.RoutedDB, clock schedule.Clock, backend control.Backend) *control.Service {
	repo := control.NewSQLiteRepository(db, clock)
	svc := control.NewService(repo, backend, clock)
	svc.BindBrightnessPolicy(internalpolicy.NewSQLiteStore(db))
	return svc
}

func ModuleWithDomainService(svc *control.Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	path, handler := controlconnect.NewControlServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "control", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return control.Schema() }
