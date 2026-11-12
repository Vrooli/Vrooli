package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/courier"
	"research-desk/internal/home"
	"research-desk/internal/launch"
	"research-desk/internal/peer"
	"research-desk/internal/testutil"
	"research-desk/internal/writer"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/activity"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"
)

// [REQ:RD-P0-010]
// Only the lifecycle-managed live instance writes. The environment is the
// control plane's (a live research-desk process, 2026-11-12); a boot check
// sets VROOLI_LIFECYCLE_MANAGED but no instance identity, and a shadow variant
// has its own namespace.
func TestOnlyTheLiveLifecycleInstanceWrites(t *testing.T) {
	live := map[string]string{
		"VROOLI_LIFECYCLE_MANAGED":   "true",
		"VROOLI_RUNTIME_INSTANCE_ID": "inst-1241f7a02260bf3c6f2a3128b3a9bc20",
		"VROOLI_STORAGE_NAMESPACE":   "research-desk",
		"VROOLI_VARIANT":             "live",
	}
	with := func(key, value string) func(string) string {
		return func(k string) string {
			if k == key {
				return value
			}
			return live[k]
		}
	}
	if err := liveInstance(with("", "")); err != nil {
		t.Fatalf("the live instance was refused: %v", err)
	}
	for name, env := range map[string]func(string) string{
		"go test or go run": with("VROOLI_LIFECYCLE_MANAGED", ""),
		"boot check":        with("VROOLI_RUNTIME_INSTANCE_ID", ""),
		"shadow variant":    with("VROOLI_STORAGE_NAMESPACE", "research-desk_shadow"),
	} {
		if liveInstance(env) == nil {
			t.Errorf("%s: treated as the live instance", name)
		}
	}

	root := t.TempDir()
	w := writer.New(root, root+"/state", testutil.EntryChars)
	w.ReadOnly("boot check")
	if err := w.Held(); !errors.Is(err, writer.ErrReadOnly) {
		t.Fatalf("a read-only instance took the lock: %v", err)
	}
	if err := w.Role(); !errors.Is(err, writer.ErrReadOnly) {
		t.Fatalf("role = %v", err)
	}
}

// The overview shows how the desk reaches Web Console and that a
// refusal turns automatic launch off; a read-only instance says it runs no
// courier.
func TestRoutesShowTheWebConsoleAccessAndARefusal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"permission_denied","message":"missing web-console:write"}`))
	}))
	defer server.Close()
	root := t.TempDir()
	w := &wiring{
		webConsole: &peer.Peer{Name: "web-console", BaseURL: func(context.Context) (string, error) { return server.URL, nil }, Credential: webConsoleCredential},
		writer:     writer.New(root, root+"/state", testutil.EntryChars),
	}
	w.writer.ReadOnly("boot check")
	ctx := context.Background()
	if a := w.webConsoleAccess(ctx); !a.Automatic || a.Said != "Web Console access: "+peer.AccessAvailability || a.Kind != activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_AVAILABILITY_POLICY {
		t.Fatalf("before any refusal: %+v", a)
	}
	client, base, opt, err := w.webConsole.Dial(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = sessionsconnect.NewSessionsServiceClient(client, base, opt).List(ctx, connect.NewRequest(&sessionsv1.ListRequest{}))
	if a := w.webConsoleAccess(ctx); a.Automatic || a.Kind != activityv1.WebConsoleAccess_WEB_CONSOLE_ACCESS_AVAILABILITY_POLICY || !strings.Contains(a.Said, "manual launch is the whole flow") || !strings.Contains(a.Said, "missing web-console:write") {
		t.Fatalf("a refusal turns automatic launch off and says so: %+v", a)
	}
	routes := map[string]home.Lens{}
	for _, l := range w.routes(ctx) {
		routes[l.Name] = l
	}
	if c := routes[courierRoute]; c.Available || !strings.Contains(c.Reason, "only reads") {
		t.Fatalf("a read-only instance runs no courier: %+v", c)
	}
	if r := routes["capability registry"]; r.Available || !strings.Contains(r.Reason, "no optional scenario is reached") || strings.Contains(r.Reason, "VROOLI_ROOT") || !strings.Contains(r.Command, "VROOLI_ROOT") {
		t.Fatalf("without service.json the overview says nothing is reached: %+v", routes)
	}
}

// The courier runs on a context the server's shutdown cancels; stop
// ends a pass in progress and releases the write lock for the next instance.
func TestShutdownStopsTheCourierAndReleasesTheLock(t *testing.T) {
	h := testutil.NewHome(t)
	clock := schedule.NewFake(testutil.Base)
	reader := home.NewReader(home.Config{ResearchDir: h.Research, CodexDir: h.Codex}, home.Deps{Clock: clock, Procs: home.FixedProcs{}})
	w := &wiring{writer: writer.New(h.Research, h.State, testutil.EntryChars)}
	quiet := log.New(io.Discard, "", 0)
	hub := &blockingHub{entered: make(chan struct{}, 1)}
	w.courier = &courier.Courier{Reader: reader, Writer: w.writer, Hub: hub, Clock: clock, Logger: quiet}
	w.launch = &launch.Service{Reader: reader, Writer: w.writer, Clock: clock, Logger: quiet}

	if err := w.writer.Held(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.courier.Set(context.Background(), "", true); err != nil {
		t.Fatal(err)
	}
	stop := w.start()
	select {
	case <-hub.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the courier never reached the Hub")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	next := writer.New(h.Research, h.State, testutil.EntryChars)
	defer next.Close()
	if err := next.Held(); err != nil {
		t.Fatalf("the next instance cannot take the lock: %v", err)
	}
}

// blockingHub holds every call until its context ends, as a Hub that stopped
// answering would.
type blockingHub struct{ entered chan struct{} }

func (b *blockingHub) wait(ctx context.Context) error {
	select {
	case b.entered <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return ctx.Err()
}
func (b *blockingHub) Sendable(ctx context.Context) error                     { return b.wait(ctx) }
func (b *blockingHub) Ask(ctx context.Context, _ courier.AskSpec) error       { return b.wait(ctx) }
func (b *blockingHub) Notify(ctx context.Context, _ courier.NoticeSpec) error { return b.wait(ctx) }
func (b *blockingHub) Asks(ctx context.Context, _ string) ([]courier.Ask, error) {
	return nil, b.wait(ctx)
}

func (b *blockingHub) Notices(ctx context.Context) (map[string]courier.Notice, error) {
	return nil, b.wait(ctx)
}

// One version, service.version in .vrooli/service.json.
func TestTheVersionIsServiceJSONs(t *testing.T) {
	if v := serviceVersion("../.vrooli/service.json"); v == "unknown" || v == "" || v == "1.0.0" {
		t.Fatalf("version = %q", v)
	}
	if v := serviceVersion(t.TempDir() + "/service.json"); v != "unknown" {
		t.Fatalf("an unreadable service.json = %q", v)
	}
}
