package perfbudget

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Harness drives read-only HTTP load and samples the management process
// through the four phases. Paths are relative to the base URL and must be
// GET endpoints: the harness never sends a body and never uses another
// method, so it cannot mutate a target by construction.
type Harness struct {
	// Profile names the fixture profile the run is measured against.
	Profile string
	// Paths are the read endpoints (already resolved, no {id} placeholders).
	Paths []string
	// Concurrency is the number of concurrent readers in the sustained
	// phase; the peak phase doubles it. Zero means budgets reader_concurrency.
	Concurrency int
	// Sustained and Peak are wall-clock windows. Zero means 1s each (tests).
	Sustained time.Duration
	Peak      time.Duration
	// Baseline and PostCleanup are quiet windows before and after load.
	// Zero means 50ms (tests).
	Baseline    time.Duration
	PostCleanup time.Duration
	// Headers are sent with every request (bearer for a live target).
	Headers http.Header
	// MinSamples is the percentile threshold; zero means the budgets value.
	MinSamples int
	// Client overrides the HTTP client for url targets.
	Client *http.Client
}

// Target is where the harness sends requests. Exactly one of Handler or
// BaseURL is set.
type Target struct {
	Handler http.Handler
	BaseURL string
	// Explicit records that BaseURL came from an operator-supplied
	// --target argument. RunURL refuses a non-explicit target.
	Explicit bool
}

// ErrTargetRequired is returned when a url run has no explicit target.
var ErrTargetRequired = errors.New("perfbudget: an explicit --target base URL is required; there is no default target")

// TargetFromFlag builds an explicit url target from the --target value.
// Empty is refused, as is anything that is not an absolute http(s) URL.
func TargetFromFlag(value string) (Target, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Target{}, ErrTargetRequired
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Target{}, fmt.Errorf("perfbudget: --target %q is not an absolute http(s) URL", value)
	}
	return Target{BaseURL: strings.TrimRight(u.String(), "/"), Explicit: true}, nil
}

// Run executes the four phases against the target and returns a complete
// Measurement. For a url target without Explicit it returns
// ErrTargetRequired before sending anything.
func (h Harness) Run(ctx context.Context, target Target, b *Budgets) (*Measurement, error) {
	if b == nil {
		return nil, fmt.Errorf("budgets are required")
	}
	if _, ok := b.Profile(h.Profile); !ok {
		return nil, fmt.Errorf("profile %q has no frozen budget", h.Profile)
	}
	if len(h.Paths) == 0 {
		return nil, fmt.Errorf("no read paths configured")
	}
	for _, p := range h.Paths {
		if strings.Contains(p, "{") {
			return nil, fmt.Errorf("path %q still carries a placeholder", p)
		}
	}
	h = h.withDefaults(b)

	desc := TargetDescriptor{}
	var base string
	var client *http.Client
	switch {
	case target.Handler != nil:
		srv := httptest.NewServer(target.Handler)
		defer srv.Close()
		base = srv.URL
		client = srv.Client()
		desc = TargetDescriptor{Kind: "handler", Explicit: true}
	case target.BaseURL != "":
		if !target.Explicit {
			return nil, ErrTargetRequired
		}
		base = strings.TrimRight(target.BaseURL, "/")
		client = h.Client
		if client == nil {
			client = &http.Client{Timeout: 30 * time.Second}
		}
		desc = TargetDescriptor{Kind: "url", BaseURL: base, Explicit: true}
	default:
		return nil, ErrTargetRequired
	}

	m := &Measurement{SchemaVersion: 1, Profile: h.Profile, Target: desc, StartedAt: time.Now().UTC(), Concurrency: h.Concurrency, Phases: map[string]*Phase{}}

	quiet := func(name string, d time.Duration) error {
		ph := &Phase{Name: name, StartedAt: time.Now().UTC(), ProcessAt: SnapshotProcess()}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
		runtime.GC()
		ph.Latency = Summarise(nil, 0, h.MinSamples)
		ph.Process = SnapshotProcess()
		ph.FinishedAt = time.Now().UTC()
		m.Phases[name] = ph
		return nil
	}
	load := func(name string, concurrency int, d time.Duration) error {
		ph := &Phase{Name: name, StartedAt: time.Now().UTC(), ProcessAt: SnapshotProcess()}
		lat, errs, err := h.drive(ctx, client, base, concurrency, d)
		if err != nil {
			return err
		}
		ph.Latency = Summarise(lat, errs, h.MinSamples)
		ph.Process = SnapshotProcess()
		ph.FinishedAt = time.Now().UTC()
		m.Phases[name] = ph
		return nil
	}

	if err := quiet(PhaseBaseline, h.Baseline); err != nil {
		return nil, err
	}
	if err := load(PhaseSustained, h.Concurrency, h.Sustained); err != nil {
		return nil, err
	}
	if err := load(PhasePeak, h.Concurrency*2, h.Peak); err != nil {
		return nil, err
	}
	if err := quiet(PhasePostCleanup, h.PostCleanup); err != nil {
		return nil, err
	}
	m.FinishedAt = time.Now().UTC()
	return m, nil
}

