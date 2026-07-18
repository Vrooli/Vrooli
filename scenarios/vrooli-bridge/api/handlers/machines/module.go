package machines

import (
	"context"
	"log"

	"vrooli-bridge/internal/clock"
	internalmachines "vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/module"
	internalonboard "vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/ssh"
	"vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
)

func Module(db *database.RoutedDB, clk clock.Clock, sshSvc *ssh.Service, registrySvc registry.Service, pairingSvc *pairing.Service, presenceHub *presence.Hub, logger *log.Logger) module.Module {
	repo := internalmachines.NewSQLiteRepository(db, clk)
	svc := internalmachines.NewService(repo)
	audit, _ := repo.(auditAppender)
	attempts, _ := internalonboard.NewSQLiteRepository(db, clk).(attemptReader)
	path, handler := machinesconnect.NewMachineServiceHandler(NewConnectHandler(Deps{Service: svc, Attempts: attempts, Projection: composedProjection{registry: registrySvc, presence: presenceHub}, Audit: audit, HostKeyResetter: sshSvc, NodeRevoker: nodeRevoker{registry: registrySvc, pairing: pairingSvc, presence: presenceHub}, Logger: logger}))
	return module.Module{
		Name:      "machines",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) },
		Endpoints: Endpoints,
	}
}

type composedProjection struct {
	registry registry.Service
	presence *presence.Hub
}

func (p composedProjection) Compose(ctx context.Context, machine internalmachines.Machine) (internalmachines.Projection, error) {
	return internalmachines.Compose(ctx, machine, registryProjectionReader{service: p.registry}, presenceProjectionReader{hub: p.presence})
}

type registryProjectionReader struct{ service registry.Service }

func (r registryProjectionReader) GetNode(ctx context.Context, nodeID string) (internalmachines.NodeSnapshot, error) {
	node, err := r.service.Get(ctx, nodeID)
	if err != nil {
		return internalmachines.NodeSnapshot{}, err
	}
	return internalmachines.NodeSnapshot{ID: node.ID, Name: node.Name, Capabilities: append([]string(nil), node.Capabilities...), ApprovedScopes: append([]string(nil), node.Scopes...)}, nil
}

type presenceProjectionReader struct{ hub *presence.Hub }

func (r presenceProjectionReader) GetPresence(_ context.Context, nodeID string) (internalmachines.PresenceSnapshot, error) {
	if r.hub == nil {
		return internalmachines.PresenceSnapshot{}, nil
	}
	for _, onlineID := range r.hub.OnlineNodes() {
		if onlineID == nodeID {
			return internalmachines.PresenceSnapshot{Connected: true}, nil
		}
	}
	return internalmachines.PresenceSnapshot{}, nil
}

type nodeRevoker struct {
	registry registry.Service
	pairing  *pairing.Service
	presence *presence.Hub
}

func (r nodeRevoker) RevokeMachineNode(ctx context.Context, nodeID string) error {
	if _, err := r.registry.Revoke(ctx, nodeID); err != nil {
		return err
	}
	// Once durable local revocation succeeds, the live channel must be cut even
	// when the credential store is temporarily unavailable. The returned error
	// still tells the operator that credential cleanup needs attention, but it
	// cannot leave a revoked Node connected in the meantime.
	var credentialErr error
	if r.pairing != nil {
		credentialErr = r.pairing.RevokeCredential(ctx, nodeID)
	}
	if r.presence != nil {
		r.presence.Disconnect(nodeID)
	}
	return credentialErr
}

func Schema() string { return internalmachines.Schema() }
