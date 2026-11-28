package capacitysync

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/packages/capacity/companion"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
)

// Feature: ollama reports its loaded footprint, and nothing else
//
//	As the ollama CLI
//	I want to implement only the /api/ps observation
//	So that the claim, resize, release and heartbeat logic is the same code
//	every other accelerated resource runs, fixed once rather than three times.

type fakePS struct {
	running []ensure.RunningModel
	err     error
}

func (f fakePS) ListRunning(context.Context) ([]ensure.RunningModel, error) {
	return f.running, f.err
}

func handlersFor(ps psClient) *Handlers {
	return &Handlers{
		GetEnv:    func(string) string { return "" },
		NewClient: func() psClient { return ps },
		Exec:      func(context.Context, string, ...string) ([]byte, error) { return []byte(`{}`), nil },
	}
}

// Scenario: the observed footprint is the sum of the loaded models.
func TestObserveSumsLoadedModelFootprints(t *testing.T) {
	// Given two models resident on the device
	h := handlersFor(fakePS{running: []ensure.RunningModel{
		{Name: "qwen3.5:9b", SizeVRAM: 8 << 30},
		{Name: "nomic-embed-text", SizeVRAM: 1 << 30},
	}})

	// When the observer looks
	footprint, err := h.Observe(context.Background())
	// Then it reports their total
	if err != nil {
		t.Fatalf("Observe() = %v, want nil", err)
	}
	if want := int64(9) << 30; footprint.Bytes != want {
		t.Fatalf("Bytes = %d, want %d", footprint.Bytes, want)
	}
}

// Scenario: nothing loaded is an honest zero.
func TestObserveReportsZeroWhenNothingIsLoaded(t *testing.T) {
	// Given no resident models
	h := handlersFor(fakePS{})

	// When the observer looks
	footprint, err := h.Observe(context.Background())

	// Then zero is reported, which the shared loop turns into a release
	if err != nil || footprint.Bytes != 0 {
		t.Fatalf("Observe() = %+v, %v; want an empty footprint and no error", footprint, err)
	}
}

// Scenario: an unreachable ollama is an error, never a zero.
//
// "Everything unloaded" and "cannot reach ollama" look identical from a zero,
// and only one of them should release a live reservation.
func TestObserveSurfacesAPollFailureRatherThanReportingZero(t *testing.T) {
	// Given ollama's API unavailable
	h := handlersFor(fakePS{err: context.DeadlineExceeded})

	// When the observer looks
	_, err := h.Observe(context.Background())

	// Then the failure surfaces
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Observe() = %v, want the poll failure", err)
	}
}

// Scenario: ollama declares only its observation. The claim's tier, ladder and
// idle policy come from resource.json through the shared companion, so there is
// no per-resource copy to fall out of step with the manifest.
func TestConfigDeclaresOnlyTheObservation(t *testing.T) {
	// Given the default handlers
	h := handlersFor(fakePS{})

	// When the companion declaration is built
	cfg := h.config()

	// Then it names ollama and carries no claim policy of its own
	if cfg.Resource != resourceName {
		t.Fatalf("resource = %q, want %q", cfg.Resource, resourceName)
	}
	if cfg.Priority != "" || cfg.YieldWhenIdle || cfg.IdleGrace != 0 || cfg.IdleUnloadTTL != 0 || cfg.PreferredBytes != 0 || cfg.FloorBytes != 0 || cfg.Profile != "" {
		t.Fatalf("config = %+v, want the claim policy left to resource.json", cfg)
	}
	// And ollama satisfies the shared observer contract
	var _ companion.Observer = h
}
