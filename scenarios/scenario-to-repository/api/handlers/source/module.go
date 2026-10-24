package source

import (
	"log"

	"scenario-to-repository/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	sourcev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-repository/v1/source/source_v1connect"
)

func Module(store *Store, outputDir string, logger *log.Logger) module.Module {
	path, handler := sourcev1connect.NewSourceRepositoryServiceHandler(NewHandler(store, outputDir, logger))
	return module.Module{Name: "source", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "source_analyze_closure", Path: sourcev1connect.SourceRepositoryServiceAnalyzeClosureProcedure, Method: "POST", Summary: "Analyze source closure", Description: "Resolves admitted scenario source and explicit runtime obligations without reading Git history.", Category: "source", Errors: []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid source root or closure"}, {Status: 500, Code: "internal", Description: "Closure analysis failed"}}},
	{ID: "source_assemble_export", Path: sourcev1connect.SourceRepositoryServiceAssembleExportProcedure, Method: "POST", Summary: "Assemble deterministic export", Description: "Builds a deterministic tar.gz source artifact after policy and closure checks.", Category: "source", Errors: []module.ErrorDesc{{Status: 412, Code: "failed_precondition", Description: "Policy or closure refused export"}}},
	{ID: "source_verify_export", Path: sourcev1connect.SourceRepositoryServiceVerifyExportProcedure, Method: "POST", Summary: "Verify exact artifact", Description: "Checks the recorded archive digest and returns an attributable verification receipt.", Category: "source"},
	{ID: "source_prepare_publication", Path: sourcev1connect.SourceRepositoryServicePreparePublicationProcedure, Method: "POST", Summary: "Prepare human publication", Description: "Creates a bounded publication handoff and never writes a remote repository.", Category: "source", Errors: []module.ErrorDesc{{Status: 403, Code: "permission_denied", Description: "Automated publication is prohibited"}}},
	{ID: "source_list_distributions", Path: sourcev1connect.SourceRepositoryServiceListDistributionsProcedure, Method: "POST", Summary: "List source distributions", Description: "Reads authoritative distribution records.", Category: "source"},
	{ID: "source_get_distribution", Path: sourcev1connect.SourceRepositoryServiceGetDistributionProcedure, Method: "POST", Summary: "Get distribution", Description: "Reads durable distribution identity and its independent verification/publication standing.", Category: "source"},
	{ID: "source_get_contents", Path: sourcev1connect.SourceRepositoryServiceGetDistributionContentsProcedure, Method: "POST", Summary: "Get distribution contents", Description: "Reads safe admitted and excluded contents without secret values.", Category: "source"},
	{ID: "source_get_handoff", Path: sourcev1connect.SourceRepositoryServiceGetPublicationHandoffProcedure, Method: "POST", Summary: "Get publication handoff", Description: "Reads human-only publication preconditions.", Category: "source"},
	{ID: "source_get_drift", Path: sourcev1connect.SourceRepositoryServiceGetDistributionDriftProcedure, Method: "POST", Summary: "Get distribution drift", Description: "Reads source and destination drift standing.", Category: "source"},
}
