package artifacts

import (
	"context"
	"io"
	"log"
	"net/http"

	internalartifacts "vrooli-bridge/internal/artifacts"
	internalmachines "vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/registry"
	internalruns "vrooli-bridge/internal/runs"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"

	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts/artifacts_v1connect"
)

// Module returns the artifacts domain's contribution to the API: the generated
// Connect-RPC ArtifactsService handler. The artifacts domain ships non-git
// artifacts to nodes via device-sync-hub directed delivery (OT-P1-003); this
// module is the single place its proto-free seams are bound to the concrete
// registry (revocation) and the device-sync-hub directed-delivery client. It
// owns its own durable distributions table, so it re-exports Schema().
type ArtifactReceiptRecorder interface {
	RecordDeliveryReceipt(context.Context, internalartifacts.DeliveryReceipt) error
}

func Module(db internalartifacts.SQLExecutor, clk schedule.Clock, registrySvc registry.Service, runsSvc internalruns.Service, verifier *nodeauth.Verifier, logger *log.Logger, placementPusher ArtifactPlacementPusher) (module.Module, ArtifactReceiptRecorder, internalartifacts.Service, *internalartifacts.ClosureShipmentService) {
	svc := internalartifacts.NewService(
		internalartifacts.NewSQLiteRepository(db, clk),
		nodeReaderAdapter{svc: registrySvc},
		newDeviceSyncDelivery(placementPusher),
		clk,
		internalartifacts.WithProducedRepository(internalartifacts.NewSQLiteProducedRepository(db, clk)),
		internalartifacts.WithRunReader(runReaderAdapter{svc: runsSvc}),
	)
	closureShipments := internalartifacts.NewClosureShipmentService(internalartifacts.NewSQLiteClosureShipmentRepository(db), nil, clk.Now)
	machineSvc := internalmachines.NewService(internalmachines.NewSQLiteRepository(db, clk))
	selectionVerifier := onboardingSelectionVerifier{resolve: func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "vrooli-onboarding")
	}}
	closureShipments.AuthorizeSelection = selectionVerifier.Authorize
	closureShipments.TargetPolicy = func(ctx context.Context, shipment internalartifacts.ClosureShipment) error {
		return validateClosureTarget(ctx, shipment, nodeReaderAdapter{svc: registrySvc}, machineSvc)
	}
	var closurePusher internalartifacts.ClosureShipmentFramePusher
	if p, ok := placementPusher.(ClosureShipmentPusher); ok {
		closurePusher = p
	}
	closureCoordinator := internalartifacts.NewClosureShipmentCoordinator(closureShipments, closurePusher, func(ctx context.Context, source string) (io.ReadCloser, error) {
		return internalartifacts.OpenClosureSource(ctx, source, http.DefaultClient)
	})
	path, handler := artifactsconnect.NewArtifactsServiceHandler(NewConnectHandler(Deps{
		Service: svc, ClosureShipments: closureCoordinator,
		Verifier: verifier, Logger: logger,
	}))
	return module.Module{
		Name: "artifacts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}, svc, svc, closureShipments
}

type runReaderAdapter struct{ svc internalruns.Service }

var _ internalartifacts.RunReader = runReaderAdapter{}

func (a runReaderAdapter) GetRunTarget(ctx context.Context, id string) (internalartifacts.RunTarget, error) {
	run, _, err := a.svc.Get(ctx, id)
	if err != nil {
		return internalartifacts.RunTarget{}, err
	}
	return internalartifacts.RunTarget{ID: run.ID, NodeID: run.NodeID}, nil
}

// Schema re-exports the artifacts domain's SQL contribution so the modules
// registry collects endpoint descriptors and schema from one handler package.
func Schema() string { return internalartifacts.Schema() }
