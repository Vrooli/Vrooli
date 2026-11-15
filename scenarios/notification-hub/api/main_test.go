package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	notificationH "notification-hub/handlers/notifications"
	recipientsH "notification-hub/handlers/recipients"
	"notification-hub/internal/hub"
	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/filedb"
	"notification-hub/internal/testutil/hubtest"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	recipientsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients"
	recipientsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients/recipients_v1connect"
)

func TestLeasedTestModeRequestReadsOnlyTheDisposableAsk(t *testing.T) {
	ctx := context.Background()
	routed := filedb.Open(t, "primary.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), "e1-fixture-lease", 0))

	service := hubtest.New(t, routed, schedule.System(), log.New(io.Discard, "", 0))
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

func TestLeasedTestModeAcceptsAndReadsRealProducerNotification(t *testing.T) {
	ctx := context.Background()
	routed := filedb.Open(t, "primary.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	leaseID := "cf5-notification-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), leaseID, 0))

	service := hubtest.New(t, routed, schedule.System(), log.New(io.Discard, "", 0))
	router := mux.NewRouter()
	notificationH.Module(service).Mount(router)
	server := httptest.NewServer(apihttp.TestModeMiddleware(requireRoutedTestLease(routed, router)))
	defer server.Close()
	client := notificationsconnect.NewNotificationsServiceClient(server.Client(), server.URL)
	send := connect.NewRequest(&notificationsv1.SendRequest{
		Title: "Treasury funds received", Body: "Received 1.000000 USDC and credited the rehearsal budget.",
		Urgency: "normal", SensitivityLabel: "financial", IdempotencyKey: "cf5-receipt-1:funding-notice", DedupeKey: "cf5-receipt-1",
	})
	send.Header().Set("X-Vrooli-Test-Mode", "1")
	created, err := client.Send(ctx, send)
	require.NoError(t, err)
	require.NotEmpty(t, created.Msg.GetNotification().GetId())

	read := connect.NewRequest(&notificationsv1.GetRequest{Id: created.Msg.GetNotification().GetId()})
	read.Header().Set("X-Vrooli-Test-Mode", "1")
	got, err := client.Get(ctx, read)
	require.NoError(t, err)
	require.Equal(t, "Treasury funds received", got.Msg.GetNotification().GetTitle())
	require.Contains(t, got.Msg.GetNotification().GetBody(), "credited the rehearsal budget")
	stats := routed.LeaseStats()
	require.Greater(t, stats.TestPoolRequests, int64(0))
	require.Zero(t, stats.PrimaryDuringTestModeRequests)
	var primaryCount int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM notifications`).Scan(&primaryCount))
	require.Zero(t, primaryCount, "notification writes remain in the disposable routed lease")
	t.Logf("lease_id=%s notification_id=%s test_pool_requests=%d primary_during_test_mode_requests=%d", leaseID, created.Msg.GetNotification().GetId(), stats.TestPoolRequests, stats.PrimaryDuringTestModeRequests)
}

func TestTestModeRequestsFailClosedWithoutAnActiveLease(t *testing.T) {
	ctx := context.Background()
	routed := filedb.Open(t, "primary.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	service := hubtest.New(t, routed, schedule.System(), log.New(io.Discard, "", 0))
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
	require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), leaseID, 0))
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
	routed := filedb.Open(t, "primary.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	const leaseID = "e4-delivery-analytics-failure-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), leaseID, 0))
	service := hubtest.New(t, routed, schedule.System(), log.New(io.Discard, "", 0))
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
	routed := filedb.Open(t, "primary.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	const leaseID = "e4-delivery-unknown-state-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), leaseID, 0))
	service := hubtest.New(t, routed, schedule.System(), log.New(io.Discard, "", 0))
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
			routed := filedb.Open(t, "primary.db")
			defer routed.Close()
			require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
			routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
				if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
					return err
				}
				return seedTestRecipient(ctx, pool)
			})
			leaseID := "e4-fixture-profile-lease"
			require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), leaseID, 0))
			service := hubtest.New(t, routed, schedule.System(), log.New(io.Discard, "", 0))
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

func TestE4DecisionLinksPhoneFixtureChangesOnlyTheLeasedAskAndRestoresIt(t *testing.T) {
	ctx := context.Background()
	routed := filedb.Open(t, "primary.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	leaseID := "e4-decision-links-phone-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), leaseID, 0))
	ctx, err := routed.PinTestPool(ctx, leaseID)
	require.NoError(t, err)
	service := hubtest.New(t, routed, schedule.System(), log.New(io.Discard, "", 0))
	service.SetAttachmentStore(hub.StaticBlobStore(blobstore.NewFilesystemBlobStore(t.TempDir())))
	var primaryBefore int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM asks WHERE id = ?`, hub.TestAskID).Scan(&primaryBefore))
	require.Zero(t, primaryBefore)
	var leasedBefore struct {
		question, allowedAnswers, optionsJSON, recommended, recommendationReason, defaultAnswer string
	}
	require.NoError(t, routed.QueryRowContext(ctx, `SELECT question, allowed_answers, options_json, recommended, recommendation_reason, default_answer FROM asks WHERE id = ?`, hub.TestAskID).Scan(&leasedBefore.question, &leasedBefore.allowedAnswers, &leasedBefore.optionsJSON, &leasedBefore.recommended, &leasedBefore.recommendationReason, &leasedBefore.defaultAnswer))

	var fixtureQuestion, fixtureOptions, fixtureRecommended string
	var fixtureAttachmentName, fixtureAttachmentPath string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := routed.QueryRowContext(r.Context(), `SELECT question, options_json, recommended FROM asks WHERE id = ?`, hub.TestAskID).Scan(&fixtureQuestion, &fixtureOptions, &fixtureRecommended)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := routed.QueryRowContext(r.Context(), `SELECT name, source_path FROM ask_attachments WHERE ask_id = ?`, hub.TestAskID).Scan(&fixtureAttachmentName, &fixtureAttachmentPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(fixtureQuestion))
	})
	request := httptest.NewRequest(http.MethodPost, "/vrooli.notification_hub.v1.conversations.ConversationsService/GetAsk", nil)
	request.Header.Set("X-Vrooli-Test-Mode", "1")
	request.Header.Set("X-Notification-Hub-E4-Fixture", "decision-links-phone")
	request.Header.Set("Referer", "http://notification-hub.invalid/asks/"+hub.TestAskID)
	response := httptest.NewRecorder()
	apihttp.TestModeMiddleware(requireRoutedTestLease(routed, next, service)).ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, fixtureQuestion, "[phone-ready project brief](/work/phone-ready-project-brief.md)")
	require.Contains(t, fixtureOptions, `"label":"Approve the brief"`)
	require.Equal(t, "continue", fixtureRecommended)
	require.Equal(t, "phone-ready-project-brief.md", fixtureAttachmentName)
	require.Equal(t, "/work/phone-ready-project-brief.md", fixtureAttachmentPath)

	var leasedAfter struct {
		question, allowedAnswers, optionsJSON, recommended, recommendationReason, defaultAnswer string
	}
	require.NoError(t, routed.QueryRowContext(ctx, `SELECT question, allowed_answers, options_json, recommended, recommendation_reason, default_answer FROM asks WHERE id = ?`, hub.TestAskID).Scan(&leasedAfter.question, &leasedAfter.allowedAnswers, &leasedAfter.optionsJSON, &leasedAfter.recommended, &leasedAfter.recommendationReason, &leasedAfter.defaultAnswer))
	var primaryAfter int
	require.NoError(t, routed.Primary().QueryRowContext(ctx, `SELECT count(*) FROM asks WHERE id = ?`, hub.TestAskID).Scan(&primaryAfter))
	require.Equal(t, leasedBefore, leasedAfter)
	var attachmentsAfter int
	require.NoError(t, routed.QueryRowContext(ctx, `SELECT count(*) FROM ask_attachments WHERE ask_id = ?`, hub.TestAskID).Scan(&attachmentsAfter))
	require.Zero(t, attachmentsAfter, "the temporary artifact is removed from the lease after each fixture request")
	require.Equal(t, primaryBefore, primaryAfter)
}

