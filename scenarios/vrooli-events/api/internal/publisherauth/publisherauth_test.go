package publisherauth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventpublisher"
	apiidentity "github.com/vrooli/api-core/identity"
	_ "modernc.org/sqlite"
)

type verifierFunc func(token string) (apiidentity.Principal, error)

func (f verifierFunc) Verify(_ context.Context, token string) (apiidentity.Principal, error) {
	return f(token)
}

func publisher(name string) apiidentity.Principal {
	return apiidentity.Principal{
		Kind: apiidentity.ActorService, Subject: name, Verified: true,
		Audience: eventpublisher.Audience, Scopes: []string{eventpublisher.Scope},
	}
}

func bearer(token string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+token)
	return h
}

func testVerifier() verifierFunc {
	return func(token string) (apiidentity.Principal, error) {
		switch token {
		case "storage":
			return publisher("storage-manager-event-publisher"), nil
		case "agent":
			return publisher("agent-manager-event-publisher"), nil
		case "not-a-publisher":
			return publisher("storage-manager"), nil
		case "down":
			return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureUnavailable, apiidentity.SourceScenarioAuthenticator)
		}
		return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureInvalid, apiidentity.SourceScenarioAuthenticator)
	}
}

func newObserver(t *testing.T) *Observer {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	return New(testVerifier(), db)
}

func TestEveryClassIsTellableFromTheProofAnEventCameWith(t *testing.T) {
	o := newObserver(t)
	ctx := context.Background()
	cases := []struct {
		name          string
		headers       http.Header
		claimed       string
		class         Class
		authenticated string
	}{
		{"own source", bearer("storage"), "storage-manager", Authenticated, "storage-manager"},
		{"another source", bearer("storage"), "agent-manager", Mismatch, "storage-manager"},
		{"no token", http.Header{}, "storage-manager", Unauthenticated, ""},
		{"blank authorization", http.Header{"Authorization": {"  "}}, "storage-manager", Unauthenticated, ""},
		{"unknown token", bearer("forged"), "storage-manager", Invalid, ""},
		{"verified but not a publisher", bearer("not-a-publisher"), "storage-manager", Invalid, ""},
		{"not a bearer", http.Header{"Authorization": {"Basic abc"}}, "storage-manager", Invalid, ""},
		{"authenticator unreachable", bearer("down"), "storage-manager", Unverifiable, ""},
	}
	for _, c := range cases {
		class, authenticated := o.Classify(ctx, c.headers, c.claimed)
		if class != c.class || authenticated != c.authenticated {
			t.Fatalf("%s: Classify = %s/%q, want %s/%q", c.name, class, authenticated, c.class, c.authenticated)
		}
	}
	if class, _ := New(nil, nil).Classify(ctx, bearer("storage"), "storage-manager"); class != Unverifiable {
		t.Fatalf("no verifier: %s, want unverifiable", class)
	}
}

func TestEventTypesAreGroupedByWhatForgingThemWouldCost(t *testing.T) {
	for eventType, want := range map[string]Family{
		"notification_hub.ask.resolved.v1":                     Answer,
		"notification_hub.ask.context_requested.v1":            Answer,
		"agent_manager.run.decision_requested.v1":              Decision,
		"storage-manager.deletion-grant.decision_withdrawn.v1": Decision,
		"agent_manager.run.notification.v1":                    Notification,
		"incident.opened.v1":                                   Notification,
		"prompt-manager.heartbeat.attempt.v1":                  Notification,
		"storage.recovery.started":                             Other,
		"vrooli.program_runtime.program_event.v1":              Other,
		"some.decision_requested.v1.but.not.a.suffix":          Other,
		"notification_hub.something.else.v1":                   Other,
	} {
		if got := FamilyOf(eventType); got != want {
			t.Fatalf("FamilyOf(%q) = %s, want %s", eventType, got, want)
		}
	}
}

func TestObservedEventsAreCountedAndSurviveAFlush(t *testing.T) {
	o := newObserver(t)
	clock := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	o.now = func() time.Time { return clock }
	ctx := context.Background()
	o.Observe(ctx, http.Header{}, "storage-manager", "storage-manager.deletion-grant.decision_requested.v1", "e1")
	o.Observe(ctx, http.Header{}, "storage-manager", "storage-manager.deletion-grant.decision_requested.v1", "e2")
	clock = clock.Add(time.Minute)
	o.Observe(ctx, bearer("storage"), "storage-manager", "storage-manager.deletion-grant.decision_requested.v1", "e3")
	o.Observe(ctx, bearer("storage"), "agent-manager", "notification_hub.ask.resolved.v1", "e4")
	if err := o.Flush(ctx); err != nil {
		t.Fatal(err)
	}
	o.Observe(ctx, http.Header{}, "storage-manager", "storage-manager.deletion-grant.decision_requested.v1", "e5")

	report, err := o.Report(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Mode != "observe" || report.Totals[Unauthenticated] != 3 || report.Totals[Authenticated] != 1 || report.Totals[Mismatch] != 1 {
		t.Fatalf("totals = %+v", report.Totals)
	}
	if report.ByFamily[0].Family != Answer || report.ByFamily[0].Totals[Mismatch] != 1 || report.ByFamily[1].Family != Decision || report.ByFamily[1].Totals[Unauthenticated] != 3 {
		t.Fatalf("by family = %+v", report.ByFamily)
	}
	// High-consequence rows first; a mismatch names both sources and a sample event.
	first := report.Rows[0]
	if first.Family != Answer || first.Class != Mismatch || first.ClaimedSource != "agent-manager" || first.AuthenticatedSource != "storage-manager" || first.SampleEventID != "e4" {
		t.Fatalf("first row = %+v", first)
	}
	var unauthenticated Row
	for _, r := range report.Rows {
		if r.Class == Unauthenticated {
			unauthenticated = r
		}
	}
	if unauthenticated.Count != 3 || unauthenticated.SampleEventID != "e5" || unauthenticated.FirstSeen == unauthenticated.LastSeen {
		t.Fatalf("unauthenticated row = %+v", unauthenticated)
	}
	if report.Since == "" {
		t.Fatal("since must say when counting started")
	}
}

// A database that cannot be written keeps its counts for the next flush.
func TestAFailedFlushKeepsTheCountsForTheNextOne(t *testing.T) {
	o := newObserver(t)
	ctx := context.Background()
	o.Observe(ctx, http.Header{}, "storage-manager", "x.y.notification.v1", "e1")
	good := o.db
	o.db = failingDB{}
	if err := o.Flush(ctx); err == nil {
		t.Fatal("want the write error")
	}
	o.db = good
	report, err := o.Report(ctx)
	if err != nil || report.Totals[Unauthenticated] != 1 {
		t.Fatalf("report = %+v, err = %v", report.Totals, err)
	}
}

type failingDB struct{}

func (failingDB) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, errors.New("disk full")
}

func (failingDB) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("disk full")
}
