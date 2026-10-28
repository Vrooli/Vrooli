package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"notification-hub/internal/hub"
	internalidentity "notification-hub/internal/identity"
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
	require.Len(t, timeline, 9)
	byID := make(map[string]hub.Notification, len(timeline))
	for _, item := range timeline {
		byID[item.ID] = item
	}
	require.Equal(t, "pending", byID[hub.TestNotificationID].State)
	require.Equal(t, "delivered", byID["e4-test-delivered"].State)
	require.Equal(t, "answered", byID["e4-test-history"].State)
	for id, state := range map[string]string{
		"DEMO-N043": "pending", "e4-test-held": "held", "e4-test-routed": "routed",
		"e4-test-failed": "failed", "e4-test-unroutable": "unroutable", "e4-test-suppressed": "suppressed",
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
	request.Header.Set("Referer", "http://notification-hub.invalid/settings?e22_fixture=settings-owner")
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

func TestE4DeliveryAnalyticsFailureIsOneShotLeaseOnlyAndRecordsRemainAvailable(t *testing.T) {
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
	const leaseID = "e4-delivery-analytics-failure-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), leaseID, 0))
	service := hub.New(routed, schedule.System(), log.New(io.Discard, "", 0))
	service.SetDefaultRecipient(hub.TestRecipientSubject)

	const asksPath = "/vrooli.notification_hub.v1.conversations.ConversationsService/ListAsks"
	const analyticsPath = "/vrooli.notification_hub.v1.delivery.DeliveryService/GetAnalytics"
	const timelinePath = "/vrooli.notification_hub.v1.delivery.DeliveryService/GetTimeline"
	var analyticsErr error
	var notificationCount int
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case asksPath:
			_, err := service.ListAsks(r.Context(), service.ResolveRecipient(r.Context()), true, 50)
			if err != nil {
				http.Error(w, "fixture selector read failed", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case analyticsPath:
			_, analyticsErr = service.AnalyticsForRecipient(r.Context(), "", "", service.ResolveRecipient(r.Context()))
			if analyticsErr != nil {
				http.Error(w, "analytics read unavailable", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case timelinePath:
			records, err := service.ListForRecipient(r.Context(), service.ResolveRecipient(r.Context()), 50)
			if err != nil {
				http.Error(w, "delivery records unavailable", http.StatusInternalServerError)
				return
			}
			notificationCount = len(records)
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
	handler := apihttp.TestModeMiddleware(requireRoutedTestLease(routed, next))
	request := func(path, referer string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("X-Vrooli-Test-Mode", "1")
		if referer != "" {
			req.Header.Set("Referer", referer)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	selected := request(asksPath, "http://notification-hub.invalid/updates?vrooli_test_mode=1&e4_fixture=delivery-analytics-read-failure")
	require.Equal(t, http.StatusNoContent, selected.Code)

	failed := request(analyticsPath, "http://notification-hub.invalid/delivery?vrooli_test_mode=1")
	require.Equal(t, http.StatusServiceUnavailable, failed.Code)
	require.Error(t, analyticsErr, "the selected metrics failure must come from the actual analytics SQL read")
	require.NotContains(t, failed.Body.String(), "machine")
	require.NotContains(t, failed.Body.String(), "provider")

	// A record read is a separate request and remains available while the
	// one-shot analytics fault has been consumed.
	records := request(timelinePath, "http://notification-hub.invalid/delivery?vrooli_test_mode=1")
	require.Equal(t, http.StatusNoContent, records.Code)
	require.Greater(t, notificationCount, 0)

	analyticsErr = nil
	retried := request(analyticsPath, "http://notification-hub.invalid/delivery?vrooli_test_mode=1")
	require.Equal(t, http.StatusNoContent, retried.Code, "the selected retry reads the restored analytics table")
	require.NoError(t, analyticsErr)

	testCtx, err := routed.PinTestPool(ctx, leaseID)
	require.NoError(t, err)
	var pending, pendingReceipts, backupTables, restoredAttemptRows int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM notifications WHERE id = 'DEMO-N043' AND requested_by = ?`, hub.TestRecipientSubject).Scan(&pending))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM receipts WHERE notification_id = 'DEMO-N043'`).Scan(&pendingReceipts))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='delivery_attempts_e4_fixture_backup'`).Scan(&backupTables))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM delivery_attempts WHERE notification_id='e4-test-delivered'`).Scan(&restoredAttemptRows))
	require.Equal(t, 1, pending)
	require.Zero(t, pendingReceipts, "Pending remains truthful without an adapter receipt")
	require.Zero(t, backupTables, "fixture cleanup restores the routed pool table")
	require.Equal(t, 1, restoredAttemptRows)

	stats := routed.LeaseStats()
	require.Greater(t, stats.TestPoolRequests, int64(0))
	require.Zero(t, stats.PrimaryDuringTestModeRequests)
	var primaryNotifications int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM notifications`).Scan(&primaryNotifications))
	require.Zero(t, primaryNotifications, "E4 records and the injected read fault never touch primary storage")
	t.Logf("lease_id=%s test_pool_requests=%d primary_during_test_mode_requests=%d primary_notifications=%d test_root_writes=0 external_effects=0", leaseID, stats.TestPoolRequests, stats.PrimaryDuringTestModeRequests, primaryNotifications)
}

func TestE4DeliveryUnknownStateTransitionsSelectedRecordOnlyInLeasedPool(t *testing.T) {
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
	const leaseID = "e4-delivery-unknown-state-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), leaseID, 0))
	service := hub.New(routed, schedule.System(), log.New(io.Discard, "", 0))
	service.SetDefaultRecipient(hub.TestRecipientSubject)
	const asksPath = "/vrooli.notification_hub.v1.conversations.ConversationsService/ListAsks"
	const timelinePath = "/vrooli.notification_hub.v1.delivery.DeliveryService/GetTimeline"
	var selectedState string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == asksPath {
			if _, err := service.ListAsks(r.Context(), service.ResolveRecipient(r.Context()), true, 50); err != nil {
				http.Error(w, "fixture selector read failed", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == timelinePath {
			records, err := service.ListForRecipient(r.Context(), service.ResolveRecipient(r.Context()), 50)
			if err != nil {
				http.Error(w, "delivery records unavailable", http.StatusInternalServerError)
				return
			}
			for _, record := range records {
				if record.ID == "DEMO-N043" {
					selectedState = record.State
				}
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	})
	handler := apihttp.TestModeMiddleware(requireRoutedTestLease(routed, next))
	request := func(path, referer string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("X-Vrooli-Test-Mode", "1")
		if referer != "" {
			req.Header.Set("Referer", referer)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	selected := request(asksPath, "http://notification-hub.invalid/updates?vrooli_test_mode=1&e4_fixture=delivery-unknown-state")
	require.Equal(t, http.StatusNoContent, selected.Code)
	unknown := request(timelinePath, "http://notification-hub.invalid/updates?vrooli_test_mode=1")
	require.Equal(t, http.StatusNoContent, unknown.Code)
	require.Equal(t, "future-state", selectedState)
	selectedState = ""
	reset := request(asksPath, "http://notification-hub.invalid/updates?vrooli_test_mode=1&e4_fixture=delivery-pending-state")
	require.Equal(t, http.StatusNoContent, reset.Code)
	pending := request(timelinePath, "http://notification-hub.invalid/updates?vrooli_test_mode=1")
	require.Equal(t, http.StatusNoContent, pending.Code)
	require.Equal(t, "pending", selectedState)
	testCtx, err := routed.PinTestPool(ctx, leaseID)
	require.NoError(t, err)
	var receipts, primaryRecords int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM receipts WHERE notification_id = 'DEMO-N043'`).Scan(&receipts))
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM notifications`).Scan(&primaryRecords))
	require.Zero(t, receipts, "the selected record remains without an adapter receipt")
	require.Zero(t, primaryRecords, "state transition remains inside the disposable leased pool")
	stats := routed.LeaseStats()
	require.Greater(t, stats.TestPoolRequests, int64(0))
	require.Zero(t, stats.PrimaryDuringTestModeRequests)
	t.Logf("lease_id=%s test_pool_requests=%d primary_during_test_mode_requests=%d primary_notifications=%d test_root_writes=0 external_effects=0", leaseID, stats.TestPoolRequests, stats.PrimaryDuringTestModeRequests, primaryRecords)
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

func TestE4NH162HistoryFixtureIsDeterministicAndLeasePinned(t *testing.T) {
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
	leaseID := "e4-nh162-history-fixture-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), leaseID, 0))
	service := hub.New(routed, schedule.System(), log.New(io.Discard, "", 0))
	var records []hub.AskRecord
	var listErr error
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		records, listErr = service.ListAsks(r.Context(), service.ResolveRecipient(r.Context()), false, 200)
		if listErr != nil {
			http.Error(w, "history read failed", http.StatusInternalServerError)
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
	assertInventory := func() {
		require.NoError(t, listErr)
		require.Len(t, records, 200, "the selected latest-200 fixture must fill the bounded History request")
		for index, record := range records {
			rank := 200 - index
			require.Equal(t, "answered", record.State, "every selected synthetic row represents an answered decision")
			require.Equal(t, fmt.Sprintf("DEMO-H%d", rank), record.Source)
			require.Equal(t, "continue", record.Answer)
			if rank == 200 {
				require.Equal(t, "e4-test-answered-ask", record.ID, "the first selected row keeps the exact E4 answered-detail route")
			} else {
				require.Equal(t, fmt.Sprintf("e4-test-nh162-ask-%03d", rank), record.ID)
			}
		}
		require.Equal(t, "Validate the recording fix", records[0].Question)
		require.Equal(t, "Confirm the release checklist", records[1].Question)
		require.Equal(t, "Review capture check 198", records[2].Question)
		require.Equal(t, "Review capture check 1", records[199].Question)
		for index := 1; index < len(records); index++ {
			require.True(t, records[index-1].CreatedAt > records[index].CreatedAt, "the full fixture order is strict and deterministic at index %d", index)
		}
	}
	response := sendList("http://notification-hub.invalid/updates?vrooli_test_mode=1&e4_fixture=nh16.2-history")
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.NoError(t, listErr)
	require.Len(t, records, 2, "the selected History fixture is deferred while Updates reads the common E4 seed")
	response = sendList("http://notification-hub.invalid/history?vrooli_test_mode=1")
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	assertInventory()
	response = sendList("http://notification-hub.invalid/updates?vrooli_test_mode=1")
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	assertInventory() // the consumed History selection persists in the disposable pool

	stats := routed.LeaseStats()
	require.GreaterOrEqual(t, stats.TestPoolRequests, int64(3))
	require.Zero(t, stats.PrimaryDuringTestModeRequests)
	var primaryAsks, primaryAnswers, primaryNotifications int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM asks`).Scan(&primaryAsks))
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM answers`).Scan(&primaryAnswers))
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM notifications`).Scan(&primaryNotifications))
	require.Zero(t, primaryAsks, "fixture creation never writes primary asks")
	require.Zero(t, primaryAnswers, "fixture creation never writes primary answers")
	require.Zero(t, primaryNotifications, "fixture creation never writes primary notifications")
	testCtx, err := routed.PinTestPool(ctx, leaseID)
	require.NoError(t, err)
	archivePage, err := service.ListAsksPage(testCtx, hub.TestRecipientSubject, hub.AskListOptions{Query: "Archived review 30", StateFilter: "answered", Limit: 50})
	require.NoError(t, err)
	require.Len(t, archivePage.Asks, 1, "the leased fixture supports searching beyond the old 200-record boundary")
	require.Equal(t, "DEMO-ARCHIVE-30", archivePage.Asks[0].Source)
	require.Empty(t, archivePage.NextCursor)
	var testAsks, testAnswers int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM asks WHERE id = 'e4-test-answered-ask' OR id LIKE 'e4-test-nh162-ask-%'`).Scan(&testAsks))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM answers WHERE ask_id = 'e4-test-answered-ask' OR ask_id LIKE 'e4-test-nh162-ask-%'`).Scan(&testAnswers))
	require.Equal(t, 230, testAsks)
	require.Equal(t, 230, testAnswers)
	t.Logf("fixture=nh16.2-history lease_id=%s list_count=%d test_pool_requests=%d primary_during_test_mode_requests=%d primary_asks=%d primary_answers=%d primary_notifications=%d", leaseID, len(records), stats.TestPoolRequests, stats.PrimaryDuringTestModeRequests, primaryAsks, primaryAnswers, primaryNotifications)
}

