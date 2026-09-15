package machines

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	internalmachines "vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/module"
	internalonboard "vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/ssh"
	"vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
)

func Module(db *database.RoutedDB, clk schedule.Clock, sshSvc *ssh.Service, registrySvc registry.Service, pairingSvc *pairing.Service, presenceHub *presence.Hub, onboardSvc internalonboard.Service, logger *log.Logger) module.Module {
	repo := internalmachines.NewSQLiteRepository(db, clk)
	svc := internalmachines.NewService(repo)
	audit, _ := repo.(auditAppender)
	attempts, _ := internalonboard.NewSQLiteRepository(db, clk).(attemptReader)
	path, handler := machinesconnect.NewMachineServiceHandler(NewConnectHandler(Deps{Service: svc, Attempts: attempts, Projection: composedProjection{registry: registrySvc, presence: presenceHub}, Audit: audit, HostKeyResetter: sshSvc, NodeRevoker: nodeRevoker{registry: registrySvc, pairing: pairingSvc, presence: presenceHub}, Repairer: machineRepairer{service: onboardSvc, trust: svc}, StoreRotator: newStoreRotator(onboardSvc, svc, registrySvc, sshSvc), Logger: logger}))
	return module.Module{
		Name:      "machines",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) },
		Endpoints: Endpoints,
	}
}

type machineTrustReader interface {
	GetTrust(context.Context, string) (internalmachines.TrustRecord, error)
}

type machineEnrollmentStarter interface {
	StartMachineEnrollment(context.Context, string, internalonboard.StartInput) (internalonboard.MachineEnrollmentDecision, error)
}

type machineRepairer struct {
	service machineEnrollmentStarter
	trust   machineTrustReader
}

func (r machineRepairer) Repair(ctx context.Context, machine internalmachines.Machine) (string, string, error) {
	if r.service == nil {
		return "", "", internalmachines.ErrInvalid{Field: "repair", Reason: "onboarding service unavailable"}
	}
	if r.trust == nil {
		return "", "", internalmachines.ErrInvalid{Field: "repair", Reason: "machine trust unavailable"}
	}
	trusted, err := r.trust.GetTrust(ctx, machine.ID)
	if err != nil {
		return "", "", internalmachines.ErrInvalid{Field: "repair", Reason: fmt.Sprintf("machine trust unavailable: %v", err)}
	}
	if strings.TrimSpace(trusted.SSHUser) == "" {
		return "", "", internalmachines.ErrInvalid{Field: "repair", Reason: "machine trust has no SSH user"}
	}
	host := ""
	for _, locator := range machine.Locators {
		if locator.Kind == "hostname" || locator.Kind == "ip" || locator.Kind == "ssh" {
			host = locator.Value
			break
		}
	}
	if host == "" {
		return "", "", internalmachines.ErrInvalid{Field: "repair", Reason: "machine has no hostname, IP, or SSH locator"}
	}
	port := trusted.SSHPort
	if port == 0 {
		port = 22
	}
	decision, err := r.service.StartMachineEnrollment(ctx, machine.ID, internalonboard.StartInput{
		MachineID: machine.ID, Host: host, Port: port, User: trusted.SSHUser, NodeName: machine.ID,
		TargetRevision: "@cp", ProvisionSudo: true,
	})
	if err != nil {
		return "", "", err
	}
	return decision.Decision.OpID, decision.Attempt.ID, nil
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
	var missing registry.ErrNodeNotFound
	if errors.As(err, &missing) {
		return internalmachines.NodeSnapshot{}, fmt.Errorf("%w: %v", internalmachines.ErrNodeMissing, err)
	}
	if err != nil {
		return internalmachines.NodeSnapshot{}, err
	}
	return internalmachines.NodeSnapshot{
		ID: node.ID, Name: node.Name, Capabilities: append([]string(nil), node.Capabilities...), ApprovedScopes: append([]string(nil), node.Scopes...),
		ConfigurationState: node.ConfigurationState, ConfigurationUnmet: append([]string(nil), node.ConfigurationUnmet...),
	}, nil
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

// machineStoreRotator resolves the Machine's current node, platform, and
// Bridge-managed SSH connection, then hands the rotation to the onboarding
// domain. Rotation runs only over verified SSH trust, the same boundary the
// typed SSH cleanup transport requires.
type machineStoreRotator struct {
	rotator  internalonboard.CredentialStoreRotator
	trust    machineTrustReader
	registry registry.Service
	keyDir   string
}

// newStoreRotator returns nil when the onboarding service cannot rotate (for
// example a test double), so the handler reports rotation as unavailable.
func newStoreRotator(onboardSvc internalonboard.Service, trust machineTrustReader, registrySvc registry.Service, sshSvc *ssh.Service) StoreRotator {
	rotator, ok := onboardSvc.(internalonboard.CredentialStoreRotator)
	if !ok || sshSvc == nil || registrySvc == nil {
		return nil
	}
	return machineStoreRotator{rotator: rotator, trust: trust, registry: registrySvc, keyDir: sshSvc.StateDir()}
}

func (r machineStoreRotator) Rotate(ctx context.Context, machine internalmachines.Machine) (internalonboard.RotateCredentialStoreResult, error) {
	nodeID := ""
	for _, lineage := range machine.Lineage {
		if lineage.Current {
			nodeID = lineage.NodeID
			break
		}
	}
	if nodeID == "" {
		return internalonboard.RotateCredentialStoreResult{}, internalmachines.ErrInvalid{Field: "lineage", Reason: "machine has no current node; repair it first"}
	}
	node, err := r.registry.Get(ctx, nodeID)
	if err != nil {
		return internalonboard.RotateCredentialStoreResult{}, fmt.Errorf("resolve the machine's node: %w", err)
	}
	trusted, err := r.trust.GetTrust(ctx, machine.ID)
	if err != nil {
		return internalonboard.RotateCredentialStoreResult{}, internalmachines.ErrInvalid{Field: "trust", Reason: fmt.Sprintf("machine trust unavailable: %v", err)}
	}
	if !trusted.SSHManagementEstablished() {
		return internalonboard.RotateCredentialStoreResult{}, internalmachines.ErrInvalid{Field: "ssh.management", Reason: "machine SSH trust is not verified; review its host key or repair it first"}
	}
	keyName := strings.TrimPrefix(strings.TrimSpace(trusted.ClientKeyRef), "ssh-key://")
	if keyName == "" || filepath.Base(keyName) != keyName {
		return internalonboard.RotateCredentialStoreResult{}, internalmachines.ErrInvalid{Field: "ssh.management", Reason: "machine trust has no Bridge-owned client key"}
	}
	host := ""
	for _, locator := range machine.Locators {
		if locator.Kind == "hostname" || locator.Kind == "ip" || locator.Kind == "ssh" {
			host = locator.Value
			break
		}
	}
	if host == "" {
		return internalonboard.RotateCredentialStoreResult{}, internalmachines.ErrInvalid{Field: "locators", Reason: "machine has no hostname, IP, or SSH locator"}
	}
	port := trusted.SSHPort
	if port == 0 {
		port = 22
	}
	return r.rotator.RotateCredentialStore(ctx, internalonboard.RotateCredentialStoreInput{
		MachineID: machine.ID, NodeID: nodeID,
		Conn:     internalonboard.Conn{Host: host, Port: port, User: trusted.SSHUser, KeyPath: filepath.Join(r.keyDir, keyName)},
		Platform: internalonboard.NodePlatform{OS: node.OS, Arch: node.Arch},
	})
}
