package activityproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestIsTranscription(t *testing.T) {
	cases := []struct {
		method, path string
		want         bool
	}{
		{http.MethodPost, "/asr", true},
		{http.MethodPost, "/asr?task=transcribe", true},
		{http.MethodPost, "/detect-language", true},
		{http.MethodGet, "/asr", false}, // GET is not work
		{http.MethodGet, "/", false},    // health/docs
		{http.MethodPost, "/", false},
	}
	for _, c := range cases {
		r, _ := http.NewRequest(c.method, "http://x"+c.path, nil)
		if got := isTranscription(r); got != c.want {
			t.Errorf("isTranscription(%s %s) = %v, want %v", c.method, c.path, got, c.want)
		}
	}
}

func TestProxyForwardsTransparentlyAndBrackets(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "yes")
		_, _ = io.WriteString(w, "ok:"+r.URL.Path)
	}))
	defer upstream.Close()

	var mu sync.Mutex
	var states []string
	h := &Handlers{
		Stdout: io.Discard, Stderr: io.Discard,
		GetEnv:   func(string) string { return "" },
		Debounce: 30 * time.Millisecond,
		Exec: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if len(args) >= 2 && args[1] == "list" {
				return []byte(`{"claims":[{"claim_id":"clm-w","owner_id":"whisper"}]}`), nil
			}
			if len(args) >= 2 && args[1] == "activity" {
				mu.Lock()
				for i, a := range args {
					if a == "--state" && i+1 < len(args) {
						states = append(states, args[i+1])
					}
				}
				mu.Unlock()
			}
			return []byte(`{}`), nil
		},
	}
	srv, err := h.server("127.0.0.1:0", strings.TrimPrefix(upstream.URL, "http://"), 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(srv.Handler)
	defer front.Close()

	// GET / is transparent and does NOT bracket.
	resp, err := http.Get(front.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "ok:/" || resp.Header.Get("X-Upstream") != "yes" {
		t.Fatalf("GET / not transparent: body=%q header=%q", body, resp.Header.Get("X-Upstream"))
	}
	if got := snapshot(&mu, &states); len(got) != 0 {
		t.Fatalf("GET / must not report activity; got %v", got)
	}

	// POST /asr is transparent AND brackets: active now, idle after debounce.
	resp, err = http.Post(front.URL+"/asr", "audio/wav", strings.NewReader("audio"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "ok:/asr" {
		t.Fatalf("POST /asr not forwarded: body=%q", body)
	}

	if !waitForState(&mu, &states, "active", time.Second) {
		t.Fatalf("active never reported; states=%v", snapshot(&mu, &states))
	}
	if !waitForState(&mu, &states, "idle", time.Second) {
		t.Fatalf("idle never reported after debounce; states=%v", snapshot(&mu, &states))
	}
}

func TestNativeAdapterMapsHistoricalASRContract(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inference" {
			t.Fatalf("native path=%q, want /inference", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse native form: %v", err)
		}
		if len(r.MultipartForm.File["file"]) != 1 || len(r.MultipartForm.File["audio_file"]) != 0 {
			t.Fatalf("native files=%v, want file only", r.MultipartForm.File)
		}
		if got := r.FormValue("response_format"); got != "verbose_json" {
			t.Fatalf("response_format=%q, want verbose_json", got)
		}
		if got := r.FormValue("language"); got != "en" {
			t.Fatalf("language=%q, want en from query", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"text":"hello","segments":[]}`)
	}))
	defer upstream.Close()

	h := &Handlers{
		Stdout: io.Discard, Stderr: io.Discard, Native: true,
		GetEnv: func(string) string { return "" },
		Exec:   func(context.Context, string, ...string) ([]byte, error) { return []byte(`{}`), nil },
	}
	srv, err := h.server("127.0.0.1:0", strings.TrimPrefix(upstream.URL, "http://"), 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(srv.Handler)
	defer front.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("audio_file", "speech.wav")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(part, "wav")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, front.URL+"/asr?output=json&language=en", &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", response.StatusCode)
	}
	data, _ := io.ReadAll(response.Body)
	if string(data) != `{"text":"hello","segments":[]}` {
		t.Fatalf("response=%q", data)
	}
}

func TestNativeAdapterMapsDetectLanguageResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inference" {
			t.Fatalf("native path=%q, want /inference", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"language":"english","detected_language":"english","detected_language_probability":0.97}`)
	}))
	defer upstream.Close()

	h := &Handlers{
		Stdout: io.Discard, Stderr: io.Discard, Native: true,
		GetEnv: func(string) string { return "" },
		Exec:   func(context.Context, string, ...string) ([]byte, error) { return []byte(`{}`), nil },
	}
	srv, err := h.server("127.0.0.1:0", strings.TrimPrefix(upstream.URL, "http://"), 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(srv.Handler)
	defer front.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("audio_file", "speech.wav")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(part, "wav")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, front.URL+"/detect-language", &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if got := payload["language_code"]; got != "en" {
		t.Fatalf("language_code=%v, want en", got)
	}
	if got := payload["confidence"]; got != 0.97 {
		t.Fatalf("confidence=%v, want 0.97", got)
	}
}

// Fail-open: when every capacity call errors, transcription still forwards.
func TestProxyFailOpenOnCapacityError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "transcribed")
	}))
	defer upstream.Close()

	h := &Handlers{
		Stdout: io.Discard, Stderr: io.Discard,
		GetEnv:   func(string) string { return "" },
		Debounce: 10 * time.Millisecond,
		Exec: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, context.DeadlineExceeded // broker down / vrooli missing
		},
	}
	srv, _ := h.server("127.0.0.1:0", strings.TrimPrefix(upstream.URL, "http://"), 10*time.Millisecond)
	front := httptest.NewServer(srv.Handler)
	defer front.Close()

	resp, err := http.Post(front.URL+"/asr", "audio/wav", strings.NewReader("audio"))
	if err != nil {
		t.Fatalf("request must succeed even when capacity is down: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "transcribed" {
		t.Fatalf("body=%q, want transcribed (fail-open)", body)
	}
}

