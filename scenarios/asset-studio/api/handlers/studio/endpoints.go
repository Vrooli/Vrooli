package studio

import (
	"asset-studio/internal/module"
	studioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio/studio_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "studio_list_identities", Path: studioconnect.StudioServiceListIdentitiesProcedure, Method: "POST", Category: "studio", Summary: "List identities"},
	{ID: "studio_create_identity", Path: studioconnect.StudioServiceCreateIdentityProcedure, Method: "POST", Category: "studio", Summary: "Create identity"},
	{ID: "studio_revise_identity", Path: studioconnect.StudioServiceReviseIdentityProcedure, Method: "POST", Category: "studio", Summary: "Revise identity"},
	{ID: "studio_resolve_spec", Path: studioconnect.StudioServiceResolveSpecProcedure, Method: "POST", Category: "studio", Summary: "Resolve deterministic spec"},
	{ID: "studio_create_render", Path: studioconnect.StudioServiceCreateRenderProcedure, Method: "POST", Category: "studio", Summary: "Create render candidates"},
	{ID: "studio_regenerate_render", Path: studioconnect.StudioServiceRegenerateRenderProcedure, Method: "POST", Category: "studio", Summary: "Regenerate a render from recorded intent"},
	{ID: "studio_analyze_conformance", Path: studioconnect.StudioServiceAnalyzeConformanceProcedure, Method: "POST", Category: "studio", Summary: "Record advisory Image Tools conformance analysis"},
	{ID: "studio_commission_agent", Path: studioconnect.StudioServiceCommissionAgentProcedure, Method: "POST", Category: "studio", Summary: "Commission an untrusted Agent Manager proposal"},
	{ID: "studio_set_campaign_budget", Path: studioconnect.StudioServiceSetCampaignBudgetProcedure, Method: "POST", Category: "studio", Summary: "Set a campaign media budget"},
	{ID: "studio_get_render", Path: studioconnect.StudioServiceGetRenderProcedure, Method: "POST", Category: "studio", Summary: "Inspect durable render receipt and candidates"},
	{ID: "studio_select_candidate", Path: studioconnect.StudioServiceSelectCandidateProcedure, Method: "POST", Category: "studio", Summary: "Select render candidate"},
	{ID: "studio_judge_conformance", Path: studioconnect.StudioServiceJudgeConformanceProcedure, Method: "POST", Category: "studio", Summary: "Record operator conformance verdict"},
	{ID: "studio_release_asset", Path: studioconnect.StudioServiceReleaseAssetProcedure, Method: "POST", Category: "studio", Summary: "Release an asset"},
	{ID: "studio_get_reference", Path: studioconnect.StudioServiceGetReleasedAssetReferenceProcedure, Method: "POST", Category: "studio", Summary: "Read released asset reference"},
	{ID: "studio_import_canon", Path: studioconnect.StudioServiceImportCanonProcedure, Method: "POST", Category: "studio", Summary: "Import canon identities"},
}
