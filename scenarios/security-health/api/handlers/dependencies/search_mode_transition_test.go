package dependencies

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	depdomain "security-health/internal/dependencies"
	"security-health/internal/dependencies/aisearch"
	apiserver "security-health/internal/server"

	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	dependenciesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/dependencies/dependencies_v1connect"
)

type modeTransitionIndex struct {
	available bool
	queryErr  error
	queries   int
}

func (i *modeTransitionIndex) EnsureCollection(context.Context) error { return nil }
func (i *modeTransitionIndex) Sync(context.Context, []aisearch.Item) (int, int, error) {
	return 0, 0, nil
}
func (i *modeTransitionIndex) Query(context.Context, string, int) ([]aisearch.KeyScore, error) {
	i.queries++
	if i.queryErr != nil {
		return nil, i.queryErr
	}
	return []aisearch.KeyScore{{Key: "npm|react|18.0.0", Score: 0.92}}, nil
}
func (i *modeTransitionIndex) CountPoints(context.Context) (int, error) { return 1, nil }
func (i *modeTransitionIndex) Available(context.Context) (bool, bool) {
	return i.available, i.available
}

func TestSearchRouteRetainsCorpusAcrossExplicitModeAndProviderTransition(t *testing.T) {
	ctx := context.Background()
	storeDB := db.NewSQLite(t)
	if _, err := storeDB.ExecContext(ctx, depdomain.Schema()); err != nil {
		t.Fatal(err)
	}
	store := depdomain.NewStore(storeDB)
	if err := store.Apply(ctx, "", []depdomain.DependencyRecord{{
		Scenario: "alpha", Ecosystem: depdomain.EcosystemNPM, Name: "react", Version: "18.0.0", SourceFile: "ui/pnpm-lock.yaml",
	}}, "2026-11-21T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	index := &modeTransitionIndex{available: true}
	service := depdomain.NewService(depdomain.Deps{Store: store, RepoRoot: t.TempDir(), Clock: schedule.System(), Index: index})
	logger := log.New(io.Discard, "", 0)
	router := apiserver.New(apiserver.Deps{Clock: schedule.System(), Logger: logger}, Module(logger, service))
	httpServer := httptest.NewServer(router.Handler())
	t.Cleanup(httpServer.Close)
	client := dependenciesconnect.NewDependencyServiceClient(httpServer.Client(), httpServer.URL)

	cases := []struct {
		name        string
		mode        dependenciesv1.Mode
		providerUp  bool
		providerErr error
		wantMode    dependenciesv1.Mode
		wantQueries int
	}{
		{name: "AI selected", mode: dependenciesv1.Mode_MODE_AI, providerUp: true, wantMode: dependenciesv1.Mode_MODE_AI, wantQueries: 1},
		{name: "AI degrades during loss", mode: dependenciesv1.Mode_MODE_AI, providerErr: errors.New("semantic provider unavailable"), wantMode: dependenciesv1.Mode_MODE_TEXT, wantQueries: 2},
		{name: "operator selects TEXT while provider is down", mode: dependenciesv1.Mode_MODE_TEXT, providerErr: errors.New("semantic provider unavailable"), wantMode: dependenciesv1.Mode_MODE_TEXT, wantQueries: 2},
		{name: "AI resumes after recovery", mode: dependenciesv1.Mode_MODE_AI, providerUp: true, wantMode: dependenciesv1.Mode_MODE_AI, wantQueries: 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			index.available, index.queryErr = tc.providerUp, tc.providerErr
			resp, err := client.Search(ctx, connect.NewRequest(&dependenciesv1.SearchRequest{Query: "react", Mode: tc.mode}))
			if err != nil {
				t.Fatal(err)
			}
			if resp.Msg.GetModeUsed() != tc.wantMode || len(resp.Msg.GetResults()) != 1 {
				t.Fatalf("search mode/results = %v/%+v, want %v and one retained corpus result", resp.Msg.GetModeUsed(), resp.Msg.GetResults(), tc.wantMode)
			}
			record := resp.Msg.GetResults()[0].GetRecord()
			if record.GetScenario() != "alpha" || record.GetName() != "react" || record.GetVersion() != "18.0.0" || record.GetSourceFile() != "ui/pnpm-lock.yaml" {
				t.Fatalf("same SQLite corpus row not retained: %+v", record)
			}
			if index.queries != tc.wantQueries {
				t.Fatalf("semantic queries = %d, want %d", index.queries, tc.wantQueries)
			}
			if tc.providerErr != nil {
				status, err := client.Status(ctx, connect.NewRequest(&dependenciesv1.StatusRequest{}))
				if err != nil || status.Msg.GetIndexReady() || !status.Msg.GetAvailable() || status.Msg.GetIndexedCount() != 1 {
					t.Fatalf("degraded status lost TEXT corpus: (%+v, %v)", status, err)
				}
			}
		})
	}
	status, err := client.Status(ctx, connect.NewRequest(&dependenciesv1.StatusRequest{}))
	if err != nil || !status.Msg.GetAvailable() || !status.Msg.GetIndexReady() || status.Msg.GetIndexedCount() != 1 {
		t.Fatalf("recovered status/corpus = (%+v, %v)", status, err)
	}
}
