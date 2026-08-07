package forest

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	forestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest"
	forestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest/forest_v1connect"
	internalforest "source-ledger/internal/forest"
	"source-ledger/internal/inference"
	"source-ledger/internal/module"
	"source-ledger/internal/policy"
)

// NewService builds the compaction service. The composition root owns the
// single instance because the scheduled maintenance pass and the operator RPC
// must share one run mutex; two instances would compact concurrently.
func NewService(db *database.RoutedDB, client inference.Client, target int, registry *policy.Registry) *internalforest.Service {
	svc := internalforest.NewService(internalforest.NewSQLiteRepository(db.Primary()), internalforest.NewSQLiteCandidateSource(db.Primary()), client, internalforest.Config{Target: target})
	svc.SetPolicyRegistry(registry)
	return svc
}

func Module(svc *internalforest.Service, logger *log.Logger) module.Module {
	path, h := forestconnect.NewForestServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "forest", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

var ProtoFile = forestv1.File_source_ledger_v1_forest_forest_proto
