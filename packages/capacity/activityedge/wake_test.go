package activityedge

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestWakeRoundTripperWakesOnlyWorkAfterConnectionRefused(t *testing.T) {
	var calls, wakes atomic.Int32
	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	roundTripper := wakeRoundTripper{Base: base, Wake: func(context.Context) error { wakes.Add(1); return nil }}
	request := httptest.NewRequest(http.MethodPost, "http://whisper/asr", nil)
	request = request.WithContext(context.WithValue(request.Context(), wakeRequestKey{}, true))
	response, err := roundTripper.RoundTrip(request)
	if err != nil || response == nil || response.StatusCode != http.StatusOK || calls.Load() != 2 || wakes.Load() != 1 {
		t.Fatalf("response=%v err=%v calls=%d wakes=%d", response, err, calls.Load(), wakes.Load())
	}
}

func TestInflightRefusalDoesNotWakeWhileDrainOwnsLifecycle(t *testing.T) {
	tr := &tracker{drained: make(chan struct{})}
	close(tr.drained)
	if !tr.begin() {
		t.Fatal("tracker refused initial request")
	}
	var starts, canceled atomic.Int32
	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	})
	permitAcquired := make(chan struct{})
	returnPermit := make(chan struct{})
	roundTripper := wakeRoundTripper{
		Base: base,
		Wake: func(ctx context.Context) error {
			if ctx.Err() != nil {
				canceled.Add(1)
				return ctx.Err()
			}
			starts.Add(1)
			return nil
		},
		WakeContext: func(parent context.Context) (context.Context, func(), bool) {
			ctx, release, allowed := tr.beginWake(parent)
			close(permitAcquired)
			<-returnPermit
			return ctx, release, allowed
		},
	}
	request := httptest.NewRequest(http.MethodPost, "http://whisper/asr", nil)
	request = request.WithContext(context.WithValue(request.Context(), wakeRequestKey{}, true))
	roundTripDone := make(chan error, 1)
	go func() {
		_, err := roundTripper.RoundTrip(request)
		tr.end()
		roundTripDone <- err
	}()
	<-permitAcquired
	drainDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		drainDone <- tr.drain(ctx)
	}()
	deadline := time.Now().Add(time.Second)
	for !tr.isDraining() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !tr.isDraining() {
		t.Fatal("drain did not close the wake gate")
	}
	close(returnPermit)
	if err := <-roundTripDone; err == nil {
		t.Fatal("drained wake should return its cancellation")
	}
	if err := <-drainDone; err != nil {
		t.Fatalf("drain after refused request: %v", err)
	}
	if starts.Load() != 0 || canceled.Load() != 1 {
		t.Fatalf("wake starts=%d canceled permits=%d, want no start and one cancellation", starts.Load(), canceled.Load())
	}
}

func TestWakeRoundTripperReplaysAdaptedWhisperMultipartBody(t *testing.T) {
	var publicBody bytes.Buffer
	writer := multipart.NewWriter(&publicBody)
	part, err := writer.CreateFormFile("audio_file", "speech.wav")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, "wav-audio-bytes"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://whisper/asr?language=en", &publicBody)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	adapted, err := adaptNativeRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	want, err := adapted.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := io.ReadAll(want)
	_ = want.Close()
	if err != nil {
		t.Fatal(err)
	}
	adapted = adapted.WithContext(context.WithValue(adapted.Context(), wakeRequestKey{}, true))
	var calls, wakes atomic.Int32
	base := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
		}
		got, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(got, wantBytes) {
			return nil, errors.New("retry body differs from original adapted audio")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	rt := wakeRoundTripper{Base: base, Wake: func(context.Context) error { wakes.Add(1); return nil }}
	response, err := rt.RoundTrip(adapted)
	if err != nil || response == nil || response.StatusCode != http.StatusOK || calls.Load() != 2 || wakes.Load() != 1 {
		t.Fatalf("response=%v err=%v calls=%d wakes=%d", response, err, calls.Load(), wakes.Load())
	}
}

func TestWakeRoundTripperRefusesToReplayNonRewindableBody(t *testing.T) {
	var calls, wakes atomic.Int32
	request := httptest.NewRequest(http.MethodPost, "http://whisper/asr", strings.NewReader("audio"))
	request.GetBody = nil
	request = request.WithContext(context.WithValue(request.Context(), wakeRequestKey{}, true))
	rt := wakeRoundTripper{
		Base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}),
		Wake: func(context.Context) error { wakes.Add(1); return nil },
	}
	if _, err := rt.RoundTrip(request); err == nil || calls.Load() != 1 || wakes.Load() != 1 {
		t.Fatalf("non-replayable body result err=%v calls=%d wakes=%d", err, calls.Load(), wakes.Load())
	}
}

func TestMatchedDemandRecoversDrainedEdgeThroughWakeOwner(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/asr" {
			t.Fatalf("upstream path=%q, want /asr", r.URL.Path)
		}
		_, _ = io.WriteString(w, "transcribed")
	}))
	defer upstream.Close()
	var wakes atomic.Int32
	var tracker *tracker
	h := &Handlers{
		Resource: "whisper",
		Match:    func(r *http.Request) bool { return r.Method == http.MethodPost && r.URL.Path == "/asr" },
		Wake:     func(context.Context) error { wakes.Add(1); tracker.resume(); return nil },
	}
	srv, trackerValue, err := h.serverWithTrackerMode("127.0.0.1:0", strings.TrimPrefix(upstream.URL, "http://"), 0, false)
	if err != nil {
		t.Fatal(err)
	}
	tracker = trackerValue
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := tracker.drain(ctx); err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(srv.Handler)
	defer front.Close()
	response, err := http.Post(front.URL+"/asr", "application/octet-stream", strings.NewReader("audio"))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || string(body) != "transcribed" || wakes.Load() != 1 {
		t.Fatalf("response=%d/%q wakes=%d", response.StatusCode, body, wakes.Load())
	}
}

func TestWakeRoundTripperDoesNotWakeProbesOrAmbiguousFailures(t *testing.T) {
	for _, tc := range []struct {
		name    string
		work    bool
		failure error
	}{
		{name: "probe", failure: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}},
		{name: "uncertain request result", work: true, failure: errors.New("connection reset after write")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var wakes atomic.Int32
			request := httptest.NewRequest(http.MethodGet, "http://whisper/", nil)
			if tc.work {
				request = request.WithContext(context.WithValue(request.Context(), wakeRequestKey{}, true))
			}
			rt := wakeRoundTripper{Base: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, tc.failure }), Wake: func(context.Context) error { wakes.Add(1); return nil }}
			if _, err := rt.RoundTrip(request); err == nil {
				t.Fatal("expected upstream error")
			}
			if wakes.Load() != 0 {
				t.Fatalf("wake count=%d, want 0", wakes.Load())
			}
		})
	}
}
