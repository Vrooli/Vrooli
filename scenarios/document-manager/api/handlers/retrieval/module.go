package retrieval

import (
	"document-manager/internal/module"
	"document-manager/internal/retrieval"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	retrievalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/retrieval/retrieval_v1connect"
)

func Module(db *database.RoutedDB) module.Module {
	repo := retrieval.NewSQLiteRepository(db)
	service := retrieval.NewService(repo, retrieval.NewSQLiteVectorStore(db))
	path, handler := retrievalconnect.NewRetrievalServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "retrieval", Mount: func(r *mux.Router) { r.PathPrefix(path).Handler(handler) }, Endpoints: Endpoints}
}

func Schema() string { return retrieval.Schema() }
