package channel

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	desktopconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	companionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/companion"
)

const linuxCompanionService = "vrooli-desktop-companion.service"
const linuxCompanionSocket = "vrooli-desktop-companion.socket"

func (a localCompanionLifecycleAdapter) handleLinuxCompanion(ctx context.Context, command *companionv1.CompanionCommand, response *companionv1.CompanionResponse) *companionv1.CompanionResponse {
	home, err := a.resolveHome()
	if err != nil {
		return companionFailure(response, "user_home_unavailable", "repair_user_session")
	}
	artifact := filepath.Join(home, companionArtifactRel)
	run := func(args ...string) bool {
		code, err := a.run(ctx, append([]string{"systemctl", "--user"}, args...))
		return code == 0 && err == nil
	}
	switch command.GetKind() {
	case companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_REVOKE,
		companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_REMOVE:
		if !run("disable", "--now", linuxCompanionSocket) || !run("stop", linuxCompanionService) {
			return companionFailure(response, "companion_stop_failed", "inspect_user_systemd")
		}
		response.State = companionv1.CompanionState_COMPANION_STATE_REVOKED
		if command.GetKind() == companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_REMOVE {
			if err := os.Remove(artifact); err != nil && !errors.Is(err, os.ErrNotExist) {
				return companionFailure(response, "companion_remove_failed", "inspect_user_artifact")
			}
			response.State = companionv1.CompanionState_COMPANION_STATE_REMOVED
		}
		return response
	}
	info, err := os.Lstat(artifact)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		response.State = companionv1.CompanionState_COMPANION_STATE_ABSENT
		response.ReasonCode, response.Recovery = "artifact_not_installed", "deliver_verified_companion_artifact"
		return response
	}
	switch command.GetKind() {
	case companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_INSTALL,
		companionv1.CompanionOperationKind_COMPANION_OPERATION_KIND_UPGRADE:
		// Local provisioning owns the display binding and the agent's opt-in.
		// These remotely delivered commands may start only those fixed units.
		if !run("enable", "--now", linuxCompanionSocket) || !run("restart", linuxCompanionService) {
			return companionFailure(response, "linux_service_not_provisioned", "provision_linux_desktop_companion")
		}
	}
	if !run("is-active", "--quiet", linuxCompanionSocket) {
		response.State = companionv1.CompanionState_COMPANION_STATE_DEGRADED
		response.ReasonCode, response.Recovery = "companion_socket_inactive", "start_desktop_companion"
		return response
	}
	probe, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	client := desktopconnect.NewDesktopSessionServiceClient(desktopOwnerClient, "http://desktop-owner")
	ready, err := client.GetReadiness(probe, connect.NewRequest(&desktopv1.GetReadinessRequest{Surface: &commonv1.SurfaceRef{}}))
	response.State = companionv1.CompanionState_COMPANION_STATE_DEGRADED
	if err != nil {
		response.ReasonCode, response.Recovery = "companion_probe_failed", "inspect_user_systemd"
		return response
	}
	response.ReasonCode, response.Recovery = ready.Msg.GetReasonCode(), ready.Msg.GetRecovery()
	response.Version = ready.Msg.GetCompanionVersion()
	if ready.Msg.GetState() == desktopv1.ReadinessState_READINESS_STATE_READY {
		response.State = companionv1.CompanionState_COMPANION_STATE_READY
	}
	return response
}
