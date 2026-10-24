package sessions

import (
	"context"
	"strings"
	"testing"
	"time"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/session"
)

func TestSessionCreateGetListPreserveUTCCreationTime(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	mgr := session.NewManagerWithFactory(ptyfake.NewFactory())
	mgr.SetStore(store)
	adapter := &Adapter{Manager: mgr, Store: store, Events: events.NewLogger(20), Metrics: metrics.New()}
	handler := NewConnectHandler(Deps{Service: adapter})

	created, err := handler.Create(ctx, request(&sessionsv1.CreateRequest{
		Shell: "/bin/sh", Cols: 80, Rows: 24, Backend: "standard",
		Origin: sessionsv1.SessionOrigin_SESSION_ORIGIN_UI,
	}))
	if err != nil {
		t.Fatal(err)
	}
	id := created.Msg.Session.Id
	t.Cleanup(func() { _ = mgr.Delete(ctx, id) })
	live, ok := mgr.Get(id)
	if !ok {
		t.Fatal("created session is missing from the real manager")
	}
	parsed, err := time.Parse(time.RFC3339Nano, created.Msg.Session.CreatedAt)
	if err != nil || !parsed.Equal(live.CreatedAt) || !strings.HasSuffix(created.Msg.Session.CreatedAt, "Z") {
		t.Fatalf("create time = %q, parsed=%v err=%v, want UTC creation instant %v", created.Msg.Session.CreatedAt, parsed, err, live.CreatedAt)
	}

	// Control the live session timestamp for read projection; creation above
	// uses the manager's actual clock. The PTY is outside this wire-time claim.
	live.CreatedAt = time.Date(2026, time.August, 25, 23, 34, 56, 7, time.FixedZone("UTC-0700", -7*60*60))
	const want = "2026-08-26T06:34:56.000000007Z"
	got, err := handler.Get(ctx, request(&sessionsv1.GetRequest{Id: id}))
	if err != nil || got.Msg.Session.CreatedAt != want {
		t.Fatalf("get = %+v err=%v, want created_at %q", got, err, want)
	}
	listed, err := handler.List(ctx, request(&sessionsv1.ListRequest{}))
	if err != nil || len(listed.Msg.Sessions) != 1 || listed.Msg.Sessions[0].CreatedAt != want {
		t.Fatalf("list = %+v err=%v, want one session with created_at %q", listed, err, want)
	}
}
