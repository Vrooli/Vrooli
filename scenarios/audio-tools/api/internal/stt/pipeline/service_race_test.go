package pipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"audio-tools/internal/audioformat"
)

// TestServiceDependencySettersAreRaceFree pins the mutable-dependency contract:
// a request may run while a test/operator seam updates the URL, transport,
// engine, recovery policy, or concurrency bound. Before the depMu snapshot,
// these reads raced the setters under -race.
func TestServiceDependencySettersAreRaceFree(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"text":"ok"}`))
	}))
	defer srv.Close()

	whisperURL := srv.URL + "/asr?output=json"
	svc := newServiceForTest()
	svc.SetWhisperURL(whisperURL)
	svc.SetHTTPClient(srv.Client())

	engineA := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))
	engineB := audioformat.New(audioformat.WithFfmpegProbe(func() bool { return false }))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			svc.SetWhisperURL(whisperURL)
			svc.SetHTTPClient(srv.Client())
			if i%2 == 0 {
				svc.SetEngine(engineA)
			} else {
				svc.SetEngine(engineB)
			}
			svc.SetWhisperConcurrency(4)
			svc.SetAutoEnsure(i%2 == 0)
		}
	}()

	for i := 0; i < 100; i++ {
		res, err := svc.Transcribe(context.Background(), []byte{0x00, 0x00}, "pcm_s16le", "", "", false)
		if err != nil {
			t.Fatalf("transcribe %d: %v", i, err)
		}
		if res.Text != "ok" {
			t.Fatalf("transcribe %d: text = %q", i, res.Text)
		}
	}
	wg.Wait()
}
