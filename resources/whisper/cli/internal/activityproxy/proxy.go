// Package activityproxy is whisper's activity edge: a host-side reverse proxy
// that sits in front of the third-party whisper-asr-webservice container and
// reports transcription activity to the platform capacity broker.
//
// Why the edge and not the caller (plan §2, internal/capacity/doc.go
// activity-source contract): whisper is request/response HTTP, and the two
// consumers that dominate live use — the host dictation tool and the browser
// WhisperProvider — are clients Vrooli does not own and cannot instrument
// caller-side. The ONE place that sees every `POST /asr` is whisper's own request
// boundary. The proxy brackets each transcription (active on request-in, idle
// after a debounce on the last response-out) via the sanctioned
// `vrooli capacity activity` CLI, so the broker can protect whisper while it is
// working and reclaim its VRAM when it is idle.
//
// It is FAIL-OPEN and transparent: it forwards every request/response unchanged
// (headers, status, streaming, multipart) and swallows every capacity error —
// transcription must never break because the broker is down. Capacity calls run
// OFF the request path (a background reporter goroutine), so the proxy adds
// negligible latency.
//
// The edge owns the canonical client port (8090). The compose container listens
// on the internal host port (18090), so "container healthy but 8090 down" means
// the companion is down: STT is unavailable and whisper capacity activity is not
// being reported until the companion is respawned.
package activityproxy

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	// defaultListen is the canonical whisper port the proxy owns; clients keep
	// hitting WHISPER_URL unchanged.
	defaultListen = "127.0.0.1:8090"
	// defaultUpstream is the internal host port the container republishes to.
	defaultUpstream = "127.0.0.1:18090"
	// resourceName is the capacity owner id whose claim the edge reports against.
	resourceName = "whisper"

	envListen   = "WHISPER_PROXY_LISTEN"
	envUpstream = "WHISPER_PROXY_UPSTREAM"
)

// Handlers owns the proxy's dependencies. Tests inject the seams so no real
// server, exec, or clock is needed.
type Handlers struct {
	Stdout io.Writer
	Stderr io.Writer
	GetEnv func(string) string
	// Exec runs a `vrooli capacity …` call and returns its stdout. Tests inject a
	// fake; production shells the on-PATH vrooli binary.
	Exec func(ctx context.Context, name string, args ...string) ([]byte, error)
	// Debounce overrides the idle debounce (default mirrors idle_grace's spirit:
	// long enough that back-to-back dictation segments don't flap the signal).
	Debounce time.Duration
	// Listen is the network-listener seam. Production uses net.Listen.
	Listen func(network, address string) (net.Listener, error)
	// SignalCh overrides OS signal delivery in tests.
	SignalCh <-chan os.Signal
	// HeartbeatInterval overrides the liveness log interval. Production uses 60s.
	HeartbeatInterval time.Duration
	// Now is the wall-clock seam for liveness logs.
	Now func() time.Time
}

// Default returns Handlers wired to the real shell.
func Default() *Handlers {
	return &Handlers{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		GetEnv: os.Getenv,
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		},
	}
}

// defaultDebounce is the dwell after the last in-flight /asr completes before the
// edge reports idle. It coalesces back-to-back dictation segments so the signal
// does not flap; the broker still waits its own idle_grace on top of this before
// the claim becomes reclaim-eligible.
const defaultDebounce = 5 * time.Second

const defaultHeartbeatInterval = 60 * time.Second

// Command returns the `activity-proxy` command for registration.
func Command(h *Handlers) cliapp.Command {
	if h == nil {
		h = Default()
	}
	return cliapp.Command{
		Name:        "activity-proxy",
		Description: "Run whisper's activity edge: a fail-open reverse proxy that brackets each /asr and reports capacity activity",
		Usage:       "whisper activity-proxy [--listen 127.0.0.1:8090] [--upstream 127.0.0.1:18090] [--debounce 5s]",
		Run:         h.Run,
	}
}

// Run parses flags and serves until signaled.
func (h *Handlers) Run(args []string) error {
	fs := flag.NewFlagSet("activity-proxy", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	listen := fs.String("listen", h.env(envListen, defaultListen), "host:port the proxy listens on (the canonical whisper port)")
	upstream := fs.String("upstream", h.env(envUpstream, defaultUpstream), "host:port of the whisper container (internal)")
	debounce := fs.Duration("debounce", h.debounce(), "idle debounce after the last in-flight /asr completes")
	if err := fs.Parse(args); err != nil {
		return err
	}

	started := h.now()
	srv, tr, err := h.serverWithTracker(*listen, *upstream, *debounce)
	if err != nil {
		fmt.Fprintf(h.Stderr, "activity-edge exit: reason=server-build err=%v\n", err)
		return err
	}

	ln, err := h.listen("tcp", *listen)
	if err != nil {
		err = fmt.Errorf("listen on %s: %w", *listen, err)
		fmt.Fprintf(h.Stderr, "activity-edge exit: reason=listen-error err=%v\n", err)
		return err
	}
	fmt.Fprintf(h.Stdout, "whisper activity edge: %s -> %s (idle debounce %s)\n", *listen, *upstream, *debounce)

	sigCh, stopSignals := h.signals()
	if stopSignals != nil {
		defer stopSignals()
	}
	heartbeatStop := h.startHeartbeat(tr, started)
	defer heartbeatStop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()

	select {
	case sig := <-sigCh:
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := srv.Shutdown(shutCtx)
		if err != nil {
			fmt.Fprintf(h.Stderr, "activity-edge exit: reason=signal sig=%s shutdown_error=%v\n", sig, err)
			return err
		}
		fmt.Fprintf(h.Stderr, "activity-edge exit: reason=signal sig=%s\n", sig)
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			fmt.Fprintln(h.Stderr, "activity-edge exit: reason=server-closed")
			return nil
		}
		fmt.Fprintf(h.Stderr, "activity-edge exit: reason=serve-error err=%v\n", err)
		return err
	}
}

