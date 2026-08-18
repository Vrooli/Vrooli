package corpus

import (
	"document-manager/internal/corpus"
	"document-manager/internal/module"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	corpusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/corpus/corpus_v1connect"
)

func Module(db *database.RoutedDB) module.Module {
	service := corpus.NewService(corpus.NewSQLiteRepository(db))
	path, handler := corpusconnect.NewCorpusServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "corpus", Mount: func(r *mux.Router) { r.PathPrefix(path).Handler(handler) }, Endpoints: Endpoints}
}

func Schema() string { return corpus.Schema() }
