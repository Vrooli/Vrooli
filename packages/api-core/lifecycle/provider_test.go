package lifecycle

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// hangingProvider returns a provider whose application handler blocks until the
// returned release function runs, plus a client for its lifecycle service.
func hangingProvider(t *testing.T, hooks Hooks) (*Provider, *Client, *httptest.Server, func()) {
	t.Helper()
	p, err := NewProvider("fixture", "instance", hooks)
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}, 8), make(chan struct{})
	server := httptest.NewServer(p.Mount(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		<-release
		w.WriteHeader(http.StatusNoContent)
	})))
	t.Cleanup(server.Close)
	client, err := NewClient(server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	stop := func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}
	t.Cleanup(stop)
	// Start one hung request and wait until its handler is running.
	go func() {
		resp, err := server.Client().Post(server.URL+"/api/v1/slow", "text/plain", strings.NewReader(""))
		if err == nil {
			resp.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("hung request never reached its handler")
	}
	return p, client, server, stop
}

// Admitted requests and detached owner work are separate counters: a control
// plane that accepts the risk of interrupting requests must be able to tell the
// two apart, and the standing must say which requests are still running.
func TestStandingSeparatesAdmittedRequestsFromOwnerWorkAndNamesThem(t *testing.T) {
	_, client, _, _ := hangingProvider(t, Hooks{})
	ctx := context.Background()
	prepared, err := client.Prepare(ctx, &commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "restart"})
	if err != nil {
		t.Fatal(err)
	}
	standing := prepared.GetStanding()
	if standing.GetAdmitting() != 1 || standing.GetRemaining() != 0 || !standing.GetInventoryComplete() || standing.GetDrained() {
		t.Fatalf("standing = admitting %d remaining %d complete %t drained %t, want 1/0/true/false", standing.GetAdmitting(), standing.GetRemaining(), standing.GetInventoryComplete(), standing.GetDrained())
	}
	if len(standing.GetWork()) != 1 {
		t.Fatalf("work = %v, want the one hung request", standing.GetWork())
	}
	item := standing.GetWork()[0]
	if item.GetKind() != "http_request" || item.GetId() != "POST /api/v1/slow" || !strings.Contains(item.GetStatus(), "since") {
		t.Fatalf("work item = %v, want the hung POST /api/v1/slow request with its start time", item)
	}
}

func TestStandingCountsOwnerWorkWithoutAdmittedRequests(t *testing.T) {
	p, err := NewProvider("fixture", "instance", Hooks{Snapshot: func(context.Context) (Inventory, error) {
		return Inventory{Complete: true, Remaining: 3, Work: []*commonv1.LifecycleWorkItem{{Id: "job-1", Kind: "job"}}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "restart"}))
	if err != nil {
		t.Fatal(err)
	}
	standing := resp.Msg.GetStanding()
	if standing.GetAdmitting() != 0 || standing.GetRemaining() != 3 || len(standing.GetWork()) != 1 {
		t.Fatalf("standing = admitting %d remaining %d work %v, want 0/3/[job-1]", standing.GetAdmitting(), standing.GetRemaining(), standing.GetWork())
	}
}

func TestClosingSignalTracksAdmissionGenerations(t *testing.T) {
	p, err := NewProvider("fixture", "instance", Hooks{})
	if err != nil {
		t.Fatal(err)
	}
	var signal <-chan struct{}
	handler := p.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		signal = Closing(r.Context())
	}))
	request := func() int {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/work", nil))
		return w.Code
	}
	assertOpen := func(ch <-chan struct{}) {
		t.Helper()
		if ch == nil {
			t.Fatal("missing closing signal")
		}
		select {
		case <-ch:
			t.Fatal("closing signal closed early")
		default:
		}
	}
	request()
	first := signal
	assertOpen(first)
	prepared, err := p.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "restart"}))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-first:
	default:
		t.Fatal("prepare did not close admitted signal")
	}
	if got := request(); got != http.StatusServiceUnavailable {
		t.Fatalf("closed admission status = %d", got)
	}
	fence := prepared.Msg.GetStanding()
	_, err = p.Resume(context.Background(), connect.NewRequest(&commonv1.LifecycleResumeRequest{OperationId: "op", FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision()}))
	if err != nil {
		t.Fatal(err)
	}
	request()
	second := signal
	assertOpen(second)
	if first == second {
		t.Fatal("resume reused the closed signal")
	}
	_, err = p.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "next", Reason: "restart"}))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-second:
	default:
		t.Fatal("second prepare did not close new signal")
	}
}

func TestDrainTimeoutNamesTheRequestsHoldingTheFence(t *testing.T) {
	_, client, _, _ := hangingProvider(t, Hooks{})
	ctx := context.Background()
	prepared, err := client.Prepare(ctx, &commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "restart"})
	if err != nil {
		t.Fatal(err)
	}
	fence := prepared.GetStanding()
	_, err = client.Drain(ctx, &commonv1.LifecycleDrainRequest{OperationId: "op", FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision(), Timeout: durationpb.New(150 * time.Millisecond)})
	if connect.CodeOf(err) != connect.CodeDeadlineExceeded {
		t.Fatalf("drain error = %v, want deadline exceeded", err)
	}
	for _, want := range []string{"1 admitted request(s) still running", "POST /api/v1/slow", "since 20"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("drain error %q does not contain %q", err, want)
		}
	}
}

func TestDrainWithoutRequestTimeoutIsStillBounded(t *testing.T) {
	var deadline time.Time
	callerCtx, cancelCaller := context.WithCancel(context.Background())
	defer cancelCaller()
	p, _, _, _ := hangingProvider(t, Hooks{
		Quiesce: func(ctx context.Context) error {
			deadline, _ = ctx.Deadline()
			cancelCaller() // the hung request would otherwise hold this drain for the whole bound
			return nil
		},
		Resume: func(context.Context) error { return nil },
	})
	prepared, err := p.Prepare(context.Background(), connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "op", Reason: "restart"}))
	if err != nil {
		t.Fatal(err)
	}
	fence := prepared.Msg.GetStanding()
	_, _ = p.Drain(callerCtx, connect.NewRequest(&commonv1.LifecycleDrainRequest{OperationId: "op", FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision(), QuiesceComponents: true}))
	if deadline.IsZero() {
		t.Fatal("drain without a timeout ran unbounded; want DefaultDrainTimeout applied")
	}
	if remaining := time.Until(deadline); remaining < DefaultDrainTimeout-time.Minute || remaining > DefaultDrainTimeout {
		t.Fatalf("drain deadline in %s, want about %s", remaining, DefaultDrainTimeout)
	}
}

func TestFinishedRequestsLeaveTheInventory(t *testing.T) {
	p, _, _, stop := hangingProvider(t, Hooks{})
	p.mu.Lock()
	before := len(p.inflight)
	p.mu.Unlock()
	if before != 1 {
		t.Fatalf("inflight = %d, want 1", before)
	}
	stop()
	deadline := time.Now().Add(5 * time.Second)
	for {
		p.mu.Lock()
		left := len(p.inflight)
		p.mu.Unlock()
		if left == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("inflight = %d after the handler returned, want 0", left)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
