package procsampler

import (
	"context"
	"testing"
	"time"
)

type countingSampler struct {
	calls   int
	samples []ProcessSample
}

func (s *countingSampler) Sample(context.Context) ([]ProcessSample, error) {
	s.calls++
	return cloneSamples(s.samples), nil
}

func TestCachedSamplerSharesFreshSampleAndDefendsCopies(t *testing.T) {
	base := &countingSampler{samples: []ProcessSample{{PID: 10, Comm: "worker", CPUPct: 12}}}
	now := time.Unix(100, 0)
	sampler := NewCachedSampler(base, time.Second)
	sampler.now = func() time.Time { return now }

	first, err := sampler.Sample(context.Background())
	if err != nil {
		t.Fatalf("first sample: %v", err)
	}
	first[0].Comm = "mutated"

	second, err := sampler.Sample(context.Background())
	if err != nil {
		t.Fatalf("second sample: %v", err)
	}
	if base.calls != 1 {
		t.Fatalf("base calls = %d, want 1", base.calls)
	}
	if got := second[0].Comm; got != "worker" {
		t.Fatalf("cached sample was externally mutated: %q", got)
	}

	now = now.Add(2 * time.Second)
	if _, err := sampler.Sample(context.Background()); err != nil {
		t.Fatalf("third sample: %v", err)
	}
	if base.calls != 2 {
		t.Fatalf("base calls after expiry = %d, want 2", base.calls)
	}
}