func TestE4DeliveryFixtureStartsFromTheCommonSeedAfterTheHistoryFixture(t *testing.T) {
	ctx := context.Background()
	routed := filedb.Open(t, "primary.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	const leaseID = "e4-delivery-after-history-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), leaseID, 0))
	testCtx, err := routed.PinTestPool(ctx, leaseID)
	require.NoError(t, err)
	countSynthetic := func() int {
		t.Helper()
		var n int
		require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM notifications WHERE id LIKE 'e4-nh162-notification-%'`).Scan(&n))
		return n
	}
	var before int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM notifications`).Scan(&before))

	require.NoError(t, seedE4NH162HistoryFixture(testCtx, routed))
	require.Positive(t, countSynthetic(), "the History fixture adds synthetic records to the leased pool")

	restore, err := applyE4TestFixture(testCtx, routed, "delivery-pending-state")
	require.NoError(t, err)
	restore()
	require.Zero(t, countSynthetic(), "a Delivery fixture must not inherit the History fixture's records")
	var after, asks, answers int
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM notifications`).Scan(&after))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM asks WHERE id LIKE 'e4-test-nh162-ask-%'`).Scan(&asks))
	require.NoError(t, routed.QueryRowContext(testCtx, `SELECT count(*) FROM answers WHERE ask_id LIKE 'e4-test-nh162-ask-%'`).Scan(&answers))
	require.Equal(t, before, after, "the leased pool is back to the common E4 seed's records")
	require.Zero(t, asks)
	require.Zero(t, answers)
}