// server builds the transparent reverse proxy + the /asr bracketing handler.
func (h *Handlers) server(listen, upstream string, debounce time.Duration) (*http.Server, error) {
	srv, _, err := h.serverWithTracker(listen, upstream, debounce)
	return srv, err
}

func (h *Handlers) serverWithTracker(listen, upstream string, debounce time.Duration) (*http.Server, *tracker, error) {
	target, err := url.Parse("http://" + strings.TrimPrefix(upstream, "http://"))
	if err != nil {
		return nil, nil, fmt.Errorf("parse upstream %q: %w", upstream, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	// FlushInterval -1 streams responses (and SSE/chunked) through immediately
	// rather than buffering — transparency for any streaming transcription.
	proxy.FlushInterval = -1
	// A dead upstream must not crash the proxy: surface 502 and keep serving.
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
		w.WriteHeader(http.StatusBadGateway)
	}

	tr := &tracker{
		debounce:   debounce,
		markActive: func() { h.report(resourceName, "active") },
		markIdle:   func() { h.report(resourceName, "idle") },
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if isTranscription(r) {
			tr.begin()
			defer tr.end()
		}
		proxy.ServeHTTP(w, r)
	})
	return &http.Server{Handler: mux, ReadHeaderTimeout: 15 * time.Second}, tr, nil
}

// isTranscription reports whether a request is a unit of transcription work that
// should hold whisper active. The webservice exposes POST /asr (transcribe) and
// POST /detect-language; GET / is the health/docs path and must NOT bracket.
func isTranscription(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	p := strings.ToLower(r.URL.Path)
	return strings.Contains(p, "/asr") || strings.Contains(p, "detect-language")
}

// report resolves whisper's active capacity claim and reports the given activity
// state. Best-effort: a missing vrooli binary, no live claim, or any CLI error is
// swallowed — the edge never lets capacity reporting affect transcription.
func (h *Handlers) report(resource, state string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	out, err := h.Exec(ctx, "vrooli", "capacity", "list", "--owner", resource, "--active", "--json")
	if err != nil {
		return
	}
	var payload struct {
		Claims []struct {
			ClaimID string `json:"claim_id"`
			OwnerID string `json:"owner_id"`
		} `json:"claims"`
	}
	if json.Unmarshal(out, &payload) != nil {
		return
	}
	for _, c := range payload.Claims {
		if c.OwnerID == resource && c.ClaimID != "" {
			// activity auto-resolves the generation server-side (last-writer-wins),
			// so the edge does not pass --generation.
			_, _ = h.Exec(ctx, "vrooli", "capacity", "activity", "--claim-id", c.ClaimID, "--state", state, "--json")
			return
		}
	}
}

func (h *Handlers) env(key, fallback string) string {
	if h.GetEnv == nil {
		return fallback
	}
	if v := strings.TrimSpace(h.GetEnv(key)); v != "" {
		return v
	}
	return fallback
}

func (h *Handlers) debounce() time.Duration {
	if h.Debounce > 0 {
		return h.Debounce
	}
	return defaultDebounce
}

func (h *Handlers) listen(network, address string) (net.Listener, error) {
	if h.Listen != nil {
		return h.Listen(network, address)
	}
	return net.Listen(network, address)
}

func (h *Handlers) signals() (<-chan os.Signal, func()) {
	if h.SignalCh != nil {
		return h.SignalCh, nil
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch, func() {
		signal.Stop(ch)
		close(ch)
	}
}

func (h *Handlers) heartbeatInterval() time.Duration {
	if h.HeartbeatInterval > 0 {
		return h.HeartbeatInterval
	}
	return defaultHeartbeatInterval
}

func (h *Handlers) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h *Handlers) startHeartbeat(tr *tracker, since time.Time) func() {
	interval := h.heartbeatInterval()
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				h.logHeartbeat(tr, since)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
}

func (h *Handlers) logHeartbeat(tr *tracker, since time.Time) {
	requests, active := tr.snapshot()
	fmt.Fprintf(h.Stdout, "activity-edge alive: requests=%d active=%t since=%s\n", requests, active, since.Format(time.RFC3339))
}

// tracker refcounts concurrent in-flight transcriptions and fires the activity
// transitions on the edges: active on the first concurrent request, idle after
// the last one finishes plus a debounce (so back-to-back segments don't flap).
// markActive/markIdle run on their own goroutines so the request path never
// blocks on a capacity call.
type tracker struct {
	mu        sync.Mutex
	inflight  int
	active    bool
	requests  uint64
	idleTimer *time.Timer
	debounce  time.Duration

	markActive func()
	markIdle   func()
}

func (t *tracker) begin() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.idleTimer != nil {
		t.idleTimer.Stop()
		t.idleTimer = nil
	}
	t.requests++
	t.inflight++
	if !t.active {
		t.active = true
		go t.markActive()
	}
}

func (t *tracker) end() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.inflight > 0 {
		t.inflight--
	}
	if t.inflight > 0 {
		return
	}
	if t.idleTimer != nil {
		t.idleTimer.Stop()
	}
	t.idleTimer = time.AfterFunc(t.debounce, func() {
		t.mu.Lock()
		if t.inflight > 0 { // a new segment arrived during the debounce — stay active
			t.mu.Unlock()
			return
		}
		t.active = false
		t.mu.Unlock()
		t.markIdle()
	})
}

func (t *tracker) snapshot() (uint64, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.requests, t.active
}
