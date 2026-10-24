package stats_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	statsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/stats"
	statsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/stats/stats_v1connect"
	_ "modernc.org/sqlite"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/stats"
)

func TestConnectPortfolioStatsUsesProducerObservationTime(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	repo := eventlog.NewSQLiteRepository(database.NewFromPrimary(db))
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	engine := stats.NewEngine(repo)
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	router := mux.NewRouter()
	stats.RegisterConnectRoutes(router, engine)
	srv := httptest.NewServer(router)
	defer srv.Close()

	client := statsconnect.NewStatsServiceClient(http.DefaultClient, srv.URL)
	response, err := client.GetPortfolioStats(context.Background(), connect.NewRequest(&statsv1.GetPortfolioStatsRequest{}))
	if err != nil {
		t.Fatalf("get portfolio stats: %v", err)
	}
	if response.Msg.GetObservedAt() == nil || response.Msg.GetObservedAt().AsTime().IsZero() {
		t.Fatal("producer did not return an observation timestamp")
	}
}
