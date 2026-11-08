package localworkbridge

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type runnerFunc func(context.Context, Request) (Result, error)

func (f runnerFunc) Run(c context.Context, r Request) (Result, error) { return f(c, r) }

type boundedRecorder struct{ *httptest.ResponseRecorder }

func (boundedRecorder) SetReadDeadline(time.Time) error { return nil }
func request(body string) *http.Request {
	r := httptest.NewRequest("POST", Path, strings.NewReader(body))
	r.Header.Set("X-Vrooli-Events-Signature", "sha256=fixture")
	r.Header.Set("X-Vrooli-Events-Event-ID", "event-1")
	r.Header.Set("X-Vrooli-Events-Idempotency-Key", "event-1")
	return r
}
func enabled(t *testing.T, f runnerFunc) *Host {
	t.Helper()
	h, err := New(Config{Enabled: true, Interval: time.Second, Timeout: time.Second, Runner: f})
	if err != nil {
		t.Fatal(err)
	}
	return h
}
func TestDisabledNeverTouchesDependencies(t *testing.T) {
	h, err := New(Config{Runner: runnerFunc(func(context.Context, Request) (Result, error) { t.Fatal("runner touched"); return Result{}, nil })})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request("x"))
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
	if err := h.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := h.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestEnabledRequiresExplicitBoundsAndRunner(t *testing.T) {
	for _, c := range []Config{{Enabled: true}, {Enabled: true, Interval: time.Second, Timeout: 181 * time.Second, Runner: runnerFunc(nil)}} {
		if _, err := New(c); err == nil {
			t.Fatal("accepted incomplete configuration")
		}
	}
}
func TestExactBytesAndDurableStatusBoundary(t *testing.T) {
	raw := " {\"payload\": 1}\n"
	h := enabled(t, func(ctx context.Context, r Request) (Result, error) {
		if r.Operation != "accept_native" || string(r.Body) != raw || r.EventID != "event-1" || r.IdempotencyKey != "event-1" || r.Signature != "sha256=fixture" {
			t.Fatalf("changed envelope: %+v", r)
		}
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("unbounded")
		}
		return Result{204}, nil
	})
	w := boundedRecorder{httptest.NewRecorder()}
	h.ServeHTTP(w, request(raw))
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
func TestRejectsBeforeRunner(t *testing.T) {
	h := enabled(t, func(context.Context, Request) (Result, error) { t.Fatal("runner touched"); return Result{}, nil })
	cases := []struct {
		name   string
		edit   func(*http.Request)
		status int
	}{
		{"wrong path", func(r *http.Request) { r.URL.Path += "/" }, 404},
		{"query", func(r *http.Request) { r.URL.RawQuery = "x=y" }, 404},
		{"method", func(r *http.Request) { r.Method = "GET" }, 405},
		{"missing signature", func(r *http.Request) { r.Header.Del("X-Vrooli-Events-Signature") }, 400},
		{"duplicate", func(r *http.Request) { r.Header.Add("X-Vrooli-Events-Event-ID", "event-1") }, 400},
		{"mismatch", func(r *http.Request) { r.Header.Set("X-Vrooli-Events-Idempotency-Key", "other") }, 400},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := request("{}")
			tt.edit(r)
			w := boundedRecorder{httptest.NewRecorder()}
			h.ServeHTTP(w, r)
			if w.Code != tt.status {
				t.Fatal(w.Code)
			}
		})
	}
	w := boundedRecorder{httptest.NewRecorder()}
	h.ServeHTTP(w, request(strings.Repeat("x", int(MaxBodyBytes)+1)))
	if w.Code != 413 {
		t.Fatal(w.Code)
	}
	plain := httptest.NewRecorder()
	h.ServeHTTP(plain, request("{}"))
	if plain.Code != 503 {
		t.Fatal(plain.Code)
	}
}
func TestInvalidOrLateResultNeverAcknowledged(t *testing.T) {
	for _, status := range []int{0, 200, 202, 302, 500} {
		h := enabled(t, func(context.Context, Request) (Result, error) { return Result{status}, nil })
		w := boundedRecorder{httptest.NewRecorder()}
		h.ServeHTTP(w, request("{}"))
		if w.Code != 502 {
			t.Fatal(w.Code)
		}
	}
	h := enabled(t, func(ctx context.Context, _ Request) (Result, error) { <-ctx.Done(); return Result{204}, nil })
	h.cfg.Timeout = time.Millisecond
	w := boundedRecorder{httptest.NewRecorder()}
	h.ServeHTTP(w, request("{}"))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
}
func TestSerialCancellationAndTerminalStop(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan error, 1)
	h := enabled(t, func(ctx context.Context, _ Request) (Result, error) {
		close(started)
		<-ctx.Done()
		return Result{}, ctx.Err()
	})
	go func() { _, err := h.invoke(context.Background(), Request{Operation: "tick"}); finished <- err }()
	<-started
	if _, err := h.invoke(context.Background(), Request{}); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := h.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := h.invoke(ctx, Request{}); !errors.Is(err, ErrStopped) {
		t.Fatal(err)
	}
	if err := h.Run(ctx); !errors.Is(err, ErrStopped) {
		t.Fatal(err)
	}
	if err := h.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
