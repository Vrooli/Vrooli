package testfixtures

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	"personal-planner/internal/modules"

	_ "modernc.org/sqlite"
)

func TestEnabledUsesOnlyExistingDevelopmentOrTestOverride(t *testing.T) {
	if Enabled(false, false) {
		t.Fatal("production without the explicit existing test override must not mount the fixture route")
	}
	if !Enabled(true, false) || !Enabled(false, true) {
		t.Fatal("development and the existing CI test override must retain the dev-only route")
	}
}

func TestReferenceScaleRejectsUnroutedRequestsAndExpiredLeases(t *testing.T) {
	ctx := context.Background()
	routed, primary := newRoutedDB(t, "none")
	handler := ReferenceScaleHandler(routed)
	body := `{"local_date":"2026-10-18"}`

	plain := httptest.NewRequest(http.MethodPost, ReferenceScaleV1Path, strings.NewReader(body))
	plain.Header.Set(leaseHeader, "not-a-lease")
	plainRecorder := httptest.NewRecorder()
	handler.ServeHTTP(plainRecorder, plain)
	if plainRecorder.Code != http.StatusForbidden {
		t.Fatalf("plain request status=%d body=%s, want forbidden", plainRecorder.Code, plainRecorder.Body.String())
	}

	missing := httptest.NewRequest(http.MethodPost, ReferenceScaleV1Path, strings.NewReader(body)).WithContext(database.WithTestMode(ctx))
	missing.Header.Set(leaseHeader, "missing")
	missingRecorder := httptest.NewRecorder()
	handler.ServeHTTP(missingRecorder, missing)
	if missingRecorder.Code != http.StatusConflict {
		t.Fatalf("missing lease status=%d body=%s, want conflict", missingRecorder.Code, missingRecorder.Body.String())
	}
	assertNoFixtureRows(t, primary)

	expiredDB, expiredPrimary := newRoutedDB(t, "expired")
	expiredHandler := ReferenceScaleHandler(expiredDB)
	expired := httptest.NewRequest(http.MethodPost, ReferenceScaleV1Path, strings.NewReader(body)).WithContext(database.WithTestMode(ctx))
	expired.Header.Set(leaseHeader, "expired-lease")
	expiredRecorder := httptest.NewRecorder()
	expiredHandler.ServeHTTP(expiredRecorder, expired)
	if expiredRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expired lease status=%d body=%s, want service unavailable", expiredRecorder.Code, expiredRecorder.Body.String())
	}
	assertNoFixtureRows(t, expiredPrimary)
	if stats := expiredDB.LeaseStats(); stats.PrimaryDuringTestModeRequests != 0 {
		t.Fatalf("expired fixture request touched primary pool: %+v", stats)
	}
}

