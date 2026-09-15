package retrieval

import (
	"document-manager/internal/module"

	retrievalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/retrieval/retrieval_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "retrieval_query", Path: retrievalconnect.RetrievalServiceQueryProcedure, Method: "POST", Summary: "Hybrid lexical and semantic corpus query", Category: "retrieval"},
	{ID: "retrieval_similar", Path: retrievalconnect.RetrievalServiceSimilarProcedure, Method: "POST", Summary: "Find similar corpus units", Category: "retrieval"},
}
