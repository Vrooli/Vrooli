package privilegebroker

import (
	"context"
	"strconv"
)

const managedLogStanzaPath = "/etc/logrotate.d/vrooli-log-volume-bounds"

// executeStorageAction maps each approved subject to a fixed argv. The broker
// never accepts a caller-provided executable, path, or shell expression.
func executeStorageAction(ctx context.Context, executor Executor, req Request) Result {
	var name string
	var args []string
	switch req.Action {
	case ActionLogRotateForce:
		name, args = "logrotate", []string{"-f", managedLogStanzaPath}
	case ActionJournaldVacuum:
		name, args = "journalctl", []string{"--vacuum-size=" + strconv.FormatInt(req.Journal.MaxUseBytes, 10)}
	case ActionDockerPruneUnusedImages:
		name, args = "docker", []string{"image", "prune", "-f"}
	case ActionDockerPruneUnusedVolumes:
		name = "docker"
		args = append([]string{"volume", "rm"}, req.Docker.VolumeNames...)
	default:
		return NewFailure(req.RequestID, req.Action, "action_not_allowed")
	}
	if _, err := executor.Run(ctx, name, args...); err != nil {
		return NewFailure(req.RequestID, req.Action, "privileged_action_failed")
	}
	return Result{Version: ProtocolVersion, RequestID: req.RequestID, Action: req.Action, Status: "completed", Changed: true}
}
