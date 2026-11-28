package classification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	_ "modernc.org/sqlite"

	"source-ledger/internal/journal"
)

type inferenceHandler struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	decide func(*inferencev1.DecideBatchRequest) (*inferencev1.DecideBatchResponse, error)
}

func (h inferenceHandler) DecideBatch(_ context.Context, request *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error) {
	response, err := h.decide(request.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

func TestDurableBackfillAndRetryAcrossRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ledger.db")
	db, journalRepo := openTestStore(t, path)
	repo := NewSQLiteRepository(db)
	catalog, err := LoadCatalog("../../../.vrooli/memory-categories.json")
	require.NoError(t, err)
	catalog = catalog[:1]
	for i := range batchSize + 10 {
		appendEntry(t, journalRepo, journal.Entry{ID: fmt.Sprintf("history-%d", i), Body: "ordinary history"})
	}
	var failed, unavailable atomic.Bool
	failed.Store(true)
	var calls atomic.Int32
	_, handler := inferenceconnect.NewInferenceServiceHandler(inferenceHandler{decide: func(request *inferencev1.DecideBatchRequest) (*inferencev1.DecideBatchResponse, error) {
		calls.Add(1)
		if unavailable.Load() {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("inference is down"))
		}
		if request.GetRole() != decisionRole ||
			request.GetProfile() != sharedv1.Profile_PROFILE_LOCAL_ONLY ||
			request.GetPrivacyClass() != sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL ||
			request.GetPrivacyProvenance() == "" || len(request.GetItems()) > batchSize {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid decision contract"))
		}
		var response inferencev1.DecideBatchResponse
		for _, item := range request.GetItems() {
			var state struct{ Observation string }
			if err := json.Unmarshal([]byte(item.GetStateJson()), &state); err != nil {
				return nil, err
			}
			answer := &inferencev1.DecideResponse{
				Method:   inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL,
				Provider: "local-model", Model: "test-model",
				Answers: []*inferencev1.DecisionAnswer{{
					QuestionId: request.GetQuestions()[0].GetId(),
					Type:       inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
					Choice:     "unrelated",
				}},
			}
			switch state.Observation {
			case "new defect":
				if failed.Load() {
					answer = &inferencev1.DecideResponse{Error: &inferencev1.InferenceError{Code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE}}
				} else {
					answer.Answers[0].Choice = "matches"
				}
			case "private defect", "account update":
				answer.Answers[0].Choice = "matches"
			}
			response.Results = append(response.Results, answer)
		}
		return &response, nil
	}})
	gateway := httptest.NewServer(handler)
	t.Cleanup(gateway.Close)
	client := inferenceconnect.NewInferenceServiceClient(gateway.Client(), gateway.URL)
	clock := schedule.NewFake(time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC))

	svc := NewService(repo, client, catalog)
	assertPending := func(scope string, want bool) {
		t.Helper()
		pending, err := svc.Coverage(ctx, scope, []string{"defect-report"})
		require.NoError(t, err)
		require.Equal(t, want, pending)
	}
	assertPending("agent-memory", true)
	assertPending("team:one", false)
	pass := func(wantDeferred int) {
		t.Helper()
		deferred, err := svc.runCategory(ctx, svc.definitions[0], clock)
		require.NoError(t, err)
		require.Equal(t, wantDeferred, deferred)
	}
	unavailable.Store(true)
	_, err = svc.runCategory(ctx, svc.definitions[0], clock)
	require.ErrorContains(t, err, "unavailable")
	unavailable.Store(false)
	pass(0)
	appendEntry(t, journalRepo, journal.Entry{ID: "new", Kind: "observation", Body: "new defect", Evidence: "Reproduction steps"})
	appendEntry(t, journalRepo, journal.Entry{ID: "private", Scope: "team:one", Kind: "observation", Body: "private defect"})
	appendEntry(t, journalRepo, journal.Entry{ID: "account", Kind: "observation", Body: "account update", RoutingHint: "defect-account", ImportKey: "request:remember/scenario-qa/defect-fixture/1/0"})
	appendEntry(t, journalRepo, journal.Entry{ID: "legacy", Kind: "legacy-bug-report", Body: "legacy source", ImportKey: "request:legacy-scenario-qa-bugs/v1/source-id"})
	assertPending("team:one", true)

	matches, err := svc.MatchingEntries(ctx, "agent-memory", []string{"defect-report"})
	require.NoError(t, err)
	require.Equal(t, map[string][]string{"legacy": {"defect-report"}}, matches, "typed source membership is available before the worker")

	pass(1)
	assertPending("agent-memory", true)
	assertPending("team:one", false)
	matches, err = svc.MatchingEntries(ctx, "agent-memory", []string{"defect-report"})
	require.NoError(t, err)
	require.Equal(t, map[string][]string{"account": {"defect-report"}, "legacy": {"defect-report"}}, matches)
	matches, err = svc.MatchingEntries(ctx, "team:one", []string{"defect-report"})
	require.NoError(t, err)
	require.Equal(t, map[string][]string{"private": {"defect-report"}}, matches)

	// Reopening the database and renaming a category preserves its completed
	// decisions, high-water marks and pending work.
	require.NoError(t, db.Close())
	db, _ = openTestStore(t, path)
	repo = NewSQLiteRepository(db)
	catalog[0].Name = "Defect observations"
	svc = NewService(repo, client, catalog)
	before := calls.Load()
	pass(0)
	require.Equal(t, before, calls.Load(), "undue retry and exhausted cursors do not call AI Gateway")
	failed.Store(false)
	clock.Advance(time.Minute)
	pass(0)
	pass(0)
	assertPending("agent-memory", false)
	notices, err := svc.Notices(ctx, "agent-memory", "defect-report", 0, 10)
	require.NoError(t, err)
	require.Len(t, notices, 1)
	require.Equal(t, "new", notices[0].ObservationID)
	paged, err := svc.Notices(ctx, "agent-memory", "defect-report", notices[0].Sequence, 10)
	require.NoError(t, err)
	require.Empty(t, paged)

	catalog[0].Definition += " Revised semantics."
	revised := NewService(repo, client, catalog)
	matches, err = revised.MatchingEntries(ctx, "agent-memory", []string{"defect-report"})
	require.NoError(t, err)
	require.Equal(t, map[string][]string{"legacy": {"defect-report"}}, matches, "only the unchanged typed source rule serves before the revised model pass")
	withoutSources := append([]Category(nil), catalog...)
	withoutSources[0].Sources = nil
	removed, err := NewService(repo, client, withoutSources).MatchingEntries(ctx, "agent-memory", []string{"defect-report"})
	require.NoError(t, err)
	require.Empty(t, removed, "obsolete typed and model classifications stay out after a source rule changes")
	staleNotices, err := revised.Notices(ctx, "agent-memory", "defect-report", 0, 10)
	require.NoError(t, err)
	require.Empty(t, staleNotices, "unread matches for a previous definition do not wake triage")
	_, err = revised.runCategory(ctx, revised.definitions[0], clock)
	require.NoError(t, err)
	_, err = revised.runCategory(ctx, revised.definitions[0], clock)
	require.NoError(t, err)
	revisedNotices, err := revised.Notices(ctx, "agent-memory", "defect-report", notices[0].Sequence, 10)
	require.NoError(t, err)
	require.Len(t, revisedNotices, 1)
	require.Equal(t, "new", revisedNotices[0].ObservationID)
	matches, err = svc.MatchingEntries(ctx, "agent-memory", []string{"defect-report"})
	require.NoError(t, err)
	require.Len(t, matches, 3, "later definitions never mutate historical decisions")
}

