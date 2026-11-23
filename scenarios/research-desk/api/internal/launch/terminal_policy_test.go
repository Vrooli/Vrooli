package launch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"research-desk/internal/peer"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
)

// sessionsRecorder is Web Console's SessionsService as far as Create goes.
type sessionsRecorder struct {
	sessionsconnect.UnimplementedSessionsServiceHandler
	got *sessionsv1.CreateRequest
}

func (r *sessionsRecorder) Create(_ context.Context, req *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error) {
	r.got = req.Msg
	return connect.NewResponse(&sessionsv1.CreateResponse{Session: &sessionsv1.Session{Id: "sess-1"}}), nil
}

func webConsoleRecorder(t *testing.T) (*WebConsole, *sessionsRecorder) {
	t.Helper()
	rec := &sessionsRecorder{}
	mux := http.NewServeMux()
	mux.Handle(sessionsconnect.NewSessionsServiceHandler(rec))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &WebConsole{Peer: &peer.Peer{Name: "web-console", BaseURL: func(context.Context) (string, error) { return srv.URL, nil }}}, rec
}

// A launched supervisor session carries Web Console's idle policy, so a lead
// whose work is done stops holding gigabytes once it has sat idle.
func TestCreateSendsTheIdlePolicy(t *testing.T) {
	wc, rec := webConsoleRecorder(t)
	if _, err := wc.Create(context.Background(), SessionSpec{IdempotencyKey: "c#1", Label: "research · c", Command: "codex", IdleArchiveAfter: "6h"}); err != nil {
		t.Fatal(err)
	}
	if !rec.got.GetHasPolicy() || rec.got.GetPolicy().GetMode() != "idle" || rec.got.GetPolicy().GetDuration() != "6h" {
		t.Fatalf("create policy = has:%v %+v, want idle 6h", rec.got.GetHasPolicy(), rec.got.GetPolicy())
	}
}

func TestCreateWithoutIdleWindowLeavesWebConsoleDefault(t *testing.T) {
	wc, rec := webConsoleRecorder(t)
	if _, err := wc.Create(context.Background(), SessionSpec{IdempotencyKey: "t#1", Label: "launch test", Command: "codex"}); err != nil {
		t.Fatal(err)
	}
	if rec.got.GetHasPolicy() || rec.got.GetPolicy() != nil {
		t.Fatalf("create policy = has:%v %+v, want none", rec.got.GetHasPolicy(), rec.got.GetPolicy())
	}
}

func TestIdleArchiveWindowFromLimits(t *testing.T) {
	for value, want := range map[string]string{"": DefaultIdleArchiveAfter, "6h": "6h", " 12h ": "12h", "15m": "15m"} {
		got, err := idleArchiveWindow(value)
		if err != nil || got != want {
			t.Errorf("idleArchiveWindow(%q) = %q, %v; want %q", value, got, err, want)
		}
	}
	for _, bad := range []string{"soon", "5m", "200h", "-1h"} {
		if _, err := idleArchiveWindow(bad); err == nil {
			t.Errorf("idleArchiveWindow(%q) accepted a window Web Console refuses", bad)
		}
	}
}
