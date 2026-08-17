package onboard

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	internalcleanup "vrooli-bridge/internal/cleanup"
	"vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/module"
	internalonboard "vrooli-bridge/internal/onboard"
	"vrooli-bridge/internal/onboard/ssh"
	"vrooli-bridge/internal/pairing"
	"vrooli-bridge/internal/presence"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	onboardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard/onboard_v1connect"
)

// NewService builds the onboard domain's application service with its proto-free
// seams bound to the concrete SSH capability (first touch + SCP + streaming
// bootstrap exec), the pairing service (server-side code issue), and the
// presence hub (ONLINE confirmation). main.go constructs it once and calls
// ResumeInterrupted on it at boot to reconcile ops orphaned by a restart.
func NewService(db internalonboard.SQLExecutor, clk schedule.Clock, pairingSvc *pairing.Service, hub *presence.Hub, sshSvc *ssh.Service, scriptPath string, opts ...internalonboard.Option) internalonboard.Service {
	opts = append(opts, internalonboard.WithEnrollmentResolver(codeIssuerAdapter{svc: pairingSvc}))
	attempts, _ := internalonboard.NewSQLiteRepository(db, clk).(internalonboard.AttemptStore)
	if attempts != nil {
		opts = append(opts, internalonboard.WithMachineLinker(machineLinkerAdapter{attempts: attempts, machines: machines.NewService(machines.NewSQLiteRepository(db, clk))}))
	}
	return internalonboard.NewService(
		internalonboard.NewSQLiteRepository(db, clk),
		internalonboard.NewSSHDriver(sshSvc, scriptPath),
		codeIssuerAdapter{svc: pairingSvc},
		newOnlineConfirmer(hub),
		clk,
		opts...,
	)
}

// NewProtectionProvisioner adapts the cleanup domain's durable typed helper
// operation to the onboarding protection step. The passphrase remains opaque
// in both domains; this adapter only coordinates dispatch and the server-owned
// terminal wait.
func NewProtectionProvisioner(cleanupSvc internalcleanup.Service) internalonboard.ProtectionProvisioner {
	return protectionProvisioner{cleanup: cleanupSvc}
}

type protectionProvisioner struct {
	cleanup internalcleanup.Service
}

func (p protectionProvisioner) ProvisionProtection(ctx context.Context, in internalonboard.ProtectionInput) (string, string, string, error) {
	if p.cleanup == nil {
		return "", "", "cleanup service is not configured", fmt.Errorf("cleanup service is not configured")
	}
	op, err := p.cleanup.ProvisionBreakGlass(ctx, internalcleanup.ProvisionInput{
		MachineID: in.MachineID, NodeID: in.NodeID, Target: in.Target, Scope: in.Scope,
		OperationID: in.CleanupOperationID, SealedPassphrase: append([]byte(nil), in.SealedPassphrase...), OperatorID: in.OperatorID,
	})
	if err != nil {
		return "", op.ID, "protection dispatch failed: " + err.Error(), err
	}
	finished, timedOut, err := p.cleanup.Wait(ctx, op.ID, 10*time.Minute)
	if err != nil {
		return "", op.ID, "protection wait failed: " + err.Error(), err
	}
	if timedOut {
		return finished.Status.String(), op.ID, "protection remains in progress after the wait budget", fmt.Errorf("protection operation %q did not reach a terminal state", op.ID)
	}
	if finished.Status != internalcleanup.StatusCompleted {
		return finished.Status.String(), op.ID, "protection finished with status " + finished.Status.String() + ": " + finished.Reason, fmt.Errorf("protection operation %q finished with status %s", op.ID, finished.Status.String())
	}
	return finished.Status.String(), op.ID, "target-bound break-glass material is established (existing material was left unchanged when present)", nil
}

// Module returns the onboard domain's contribution to the API: the generated
// Connect-RPC OnboardService handler. It owns its durable op tables, so it
// re-exports Schema().
func Module(svc internalonboard.Service, db internalonboard.SQLExecutor, clk schedule.Clock, sshSvc *ssh.Service, logger *log.Logger) module.Module {
	attempts, _ := internalonboard.NewSQLiteRepository(db, clk).(attemptLookup)
	machineService := machines.NewService(machines.NewSQLiteRepository(db, clk))
	path, handler := onboardconnect.NewOnboardServiceHandler(NewConnectHandler(Deps{
		Service:  svc,
		Attempts: attempts,
		Machines: machineService,
		Resolver: machineService,
		KeyCheck: func(ctx context.Context, host string, port int, user, keyRef string) (bool, string, string) {
			keyName := strings.TrimPrefix(strings.TrimSpace(keyRef), "ssh-key://")
			if keyName == "" || keyName != filepath.Base(keyName) {
				return false, ssh.StatusKeyError, ""
			}
			result := sshSvc.TestConnection(ctx, ssh.TestConnectionRequest{Host: host, Port: port, User: user, KeyPath: filepath.Join(sshSvc.StateDir(), keyName)})
			return result.OK, result.Status, result.Fingerprint
		},
		Logger: logger,
	}))
	return module.Module{
		Name: "onboard",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the onboard domain's SQL contribution so the modules
// registry collects endpoint descriptors and schema from one handler package.
func Schema() string { return internalonboard.Schema() }
