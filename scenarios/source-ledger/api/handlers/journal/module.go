// Package journal exposes the immutable memory journal over Connect-RPC.
package journal

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"source-ledger/internal/inference"
	internaljournal "source-ledger/internal/journal"
	"source-ledger/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	measures "github.com/vrooli/measures-go"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
)

// Module owns the production journal composition. Inference remains behind
// the scenario-owned Client seam, so no provider SDK can leak into the domain.
func Module(db *database.RoutedDB, client inference.Client, facetResolver internaljournal.FacetResolver, logger *log.Logger) module.Module {
	repo := internaljournal.NewSQLiteRepository(db.Primary())
	svc := internaljournal.NewService(repo, client, facetResolver)
	path, handler := journalconnect.NewJournalServiceHandler(NewConnectHandler(svc, logger))
	registry := measures.NewRegistry()
	decl := measures.MeasureDeclaration{Name: "entries.appended", Scenario: "source-ledger", Domain: "entries", Intent: "Number of immutable journal entries appended in a time window.", Questions: []string{"how many journal entries were appended", "source ledger append volume this week", "memory entries added last 7 days"}, Params: map[string]measures.Param{"window": {Name: "window", Type: measures.ParamTypeTimeWindow, Default: string(measures.TokenThisWeek)}}, Result: measures.Result{Kind: measures.ResultScalar, ValueField: "count", Unit: "entries", SummaryTemplate: "{count} journal entries appended ({window})"}, Effect: measures.EffectRead, RunEligible: true, Service: "JournalService", Method: "CountEntries"}
	if err := registry.Register(decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
		token := measures.TimeWindowToken(req.Params["window"])
		if token == "" {
			token = measures.TokenThisWeek
		}
		rng, err := measures.ResolveToken(token, time.Now().UTC(), time.UTC)
		if err != nil {
			return measures.MeasureResult{}, err
		}
		count, err := svc.CountInWindow(ctx, rng.From, rng.To)
		if err != nil {
			return measures.MeasureResult{}, err
		}
		return measures.MeasureResult{Value: strconv.FormatInt(count, 10), Provenance: measures.Provenance{ExecutedQuery: "SELECT COUNT(*) FROM entries WHERE created_at >= ? AND created_at < ?"}}, nil
	}); err != nil {
		panic(err)
	}
	return module.Module{Name: "journal", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		r.PathPrefix("/measures/").Handler(http.StripPrefix("/measures", registry.Handler()))
	}, Endpoints: Endpoints}
}

// ProtoFile is registered for generated-proto/Connect endpoint parity checks.
var ProtoFile = journalv1.File_source_ledger_v1_journal_journal_proto
