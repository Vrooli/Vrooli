package cleanup

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup/cleanup_v1connect"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/channelsign"
	internalcleanup "vrooli-bridge/internal/cleanup"
	internalmachines "vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/nodeauth"
	onboardssh "vrooli-bridge/internal/onboard/ssh"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"
)

type Deps struct {
	Service  internalcleanup.Service
	Verifier *nodeauth.Verifier
	Logger   *log.Logger
}

func NewService(db internalcleanup.SQLExecutor, clock schedule.Clock, registrySvc registry.Service, keys sealingKeySource, hub *presence.Hub, auditSink audit.Sink, signer channelsign.Signer, sshSvc *onboardssh.Service) internalcleanup.Service {
	repo := internalcleanup.NewSQLiteRepository(db, clock)
	sshPusher := &typedSSHPusher{machines: internalmachines.NewService(internalmachines.NewSQLiteRepository(db, clock)), ssh: sshSvc}
	service := internalcleanup.NewService(repo, nodeReader{svc: registrySvc, keys: keys}, hub, commandPusher{hub: hub, signer: signer, sshPusher: sshPusher}, auditSinkAdapter{sink: auditSink}, clock)
	sshPusher.report = service.AppendEvent
	return service
}

func Module(svc internalcleanup.Service, verifier *nodeauth.Verifier, logger *log.Logger) module.Module {
	path, handler := cleanup_v1connect.NewCleanupServiceHandler(NewConnectHandler(Deps{Service: svc, Verifier: verifier, Logger: logger}))
	return module.Module{Name: "cleanup", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

func Schema() string { return internalcleanup.Schema() }
