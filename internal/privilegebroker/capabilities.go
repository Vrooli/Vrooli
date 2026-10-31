package privilegebroker

import "time"

const temporaryObservationTimeout = 30 * time.Second

// capabilities is the compiled registry, never a caller-selected grant.
func capabilities() []string {
	return []string{
		ActionCapabilitiesInspect, ActionTemporaryActivityObserve,
		ActionBridgeUFWInspect, ActionBridgeUFWAllow, ActionBridgeUFWVerify, ActionBridgeUFWRevoke,
		ActionVolumeFilesystemCheck, ActionVolumeFilesystemRepair, ActionRuntimeHomeOwnershipRepair,
		ActionLogRotateForce, ActionJournaldVacuum, ActionDockerPruneUnusedImages, ActionDockerPruneUnusedVolumes,
		ActionAptPackagesEnsure, ActionEdgeUFWAllow, ActionProcessStopScoped, ActionEdgeCaddyValidate, ActionEdgeCaddyReload,
	}
}
