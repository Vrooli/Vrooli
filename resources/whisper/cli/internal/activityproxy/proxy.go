// Package activityproxy is whisper's activity edge: a host-side reverse proxy
// that sits in front of the supervised native whisper.cpp server and
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
// The edge owns the canonical client port (8090). The native server listens on
// the internal host port (18090), so "server healthy but 8090 down" means the
// companion is down: STT is unavailable and whisper capacity activity is not
// being reported until the companion is respawned.
package activityproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/companion"
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
	// ParentPID makes the activity edge terminate when its owning process exits.
	// Zero defaults to the process parent in Run.
	ParentPID int
	// ParentAlive is injectable for deterministic parent-death tests.
	ParentAlive func(int) bool
	// Now is the wall-clock seam for liveness logs.
	Now func() time.Time
	// Native enables the compatibility adapter from the historical /asr form to
	// whisper.cpp's /inference form. Compatibility tests and attach-only users can
	// keep the transparent proxy by leaving it false.
	Native bool
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
var defaultDebounce = tuning.ActivityDebounce()

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
	upstream := fs.String("upstream", h.env(envUpstream, defaultUpstream), "host:port of the native whisper server (internal)")
	debounce := fs.Duration("debounce", h.debounce(), "idle debounce after the last in-flight /asr completes")
	native := fs.Bool("native", h.Native, "adapt the historical /asr multipart contract to whisper.cpp")
	parent := fs.Int("parent-pid", h.ParentPID, "PID of the supervised whisper resource process")
	if err := fs.Parse(args); err != nil {
		return err
	}
	parentPID := *parent
	if parentPID <= 1 {
		return fmt.Errorf("activity edge: parent PID is required (start with --parent-pid <resource-pid>)")
	}

	started := h.now()
	srv, tr, err := h.serverWithTrackerMode(*listen, *upstream, *debounce, *native)
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
	parentTicker := time.NewTicker(h.heartbeatInterval())
	defer parentTicker.Stop()

	for {
		select {
		case <-parentTicker.C:
			if h.parentGone(parentPID) {
				shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutCtx)
				fmt.Fprintln(h.Stderr, "activity-edge exit: reason=parent_gone")
				return nil
			}
			continue
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
}

func (h *Handlers) parentGone(pid int) bool {
	if pid <= 1 {
		return false
	}
	if h.ParentAlive != nil {
		return !h.ParentAlive(pid)
	}
	err := syscall.Kill(pid, 0)
	return err != nil && err != syscall.EPERM
}

// server builds the transparent reverse proxy + the /asr bracketing handler.
func (h *Handlers) server(listen, upstream string, debounce time.Duration) (*http.Server, error) {
	srv, _, err := h.serverWithTracker(listen, upstream, debounce)
	return srv, err
}

func (h *Handlers) serverWithTracker(listen, upstream string, debounce time.Duration) (*http.Server, *tracker, error) {
	return h.serverWithTrackerMode(listen, upstream, debounce, h.Native)
}

func (h *Handlers) serverWithTrackerMode(listen, upstream string, debounce time.Duration, native bool) (*http.Server, *tracker, error) {
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
		if native && isTranscription(r) {
			adapted, err := adaptNativeRequest(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if strings.Contains(strings.ToLower(r.URL.Path), "detect-language") {
				rec := httptest.NewRecorder()
				proxy.ServeHTTP(rec, adapted)
				copyRecordedResponse(w, rec, true)
				return
			}
			proxy.ServeHTTP(w, adapted)
			return
		}
		proxy.ServeHTTP(w, r)
	})
	return &http.Server{Handler: mux, ReadHeaderTimeout: tuning.ActivityReadHeaderTimeout()}, tr, nil
}

