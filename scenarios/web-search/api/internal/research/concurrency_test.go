package research_test

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"web-search/internal/research"
)

type boundedFetcher struct{ active, peak atomic.Int32 }

type countingFetcher struct{ calls atomic.Int32 }

type fetcherFunc func(context.Context, string) (string, error)

func (f fetcherFunc) Fetch(ctx context.Context, url string) (string, error) { return f(ctx, url) }

func (f *countingFetcher) Fetch(context.Context, string) (string, error) {
	f.calls.Add(1)
	return "page text", nil
}

func (f *boundedFetcher) Fetch(ctx context.Context, url string) (string, error) {
	current := f.active.Add(1)
	for {
		old := f.peak.Load()
		if current <= old || f.peak.CompareAndSwap(old, current) {
			break
		}
	}
	defer f.active.Add(-1)
	select {
	case <-time.After(10 * time.Millisecond):
		return "page text", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestL2FetchesWithConfiguredBoundAndPreservesFailures(t *testing.T) {
	fetcher := &boundedFetcher{}
	searcher := &fakeSearcher{candidates: []research.Candidate{{URL: "https://example.test/1"}, {URL: "https://example.test/2"}, {URL: "https://example.test/3"}, {URL: "https://example.test/4"}}}
	svc := research.NewService(research.Deps{Searcher: searcher, Fetcher: fetcher, Synthesizer: &fakeSynthesizer{result: research.Synthesis{Text: "answer", Citations: []research.Citation{{ResultIndex: 0, URL: "https://example.test/1"}}}}, FetchConcurrency: 2})
	out, err := svc.RunL2(context.Background(), "q", 4, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := fetcher.peak.Load(); got > 2 {
		t.Fatalf("peak fetch concurrency = %d, want <= 2", got)
	}
	if len(out.FetchFailures) != 0 {
		t.Fatalf("unexpected failures: %+v", out.FetchFailures)
	}
}

func TestControlledConcurrencyReducesMultiHostWaitTime(t *testing.T) {
	searcher := &fakeSearcher{candidates: []research.Candidate{
		{URL: "https://one.example/1"}, {URL: "https://two.example/2"},
		{URL: "https://three.example/3"}, {URL: "https://four.example/4"},
	}}
	run := func(concurrency int) time.Duration {
		started := time.Now()
		_, err := research.NewService(research.Deps{
			Searcher: searcher, Fetcher: &boundedFetcher{},
			Synthesizer:      &fakeSynthesizer{result: research.Synthesis{Text: "answer", Citations: []research.Citation{{ResultIndex: 0, URL: "https://one.example/1"}}}},
			FetchConcurrency: concurrency,
		}).RunL2(context.Background(), "q", 4, false)
		if err != nil {
			t.Fatal(err)
		}
		return time.Since(started)
	}
	serial := run(1)
	parallel := run(2)
	t.Logf("controlled multi-host wait: serial=%s parallel=%s", serial, parallel)
	if parallel >= serial {
		t.Fatalf("bounded concurrency did not reduce controlled wait: serial=%s parallel=%s", serial, parallel)
	}
}

func TestL2EnforcesPerHostConcurrency(t *testing.T) {
	type hostFetcher struct {
		active, peak atomic.Int32
	}
	fetcher := &hostFetcher{}
	// All URLs share a host, so the per-host limit is observable even when the
	// global limit allows four workers.
	searcher := &fakeSearcher{candidates: []research.Candidate{
		{URL: "https://same.example/1"}, {URL: "https://same.example/2"},
		{URL: "https://same.example/3"}, {URL: "https://same.example/4"},
	}}
	_, err := research.NewService(research.Deps{
		Searcher: searcher,
		Fetcher: fetcherFunc(func(ctx context.Context, _ string) (string, error) {
			current := fetcher.active.Add(1)
			for {
				old := fetcher.peak.Load()
				if current <= old || fetcher.peak.CompareAndSwap(old, current) {
					break
				}
			}
			defer fetcher.active.Add(-1)
			select {
			case <-time.After(5 * time.Millisecond):
				return "page text", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}),
		FetchConcurrency: 4, FetchPerHostConcurrency: 1,
		Synthesizer: &fakeSynthesizer{result: research.Synthesis{Text: "answer", Citations: []research.Citation{{ResultIndex: 0}}}},
	}).RunL2(context.Background(), "q", 4, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := fetcher.peak.Load(); got != 1 {
		t.Fatalf("same-host peak fetch concurrency = %d, want 1", got)
	}
}

func TestL2CancellationBeforeQueuedFetchMakesNoNetworkRequest(t *testing.T) {
	fetcher := &countingFetcher{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := research.NewService(research.Deps{
		Searcher: &fakeSearcher{candidates: []research.Candidate{{URL: "https://cancel.example/1"}, {URL: "https://cancel.example/2"}}},
		Fetcher:  fetcher, FetchConcurrency: 1,
		Synthesizer: &fakeSynthesizer{result: research.Synthesis{Text: strings.Repeat("answer ", 4)}},
	}).RunL2(ctx, "q", 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := fetcher.calls.Load(); got != 0 {
		t.Fatalf("queued cancellation invoked %d network fetches", got)
	}
}

func TestL2RepeatedCancellationReleasesQueuedPermits(t *testing.T) {
	for run := 0; run < 20; run++ {
		var calls atomic.Int32
		started := make(chan struct{})
		fetcher := fetcherFunc(func(ctx context.Context, _ string) (string, error) {
			calls.Add(1)
			select {
			case <-started:
			default:
				close(started)
			}
			<-ctx.Done()
			return "", ctx.Err()
		})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, _ = research.NewService(research.Deps{
				Searcher: &fakeSearcher{candidates: []research.Candidate{
					{URL: "https://cancel.example/1"}, {URL: "https://cancel.example/2"},
					{URL: "https://cancel.example/3"}, {URL: "https://cancel.example/4"},
				}},
				Fetcher: fetcher, FetchConcurrency: 1,
				Synthesizer: &fakeSynthesizer{result: research.Synthesis{Text: "unused"}},
			}).RunL2(ctx, "q", 4, false)
		}()
		select {
		case <-started:
		case <-time.After(time.Second):
			cancel()
			t.Fatal("first fetch did not start")
		}
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatalf("cancellation run %d did not release queued work", run)
		}
		if got := calls.Load(); got != 1 {
			t.Fatalf("cancellation run %d admitted %d fetches, want 1", run, got)
		}
	}
}

func TestL2TimeoutPreservesCompletedSources(t *testing.T) {
	searcher := &fakeSearcher{candidates: []research.Candidate{
		{URL: "https://fast.example", Title: "Fast"},
		{URL: "https://slow.example", Title: "Slow"},
	}}
	fetcher := fetcherFunc(func(ctx context.Context, rawURL string) (string, error) {
		if rawURL == "https://fast.example" {
			return "completed source", nil
		}
		<-ctx.Done()
		return "", ctx.Err()
	})
	syn := &fakeSynthesizer{result: research.Synthesis{Text: "from completed source", Citations: []research.Citation{{ResultIndex: 0}}}}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	out, err := research.NewService(research.Deps{
		Searcher: searcher, Fetcher: fetcher, Synthesizer: syn, FetchConcurrency: 2,
	}).RunL2(ctx, "q", 2, false)
	require.NoError(t, err)
	require.Len(t, syn.gotDocs, 1)
	require.Equal(t, "https://fast.example", syn.gotDocs[0].URL)
	require.Len(t, out.FetchFailures, 1)
	require.Equal(t, "https://slow.example", out.FetchFailures[0].URL)
}
