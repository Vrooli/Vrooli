package modules

import (
	"deployment-manager/internal/module"
	approvalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/approvals/approvalsv1connect"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/dependencies/dependenciesv1connect"
	deploymentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/deployments/deploymentsv1connect"
	evidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/evidence/evidencev1connect"
	fitnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/fitness/fitnessv1connect"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/lpbs/lpbsv1connect"
	migrationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/migration/migrationv1connect"
	profilesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/profiles/profilesv1connect"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	swapsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/swaps/swapsv1connect"
	telemetryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/telemetry/telemetryv1connect"
)

// AllEndpoints is the generated-contract registry. Connect procedures are
// referenced from generated bindings so a renamed RPC cannot silently drift.
// The health probe is the sole remaining REST exception in this registry.
func AllEndpoints() []module.EndpointDescriptor {
	return []module.EndpointDescriptor{
		{
			ID: "health", Path: "/health", Method: "GET", Summary: "Service health check",
			Description: "Returns API readiness and dependency status.", Category: "system",
			RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Lifecycle and load-balancer probes must reach health without a generated client.", ProtoPayloads: &module.RESTProtoPayloads{
				Request: module.RESTPayload{Transport: "none", Conformance: "none"}, Response: module.RESTPayload{Transport: "json", Conformance: "transport_only"}, Error: module.RESTPayload{Transport: "json", Conformance: "transport_only"},
			}},
		},
		connectEndpoint("list_profiles", profilesconnect.ProfilesServiceListProfilesProcedure, "List deployment profiles", "profiles"),
		connectEndpoint("get_profile", profilesconnect.ProfilesServiceGetProfileProcedure, "Get a deployment profile", "profiles"),
		connectEndpoint("create_profile", profilesconnect.ProfilesServiceCreateProfileProcedure, "Create a deployment profile", "profiles"),
		connectEndpoint("update_profile", profilesconnect.ProfilesServiceUpdateProfileProcedure, "Update a deployment profile", "profiles"),
		connectEndpoint("delete_profile", profilesconnect.ProfilesServiceDeleteProfileProcedure, "Delete a deployment profile", "profiles"),
		connectEndpoint("list_profile_versions", profilesconnect.ProfilesServiceListProfileVersionsProcedure, "List profile versions", "profiles"),
		connectEndpoint("report_target_verdict", evidenceconnect.EvidenceServiceReportTargetVerdictProcedure, "Report target evidence", "evidence"),
		connectEndpoint("list_target_verdicts", evidenceconnect.EvidenceServiceListTargetVerdictsProcedure, "List target evidence", "evidence"),
		connectEndpoint("get_evidence_review", evidenceconnect.EvidenceServiceGetEvidenceReviewProcedure, "Review release evidence", "evidence"),
		connectEndpoint("analyze_dependencies", dependenciesconnect.DependenciesServiceAnalyzeProcedure, "Analyze scenario dependencies", "dependencies"),
		connectEndpoint("score_fitness", fitnessconnect.FitnessServiceScoreProcedure, "Score deployment fitness", "fitness"),
		connectEndpoint("deploy", deploymentsconnect.DeploymentsServiceDeployProcedure, "Deploy a profile", "deployments"),
		connectEndpoint("deploy_desktop", deploymentsconnect.DeploymentsServiceDeployDesktopProcedure, "Deploy a desktop profile", "deployments"),
		connectEndpoint("deployment_status", deploymentsconnect.DeploymentsServiceStatusProcedure, "Get deployment status", "deployments"),
		connectEndpoint("list_swaps", swapsconnect.SwapsServiceListProcedure, "List dependency swaps", "swaps"),
		connectEndpoint("analyze_swap", swapsconnect.SwapsServiceAnalyzeProcedure, "Analyze a dependency swap", "swaps"),
		connectEndpoint("cascade_swap", swapsconnect.SwapsServiceCascadeProcedure, "Analyze swap cascade", "swaps"),
		connectEndpoint("apply_swap", swapsconnect.SwapsServiceApplyProcedure, "Apply a dependency swap", "swaps"),
		connectEndpoint("apply_swap_to_profile", swapsconnect.SwapsServiceApplyToProfileProcedure, "Apply a swap to a profile", "swaps"),
		connectEndpoint("list_telemetry", telemetryconnect.TelemetryServiceListProcedure, "List telemetry", "telemetry"),
		connectEndpoint("upload_telemetry", telemetryconnect.TelemetryServiceUploadProcedure, "Upload telemetry", "telemetry"),
		connectEndpoint("report_migration", migrationconnect.MigrationServiceReportProcedure, "Report a migration task", "migration"),
		connectEndpoint("migration_status", migrationconnect.MigrationServiceStatusProcedure, "Get migration task status", "migration"),
		connectEndpoint("list_approvals", approvalconnect.ApprovalsServiceListProcedure, "List approvals", "approvals"),
		connectEndpoint("get_approval", approvalconnect.ApprovalsServiceGetProcedure, "Get an approval", "approvals"),
		connectEndpoint("create_approval", approvalconnect.ApprovalsServiceCreateProcedure, "Create an approval", "approvals"),
		connectEndpoint("decide_approval", approvalconnect.ApprovalsServiceDecideProcedure, "Decide an approval", "approvals"),
		connectEndpoint("check_release_gate", approvalconnect.ApprovalsServiceCheckReleaseGateProcedure, "Check release gate", "approvals"),
		connectEndpoint("set_required_platforms", approvalconnect.ApprovalsServiceSetRequiredPlatformsProcedure, "Set required platforms", "approvals"),
		connectEndpoint("get_required_platforms", approvalconnect.ApprovalsServiceGetRequiredPlatformsProcedure, "Get required platforms", "approvals"),
		connectEndpoint("get_lpbs_config", lpbsconnect.LPBSServiceGetConfigProcedure, "Get LPBS release configuration", "releases"),
		connectEndpoint("save_lpbs_config", lpbsconnect.LPBSServiceSaveConfigProcedure, "Save LPBS release configuration", "releases"),
		connectEndpoint("list_releases", releasesconnect.ReleasesServiceListProcedure, "List releases", "releases"),
		connectEndpoint("get_release", releasesconnect.ReleasesServiceGetProcedure, "Get a release", "releases"),
		connectEndpoint("reverify_release", releasesconnect.ReleasesServiceReverifyProcedure, "Reverify a release", "releases"),
		connectEndpoint("start_release", releasesconnect.ReleasesServiceStartProcedure, "Start a release", "releases"),
	}
}

func connectEndpoint(id, path, summary, category string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Description: summary + " through the generated Connect contract.", Category: category}
}
