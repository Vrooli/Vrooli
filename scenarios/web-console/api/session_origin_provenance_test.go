package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/sessionstore"
)

// newProvenanceServer wires a fake-PTY server to an in-memory store so both the
// manager's base Save and the adapter's provenance patch land in one place.
func newProvenanceServer(t *testing.T) *Server {
	t.Helper()
	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	srv.sessions.SetStore(srv.sessionStore)
	return srv
}

func createWithProvenance(t *testing.T, srv *Server, req *sessionsv1.CreateRequest) *sessionsv1.Session {
	t.Helper()
	resp, err := newSessionsConnectHandlerForServer(srv).Create(context.Background(), connect.NewRequest(req))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return resp.Msg.GetSession()
}

// TestCreateSession_ProvenanceRoundTrip covers the acceptance case: a
// programmatic create with an owner persists and returns via Create, Get, and
// List with those values.
func TestCreateSession_ProvenanceRoundTrip(t *testing.T) {
	srv := newProvenanceServer(t)

	created := createWithProvenance(t, srv, &sessionsv1.CreateRequest{
		Origin:       sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC,
		Owner:        "agent-manager",
		DisplayLabel: "Nightly build",
	})
	defer func() { _ = srv.sessions.Delete(context.Background(), created.GetId()) }()

	assertProvenance := func(where string, s *sessionsv1.Session) {
		if s.GetOrigin() != sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC {
			t.Errorf("%s: origin = %v, want PROGRAMMATIC", where, s.GetOrigin())
		}
		if s.GetOwner() != "agent-manager" {
			t.Errorf("%s: owner = %q, want agent-manager", where, s.GetOwner())
		}
		if s.GetDisplayLabel() != "Nightly build" {
			t.Errorf("%s: display_label = %q, want Nightly build", where, s.GetDisplayLabel())
		}
	}
	assertProvenance("create", created)

	got, err := callGet(t, srv, created.GetId())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	assertProvenance("get", got)

	list, err := callList(t, srv)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found *sessionsv1.Session
	for _, s := range list {
		if s.GetId() == created.GetId() {
			found = s
		}
	}
	if found == nil {
		t.Fatalf("created session %s missing from list", created.GetId())
	}
	assertProvenance("list", found)
}

// TestCreateSession_UnspecifiedOriginNormalizesToProgrammatic covers the
// normalization rule: an origin-less create can only be programmatic.
func TestCreateSession_UnspecifiedOriginNormalizesToProgrammatic(t *testing.T) {
	srv := newProvenanceServer(t)

	created := createWithProvenance(t, srv, &sessionsv1.CreateRequest{})
	defer func() { _ = srv.sessions.Delete(context.Background(), created.GetId()) }()

	if created.GetOrigin() != sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC {
		t.Errorf("create origin = %v, want PROGRAMMATIC", created.GetOrigin())
	}
	// The stored row must carry the normalized value, not the empty wire zero.
	stored, err := srv.sessionStore.Get(context.Background(), created.GetId())
	if err != nil {
		t.Fatalf("store get: %v", err)
	}
	if stored.Origin != sessionstore.OriginProgrammatic {
		t.Errorf("stored origin = %q, want programmatic", stored.Origin)
	}

	got, err := callGet(t, srv, created.GetId())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.GetOrigin() != sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC {
		t.Errorf("get origin = %v, want PROGRAMMATIC", got.GetOrigin())
	}
}

// TestCreateSession_UIOriginPreserved proves an explicit UI origin is stored and
// returned as UI (never normalized to programmatic).
func TestCreateSession_UIOriginPreserved(t *testing.T) {
	srv := newProvenanceServer(t)

	created := createWithProvenance(t, srv, &sessionsv1.CreateRequest{
		Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_UI,
	})
	defer func() { _ = srv.sessions.Delete(context.Background(), created.GetId()) }()

	if created.GetOrigin() != sessionsv1.SessionOrigin_SESSION_ORIGIN_UI {
		t.Errorf("origin = %v, want UI", created.GetOrigin())
	}
}

// TestCreateSession_EmitsProvenanceEvent proves the session.created event
// carries origin/owner/label for downstream (live sidebar) consumers.
func TestCreateSession_EmitsProvenanceEvent(t *testing.T) {
	srv := newProvenanceServer(t)

	created := createWithProvenance(t, srv, &sessionsv1.CreateRequest{
		Origin:       sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC,
		Owner:        "agent-manager",
		DisplayLabel: "Nightly build",
	})
	defer func() { _ = srv.sessions.Delete(context.Background(), created.GetId()) }()

	var details map[string]string
	for _, e := range srv.events.Recent(0) {
		if e.Type == "session.created" && e.SessionID == created.GetId() {
			details = e.Details
		}
	}
	if details == nil {
		t.Fatalf("no session.created event for %s", created.GetId())
	}
	if details["origin"] != "programmatic" || details["owner"] != "agent-manager" || details["label"] != "Nightly build" {
		t.Errorf("event provenance = %q/%q/%q, want programmatic/agent-manager/Nightly build",
			details["origin"], details["owner"], details["label"])
	}
}