type decideFunc func(context.Context, *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error)

func (f decideFunc) DecideBatch(ctx context.Context, req *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error) {
	return f(ctx, req)
}

func TestTypedImportStillClassifiesDuringModelOutage(t *testing.T) {
	ctx := context.Background()
	db, journalRepo := openTestStore(t, filepath.Join(t.TempDir(), "ledger.db"))
	catalog, err := LoadCatalog("../../../.vrooli/memory-categories.json")
	require.NoError(t, err)
	clock := schedule.NewFake(time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC))
	client := decideFunc(func(_ context.Context, request *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error) {
		require.Len(t, request.Msg.GetItems(), 1)
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("offline"))
	})

	appendEntry(t, journalRepo, journal.Entry{ID: "normal", Body: "ordinary observation"})
	appendEntry(t, journalRepo, journal.Entry{ID: "legacy", Kind: "legacy-bug-report", Body: "legacy source", ImportKey: "request:legacy-scenario-qa-bugs/v1/source-id"})
	svc := NewService(NewSQLiteRepository(db), client, catalog)
	deferred, err := svc.runCategory(ctx, svc.definitions[0], clock)
	require.ErrorContains(t, err, "unavailable")
	require.Equal(t, 1, deferred)
	matches, err := svc.MatchingEntries(ctx, "agent-memory", []string{"defect-report"})
	require.NoError(t, err)
	require.Equal(t, map[string][]string{"legacy": {"defect-report"}}, matches)
	pending, err := svc.Coverage(ctx, "agent-memory", []string{"defect-report"})
	require.NoError(t, err)
	require.True(t, pending)
	deferred, err = svc.runCategory(ctx, svc.definitions[0], clock)
	require.NoError(t, err)
	require.Zero(t, deferred)
}

