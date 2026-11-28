package recall

import (
	"context"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"

	internalfacets "source-ledger/internal/facets"
	"source-ledger/internal/inference"
	internaljournal "source-ledger/internal/journal"
	"source-ledger/internal/module"
	"source-ledger/internal/policy"
	internalrecall "source-ledger/internal/recall"
	vectorcodec "source-ledger/internal/vector"
)

func Module(db *database.RoutedDB, client inference.Client, config internalrecall.Config, registry *policy.Registry, logger *log.Logger, generations *vectorcodec.SQLiteGenerationServing, exact internalrecall.RecipeQueryEmbedder, categories internalrecall.ClassificationIndex) module.Module {
	var embedder internalrecall.QueryEmbedder = inference.Embedder{Client: client}
	if exact != nil {
		// Recall uses the exact persisted recipe whenever a generation is active;
		// the gateway default remains the pre-generation fallback.
		embedder = exactQueryEmbedder{fallback: inference.Embedder{Client: client}, exact: exact}
	}
	svc := internalrecall.NewService(internalrecall.NewSQLiteSource(db.Reader(), generations), embedder, config)
	svc.SetPolicyRegistry(registry)
	svc.SetClassificationIndex(categories)
	journal := internaljournal.NewService(internaljournal.NewSQLiteRepository(db.Primary()), client)
	handler := NewConnectHandler(svc, logger, journal)
	handler.SetUsageRecorder(internalfacets.NewService(internalfacets.NewSQLiteRepository(db.Primary())))
	path, h := recallconnect.NewRecallServiceHandler(handler)
	return module.Module{Name: "recall", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

type exactQueryEmbedder struct {
	fallback inference.Embedder
	exact    internalrecall.RecipeQueryEmbedder
}

func (e exactQueryEmbedder) EmbedQuery(ctx context.Context, query string) ([]float64, error) {
	return e.fallback.EmbedQueryFast(ctx, query)
}

func (e exactQueryEmbedder) EmbedQueryForRecipe(ctx context.Context, recipeID, query string) ([]float64, error) {
	return e.exact.EmbedQueryForRecipe(ctx, recipeID, query)
}

var ProtoFile = recallv1.File_source_ledger_v1_recall_recall_proto