func TestE22SettingsOwnerFixtureIsLeasePinnedAndPersistsOnlyInTheTestPool(t *testing.T) {
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
	const leaseID = "e22-settings-owner-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), leaseID, 0))
	service := hub.New(routed, schedule.System(), log.New(io.Discard, "", 0))
	type ownerResult struct {
		Subject string            `json:"subject"`
		Quiet   []hub.QuietWindow `json:"quiet"`
		Order   []string          `json:"order"`
		Devices []hub.Device      `json:"devices"`
	}
	var observed ownerResult
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, err := internalidentity.Owner(r.Context(), r.Header, nil)
		if err != nil {
			http.Error(w, "owner is not authenticated", http.StatusUnauthorized)
			return
		}
		observed.Subject = owner.Subject
		recipient := service.ResolveRecipient(r.Context())
		observed.Quiet, err = service.ListQuietWindows(r.Context(), recipient)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		chain, err := service.GetEscalationChainReadOnly(r.Context(), recipient)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, step := range chain.Steps {
			observed.Order = append(observed.Order, step.Channel)
		}
		observed.Devices, err = service.ListDevices(r.Context(), recipient)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(observed)
	})
	mux := apihttp.TestModeMiddleware(requireRoutedTestLease(routed, next))
	request := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.identity.IdentityService/GetSession", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	request.Header.Set("Referer", "http://notification-hub.invalid/settings?e22_fixture=settings-owner")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, "fixture request failed: %s", response.Body.String())
	require.Equal(t, hub.TestRecipientSubject, observed.Subject, "the test owner is the isolated fixture recipient")
	require.Len(t, observed.Quiet, 2)
	require.Equal(t, "America/New_York", observed.Quiet[0].Timezone)
	require.Equal(t, "Asia/Tokyo", observed.Quiet[1].Timezone)
	require.False(t, observed.Quiet[0].CriticalOverride)
	require.False(t, observed.Quiet[1].CriticalOverride)
	require.Equal(t, []string{"web_push", "x-legacy-route"}, observed.Order)
	require.Len(t, observed.Devices, 1)
	require.ElementsMatch(t, []string{"email", "webhook", "browser", "web_push"}, observed.Devices[0].Channels)

	// Re-reading with the same lease observes the same rows; fixture seeding is
	// one-time, so later editor writes are not overwritten on each API request.
	observed = ownerResult{}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, []string{"web_push", "x-legacy-route"}, observed.Order)

	stats := routed.LeaseStats()
	require.Greater(t, stats.TestPoolRequests, int64(0))
	require.Zero(t, stats.PrimaryDuringTestModeRequests)
	testCtx, err := routed.PinTestPool(ctx, leaseID)
	require.NoError(t, err)
	var leasedWindows, leasedChannels int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM quiet_windows WHERE id LIKE 'e22-%'`).Scan(&leasedWindows))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM escalation_chains WHERE id LIKE 'e22-%'`).Scan(&leasedChannels))
	require.Equal(t, 2, leasedWindows)
	require.Equal(t, 2, leasedChannels)
	var primaryWindows, primaryDevices int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM quiet_windows WHERE id LIKE 'e22-%'`).Scan(&primaryWindows))
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM devices WHERE id='e22-settings-device'`).Scan(&primaryDevices))
	require.Zero(t, primaryWindows)
	require.Zero(t, primaryDevices)

	// The selected order-state fixture changes only the leased order and is
	// consumed once; repeated page reads must not overwrite later owner edits.
	canonical := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.recipients.RecipientsService/GetEscalationChain", nil)
	canonical.Header.Set("X-Vrooli-Test-Mode", "1")
	canonical.Header.Set("Referer", "http://notification-hub.invalid/settings?vrooli_test_mode=1&e22_fixture=settings-owner&e22_order=canonical")
	response = httptest.NewRecorder()
	observed = ownerResult{}
	mux.ServeHTTP(response, canonical)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, []string{"web_push", "email"}, observed.Order)
	_, err = service.SetEscalationChain(testCtx, hub.TestRecipientSubject, []string{"email"})
	require.NoError(t, err)
	response = httptest.NewRecorder()
	observed = ownerResult{}
	mux.ServeHTTP(response, canonical)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, []string{"email"}, observed.Order, "the fixture must not overwrite a later owner edit")

	// The selected Add channel state starts with Web Push only, leaving Email
	// available in the picker. Its seed is also one-time per lease.
	single := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.recipients.RecipientsService/GetEscalationChain", nil)
	single.Header.Set("X-Vrooli-Test-Mode", "1")
	single.Header.Set("Referer", "http://notification-hub.invalid/settings?vrooli_test_mode=1&e22_fixture=settings-owner&e22_order=single")
	response = httptest.NewRecorder()
	observed = ownerResult{}
	mux.ServeHTTP(response, single)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, []string{"web_push"}, observed.Order)
	rows, err := routed.QueryContext(testCtx, `SELECT channel FROM channel_addresses WHERE device_id='e22-settings-device' ORDER BY channel`)
	require.NoError(t, err)
	var availableChannels []string
	for rows.Next() {
		var channel string
		require.NoError(t, rows.Scan(&channel))
		availableChannels = append(availableChannels, channel)
	}
	require.NoError(t, rows.Err())
	require.NoError(t, rows.Close())
	require.Equal(t, []string{"email", "web_push"}, availableChannels, "single-order fixture should leave only Email as an available addition")
	_, err = service.SetEscalationChain(testCtx, hub.TestRecipientSubject, []string{"email"})
	require.NoError(t, err)
	response = httptest.NewRecorder()
	observed = ownerResult{}
	mux.ServeHTTP(response, single)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, []string{"email"}, observed.Order, "the one-channel fixture must not overwrite a later owner edit")
	stats = routed.LeaseStats()
	require.Greater(t, stats.TestPoolRequests, int64(0))
	require.Zero(t, stats.PrimaryDuringTestModeRequests)
	var primaryOrderRows int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM escalation_chains WHERE id LIKE 'e22-%'`).Scan(&primaryOrderRows))
	require.Zero(t, primaryOrderRows)

	// The fixture header alone cannot grant owner identity on an ordinary
	// request, even while a disposable lease remains installed.
	ordinary := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.identity.IdentityService/GetSession", nil)
	ordinary.Header.Set("Referer", "http://notification-hub.invalid/settings?e22_fixture=settings-owner")
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, ordinary)
	require.Equal(t, http.StatusUnauthorized, response.Code)

	requestWithFault := func(path, fault string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("X-Vrooli-Test-Mode", "1")
		req.Header.Set("Referer", "http://notification-hub.invalid/settings?vrooli_test_mode=1&e22_fixture=settings-owner&e22_fault="+fault)
		out := httptest.NewRecorder()
		mux.ServeHTTP(out, req)
		return out
	}
	setPath := "/vrooli.notification_hub.v1.recipients.RecipientsService/SetQuietWindow"
	listPath := "/vrooli.notification_hub.v1.recipients.RecipientsService/ListQuietWindows"
	setOrderPath := "/vrooli.notification_hub.v1.recipients.RecipientsService/SetEscalationChain"
	getOrderPath := "/vrooli.notification_hub.v1.recipients.RecipientsService/GetEscalationChain"
	require.Equal(t, http.StatusOK, requestWithFault(setPath, "partial").Code)
	require.Equal(t, http.StatusOK, requestWithFault(setPath, "partial").Code, "the second weekday write completes")
	require.Equal(t, http.StatusOK, requestWithFault(setPath, "partial").Code, "the third weekday write completes before its confirmation is hidden")
	require.Equal(t, http.StatusServiceUnavailable, requestWithFault(listPath, "partial").Code, "the third-day confirming read is unavailable after its write")
	require.Equal(t, http.StatusServiceUnavailable, requestWithFault(listPath, "partial").Code, "the immediate follow-up read remains explicitly unknown")
	require.Equal(t, http.StatusOK, requestWithFault(listPath, "partial").Code, "later refresh reads authoritative state normally")
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM quiet_windows WHERE recipient_id=(SELECT id FROM recipients WHERE subject=?)`, hub.TestRecipientSubject).Scan(&leasedWindows))
	require.Equal(t, 5, leasedWindows, "the five-day partial fixture replaces only the leased owner's quiet windows")
	require.Equal(t, http.StatusOK, requestWithFault(setPath, "unknown-readback").Code)
	require.Equal(t, http.StatusServiceUnavailable, requestWithFault(listPath, "unknown-readback").Code, "one confirming read is hidden after a write")
	require.Equal(t, http.StatusOK, requestWithFault(listPath, "unknown-readback").Code, "later refresh reads authoritative state normally")
	require.Equal(t, http.StatusOK, requestWithFault(setOrderPath, "unknown-order-readback").Code)
	require.Equal(t, http.StatusServiceUnavailable, requestWithFault(getOrderPath, "unknown-order-readback").Code, "one confirming escalation-order read is hidden after a write")
	require.Equal(t, http.StatusOK, requestWithFault(getOrderPath, "unknown-order-readback").Code, "later order refresh reads authoritative state normally")
	require.Equal(t, http.StatusOK, requestWithFault(setOrderPath, "unknown-order-readback").Code, "later order writes remain available after the one-shot fault")
	require.Equal(t, http.StatusOK, requestWithFault(getOrderPath, "unknown-order-readback").Code, "later writes do not re-arm the spent fault")

	// A new lease over this same disposable pool must start from the selected
	// fixture baseline even when an earlier lease left owner edits behind.
	_, err = service.SetQuietWindow(testCtx, hub.TestRecipientSubject, hub.QuietWindow{Weekday: 1, Start: "13:00", End: "14:00", Timezone: "Etc/UTC"})
	require.NoError(t, err)
	_, err = service.SetEscalationChain(testCtx, hub.TestRecipientSubject, []string{"email"})
	require.NoError(t, err)
	_, err = routed.ExecContext(testCtx, `UPDATE devices SET name='device-demo-email', machine_id='' WHERE id='e22-settings-device'`)
	require.NoError(t, err)
	require.NoError(t, seedE22SettingsFixture(testCtx, routed))
	var fixtureDeviceName, fixtureMachineID string
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT name, machine_id FROM devices WHERE id='e22-settings-device'`).Scan(&fixtureDeviceName, &fixtureMachineID))
	require.Equal(t, "E22 settings fixture device", fixtureDeviceName, "the next fixture baseline restores its device label")
	require.Equal(t, "e22-fixture-machine", fixtureMachineID, "the next fixture baseline restores its machine ID")
	require.NoError(t, seedE21DeviceRemovalReviewFixture(testCtx, routed))
	var selectedName, selectedMachine string
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT name, machine_id FROM devices WHERE id='device-demo-work'`).Scan(&selectedName, &selectedMachine))
	require.Equal(t, "Work browser", selectedName)
	require.Equal(t, "demo-workstation", selectedMachine)
	var selectedChannels, selectedWebPush int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*), sum(CASE WHEN channel='web_push' THEN 1 ELSE 0 END) FROM channel_addresses WHERE device_id='device-demo-work'`).Scan(&selectedChannels, &selectedWebPush))
	require.Equal(t, 1, selectedChannels, "the selected original has one known channel")
	require.Equal(t, 1, selectedWebPush)
	var remainingName, remainingMachine string
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT name, machine_id FROM devices WHERE id='demo-phone'`).Scan(&remainingName, &remainingMachine))
	require.Equal(t, "Travel phone", remainingName)
	require.Equal(t, "demo-mobile", remainingMachine)
	var remainingChannelCount int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM channel_addresses WHERE device_id='demo-phone' AND channel='web_push'`).Scan(&remainingChannelCount))
	require.Equal(t, 1, remainingChannelCount, "selected removal result retains a second disposable owner device")
	require.NoError(t, seedE22SettingsFixture(testCtx, routed))
	require.NoError(t, seedE21WebPushContentAccessFixture(testCtx, routed))
	addresses, err := service.ListChannelAddresses(testCtx, hub.TestRecipientSubject)
	require.NoError(t, err)
	webPushAddress := channelAddressByID(t, addresses, "e22-settings-web_push")
	require.Equal(t, "device-demo-work", webPushAddress.DeviceID)
	require.Equal(t, "fixture://web_push", webPushAddress.Address, "the disposable Web Push endpoint remains private fixture data")
	require.Equal(t, []string{"public", "private", "team-internal"}, webPushAddress.ApprovedLabels, "the selected editor starts with saved and preserved unknown labels")
	var primaryWebPushRows int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM channel_addresses WHERE id='e22-settings-web_push'`).Scan(&primaryWebPushRows))
	require.Zero(t, primaryWebPushRows, "the selected Web Push fixture remains only in the disposable pool")
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT name, machine_id FROM devices WHERE id='device-demo-work'`).Scan(&selectedName, &selectedMachine))
	require.Equal(t, "Work browser", selectedName)
	require.Equal(t, "demo-workstation", selectedMachine)
	require.NoError(t, seedE22SettingsFixture(testCtx, routed))
	var staleSelectedRows int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM devices WHERE id='device-demo-work'`).Scan(&staleSelectedRows))
	require.Zero(t, staleSelectedRows, "a later common fixture restores without retaining the selected-only device")
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM devices WHERE id='demo-phone'`).Scan(&staleSelectedRows))
	require.Zero(t, staleSelectedRows, "a later common fixture removes the selected-only remaining device")
	resetQuiet, err := service.ListQuietWindows(testCtx, hub.TestRecipientSubject)
	require.NoError(t, err)
	require.Len(t, resetQuiet, 2)
	require.Equal(t, "21:30", resetQuiet[0].Start)
	require.Equal(t, "America/New_York", resetQuiet[0].Timezone)
	resetOrder, err := service.GetEscalationChainReadOnly(testCtx, hub.TestRecipientSubject)
	require.NoError(t, err)
	require.Len(t, resetOrder.Steps, 2)
	require.Equal(t, []string{"web_push", "x-legacy-route"}, []string{resetOrder.Steps[0].Channel, resetOrder.Steps[1].Channel})
	var primaryFixtureRows int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT (SELECT count(*) FROM quiet_windows WHERE id LIKE 'e22-%') + (SELECT count(*) FROM escalation_chains WHERE id LIKE 'e22-%')`).Scan(&primaryFixtureRows))
	require.Zero(t, primaryFixtureRows, "restoring the selected baseline must remain inside the pinned disposable pool")

	quietCaseRequest := func(caseID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.identity.IdentityService/GetSession", nil)
		req.Header.Set("X-Vrooli-Test-Mode", "1")
		req.Header.Set("Referer", "http://notification-hub.invalid/settings?vrooli_test_mode=1&e22_fixture=settings-owner&e22_case="+caseID)
		out := httptest.NewRecorder()
		mux.ServeHTTP(out, req)
		return out
	}
	const firstQuietCase = "e22-quiet-monday-editor-dark-phone"
	const secondQuietCase = "e22-quiet-monday-editor-light-phone"
	observed = ownerResult{}
	require.Equal(t, http.StatusOK, quietCaseRequest(firstQuietCase).Code)
	require.Equal(t, "21:30", observed.Quiet[0].Start, "a selected case starts from the deterministic quiet baseline")
	_, err = service.SetQuietWindow(testCtx, hub.TestRecipientSubject, hub.QuietWindow{Weekday: 1, Start: "13:00", End: "14:00", Timezone: "Etc/UTC"})
	require.NoError(t, err)
	observed = ownerResult{}
	require.Equal(t, http.StatusOK, quietCaseRequest(firstQuietCase).Code)
	require.Equal(t, "13:00", observed.Quiet[0].Start, "same-case requests preserve edits for readback")
	observed = ownerResult{}
	require.Equal(t, http.StatusOK, quietCaseRequest(secondQuietCase).Code)
	require.Equal(t, "21:30", observed.Quiet[0].Start, "the next selected case restores its own starting baseline")

	for _, removalCase := range []string{"e22-quiet-remove-monday-light-phone", "e22-quiet-remove-monday-dark-phone"} {
		observed = ownerResult{}
		require.Equal(t, http.StatusOK, quietCaseRequest(removalCase).Code)
		require.Equal(t, "21:30", observed.Quiet[0].Start, "each removal-review case restores the selected saved-window baseline")
		_, err = service.SetQuietWindow(testCtx, hub.TestRecipientSubject, hub.QuietWindow{Weekday: 1, Start: "13:00", End: "14:00", Timezone: "Etc/UTC"})
		require.NoError(t, err)
		observed = ownerResult{}
		require.Equal(t, http.StatusOK, quietCaseRequest(removalCase).Code)
		require.Equal(t, "13:00", observed.Quiet[0].Start, "same-case removal journey keeps its exact selected baseline after setup")
	}

	const reachChannelCase = "e21-channel-address-readback"
	observed = ownerResult{}
	require.Equal(t, http.StatusOK, quietCaseRequest(reachChannelCase).Code)
	addresses, err = service.ListChannelAddresses(testCtx, hub.TestRecipientSubject)
	require.NoError(t, err)
	var emailAddress hub.ChannelAddress
	for _, address := range addresses {
		if address.ID == "e22-settings-email" {
			emailAddress = address
			break
		}
	}
	require.Equal(t, "owner@example.test", emailAddress.Address, "the disposable Reach fixture supplies a valid reserved example destination")
	_, err = service.UpdateChannelAddress(testCtx, hub.TestRecipientSubject, hub.ChannelAddress{ID: emailAddress.ID, Address: "changed@example.test", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)
	observed = ownerResult{}
	require.Equal(t, http.StatusOK, quietCaseRequest(reachChannelCase).Code)
	addresses, err = service.ListChannelAddresses(testCtx, hub.TestRecipientSubject)
	require.NoError(t, err)
	require.Equal(t, "changed@example.test", channelAddressByID(t, addresses, emailAddress.ID).Address, "same-case reads preserve an exact-ID channel-address update")
	observed = ownerResult{}
	require.Equal(t, http.StatusOK, quietCaseRequest("e22-quiet-monday-editor-light-desktop").Code)
	addresses, err = service.ListChannelAddresses(testCtx, hub.TestRecipientSubject)
	require.NoError(t, err)
	require.Equal(t, "owner@example.test", channelAddressByID(t, addresses, emailAddress.ID).Address, "the next selected case restores the channel baseline")
	_, err = service.UpdateChannelAddress(testCtx, hub.TestRecipientSubject, hub.ChannelAddress{ID: emailAddress.ID, Address: "stale@example.test", ApprovedLabels: []string{"public"}})
	require.NoError(t, err)
	observed = ownerResult{}
	require.Equal(t, http.StatusOK, quietCaseRequest("e22-quiet-monday-editor-desktop").Code)
	addresses, err = service.ListChannelAddresses(testCtx, hub.TestRecipientSubject)
	require.NoError(t, err)
	require.Equal(t, "owner@example.test", channelAddressByID(t, addresses, emailAddress.ID).Address, "a new Reach case restores the address left by an earlier lease case")

	for _, caseID := range []string{"e22-quiet-mixed-timezones-dark-phone", "e22-quiet-mixed-timezones-light-phone"} {
		observed = ownerResult{}
		require.Equal(t, http.StatusOK, quietCaseRequest(caseID).Code)
		require.Len(t, observed.Quiet, 5, "the selected mixed-zone fixture restores five weekday windows")
		for index, window := range observed.Quiet {
			require.Equal(t, index+1, window.Weekday)
			require.Equal(t, "13:00", window.Start)
			require.Equal(t, "14:00", window.End)
			wantZone := "Etc/UTC"
			if index == 0 {
				wantZone = "Europe/London"
			}
			require.Equal(t, wantZone, window.Timezone)
			require.False(t, window.CriticalOverride)
		}
	}
	for _, caseID := range []string{"e22-quiet-timezone-change-dark-phone", "e22-quiet-timezone-change-light-phone"} {
		observed = ownerResult{}
		require.Equal(t, http.StatusOK, quietCaseRequest(caseID).Code)
		require.Len(t, observed.Quiet, 2, "the selected timezone-change fixture preserves an unchanged weekday")
		require.Equal(t, hub.QuietWindow{ID: "e22-timezone-monday", Weekday: 1, Start: "13:00", End: "14:00", Timezone: "Etc/UTC"}, observed.Quiet[0])
		require.Equal(t, hub.QuietWindow{ID: "e22-timezone-friday", Weekday: 5, Start: "20:00", End: "21:00", Timezone: "Asia/Tokyo"}, observed.Quiet[1])
		_, err = service.SetQuietWindow(testCtx, hub.TestRecipientSubject, hub.QuietWindow{Weekday: 1, Start: "13:00", End: "14:00", Timezone: "Europe/London"})
		require.NoError(t, err)
		observed = ownerResult{}
		require.Equal(t, http.StatusOK, quietCaseRequest(caseID).Code)
		require.Equal(t, "Europe/London", observed.Quiet[0].Timezone, "same-case timezone edits remain available for authoritative readback")
		require.Equal(t, "Asia/Tokyo", observed.Quiet[1].Timezone, "the unchanged weekday retains its saved timezone")
	}
	var primaryTimezoneRows int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM quiet_windows WHERE id LIKE 'e22-timezone-%'`).Scan(&primaryTimezoneRows))
	require.Zero(t, primaryTimezoneRows, "selected timezone-change fixtures remain inside the pinned disposable pool")
	var primaryVisualRows int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM quiet_windows WHERE id LIKE 'e22-visual-mixed-%'`).Scan(&primaryVisualRows))
	require.Zero(t, primaryVisualRows, "selected mixed-zone visual fixtures stay inside the pinned disposable pool")

	for _, caseID := range []string{"e22-settings-owner-dark-journey", "e22-settings-owner-light-journey"} {
		_, err = service.SetQuietWindow(testCtx, hub.TestRecipientSubject, hub.QuietWindow{Weekday: 1, Start: "13:00", End: "14:00", Timezone: "Etc/UTC"})
		require.NoError(t, err)
		_, err = service.SetEscalationChain(testCtx, hub.TestRecipientSubject, []string{"email"})
		require.NoError(t, err)
		observed = ownerResult{}
		require.Equal(t, http.StatusOK, quietCaseRequest(caseID).Code)
		require.Len(t, observed.Quiet, 2, "each full owner journey restores its deterministic saved quiet baseline")
		require.Equal(t, "21:30", observed.Quiet[0].Start)
		require.Equal(t, "America/New_York", observed.Quiet[0].Timezone)
		require.Equal(t, "20:00", observed.Quiet[1].Start)
		require.Equal(t, "Asia/Tokyo", observed.Quiet[1].Timezone)
		require.Equal(t, []string{"web_push", "x-legacy-route"}, observed.Order, "each full owner journey starts from the selected unknown-channel order")
	}
	var primaryJourneyRows int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT (SELECT count(*) FROM quiet_windows WHERE id LIKE 'e22-%') + (SELECT count(*) FROM escalation_chains WHERE id LIKE 'e22-%')`).Scan(&primaryJourneyRows))
	require.Zero(t, primaryJourneyRows, "full owner journey resets remain inside the pinned disposable pool")
}

func TestE22ChannelUnknownReadbackFixtureIsExplicitlyScoped(t *testing.T) {
	page := "http://localhost:20001/settings/reach?vrooli_test_mode=1&e22_fixture=settings-owner&e22_case=e21-channel-unknown-readback&e22_fault=channel-unknown-readback"
	request := httptest.NewRequest(http.MethodGet, page, nil)
	require.Equal(t, "e21-channel-unknown-readback", e22FixtureCase(request))
	require.Empty(t, e22FixtureFault(request), "fault injection is read only from a request's owner-fixture referer")
	request.Header.Set("Referer", page)
	require.Equal(t, "channel-unknown-readback", e22FixtureFault(request))

	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings/reach?vrooli_test_mode=1&e22_case=e21-channel-unknown-readback&e22_fault=channel-unknown-readback", nil)
	require.Empty(t, e22FixtureCase(request), "fixture resets require the owner fixture selector")
	request.Header.Set("Referer", page)
	require.Equal(t, "channel-unknown-readback", e22FixtureFault(request), "fault selection only annotates the routed fixture request; it cannot write primary data")

	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings?e22_fixture=settings-owner&e22_case=e43-preferences", nil)
	require.Equal(t, "e43-preferences", e22FixtureCase(request), "the preference baseline is available only through its explicit owner fixture case")
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings/reach?e22_fixture=settings-owner&e22_case=e21-channel-create-unknown-readback", nil)
	require.Equal(t, "e21-channel-create-unknown-readback", e22FixtureCase(request), "the stable-ID create recovery baseline is an explicit fixture case")
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings/reach?e22_fixture=settings-owner&e22_case=e21-channel-address-validation-current", nil)
	require.Equal(t, "e21-channel-address-validation-current", e22FixtureCase(request), "the selected invalid-address original has a dedicated leased owner fixture")
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings/reach?vrooli_test_mode=1&e22_fixture=settings-owner&e22_case=e21-channel-address-readback&e21_fixture=webpush-content-access", nil)
	require.Equal(t, "e21-channel-address-readback", e22FixtureCase(request), "the Web Push editor reuses the established owner readback case selector")
	require.True(t, e21WebPushContentAccessFixtureSelected(request), "the selected Web Push state is separately scoped to the leased E21 journey")
	require.True(t, e22SettingsFixtureSelected(request))
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings/reach?vrooli_test_mode=1&e21_fixture=webpush-content-access", nil)
	require.True(t, e21WebPushContentAccessFixtureSelected(request))
	require.Empty(t, e22FixtureCase(request), "the Web Push flag alone must not install the owner fixture")
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/api", nil)
	request.Header.Set("X-Notification-Hub-E22-Fixture", "settings-owner")
	request.Header.Set("X-Notification-Hub-E22-Case", "e21-channel-address-readback")
	require.True(t, e22SettingsFixtureSelected(request), "the explicit routed owner fixture header survives cross-origin browser requests")
	require.Equal(t, "e21-channel-address-readback", e22FixtureCase(request), "the explicit routed case header keeps fixture seeding one-time")
	for _, caseID := range []string{"e21-device-removal-review-readback", "e21-webpush-content-access", "e21-permission-denied-dark-phone", "e21-permission-denied-light-phone"} {
		request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings/reach?e22_fixture=settings-owner&e22_case="+caseID, nil)
		require.Equal(t, caseID, e22FixtureCase(request), "mutating Reach journeys receive isolated deterministic fixture baselines")
	}
	for _, caseID := range []string{"e21-device-removal-review-readback", "e21-permission-denied-dark-phone", "e21-permission-denied-light-phone"} {
		request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings/reach?e22_fixture=settings-owner&e22_case="+caseID, nil)
		require.Equal(t, caseID, e22FixtureCase(request), "mutating Reach journeys receive isolated deterministic fixture baselines")
	}
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings?e22_fixture=settings-owner&e22_case=e22-quiet-mixed-timezones-dark-phone", nil)
	require.Equal(t, "e22-quiet-mixed-timezones-dark-phone", e22FixtureCase(request), "selected mixed-zone visual fixtures are explicitly allowlisted")
	for _, caseID := range []string{"e22-quiet-monday-editor-dark-phone", "e22-quiet-monday-editor-dark-narrow", "e22-quiet-japanese-editor-dark-phone", "e22-quiet-japanese-editor-light-phone", "e22-quiet-validation-states-dark-phone", "e22-quiet-validation-states-light-phone", "e22-order-add-channel-dark-phone", "e22-order-add-channel-light-phone", "e22-order-channel-editor-dark-desktop", "e22-order-channel-editor-dark-narrow-p1", "e22-order-channel-editor-dark-narrow-p2", "e22-order-channel-editor-dark-phone", "e22-order-channel-editor-light-desktop", "e22-order-channel-editor-light-narrow-p1", "e22-order-channel-editor-light-narrow-p2", "e22-order-channel-editor-light-phone", "e22-order-empty-draft-light-phone", "e22-quiet-remove-monday-dark-phone", "e22-quiet-remove-monday-light-phone"} {
		request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings?e22_fixture=settings-owner&e22_case="+caseID, nil)
		require.Equal(t, caseID, e22FixtureCase(request), "each routed visual case receives its own deterministic fixture baseline")
	}
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings?e22_fixture=settings-owner&e22_case=e22-settings-owner-light-journey", nil)
	require.Equal(t, "e22-settings-owner-light-journey", e22FixtureCase(request), "full owner journeys receive independent fixture baselines")
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings?e22_case=e43-preferences", nil)
	require.Empty(t, e22FixtureCase(request), "the preference fixture cannot be selected without the owner-fixture selector")
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings?e22_case=e22-quiet-mixed-timezones-dark-phone", nil)
	require.Empty(t, e22FixtureCase(request), "selected mixed-zone visual fixtures cannot be selected without the owner-fixture selector")
	request = httptest.NewRequest(http.MethodGet, "http://localhost:20001/settings?e22_case=e22-settings-owner-light-journey", nil)
	require.Empty(t, e22FixtureCase(request), "full owner journey fixtures cannot be selected without the owner-fixture selector")
}

func TestE21ChannelReadFailureFixtureIsExplicitlyScoped(t *testing.T) {
	cases := []string{
		"e21-reach-channel-read-gap-light-phone",
		"e21-reach-channel-read-gap-dark-phone",
		"e21-reach-channel-read-gap-dark-narrow",
		"e21-reach-channel-read-gap-light-narrow",
		"e21-reach-channel-read-gap-dark-desktop",
		"e21-reach-channel-read-gap-light-desktop",
	}
	for _, caseID := range cases {
		t.Run(caseID, func(t *testing.T) {
			page := "http://localhost:20001/settings/reach?vrooli_test_mode=1&e22_fixture=settings-owner&e22_case=" + caseID + "&e22_fault=channel-address-read-failure&e21_fixture=webpush-content-access"
			request := httptest.NewRequest(http.MethodPost, "http://localhost:20001/vrooli.notification_hub.v1.recipients.RecipientsService/ListChannelAddresses", nil)
			request.Header.Set("Referer", page)
			require.Equal(t, caseID, e22FixtureCase(request))
			require.True(t, e21ChannelReadGapFixtureSelected(request))
			require.Equal(t, "channel-address-read-failure", e22FixtureFault(request))
		})
	}

	request := httptest.NewRequest(http.MethodPost, "http://localhost:20001/vrooli.notification_hub.v1.recipients.RecipientsService/ListChannelAddresses", nil)
	request.Header.Set("Referer", "http://localhost:20001/settings/reach?vrooli_test_mode=1&e22_fixture=settings-owner&e22_case=e21-channel-address-readback&e22_fault=channel-address-read-failure")
	require.NotEqual(t, "e21-reach-channel-read-gap-light-phone", e22FixtureCase(request), "the synthetic read failure is reserved for its dedicated disposable cases")
	require.False(t, e21ChannelReadGapFixtureSelected(request), "ordinary channel-address reads cannot select the synthetic failure")
}

func channelAddressByID(t *testing.T, addresses []hub.ChannelAddress, id string) hub.ChannelAddress {
	t.Helper()
	for _, address := range addresses {
		if address.ID == id {
			return address
		}
	}
	t.Fatalf("channel address %q was not found", id)
	return hub.ChannelAddress{}
}

func TestE21ReachEmptyFixtureClearsOnlyLeasedOwnerRows(t *testing.T) {
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
	const leaseID = "e21-reach-empty-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), leaseID, 0))
	testCtx, err := routed.PinTestPool(ctx, leaseID)
	require.NoError(t, err)
	var testRecipientID string
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT id FROM recipients WHERE subject=?`, hub.TestRecipientSubject).Scan(&testRecipientID))
	_, err = routed.ExecContext(testCtx, `INSERT INTO devices (id, recipient_id, name, machine_id, created_at) VALUES ('stale-e21-device', ?, 'stale', 'stale-machine', '2026-01-01T00:00:00Z')`, testRecipientID)
	require.NoError(t, err)
	_, err = routed.ExecContext(testCtx, `INSERT INTO channel_addresses (id, device_id, channel, address, approved_labels, created_at) VALUES ('stale-e21-channel', 'stale-e21-device', 'email', 'owner@example.test', '["public"]', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = routed.ExecContext(testCtx, `INSERT INTO recipients (id, subject, trust_posture, created_at, updated_at) VALUES ('other-e21-recipient', 'other-e21@example.test', 'personal', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = routed.ExecContext(testCtx, `INSERT INTO devices (id, recipient_id, name, machine_id, created_at) VALUES ('other-e21-device', 'other-e21-recipient', 'Other owner', 'other-machine', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)
	_, err = routed.ExecContext(testCtx, `INSERT INTO channel_addresses (id, device_id, channel, address, approved_labels, created_at) VALUES ('other-e21-channel', 'other-e21-device', 'email', 'other@example.test', '["public"]', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)
	service := hub.New(routed, schedule.System(), log.New(io.Discard, "", 0))
	var subject string
	var devices []hub.Device
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, err := internalidentity.Owner(r.Context(), r.Header, nil)
		if err != nil {
			http.Error(w, "owner is not authenticated", http.StatusUnauthorized)
			return
		}
		subject = owner.Subject
		devices, err = service.ListDevices(r.Context(), service.ResolveRecipient(r.Context()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	mux := apihttp.TestModeMiddleware(requireRoutedTestLease(routed, next))
	request := httptest.NewRequest(http.MethodGet, "/vrooli.notification_hub.v1.recipients.RecipientsService/ListDevices", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	request.Header.Set("Referer", "http://notification-hub.invalid/settings/reach?vrooli_test_mode=1&e21_fixture=reach-empty")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, "empty Reach fixture failed: %s", response.Body.String())
	require.Equal(t, hub.TestRecipientSubject, subject)
	require.Empty(t, devices)
	stats := routed.LeaseStats()
	require.Greater(t, stats.TestPoolRequests, int64(0))
	require.Zero(t, stats.PrimaryDuringTestModeRequests)
	var testDevices, primaryDevices, otherDevices, otherChannels int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM devices WHERE recipient_id=?`, testRecipientID).Scan(&testDevices))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM devices WHERE recipient_id='other-e21-recipient'`).Scan(&otherDevices))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM channel_addresses WHERE device_id='other-e21-device'`).Scan(&otherChannels))
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM devices`).Scan(&primaryDevices))
	require.Zero(t, testDevices)
	require.Equal(t, 1, otherDevices, "Reach fixture reset must not remove another owner's device")
	require.Equal(t, 1, otherChannels, "Reach fixture reset must not remove another owner's channel")
	require.Zero(t, primaryDevices)
}

