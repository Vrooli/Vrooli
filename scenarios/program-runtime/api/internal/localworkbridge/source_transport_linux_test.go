//go:build linux

package localworkbridge

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type releaseFixture struct {
	calls int
	scope SourceBrokerScope
	run   func(context.Context, func(context.Context) error) error
}

func (r *releaseFixture) WithinSourceRelease(ctx context.Context, s SourceBrokerScope, fn func(context.Context) error) error {
	r.calls++
	r.scope = s
	if r.run != nil {
		return r.run(ctx, fn)
	}
	return fn(ctx)
}
func transportFixture(t *testing.T, b *SourceBroker) (*SourceTransport, *os.File, *os.File, *releaseFixture) {
	t.Helper()
	req, wr, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	rd, resp, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range []*os.File{req, wr, rd, resp} {
		t.Cleanup(func() { f.Close() })
	}
	boundary := &releaseFixture{}
	tr, e := NewSourceTransport(SourceTransportConfig{Enabled: true, Broker: b, Requests: req, Responses: resp, Boundary: boundary, MaxOperations: 4, Lifetime: 3 * time.Second, IOTimeout: 500 * time.Millisecond})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if e := tr.Stop(ctx); e != nil {
			t.Error(e)
		}
	})
	return tr, wr, rd, boundary
}
func writeSourceFrame(t *testing.T, w *os.File, raw []byte) {
	t.Helper()
	w.SetWriteDeadline(time.Now().Add(time.Second))
	var h [4]byte
	binary.BigEndian.PutUint32(h[:], uint32(len(raw)))
	if _, e := w.Write(append(h[:], raw...)); e != nil {
		t.Fatal(e)
	}
}
func readSourceFrame(t *testing.T, r *os.File) []byte {
	t.Helper()
	r.SetReadDeadline(time.Now().Add(time.Second))
	var h [4]byte
	if _, e := io.ReadFull(r, h[:]); e != nil {
		t.Fatal(e)
	}
	n := binary.BigEndian.Uint32(h[:])
	if n > sourceBrokerResponseLimit {
		t.Fatal(n)
	}
	raw := make([]byte, n)
	if _, e := io.ReadFull(r, raw); e != nil {
		t.Fatal(e)
	}
	return raw
}
func startSourceTransport(tr *SourceTransport) <-chan error {
	done := make(chan error, 1)
	go func() { done <- tr.Serve(context.Background()) }()
	return done
}
func awaitSourceTransport(t *testing.T, done <-chan error, failed bool) {
	t.Helper()
	select {
	case e := <-done:
		if (e != nil) != failed {
			t.Fatalf("serve: %v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("transport did not join")
	}
}
func TestSourceTransportDisabledNoIO(t *testing.T) {
	tr, e := NewSourceTransport(SourceTransportConfig{})
	if e != nil {
		t.Fatal(e)
	}
	if !errors.Is(tr.Serve(nil), ErrDisabled) || tr.Stop(nil) != nil {
		t.Fatal("disabled transport touched dependency")
	}
}
func TestSourceTransportRejectsDescriptors(t *testing.T) {
	b, _, _, _, _, _ := brokerFixture(t)
	tr, w, r, _ := transportFixture(t, b)
	cases := []struct {
		name   string
		change func(*SourceTransportConfig)
	}{
		{"missing boundary", func(c *SourceTransportConfig) { c.Boundary = nil }},
		{"wrong read direction", func(c *SourceTransportConfig) { c.Requests = w }},
		{"wrong write direction", func(c *SourceTransportConfig) { c.Responses = r }},
		{"same pipe", func(c *SourceTransportConfig) { c.Responses = w }},
		{"excess operations", func(c *SourceTransportConfig) { c.MaxOperations = 65 }},
		{"unbounded lifetime", func(c *SourceTransportConfig) { c.Lifetime = 181 * time.Second }},
		{"unbounded IO", func(c *SourceTransportConfig) { c.IOTimeout = 6 * time.Second }},
	}
	for _, v := range cases {
		t.Run(v.name, func(t *testing.T) {
			c := tr.cfg
			v.change(&c)
			if _, e := NewSourceTransport(c); e == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
	f, e := os.CreateTemp(t.TempDir(), "ordinary")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	c := tr.cfg
	c.Requests = f
	if _, e := NewSourceTransport(c); e == nil {
		t.Fatal("ordinary file accepted")
	}
}
func TestSourceTransportFiniteFramesAndRetainedIdentity(t *testing.T) {
	b, _, creds, o, _, _ := brokerFixture(t)
	tr, w, r, boundary := transportFixture(t, b)
	done := startSourceTransport(tr)
	for _, op := range []string{"read_event", "observe_collector", "admit_collector", "wait_collector"} {
		writeSourceFrame(t, w, brokerRequest(op))
		raw := readSourceFrame(t, r)
		if len(raw) == 0 || bytes.Contains(raw, []byte("Bearer")) || bytes.Contains(raw, []byte("fixture-not-a-credential")) {
			t.Fatal("invalid or secret response")
		}
	}
	awaitSourceTransport(t, done, false)
	if boundary.calls != 4 || boundary.scope != b.scope || o.calls != 4 || creds.calls != 4 || o.call.RequestKey != b.scope.RequestKey {
		t.Fatal("finite owner identity changed")
	}
	if !errors.Is(tr.Serve(context.Background()), ErrStopped) {
		t.Fatal("second lifetime admitted")
	}
}
func TestSourceTransportMalformedFramesBeforeAuthority(t *testing.T) {
	for _, raw := range [][]byte{{}, {0, 0}, {0, 0, 0, 0}, {0, 0, 16, 1}, {0, 0, 0, 5, '{', '}'}, {0, 0, 0, 2, '{', '}'}} {
		t.Run(strings.ReplaceAll(string(raw), "\x00", "_"), func(t *testing.T) {
			b, f, c, o, _, _ := brokerFixture(t)
			tr, w, _, _ := transportFixture(t, b)
			done := startSourceTransport(tr)
			w.Write(raw)
			w.Close()
			awaitSourceTransport(t, done, len(raw) > 0)
			if f.calls+c.calls+o.calls != 0 {
				t.Fatal("malformed frame reached authority")
			}
		})
	}
}
func TestSourceTransportRevocationAtReleaseWritesNothing(t *testing.T) {
	b, _, _, _, eb, _ := brokerFixture(t)
	tr, w, r, boundary := transportFixture(t, b)
	boundary.run = func(ctx context.Context, fn func(context.Context) error) error { eb.b.Revoked = true; return fn(ctx) }
	done := startSourceTransport(tr)
	writeSourceFrame(t, w, brokerRequest("read_event"))
	awaitSourceTransport(t, done, true)
	got, e := io.ReadAll(r)
	if e != nil || len(got) != 0 {
		t.Fatalf("revoked response leaked: %d %v", len(got), e)
	}
}
func TestSourceTransportExpiredOrInvalidReleaseCallback(t *testing.T) {
	for _, kind := range []string{"expired", "missing", "twice", "discarded error"} {
		t.Run(kind, func(t *testing.T) {
			b, _, _, _, _, _ := brokerFixture(t)
			tr, w, r, boundary := transportFixture(t, b)
			boundary.run = func(ctx context.Context, fn func(context.Context) error) error {
				switch kind {
				case "missing":
					return nil
				case "discarded error":
					inner, cancel := context.WithCancel(ctx)
					cancel()
					_ = fn(inner)
					return nil
				case "twice":
					if e := fn(ctx); e != nil {
						return e
					}
					return fn(ctx)
				default:
					inner, cancel := context.WithCancel(ctx)
					cancel()
					return fn(inner)
				}
			}
			done := startSourceTransport(tr)
			writeSourceFrame(t, w, brokerRequest("read_event"))
			if kind == "twice" {
				readSourceFrame(t, r)
			}
			awaitSourceTransport(t, done, true)
			raw, _ := io.ReadAll(r)
			if len(raw) != 0 {
				t.Fatal("invalid release wrote extra bytes")
			}
		})
	}
}
func TestSourceTransportBackpressureAndStopJoin(t *testing.T) {
	for _, stop := range []bool{false, true} {
		t.Run(map[bool]string{false: "deadline", true: "stop"}[stop], func(t *testing.T) {
			b, _, _, o, _, _ := brokerFixture(t)
			o.event = []byte(`{"eventId":"event-1","eventType":"agent_manager.run.notification.v1","source":{"scenario":"agent-manager"},"data":{"blob":"` + strings.Repeat("x", 200000) + `"}}`)
			tr, w, r, boundary := transportFixture(t, b)
			entered := make(chan struct{})
			boundary.run = func(ctx context.Context, fn func(context.Context) error) error { close(entered); return fn(ctx) }
			done := startSourceTransport(tr)
			writeSourceFrame(t, w, brokerRequest("read_event"))
			<-entered
			var prefix []byte
			if stop {
				prefix = make([]byte, 4)
				r.SetReadDeadline(time.Now().Add(time.Second))
				if _, err := io.ReadFull(r, prefix); err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				if e := tr.Stop(ctx); e != nil {
					t.Fatal(e)
				}
			}
			awaitSourceTransport(t, done, true)
			raw, _ := io.ReadAll(r)
			raw = append(prefix, raw...)
			if len(raw) < 4 {
				t.Fatal("did not exercise partial release")
			}
			if int(binary.BigEndian.Uint32(raw[:4])) <= len(raw)-4 {
				t.Fatal("full response escaped backpressure")
			}
		})
	}
}
func TestSourceTransportStopJoinsOwnerAndNeverReplaces(t *testing.T) {
	b, _, _, o, _, _ := brokerFixture(t)
	tr, w, _, _ := transportFixture(t, b)
	entered, exit := make(chan struct{}), make(chan struct{})
	o.run = func(context.Context, *SourcePermit) { close(entered); <-exit }
	done := startSourceTransport(tr)
	writeSourceFrame(t, w, brokerRequest("read_event"))
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if !errors.Is(tr.Stop(ctx), context.DeadlineExceeded) {
		t.Fatal("unjoined owner reported stopped")
	}
	if !errors.Is(tr.Serve(context.Background()), ErrStopped) {
		t.Fatal("replacement allowed")
	}
	close(exit)
	awaitSourceTransport(t, done, true)
	if e := tr.Stop(context.Background()); e != nil {
		t.Fatal(e)
	}
}
func TestSourceTransportStopBeforeStartAndPartialRead(t *testing.T) {
	for _, start := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "partial"}[start], func(t *testing.T) {
			b, _, _, _, _, _ := brokerFixture(t)
			tr, w, _, _ := transportFixture(t, b)
			var done <-chan error
			if start {
				done = startSourceTransport(tr)
				w.Write([]byte{0, 0})
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if e := tr.Stop(ctx); e != nil {
				t.Fatal(e)
			}
			if done != nil {
				awaitSourceTransport(t, done, true)
			}
			if !errors.Is(tr.Serve(context.Background()), ErrStopped) {
				t.Fatal("restart allowed")
			}
		})
	}
}

// These are finite test-owned helper processes, never production workers or
// confinement proof. A real helper crash verifies descriptor/Serve retirement.
func TestSourceTransportWorkerFixture(t *testing.T) {
	mode := os.Getenv("VROOLI_LOCAL_WORK_TRANSPORT_FIXTURE")
	if mode == "" {
		return
	}
	requests, responses := os.NewFile(3, "requests"), os.NewFile(4, "responses")
	if mode == "crash" {
		requests.Write([]byte{0, 0, 0})
		os.Exit(23)
	}
	writeSourceFrame(t, requests, brokerRequest("read_event"))
	raw := readSourceFrame(t, responses)
	if !bytes.Contains(raw, []byte("event-1")) || bytes.Contains(raw, []byte("Bearer")) {
		os.Exit(24)
	}
	requests.Close()
	responses.Close()
	os.Exit(0)
}
func TestSourceTransportWorkerProcessAndCrash(t *testing.T) {
	for _, mode := range []string{"roundtrip", "crash"} {
		t.Run(mode, func(t *testing.T) {
			b, _, _, o, _, _ := brokerFixture(t)
			tr, w, r, _ := transportFixture(t, b)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSourceTransportWorkerFixture$")
			cmd.Env = []string{"VROOLI_LOCAL_WORK_TRANSPORT_FIXTURE=" + mode}
			cmd.ExtraFiles = []*os.File{w, r}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			w.Close()
			r.Close()
			done := startSourceTransport(tr)
			err := cmd.Wait()
			if (err != nil) != (mode == "crash") {
				t.Fatalf("worker status: %v", err)
			}
			awaitSourceTransport(t, done, mode == "crash")
			if mode == "crash" && o.calls != 0 {
				t.Fatal("truncated crashed request reached owner")
			}
			if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
				t.Fatal("worker not joined")
			}
		})
	}
}

func TestSourceTransportCancellationCallbackIsJoined(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entered, exit, joined := make(chan struct{}), make(chan struct{}), make(chan struct{})
	stop := sourceAfterFunc(ctx, func() { close(entered); <-exit })
	cancel()
	<-entered
	go func() { stop(); close(joined) }()
	select {
	case <-joined:
		t.Fatal("callback not joined")
	case <-time.After(20 * time.Millisecond):
	}
	close(exit)
	select {
	case <-joined:
	case <-time.After(time.Second):
		t.Fatal("join did not finish")
	}
}