func TestE4NH162HistoryFixtureIsDeterministicAndLeasePinned(t *testing.T) {
	ctx := context.Background()
	routed := filedb.Open(t, "primary.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	leaseID := "e4-nh162-history-fixture-lease"
	require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), leaseID, 0))
	service := hubtest.New(t, routed, schedule.System(), log.New(io.Discard, "", 0))
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

func TestConfigurePushBudgetReadsCapacityAndRefillRate(t *testing.T) {
	service := pushBudgetSettings{hub.DefaultPushBudget}
	configurePushBudget(service, func(key string) string {
		return map[string]string{
			"VROOLI_NOTIFICATION_PUSH_BUCKET_CAPACITY":        "10",
			"VROOLI_NOTIFICATION_PUSH_BUCKET_REFILL_INTERVAL": "17s",
		}[key]
	})
	require.Equal(t, hub.PushBudgetConfig{Capacity: 10, RefillInterval: 17 * time.Second}, service[0])
}

type pushBudgetSettings []hub.PushBudgetConfig

func (c pushBudgetSettings) SetPushBudget(config hub.PushBudgetConfig) { c[0] = config }

func (c pushBudgetSettings) PushBudget() hub.PushBudgetConfig { return c[0] }

func TestTestPoolSeedDeclaresOnePreferenceSourceWithoutOwnerChoices(t *testing.T) {
	ctx := context.Background()
	routed := filedb.Open(t, "seeded.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	require.NoError(t, seedTestRecipient(ctx, routed.Primary()))

	var sources, categories, choices int
	require.NoError(t, routed.QueryRowContext(ctx, `SELECT count(*) FROM notification_sources WHERE source_scenario = ?`, testPreferenceSource).Scan(&sources))
	require.NoError(t, routed.QueryRowContext(ctx, `SELECT count(*) FROM notification_categories WHERE source_scenario = ?`, testPreferenceSource).Scan(&categories))
	require.NoError(t, routed.QueryRowContext(ctx, `SELECT count(*) FROM notification_preferences WHERE recipient_key = ?`, hub.TestRecipientSubject).Scan(&choices))
	require.Equal(t, 1, sources)
	require.Equal(t, 2, categories)
	require.Zero(t, choices, "the owner starts on every category's default")
}

// bas/actions/seed-owner.json resets devices, quiet hours and the escalation
// order through the Hub's API. It first reads GetRecipient and refuses to go on
// unless the subject is hub.TestRecipientSubject. This pins down why that is
// enough: only a leased test pool answers with that subject, and nothing else
// can reach the operator's live rows.
func TestOwnerResetCallsCanOnlyReachTheLeasedPool(t *testing.T) {
	ctx := context.Background()
	routed := filedb.Open(t, "primary.db")
	defer routed.Close()
	require.NoError(t, database.EnsureSchemas(ctx, routed.Primary(), modules.AllSchemas()...))
	routed.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool, modules.AllSchemas()...); err != nil {
			return err
		}
		return seedTestRecipient(ctx, pool)
	})
	service := hubtest.New(t, routed, schedule.System(), log.New(io.Discard, "", 0))
	const liveOwner = "owner@live.example"
	service.SetDefaultRecipient(liveOwner)
	live, err := service.UpsertDevice(ctx, liveOwner, hub.Device{Name: "Live phone", MachineID: "live"})
	require.NoError(t, err)
	_, err = service.SetQuietWindow(ctx, liveOwner, hub.QuietWindow{Weekday: 1, Start: "22:00", End: "07:00", Timezone: "UTC"})
	require.NoError(t, err)

	router := mux.NewRouter()
	recipientsH.Module(service).Mount(router)
	server := httptest.NewServer(apihttp.TestModeMiddleware(requireRoutedTestLease(routed, router)))
	defer server.Close()
	client := recipientsconnect.NewRecipientsServiceClient(server.Client(), server.URL)
	call := func(testMode bool) *connect.Request[recipientsv1.GetRecipientRequest] {
		req := connect.NewRequest(&recipientsv1.GetRecipientRequest{})
		if testMode {
			req.Header().Set("X-Vrooli-Test-Mode", "1")
		}
		return req
	}
	liveRowsIntact := func(label string) {
		t.Helper()
		devices, err := service.ListDevices(ctx, liveOwner)
		require.NoError(t, err, label)
		require.Len(t, devices, 1, label)
		require.Equal(t, live.ID, devices[0].ID, label)
		windows, err := service.ListQuietWindows(ctx, liveOwner)
		require.NoError(t, err, label)
		require.Len(t, windows, 1, label)
	}

	// Without the test-mode header the caller is the live owner's Hub: the guard
	// sees a different subject and stops, and a write has no identity to run as.
	recipient, err := client.GetRecipient(ctx, call(false))
	require.NoError(t, err)
	require.Equal(t, liveOwner, recipient.Msg.GetSubject())
	require.NotEqual(t, hub.TestRecipientSubject, recipient.Msg.GetSubject())
	_, err = client.RemoveDevice(ctx, connect.NewRequest(&recipientsv1.RemoveDeviceRequest{Id: live.ID}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	liveRowsIntact("no test-mode header")

	// Test mode requested but no lease installed: every call is refused before
	// any handler or storage is reached.
	_, err = client.GetRecipient(ctx, call(true))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	remove := connect.NewRequest(&recipientsv1.RemoveDeviceRequest{Id: live.ID})
	remove.Header().Set("X-Vrooli-Test-Mode", "1")
	_, err = client.RemoveDevice(ctx, remove)
	require.Error(t, err)
	liveRowsIntact("test mode without a lease")

	// With a lease the same calls see only the disposable owner, and a reset
	// there leaves the live rows alone even when it names the live device id.
	require.NoError(t, routed.InstallTestPool(ctx, filedb.DSN(t, "leased.db"), "owner-reset-lease", 0))
	recipient, err = client.GetRecipient(ctx, call(true))
	require.NoError(t, err)
	require.Equal(t, hub.TestRecipientSubject, recipient.Msg.GetSubject())
	upsert := connect.NewRequest(&recipientsv1.UpsertDeviceRequest{Id: "pool-device", Name: "Pool device"})
	upsert.Header().Set("X-Vrooli-Test-Mode", "1")
	_, err = client.UpsertDevice(ctx, upsert)
	require.NoError(t, err)
	list := connect.NewRequest(&recipientsv1.ListDevicesRequest{})
	list.Header().Set("X-Vrooli-Test-Mode", "1")
	devices, err := client.ListDevices(ctx, list)
	require.NoError(t, err)
	require.Len(t, devices.Msg.GetDevices(), 1, "the leased owner sees only its own device")
	require.Equal(t, "pool-device", devices.Msg.GetDevices()[0].GetId())
	for _, id := range []string{"pool-device", live.ID} {
		remove := connect.NewRequest(&recipientsv1.RemoveDeviceRequest{Id: id})
		remove.Header().Set("X-Vrooli-Test-Mode", "1")
		_, _ = client.RemoveDevice(ctx, remove)
	}
	liveRowsIntact("reset inside a lease")
	require.Zero(t, routed.LeaseStats().PrimaryDuringTestModeRequests)
}
