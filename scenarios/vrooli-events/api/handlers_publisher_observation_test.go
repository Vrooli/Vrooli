package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/eventpublisher"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/api/internal/publisherauth"
)

type publisherVerifier map[string]apiidentity.Principal

func (v publisherVerifier) Verify(_ context.Context, token string) (apiidentity.Principal, error) {
	if principal, ok := v[token]; ok {
		return principal, nil
	}
	return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureInvalid, apiidentity.SourceScenarioAuthenticator)
}

func publisherPrincipal(source string) apiidentity.Principal {
	return apiidentity.Principal{
		Kind: apiidentity.ActorService, Subject: eventpublisher.Principal(source), Verified: true,
		Audience: eventpublisher.Audience, Scopes: []string{eventpublisher.Scope},
	}
}

func ingestAs(t *testing.T, ts *httptest.Server, authorization, eventID, source, eventType string) *http.Response {
	t.Helper()
	body := `{"eventId":"` + eventID + `","eventType":"` + eventType + `","source":{"scenario":"` + source + `"}}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/events", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// Observation is the first step of requiring a publisher to prove its source: it
// counts how each accepted event fared and changes nothing about whether it is
// accepted, whatever the event presents.
func TestIngestAcceptsEveryEventWhateverItPresentsAndCountsHowItFared(t *testing.T) {
	srv, _ := newTestServer(t)
	if _, err := srv.db.ExecContext(context.Background(), publisherauth.Schema()); err != nil {
		t.Fatal(err)
	}
	srv.publishers = publisherauth.New(publisherVerifier{
		"storage": publisherPrincipal("storage-manager"), "agent": publisherPrincipal("agent-manager"),
	}, srv.db)
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)

	const resolved = "notification_hub.ask.resolved.v1"
	for i, c := range []struct{ authorization, source, eventType string }{
		{"Bearer storage", "storage-manager", "storage-manager.deletion-grant.decision_requested.v1"}, // authenticated
		{"Bearer storage", "notification-hub", resolved},                                              // mismatch: a forged answer
		{"", "notification-hub", resolved},                                                            // unauthenticated
		{"Bearer forged", "storage-manager", "storage.recovery.started"},                              // invalid
	} {
		resp := ingestAs(t, ts, c.authorization, "evt-"+string(rune('a'+i)), c.source, c.eventType)
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("case %d: status = %d, want 202 whatever the proof", i, resp.StatusCode)
		}
	}

	resp, err := http.Get(ts.URL + "/api/v1/publisher-authentication")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	report := decodeJSON[publisherauth.Report](t, resp)
	if report.Mode != "observe" || report.Totals[publisherauth.Authenticated] != 1 || report.Totals[publisherauth.Mismatch] != 1 ||
		report.Totals[publisherauth.Unauthenticated] != 1 || report.Totals[publisherauth.Invalid] != 1 {
		t.Fatalf("totals = %+v", report.Totals)
	}
	first := report.Rows[0]
	if first.Family != publisherauth.Answer || first.Class != publisherauth.Mismatch || first.ClaimedSource != "notification-hub" || first.AuthenticatedSource != "storage-manager" {
		t.Fatalf("the forged answer must lead the report: %+v", first)
	}
}

func TestADuplicateOrRejectedEventIsNotCounted(t *testing.T) {
	srv, _ := newTestServer(t)
	if _, err := srv.db.ExecContext(context.Background(), publisherauth.Schema()); err != nil {
		t.Fatal(err)
	}
	srv.publishers = publisherauth.New(publisherVerifier{}, srv.db)
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)

	if resp := ingestAs(t, ts, "", "evt-1", "storage-manager", "storage.recovery.started"); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("first: %d", resp.StatusCode)
	}
	if resp := ingestAs(t, ts, "", "evt-1", "storage-manager", "storage.recovery.started"); resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate: %d", resp.StatusCode)
	}
	if resp := ingestAs(t, ts, "", "", "storage-manager", "storage.recovery.started"); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid: %d", resp.StatusCode)
	}
	report, err := srv.publishers.Report(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Totals[publisherauth.Unauthenticated] != 1 {
		t.Fatalf("totals = %+v, want exactly the one accepted event", report.Totals)
	}
}

func TestWithoutAnObserverIngestIsUnchangedAndTheReportIsUnavailable(t *testing.T) {
	_, ts := newTestServer(t)
	if resp := ingestAs(t, ts, "Bearer anything", "evt-1", "storage-manager", "storage.recovery.started"); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("ingest: %d", resp.StatusCode)
	}
	resp, err := http.Get(ts.URL + "/api/v1/publisher-authentication")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("report: %d, want 503", resp.StatusCode)
	}
}