// adaptNativeRequest preserves the public audio_file multipart field and
// /asr path while translating to whisper.cpp's /inference endpoint. Query
// controls from the historical API are promoted into the native multipart
// fields because whisper.cpp does not consume them from the query string.
func adaptNativeRequest(r *http.Request) (*http.Request, error) {
	if r.Method != http.MethodPost || !isTranscription(r) {
		return r, nil
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return nil, fmt.Errorf("native whisper requires multipart audio: %w", err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, values := range r.MultipartForm.Value {
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				return nil, fmt.Errorf("copy whisper form field %q: %w", key, err)
			}
		}
	}
	for key, headers := range r.MultipartForm.File {
		for _, header := range headers {
			file, err := header.Open()
			if err != nil {
				return nil, fmt.Errorf("open uploaded audio: %w", err)
			}
			targetKey := key
			if key == "audio_file" {
				targetKey = "file"
			}
			part, err := writer.CreateFormFile(targetKey, header.Filename)
			if err == nil {
				_, err = io.Copy(part, file)
			}
			closeErr := file.Close()
			if err != nil {
				return nil, fmt.Errorf("copy uploaded audio: %w", err)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("close uploaded audio: %w", closeErr)
			}
		}
	}
	if _, ok := r.MultipartForm.Value["response_format"]; !ok {
		format := nativeResponseFormat(r.URL.Query().Get("output"))
		if err := writer.WriteField("response_format", format); err != nil {
			return nil, fmt.Errorf("set native response format: %w", err)
		}
	}
	for _, key := range []string{"task", "language", "prompt", "initial_prompt"} {
		if _, inBody := r.MultipartForm.Value[key]; inBody {
			continue
		}
		if value := r.URL.Query().Get(key); value != "" {
			if err := writer.WriteField(key, value); err != nil {
				return nil, fmt.Errorf("copy native query field %q: %w", key, err)
			}
		}
	}
	if strings.Contains(strings.ToLower(r.URL.Path), "detect-language") {
		if err := writer.WriteField("detect_language", "true"); err != nil {
			return nil, fmt.Errorf("set native language detection: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close native whisper form: %w", err)
	}
	adapted := r.Clone(r.Context())
	adapted.URL.Path = "/inference"
	adapted.URL.RawPath = ""
	adapted.URL.RawQuery = ""
	adapted.Body = io.NopCloser(bytes.NewReader(body.Bytes()))
	adapted.ContentLength = int64(body.Len())
	adapted.Header = adapted.Header.Clone()
	adapted.Header.Set("Content-Type", writer.FormDataContentType())
	adapted.Header.Set("Content-Length", fmt.Sprintf("%d", body.Len()))
	return adapted, nil
}

func nativeResponseFormat(output string) string {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "text", "srt", "vtt":
		return strings.ToLower(strings.TrimSpace(output))
	default:
		return "verbose_json"
	}
}

func copyRecordedResponse(w http.ResponseWriter, rec *httptest.ResponseRecorder, detectLanguage bool) {
	body := rec.Body.Bytes()
	if detectLanguage && rec.Code >= http.StatusOK && rec.Code < http.StatusMultipleChoices {
		body = adaptDetectLanguageResponse(body)
		w.Header().Set("Content-Type", "application/json")
	}
	for key, values := range rec.Header() {
		if key == "Content-Length" || (detectLanguage && key == "Content-Type") {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(rec.Code)
	_, _ = w.Write(body)
}

func adaptDetectLanguageResponse(body []byte) []byte {
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return body
	}
	language, _ := payload["detected_language"].(string)
	if language == "" {
		language, _ = payload["language"].(string)
	}
	if _, ok := payload["language_code"]; !ok {
		payload["language_code"] = languageCode(language)
	}
	if _, ok := payload["confidence"]; !ok {
		if probability, ok := payload["detected_language_probability"].(float64); ok {
			payload["confidence"] = probability
		}
	}
	adapted, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return adapted
}

func languageCode(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "english":
		return "en"
	case "spanish":
		return "es"
	case "french":
		return "fr"
	case "german":
		return "de"
	case "italian":
		return "it"
	case "portuguese":
		return "pt"
	case "dutch":
		return "nl"
	default:
		return strings.ToLower(strings.TrimSpace(language))
	}
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
// report tells the broker this resource started or stopped working.
//
// Finding the active claim and setting its state is the half every capacity
// companion shares, so it comes from packages/capacity/companion rather than
// being copied here. What stays local is the edge's own contract: this runs off
// the request path, and every failure is swallowed so a broker outage can never
// affect a transcription.
func (h *Handlers) report(resource, state string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	companion.Reporter{Resource: resource, Exec: h.Exec}.ReportActivity(ctx, state)
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
	return tuning.CompanionHeartbeatInterval()
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