func TestE21DeviceReadFailureFixtureIsExplicitlyScoped(t *testing.T) {
	page := "http://localhost:20001/settings/reach?vrooli_test_mode=1&e22_fixture=settings-owner&e22_case=e21-device-read-failure-dark-phone&e22_fault=device-read-failure"
	request := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.recipients.RecipientsService/ListDevices", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	request.Header.Set("Referer", page)
	require.True(t, e22SettingsFixtureSelected(request), "device-read recovery requires the owner fixture selector")
	require.Equal(t, "device-read-failure", e22FixtureFault(request))
	require.True(t, e21DeviceReadFailureFixtureSelected(request), "only selected light/dark E21 cases may inject device read failures")

	request = httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.recipients.RecipientsService/ListDevices", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	request.Header.Set("Referer", "http://localhost:20001/settings/reach?vrooli_test_mode=1&e22_fixture=settings-owner&e22_case=e21-device-removal-review-readback&e22_fault=device-read-failure")
	require.False(t, e21DeviceReadFailureFixtureSelected(request), "fault does not apply to unrelated owner cases")

	request = httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.recipients.RecipientsService/ListDevices", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	request.Header.Set("Referer", "http://localhost:20001/settings/reach?vrooli_test_mode=1&e22_case=e21-device-read-failure-dark-phone&e22_fault=device-read-failure")
	require.False(t, e22SettingsFixtureSelected(request), "fault cannot escape the explicit owner fixture")

	request = httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.recipients.RecipientsService/ListDevices", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	request.Header.Set("X-Notification-Hub-E22-Fixture", "settings-owner")
	request.Header.Set("X-Notification-Hub-E22-Case", "e21-device-read-failure-light-phone")
	request.Header.Set("X-Notification-Hub-E22-Fault", "device-read-failure")
	require.Equal(t, "device-read-failure", e22FixtureFault(request), "the routed client may explicitly carry the allowlisted fault when Referer is reduced")
	require.True(t, e22SettingsFixtureSelected(request))

	request.Header.Set("X-Notification-Hub-E22-Case", "e21-device-removal-review-readback")
	require.Empty(t, e22FixtureFault(request), "the fault header is ignored for unrelated cases")

	request.Header.Del("X-Notification-Hub-E22-Fault")
	request.Header.Set("X-Notification-Hub-E22-Case", "e21-device-read-failure-dark-phone")
	request.Header.Set("Referer", "http://localhost:20001/settings?view=all&vrooli_test_mode=1&e22_fixture=settings-owner&e22_case=e21-device-read-failure-dark-phone&e22_fault=device-read-failure")
	require.Empty(t, e22FixtureFault(request), "device failure does not get consumed during Settings setup")
	request.Header.Set("Referer", "http://localhost:20001/settings/reach?vrooli_test_mode=1")
	request.Header.Set("X-Notification-Hub-E22-Fault", "device-read-failure")
	require.Equal(t, "device-read-failure", e22FixtureFault(request), "the routed Reach client carries the allowlisted fault when Referer is reduced")
}

