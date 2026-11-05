package pipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
)

// TestWhisperConcurrencyBound proves the semaphore caps concurrent Whisper
// calls at DefaultWhisperConcurrency: with N (>cap) simultaneous callers,
// the in-flight count never exceeds the cap, every call still succeeds
// (queue with backpressure, never error), and all complete.
func TestWhisperConcurrencyBound(t *testing.T) {
	var inFlight, maxObserved int64
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		cur := atomic.AddInt64(&inFlight, 1)
		for {
			m := atomic.LoadInt64(&maxObserved)
			if cur <= m || atomic.CompareAndSwapInt64(&maxObserved, m, cur) {
				break
			}
		}
		<-release // hold the request open so concurrency is observable
		atomic.AddInt64(&inFlight, -1)
		_, _ = w.Write([]byte(`{"text":"ok"}`))
	}))
	defer srv.Close()

	engine := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	svc := NewService(Config{}, "", nil, "", SpeakerConfig{}, "", nil, nil,
		&atomic.Int64{}, srv.URL+"/asr",
		&http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}, engine)

	const callers = DefaultWhisperConcurrency + 4
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// PCM input → WAV wrap, no ffmpeg needed.
			_, err := svc.Transcribe(context.Background(), []byte{0x01, 0x00}, "pcm_s16le", "", "", false)
			errs <- err
		}()
	}

	// Give the goroutines time to saturate the semaphore, then let them go.
	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&maxObserved) >= int64(DefaultWhisperConcurrency)
	}, 2*time.Second, 5*time.Millisecond, "expected the cap to be reached")
	close(release)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err, "queued calls must succeed, never error")
	}
	require.LessOrEqual(t, atomic.LoadInt64(&maxObserved), int64(DefaultWhisperConcurrency),
		"concurrent Whisper calls must never exceed the cap")
}