func (h Harness) withDefaults(b *Budgets) Harness {
	if h.Concurrency <= 0 {
		h.Concurrency = b.Phase23.ReaderConcurrency
	}
	if h.MinSamples <= 0 {
		h.MinSamples = b.Phase23.MinSamplesForPercentiles
	}
	if h.Sustained <= 0 {
		h.Sustained = time.Second
	}
	if h.Peak <= 0 {
		h.Peak = time.Second
	}
	if h.Baseline <= 0 {
		h.Baseline = 50 * time.Millisecond
	}
	if h.PostCleanup <= 0 {
		h.PostCleanup = 50 * time.Millisecond
	}
	return h
}

// drive runs concurrency readers round-robin over the paths for d and
// returns every latency plus the error count. A response with status >= 400
// or a transport error is an error; its latency is still recorded.
func (h Harness) drive(ctx context.Context, client *http.Client, base string, concurrency int, d time.Duration) ([]time.Duration, int, error) {
	deadline, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	var mu sync.Mutex
	var latencies []time.Duration
	errs := 0
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			i := offset
			for {
				if deadline.Err() != nil {
					return
				}
				path := h.Paths[i%len(h.Paths)]
				i++
				start := time.Now()
				failed := h.get(deadline, client, base+path)
				elapsed := time.Since(start)
				if deadline.Err() != nil && failed {
					// The deadline cut this request; do not count a
					// cancelled request as an error of the target.
					return
				}
				mu.Lock()
				latencies = append(latencies, elapsed)
				if failed {
					errs++
				}
				mu.Unlock()
			}
		}(worker)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return nil, 0, ctx.Err()
	}
	return latencies, errs, nil
}

func (h Harness) get(ctx context.Context, client *http.Client, u string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return true
	}
	for k, vs := range h.Headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return true
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode >= 400
}

// Args is the parsed command line of a live staging pass. It exists so the
// --target guard is testable without a binary: Parse refuses an empty
// target, so nothing can point the harness at a default host.
type Args struct {
	Target      Target
	Budgets     string
	Profile     string
	Concurrency int
	Sustained   time.Duration
	Peak        time.Duration
	Paths       []string
	Bearer      string
}

// ParseArgs parses a staging pass command line. --target and --profile are
// required; --path may repeat.
func ParseArgs(argv []string) (Args, error) {
	fs := flag.NewFlagSet("perfbudget", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var a Args
	var target string
	var paths pathList
	fs.StringVar(&target, "target", "", "explicit base URL of the management API under measurement (required; no default)")
	fs.StringVar(&a.Budgets, "budgets", "certification/budgets.json", "budgets file")
	fs.StringVar(&a.Profile, "profile", "", "fixture profile (required)")
	fs.IntVar(&a.Concurrency, "concurrency", 0, "sustained readers (default: budgets reader_concurrency)")
	fs.DurationVar(&a.Sustained, "sustained", 60*time.Second, "sustained window")
	fs.DurationVar(&a.Peak, "peak", 30*time.Second, "peak window")
	fs.Var(&paths, "path", "read path (repeatable)")
	fs.StringVar(&a.Bearer, "bearer-env", "", "environment variable holding the bearer token (the value is never passed on argv)")
	if err := fs.Parse(argv); err != nil {
		return Args{}, err
	}
	t, err := TargetFromFlag(target)
	if err != nil {
		return Args{}, err
	}
	a.Target = t
	if strings.TrimSpace(a.Profile) == "" {
		return Args{}, fmt.Errorf("perfbudget: --profile is required")
	}
	a.Paths = paths
	return a, nil
}

type pathList []string

func (p *pathList) String() string     { return strings.Join(*p, ",") }
func (p *pathList) Set(v string) error { *p = append(*p, v); return nil }
