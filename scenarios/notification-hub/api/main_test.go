package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"notification-hub/internal/hub"
	"notification-hub/internal/modules"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
)

func TestLeasedTestModeRequestReadsOnlyTheDisposableAsk(t *testing.T) {
	ctx := context.Background()
	routed, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	require.NoError(t, err)
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), "e1-fixture-lease", 0))

	service := hub.New(routed, schedule.System(), log.New(io.Discard, "", 0))
	var questions []string
	var timeline []hub.Notification
	mux := apihttp.TestModeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subject := service.ResolveRecipient(r.Context())
		asks, err := service.ListAsks(r.Context(), subject, true, 50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, ask := range asks {
			questions = append(questions, ask.Question)
		}
		timeline, err = service.ListForRecipient(r.Context(), subject, 50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(questions)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/asks", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, "fixture reads failed: %s", response.Body.String())
	require.Equal(t, []string{"Choose a test-only option"}, questions)
	require.Len(t, timeline, 10)
	byID := make(map[string]hub.Notification, len(timeline))
	for _, item := range timeline {
		byID[item.ID] = item
	}
	require.Equal(t, "pending", byID[hub.TestNotificationID].State)
	require.Equal(t, "delivered", byID["e4-test-delivered"].State)
	require.Equal(t, "answered", byID["e4-test-history"].State)
	for id, state := range map[string]string{
		"e4-test-pending": "pending", "e4-test-held": "held", "e4-test-routed": "routed",
		"e4-test-failed": "failed", "e4-test-unroutable": "unroutable", "e4-test-suppressed": "suppressed",
		"e4-test-unknown": "future-state",
	} {
		require.Equal(t, state, byID[id].State, "fixture preserves every recognized and unknown delivery state")
	}

	stats := routed.LeaseStats()
	require.Greater(t, stats.TestPoolRequests, int64(0))
	require.Zero(t, stats.PrimaryDuringTestModeRequests)
	var primaryAsks int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM asks`).Scan(&primaryAsks))
	require.Zero(t, primaryAsks, "fixture must never be written to the primary database")
	var answers int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM answers`).Scan(&answers))
	require.Zero(t, answers, "the disposable fixture remains unanswered")
	testCtx, err := routed.PinTestPool(ctx, "e1-fixture-lease")
	require.NoError(t, err)
	var fixtureReceipts, fixtureAttempts, fixtureAnswers int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM receipts WHERE notification_id = ?`, "e4-test-delivered").Scan(&fixtureReceipts))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM delivery_attempts WHERE notification_id = ?`, "e4-test-delivered").Scan(&fixtureAttempts))
	var readAskAnswers int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM answers WHERE ask_id = ?`, hub.TestAskID).Scan(&readAskAnswers))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM answers WHERE ask_id = ?`, "e4-test-answered-ask").Scan(&fixtureAnswers))
	require.Equal(t, 1, fixtureReceipts, "adapter receipt is explicit fixture data")
	require.Equal(t, 1, fixtureAttempts, "delivery attempt is distinct from current state")
	require.Zero(t, readAskAnswers, "an ordinary ask read must not acknowledge or answer it")
	require.Equal(t, 1, fixtureAnswers, "History contains only a closed answered ask")
	t.Logf("lease_id=%s test_pool_requests=%d primary_during_test_mode_requests=%d primary_asks=%d primary_answers=%d", routed.ActiveLeaseID(), stats.TestPoolRequests, stats.PrimaryDuringTestModeRequests, primaryAsks, answers)
}

