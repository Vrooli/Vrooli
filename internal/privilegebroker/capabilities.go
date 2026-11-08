package privilegebroker

import "time"

const temporaryObservationTimeout = 30 * time.Second

// capabilities is the compiled registry, never a caller-selected grant.
func capabilities() []string {
	return []string{
		ActionCapabilitiesInspect, ActionTemporaryActivityObserve,
		ActionKernelMemoryInspect, ActionServiceRecover,
		ActionBridgeUFWInspect, ActionBridgeUFWAllow, ActionBridgeUFWVerify, ActionBridgeUFWRevoke,
		ActionLocalAdmissionInspect, ActionLocalAdmissionEnsure, ActionLocalAdmissionRevoke,
		ActionVolumeFilesystemCheck, ActionVolumeFilesystemRepair, ActionVolumeSMARTInspect, ActionRuntimeHomeOwnershipRepair,
		ActionLogRotateForce, ActionJournaldVacuum, ActionDockerPruneUnusedImages, ActionDockerPruneUnusedVolumes,
		ActionAptPackagesEnsure, ActionEdgeUFWAllow, ActionProcessStopScoped, ActionEdgeCaddyValidate, ActionEdgeCaddyReload,
	}
}
