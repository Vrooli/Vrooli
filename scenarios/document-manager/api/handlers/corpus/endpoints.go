package corpus

import (
	"document-manager/internal/module"

	corpusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/corpus/corpus_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "corpus_create_collection", Path: corpusconnect.CorpusServiceCreateCollectionProcedure, Method: "POST", Summary: "Create a privacy-scoped collection", Category: "corpus"},
	{ID: "corpus_get_collection", Path: corpusconnect.CorpusServiceGetCollectionProcedure, Method: "POST", Summary: "Get collection metadata", Category: "corpus"},
	{ID: "corpus_list_collections", Path: corpusconnect.CorpusServiceListCollectionsProcedure, Method: "POST", Summary: "List collections", Category: "corpus"},
	{ID: "corpus_add_document", Path: corpusconnect.CorpusServiceAddDocumentProcedure, Method: "POST", Summary: "Add a document to a collection", Category: "corpus"},
	{ID: "corpus_list_documents", Path: corpusconnect.CorpusServiceListDocumentsProcedure, Method: "POST", Summary: "List collection documents", Category: "corpus"},
	{ID: "corpus_export", Path: corpusconnect.CorpusServiceExportProcedure, Method: "POST", Summary: "Export a collection as open JSON", Category: "corpus"},
	{ID: "corpus_import", Path: corpusconnect.CorpusServiceImportProcedure, Method: "POST", Summary: "Import an open JSON collection archive", Category: "corpus"},
	{ID: "corpus_prune", Path: corpusconnect.CorpusServicePruneProcedure, Method: "POST", Summary: "Coordinate safe regeneration pruning", Category: "corpus"},
}