func TestRunLogsSignalExit(t *testing.T) {
	sigCh := make(chan os.Signal, 1)
	sigCh <- syscall.SIGTERM
	var stderr bytes.Buffer
	h := &Handlers{
		Stdout:   io.Discard,
		Stderr:   &stderr,
		GetEnv:   func(string) string { return "" },
		Exec:     func(context.Context, string, ...string) ([]byte, error) { return []byte(`{}`), nil },
		SignalCh: sigCh,
	}
	if err := h.Run([]string{"--listen", "127.0.0.1:0", "--upstream", "127.0.0.1:1"}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	got := stderr.String()
	if !strings.Contains(got, "activity-edge exit: reason=signal") || !strings.Contains(got, "sig=terminated") {
		t.Fatalf("signal exit log = %q", got)
	}
}

func TestRunLogsServeError(t *testing.T) {
	var stderr bytes.Buffer
	h := &Handlers{
		Stdout: io.Discard,
		Stderr: &stderr,
		GetEnv: func(string) string { return "" },
		Exec:   func(context.Context, string, ...string) ([]byte, error) { return []byte(`{}`), nil },
		Listen: func(string, string) (net.Listener, error) {
			return failingListener{err: errors.New("accept boom")}, nil
		},
		SignalCh: make(chan os.Signal),
	}
	if err := h.Run([]string{"--listen", "127.0.0.1:0", "--upstream", "127.0.0.1:1"}); err == nil {
		t.Fatal("Run should return the serve error")
	}
	got := stderr.String()
	if !strings.Contains(got, "activity-edge exit: reason=serve-error") || !strings.Contains(got, "accept boom") {
		t.Fatalf("serve-error log = %q", got)
	}
}

func TestHeartbeatFormatting(t *testing.T) {
	var stdout bytes.Buffer
	h := &Handlers{Stdout: &stdout}
	tr := &tracker{requests: 3, active: true}
	since := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	h.logHeartbeat(tr, since)

	want := "activity-edge alive: requests=3 active=true since=2026-06-24T12:00:00Z\n"
	if got := stdout.String(); got != want {
		t.Fatalf("heartbeat = %q, want %q", got, want)
	}
}

func snapshot(mu *sync.Mutex, states *[]string) []string {
	mu.Lock()
	defer mu.Unlock()
	return append([]string(nil), (*states)...)
}

func waitForState(mu *sync.Mutex, states *[]string, want string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, s := range snapshot(mu, states) {
			if s == want {
				return true
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

type failingListener struct {
	err error
}

func (l failingListener) Accept() (net.Conn, error) { return nil, l.err }
func (l failingListener) Close() error              { return nil }
func (l failingListener) Addr() net.Addr            { return testAddr("127.0.0.1:0") }

type testAddr string

func (a testAddr) Network() string { return "tcp" }
func (a testAddr) String() string  { return string(a) }