func TestManagedLoopNoImmediateTickAndCancellation(t *testing.T) {
	h := enabled(t, func(context.Context, Request) (Result, error) { t.Fatal("unexpected tick"); return Result{}, nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestManagedLoopOneCycleAndStopsOnError(t *testing.T) {
	sentinel := errors.New("cycle failed")
	h := enabled(t, func(_ context.Context, r Request) (Result, error) {
		if r.Operation != "tick" {
			t.Fatal(r.Operation)
		}
		return Result{}, sentinel
	})
	h.cfg.Interval = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := h.Run(ctx); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
}

func TestStopTimeoutDoesNotPretendRunnerExited(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan struct{})
	h := enabled(t, func(context.Context, Request) (Result, error) { close(started); <-release; return Result{204}, nil })
	go func() { defer close(done); _, _ = h.invoke(context.Background(), Request{Operation: "tick"}) }()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := h.Stop(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := h.invoke(context.Background(), Request{}); !errors.Is(err, ErrStopped) {
		t.Fatal(err)
	}
	close(release)
	<-done
	if err := h.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNativeBudgetAndSafeError(t *testing.T) {
	h := enabled(t, func(ctx context.Context, _ Request) (Result, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 8*time.Second {
			t.Fatal("native budget exceeds producer budget")
		}
		return Result{}, errors.New("private error must never escape")
	})
	h.cfg.Timeout = 180 * time.Second
	w := boundedRecorder{httptest.NewRecorder()}
	h.ServeHTTP(w, request("{}"))
	if w.Code != 503 || strings.Contains(w.Body.String(), "private") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestManagedLoopRejectsUnsuccessfulWorkerStatus(t *testing.T) {
	for _, status := range []int{503, 0, 200, 201, 202, 203, 205, 206, 299} {
		calls := 0
		h := enabled(t, func(context.Context, Request) (Result, error) {
			calls++
			return Result{HTTPStatus: status}, nil
		})
		h.cfg.Interval = time.Millisecond
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := h.Run(ctx)
		cancel()
		if err == nil || err.Error() != "local work bridge cycle not completed" {
			t.Fatalf("status %d: expected fixed cycle failure, got %v", status, err)
		}
		if calls != 1 {
			t.Fatalf("status %d: executed %d cycles", status, calls)
		}
	}
}

func TestManagedLoopContinuesOnlyAfter204(t *testing.T) {
	calls := 0
	sentinel := errors.New("second cycle")
	h := enabled(t, func(context.Context, Request) (Result, error) {
		calls++
		if calls == 1 {
			return Result{HTTPStatus: http.StatusNoContent}, nil
		}
		return Result{}, sentinel
	})
	h.cfg.Interval = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := h.Run(ctx); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("executed %d cycles", calls)
	}
}