func TestTestModeRequestsFailClosedWithoutAnActiveLease(t *testing.T) {
	ctx := context.Background()
	routed, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	require.NoError(t, err)
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	service := hub.New(routed, schedule.System(), log.New(io.Discard, "", 0))
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, err := service.ListAsks(r.Context(), service.ResolveRecipient(r.Context()), true, 50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux := apihttp.TestModeMiddleware(requireRoutedTestLease(routed, next))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/asks", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.False(t, called, "handler must not reach storage without a lease")
	require.Zero(t, routed.LeaseStats().PrimaryDuringTestModeRequests)

	leaseID := "cleared-fixture-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), leaseID, 0))
	require.NoError(t, routed.ClearTestPool(leaseID))
	called = false
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.False(t, called, "a cleared lease must not route test-mode requests to primary")
	require.Zero(t, routed.LeaseStats().PrimaryDuringTestModeRequests)

	var primaryAsks int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM asks`).Scan(&primaryAsks))
	require.Zero(t, primaryAsks, "failed test-mode requests must have no primary-store effect")
}

func TestE4FixtureProfilesStayInsideThePinnedTestPool(t *testing.T) {
	for _, fixture := range []string{"empty-open-asks", "asks-read-failure"} {
		t.Run(fixture, func(t *testing.T) {
			ctx := context.Background()
			routed, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
			require.NoError(t, err)
			defer routed.Close()
			require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
			routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
				if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
					return err
				}
				return seedTestRecipient(ctx, pool)
			})
			leaseID := "e4-fixture-profile-lease"
			require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), leaseID, 0))
			service := hub.New(routed, schedule.System(), log.New(io.Discard, "", 0))
			var listErr error
			var listed int
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var asks []hub.AskRecord
				asks, listErr = service.ListAsks(r.Context(), service.ResolveRecipient(r.Context()), true, 50)
				listed = len(asks)
				if listErr != nil {
					http.Error(w, "read failed", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			mux := apihttp.TestModeMiddleware(requireRoutedTestLease(routed, next))
			sendList := func(referer string) *httptest.ResponseRecorder {
				request := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.conversations.ConversationsService/ListAsks", nil)
				request.Header.Set("X-Vrooli-Test-Mode", "1")
				request.Header.Set("Referer", referer)
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				return response
			}
			firstResponse := sendList("http://notification-hub.invalid/updates?vrooli_test_mode=1&e4_fixture=" + fixture)
			require.NoError(t, listErr, "fixture selection on Updates is deferred until the Needs you list read")
			require.Equal(t, 1, listed)
			require.Equal(t, http.StatusNoContent, firstResponse.Code)
			listErr = nil
			response := sendList("http://notification-hub.invalid/asks?vrooli_test_mode=1")
			if fixture == "asks-read-failure" {
				require.Error(t, listErr, "the actual leased ListAsks query must fail")
				require.Equal(t, http.StatusInternalServerError, response.Code)
			} else {
				require.NoError(t, listErr)
				require.Zero(t, listed, "empty-open-asks profile must return an authoritative empty open list")
				require.Equal(t, http.StatusNoContent, response.Code)
			}
			listErr = nil
			repeated := sendList("http://notification-hub.invalid/asks?vrooli_test_mode=1")
			if fixture == "asks-read-failure" {
				require.Error(t, listErr, "the selected read failure remains active for follow-up list queries in the same routed case")
				require.Equal(t, http.StatusInternalServerError, repeated.Code)
			} else {
				require.NoError(t, listErr)
				require.Zero(t, listed, "follow-up list queries must retain the selected empty state")
				require.Equal(t, http.StatusNoContent, repeated.Code)
			}
			stats := routed.LeaseStats()
			require.Greater(t, stats.TestPoolRequests, int64(0))
			require.Zero(t, stats.PrimaryDuringTestModeRequests)
			testCtx, err := routed.PinTestPool(ctx, leaseID)
			require.NoError(t, err)
			var restored int
			if fixture == "asks-read-failure" {
				require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM asks`).Scan(&restored), "the leased asks table is restored after the injected read failure")
			} else {
				require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM asks WHERE id = ? AND state = 'pending'`, hub.TestAskID).Scan(&restored), "the ordinary disposable ask is restored after the empty profile read")
			}
			require.Greater(t, restored, 0)
			var primaryAsks int
			require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM asks`).Scan(&primaryAsks))
			require.Zero(t, primaryAsks, "E4 fixture mutations must never touch primary storage")
		})
	}
}