func TestNoticesDeliverDeliberateObservationsAndExplicitlyRoutedEvidence(t *testing.T) {
	ctx := t.Context()
	db, journalRepo := openTestStore(t, filepath.Join(t.TempDir(), "ledger.db"))
	category := Category{ID: "test-category", Name: "Test category", Definition: "An actionable observation."}
	client := decideFunc(func(_ context.Context, req *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error) {
		results := make([]*inferencev1.DecideResponse, len(req.Msg.GetItems()))
		for i := range results {
			results[i] = &inferencev1.DecideResponse{
				Method: inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL,
				Answers: []*inferencev1.DecisionAnswer{{
					QuestionId: category.ID,
					Type:       inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
					Choice:     "matches",
				}},
			}
		}
		return connect.NewResponse(&inferencev1.DecideBatchResponse{Results: results}), nil
	})
	svc := NewService(NewSQLiteRepository(db), client, []Category{category})
	entries := []journal.Entry{
		{ID: "captured", Kind: "observation", Body: "A deliberately captured failure."},
		{ID: "routed", Kind: "note", Body: "Evidence from another producer.", RoutingHint: "defect"},
		{ID: "work", Kind: "work-record", Body: "Historical implementation work."},
		{ID: "blank-route", Kind: "note", Body: "An ordinary note.", RoutingHint: "  "},
		{ID: "account", Kind: "observation", Body: "Investigator summary.", RoutingHint: "defect-account", ImportKey: "request:remember/scenario-qa/defect-fixture/1/0"},
		{ID: "archived-account", Kind: "legacy-bug-disposition", Body: "Imported case summary.", RoutingHint: "defect-account"},
		{ID: "unpublished", Kind: "observation", Body: "A normal capture with an advisory hint.", RoutingHint: "defect-account"},
	}
	for _, entry := range entries {
		appendEntry(t, journalRepo, entry)
	}
	_, err := svc.runCategory(ctx, svc.definitions[0], schedule.NewFake(time.Now()))
	require.NoError(t, err)
	matches, err := svc.MatchingEntries(ctx, "agent-memory", []string{category.ID})
	require.NoError(t, err)
	require.Len(t, matches, len(entries))

	notices, err := svc.Notices(ctx, "agent-memory", category.ID, 0, 2)
	require.NoError(t, err)
	require.Len(t, notices, 2)
	require.Equal(t, "captured", notices[0].ObservationID)
	require.Equal(t, "routed", notices[1].ObservationID)
	next, err := svc.Notices(ctx, "agent-memory", category.ID, notices[1].Sequence, 2)
	require.NoError(t, err)
	require.Len(t, next, 1)
	require.Equal(t, "unpublished", next[0].ObservationID)
}

func TestCatalogRejectsAmbiguousNormalizedNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "categories.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":1,"categories":[
{"id":"first","name":"First","aliases":["Bug Reports"],"definition":"first definition"},
{"id":"second","name":"bug-reports","definition":"second definition"}
]}`), 0o600))
	_, err := LoadCatalog(path)
	require.ErrorContains(t, err, `"bug-reports" is shared`)
}

func openTestStore(t *testing.T, path string) (*database.RoutedDB, *journal.SQLiteRepository) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: path, MaxOpenConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, database.EnsureSchemas(ctx, db.Primary(), database.SchemaProviderFunc(journal.Schema), database.SchemaProviderFunc(Schema)))
	return db, journal.NewSQLiteRepository(db.Primary())
}

func appendEntry(t *testing.T, repo *journal.SQLiteRepository, entry journal.Entry) {
	t.Helper()
	_, err := repo.Append(context.Background(), entry, nil)
	require.NoError(t, err)
}
