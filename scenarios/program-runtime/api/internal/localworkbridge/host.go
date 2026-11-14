// Package localworkbridge provides an inert-by-default, host-owned bridge seam.
// It does not provision credentials, start processes, or register routes.
package localworkbridge

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

const Path = "/internal/program-runtime/local-work/events"
const MaxBodyBytes int64 = 64 << 10

var (
	ErrDisabled = errors.New("local work bridge disabled")
	ErrStopped  = errors.New("local work bridge stopped")
	ErrBusy     = errors.New("local work bridge busy")
)

// Request is an in-memory boundary, not a serialized worker protocol. Body is
// preserved exactly: the worker authenticates it before durable acceptance.
type Request struct {
	Operation                          string
	Body                               []byte
	Signature, EventID, IdempotencyKey string
}
type Result struct{ HTTPStatus int }

// Runner must honor cancellation, bound output, and reap any managed process
// before returning. In-scope native 204 requires authenticated durable commit
// or an exact duplicate. Authenticated out-of-scope events may return 204 and be
// ignored without storage. Tick performs at most one bounded collect/export
// cycle. No ambient authority, PATH Python fallback, or shell/source-controlled
// commands are allowed.
type Runner interface {
	Run(context.Context, Request) (Result, error)
}

type Config struct {
	Enabled  bool
	Interval time.Duration
	Timeout  time.Duration
	Runner   Runner
	// OnFailure receives a fixed, scrubbed tick-loop failure promptly. It must
	// return without blocking; it is never called for disabled hosts.
	OnFailure func(error)
}

type Host struct {
	cfg              Config
	mu               sync.Mutex
	stopped, running bool
	stop             chan struct{}
	idle             chan struct{}
	cancel           context.CancelFunc
}

// New performs no I/O or background work. Disabled config ignores dependencies.
func New(cfg Config) (*Host, error) {
	h := &Host{cfg: cfg}
	if !cfg.Enabled {
		return h, nil
	}
	if cfg.Runner == nil || cfg.Interval < time.Second || cfg.Interval > time.Hour || cfg.Timeout <= 0 || cfg.Timeout > 180*time.Second {
		return nil, errors.New("explicit runner, interval [1s,1h], and timeout (0,180s] required")
	}
	h.stop = make(chan struct{})
	return h, nil
}

func (h *Host) invoke(ctx context.Context, req Request) (Result, error) {
	if !h.cfg.Enabled {
		return Result{}, ErrDisabled
	}
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		return Result{}, ErrStopped
	}
	if h.idle != nil {
		h.mu.Unlock()
		return Result{}, ErrBusy
	}
	timeout := h.cfg.Timeout
	if req.Operation == "accept_native" && timeout > 8*time.Second {
		timeout = 8 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	h.cancel = cancel
	idle := make(chan struct{})
	h.idle = idle
	h.mu.Unlock()
	defer func() {
		cancel()
		h.mu.Lock()
		h.cancel = nil
		h.idle = nil
		close(idle)
		h.mu.Unlock()
	}()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	result, err := h.cfg.Runner.Run(ctx, req)
	// Never turn a late completion into an acknowledged delivery.
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	return result, err
}

// Run blocks in the host-owned lifecycle. It creates no goroutine and does not
// tick immediately. A failed cycle terminates the loop for owner supervision.
// Concurrent Run calls are rejected. Stop is terminal; construct a new host to restart.
func (h *Host) Run(ctx context.Context) error {
	if !h.cfg.Enabled {
		return nil
	}
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		return ErrStopped
	}
	if h.running {
		h.mu.Unlock()
		return ErrBusy
	}
	h.running = true
	h.mu.Unlock()
	defer func() { h.mu.Lock(); h.running = false; h.mu.Unlock() }()
	timer := time.NewTicker(h.cfg.Interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.stop:
			return nil
		case <-timer.C:
			result, err := h.invoke(ctx, Request{Operation: "tick"})
			if errors.Is(err, ErrBusy) {
				continue
			}
			if errors.Is(err, ErrStopped) {
				return nil
			}
			if err != nil {
				return err
			}
			if result.HTTPStatus != http.StatusNoContent {
				return errors.New("local work bridge cycle not completed")
			}
		}
	}
}

// Stop cancels the active call and waits at most until ctx expires. A timeout
// is not proof that an injected runner stopped; the owner must not restart it.
func (h *Host) Stop(ctx context.Context) error {
	if !h.cfg.Enabled {
		return nil
	}
	h.mu.Lock()
	if !h.stopped {
		h.stopped = true
		close(h.stop)
	}
	if h.cancel != nil {
		h.cancel()
	}
	idle := h.idle
	h.mu.Unlock()
	if idle == nil {
		return nil
	}
	select {
	case <-idle:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Host) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Enabled || r.URL.Path != Path || r.URL.RawQuery != "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", 405)
		return
	}
	// Duplicate identity/signature headers are ambiguous and must not be folded.
	keys := []string{"X-Vrooli-Events-Signature", "X-Vrooli-Events-Event-ID", "X-Vrooli-Events-Idempotency-Key"}
	vals := make([]string, len(keys))
	for i, k := range keys {
		v := r.Header.Values(k)
		if len(v) != 1 || len(v[0]) == 0 || len(v[0]) > 512 {
			http.Error(w, "invalid event headers", 400)
			return
		}
		vals[i] = v[0]
	}
	if vals[1] != vals[2] {
		http.Error(w, "event identity mismatch", 400)
		return
	}
	// Fail closed without a transport read deadline; contexts alone cannot
	// interrupt a slow request Body.Read. The real net/http writer supports this.
	readTimeout := h.cfg.Timeout
	if readTimeout > 8*time.Second {
		readTimeout = 8 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), readTimeout)
	defer cancel()
	r = r.WithContext(ctx)
	controller := http.NewResponseController(w)
	if err := controller.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		http.Error(w, "bounded transport unavailable", 503)
		return
	}
	defer controller.SetReadDeadline(time.Time{})
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "event too large", 413)
		} else {
			http.Error(w, "invalid event body", 400)
		}
		return
	}
	result, err := h.invoke(r.Context(), Request{Operation: "accept_native", Body: body, Signature: vals[0], EventID: vals[1], IdempotencyKey: vals[2]})
	if err != nil {
		http.Error(w, "bridge unavailable", 503)
		return
	}
	// Never expose worker text, headers, raw payload, or arbitrary status codes.
	switch result.HTTPStatus {
	case 204:
		w.WriteHeader(204)
	case 400, 401, 403, 409, 413, 429, 503:
		http.Error(w, "event not accepted", result.HTTPStatus)
	default:
		http.Error(w, "invalid bridge result", 502)
	}
}