func TestReferenceScaleSeedsFixedDataDeterministicallyInsideLease(t *testing.T) {
	routed, primary := newRoutedDB(t, "live")
	handler := ReferenceScaleHandler(routed)
	wrongLease := httptest.NewRequest(http.MethodPost, ReferenceScaleV1Path, strings.NewReader(`{"local_date":"2026-10-18"}`)).WithContext(database.WithTestMode(context.Background()))
	wrongLease.Header.Set(leaseHeader, "different-lease")
	wrongRecorder := httptest.NewRecorder()
	handler.ServeHTTP(wrongRecorder, wrongLease)
	if wrongRecorder.Code != http.StatusConflict {
		t.Fatalf("wrong lease status=%d, want conflict", wrongRecorder.Code)
	}
	if _, err := routed.ExecContext(database.WithTestMode(context.Background()), `INSERT INTO work_items (id,title,description,remaining_minutes,original_estimate_minutes,status,completed_at,snoozed_until,source_label,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, "user-owned-work-keep", "User-owned task", "Must survive reference fixture setup", 30, 30, "open", "", "", "user", "2026-10-18T12:00:00Z", "2026-10-18T12:00:00Z"); err != nil {
		t.Fatal(err)
	}
	beforeFirst := routed.LeaseStats()
	first := postFixture(t, handler, "fixture-lease", `{"local_date":"2026-10-18"}`)
	afterFirst := routed.LeaseStats()
	if afterFirst.TestPoolRequests-beforeFirst.TestPoolRequests != 1 || afterFirst.PrimaryDuringTestModeRequests != beforeFirst.PrimaryDuringTestModeRequests {
		t.Fatalf("first fixture transaction did not route exactly once through the leased test pool: before=%+v after=%+v", beforeFirst, afterFirst)
	}
	second := postFixture(t, handler, "fixture-lease", `{"local_date":"2026-10-18"}`)
	if first.Fixture != ReferenceScaleV1ID || first.LeaseID != "fixture-lease" {
		t.Fatalf("fixture identity = %+v", first)
	}
	want := referenceScaleCounts{WorkItems: 1000, CalendarEvents: 10000, Allocations: 10000, ManualActuals: 10000, Goals: 1}
	if first.Counts != want || second.Counts != want {
		t.Fatalf("fixture counts first=%+v second=%+v want=%+v", first.Counts, second.Counts, want)
	}
	if first.AuthoredFingerprint == "" || first.AuthoredFingerprint != second.AuthoredFingerprint {
		t.Fatalf("reseed fingerprint first=%q second=%q", first.AuthoredFingerprint, second.AuthoredFingerprint)
	}
	if got := countRows(t, routed, "work_items", "user-owned-work-keep"); got != 1 {
		t.Fatalf("unrelated user-owned sentinel count=%d, want 1", got)
	}
	if got := countRows(t, routed, "work_items", "p1-reference-scale-v1-work-%"); got != 1000 {
		t.Fatalf("test pool work items=%d, want 1000 (reseed must replace, not append)", got)
	}
	if got := countRows(t, routed, "calendar_events", "p1-reference-scale-v1-event-%"); got != 10000 {
		t.Fatalf("test pool events=%d, want 10000", got)
	}
	if got := countRows(t, routed, "calendar_allocations", "p1-reference-scale-v1-allocation-%"); got != 10000 {
		t.Fatalf("test pool allocations=%d, want 10000", got)
	}
	if got := countRows(t, routed, "manual_actuals", "p1-reference-scale-v1-actual-%"); got != 10000 {
		t.Fatalf("test pool actuals=%d, want 10000", got)
	}
	assertNoFixtureRows(t, primary)
	if stats := routed.LeaseStats(); stats.TestPoolRequests == 0 || stats.PrimaryDuringTestModeRequests != 0 {
		t.Fatalf("fixture routing stats=%+v", stats)
	}
}

func postFixture(t *testing.T, handler http.Handler, leaseID, body string) referenceScaleResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, ReferenceScaleV1Path, strings.NewReader(body)).WithContext(database.WithTestMode(context.Background()))
	req.Header.Set(leaseHeader, leaseID)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("fixture response status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response referenceScaleResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode fixture response: %v", err)
	}
	return response
}

func newRoutedDB(t *testing.T, leaseMode string) (*database.RoutedDB, *sql.DB) {
	t.Helper()
	primaryPath := filepath.Join(t.TempDir(), "primary.db")
	primary, err := sql.Open("sqlite", primaryPath)
	if err != nil {
		t.Fatal(err)
	}
	primary.SetMaxOpenConns(1)
	if err := database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...); err != nil {
		t.Fatal(err)
	}
	clock := schedule.NewFake(time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC))
	routed, err := database.OpenWithClock(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		TestDriver:   database.DriverSQLite,
		DSN:          primaryPath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}, clock)
	if err != nil {
		t.Fatal(err)
	}
	routed.SetTestPoolInitializer(func(ctx context.Context, testDB *sql.DB) error {
		return database.EnsureSchemas(ctx, testDB, modules.AllSchemas()...)
	})
	if leaseMode == "live" || leaseMode == "expired" {
		ttl := time.Hour
		leaseID := "fixture-lease"
		if leaseMode == "expired" {
			leaseID = "expired-lease"
		}
		if err := routed.InstallTestPool(context.Background(), filepath.Join(t.TempDir(), "test.db"), leaseID, ttl); err != nil {
			t.Fatal(err)
		}
		if leaseMode == "expired" {
			clock.Advance(2 * time.Hour)
		}
	}
	t.Cleanup(func() { _ = routed.Close() })
	return routed, primary
}

func countRows(t *testing.T, db *database.RoutedDB, table, pattern string) int {
	t.Helper()
	query := `SELECT count(*) FROM ` + table + ` WHERE id LIKE ?`
	var count int
	if err := db.QueryRowContext(database.WithTestMode(context.Background()), query, pattern).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func assertNoFixtureRows(t *testing.T, primary *sql.DB) {
	t.Helper()
	for _, table := range []string{"work_items", "calendar_events", "calendar_allocations", "manual_actuals", "goals"} {
		var count int
		if err := primary.QueryRow(`SELECT count(*) FROM ` + table + ` WHERE id LIKE 'p1-reference-scale-v1-%'`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("primary %s contains %d fixture rows", table, count)
		}
	}
}