func TestE21WebPushFixtureHeaderSeedsSelectedOwnerDeviceInLeasedPool(t *testing.T) {
	ctx := context.Background()
	routed, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(t.TempDir(), "primary.db")})
	require.NoError(t, err)
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		pool.SetMaxOpenConns(1)
		if _, err := pool.ExecContext(ctx, `PRAGMA foreign_keys=ON`); err != nil {
			return err
		}
		var foreignKeys int
		if err := pool.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
			return err
		}
		if foreignKeys != 1 {
			return errors.New("SQLite foreign-key enforcement is not enabled for the leased fixture pool")
		}
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	const leaseID = "e21-webpush-fixture-header-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "leased.db"), leaseID, 0))
	service := hub.New(routed, schedule.System(), log.New(io.Discard, "", 0))
	var subject string
	var devices []hub.Device
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, err := internalidentity.Owner(r.Context(), r.Header, nil)
		if err != nil {
			http.Error(w, "owner is not authenticated", http.StatusUnauthorized)
			return
		}
		subject = owner.Subject
		devices, err = service.ListDevices(r.Context(), service.ResolveRecipient(r.Context()))
		if err != nil {
			http.Error(w, "owner device read failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := apihttp.TestModeMiddleware(requireRoutedTestLease(routed, next))
	request := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.recipients.RecipientsService/ListDevices?vrooli_test_mode=1&e21_fixture=webpush-content-access&e22_fixture=settings-owner&e22_case=e21-channel-address-readback", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	request.Header.Set("X-Notification-Hub-E21-Fixture", "webpush-content-access")
	request.Header.Set("X-Notification-Hub-E22-Fixture", "settings-owner")
	request.Header.Set("X-Notification-Hub-E22-Case", "e21-channel-address-readback")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, "selected fixture read failed: %s", response.Body.String())
	require.Equal(t, hub.TestRecipientSubject, subject)
	require.Equal(t, []hub.Device{{ID: "device-demo-work", Name: "Work browser", MachineID: "demo-workstation", Channels: []string{"browser", "email", "web_push", "webhook"}}}, devices)
	stats := routed.LeaseStats()
	require.Greater(t, stats.TestPoolRequests, int64(0))
	require.Zero(t, stats.PrimaryDuringTestModeRequests)
	var primaryDevices int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM devices`).Scan(&primaryDevices))
	require.Zero(t, primaryDevices, "Web Push fixture writes must stay in the leased test pool")
}

func TestE21ReachEmptyFixtureTakesPrecedenceOverSettingsSeed(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/devices?vrooli_test_mode=1&e21_fixture=reach-empty&e22_fixture=settings-owner", nil)
	if !e21ReachEmptyFixtureSelected(request) {
		t.Fatal("expected selected Reach empty fixture")
	}
	if e22SettingsFixtureSelected(request) {
		t.Fatal("Settings fixture must not reseed devices during selected Reach empty state")
	}

	request = httptest.NewRequest(http.MethodGet, "/api/devices?vrooli_test_mode=1&e22_fixture=settings-owner", nil)
	if !e22SettingsFixtureSelected(request) {
		t.Fatal("expected Settings fixture when Reach empty fixture is absent")
	}

	request = httptest.NewRequest(http.MethodGet, "/api/devices?vrooli_test_mode=1&e22_fixture=settings-owner", nil)
	request.Header.Set("X-Notification-Hub-E21-Fixture", "reach-empty")
	if !e21ReachEmptyFixtureSelected(request) || e22SettingsFixtureSelected(request) {
		t.Fatal("explicit leased Reach empty selector must take precedence over Settings seed")
	}
}
