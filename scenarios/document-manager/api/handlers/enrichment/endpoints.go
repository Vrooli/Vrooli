package enrichment

import (
	"document-manager/internal/module"
	enrichmentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/enrichment/enrichment_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "enrichment_enrich", Path: enrichmentconnect.EnrichmentServiceEnrichProcedure, Method: "POST", Summary: "Enrich a document through AI Gateway", Category: "enrichment"},
	{ID: "enrichment_embed", Path: enrichmentconnect.EnrichmentServiceEmbedProcedure, Method: "POST", Summary: "Create gateway-governed embedding metadata", Category: "enrichment"},
}
