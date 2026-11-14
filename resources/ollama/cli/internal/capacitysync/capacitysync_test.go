package capacitysync

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

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

// Scenario: the declared ladder comes from the model policy, not from the
// currently loaded set.
func TestConfigDeclaresTheModelPolicyLadder(t *testing.T) {
	// Given the default handlers
	h := handlersFor(fakePS{})

	// When the companion declaration is built
	cfg := h.config()

	// Then it names ollama at the service tier, yielding when idle
	if cfg.Resource != resourceName || cfg.Priority != "service" || !cfg.YieldWhenIdle {
		t.Fatalf("config = %+v, want ollama/service/yield-when-idle", cfg)
	}
	// And it carries a degrade ladder that ends at the floor
	var profile degradeProfile
	if err := json.Unmarshal([]byte(cfg.Profile), &profile); err != nil {
		t.Fatalf("profile is not valid JSON: %v", err)
	}
	if len(profile.Steps) == 0 {
		t.Fatal("the declared profile has no steps; the broker could never ask ollama to step down")
	}
	if profile.Steps[0].AmountBytes != cfg.PreferredBytes {
		t.Fatalf("first step = %d, want it to equal preferred %d", profile.Steps[0].AmountBytes, cfg.PreferredBytes)
	}
	if profile.Steps[len(profile.Steps)-1].AmountBytes != cfg.FloorBytes {
		t.Fatalf("last step = %d, want it to equal floor %d", profile.Steps[len(profile.Steps)-1].AmountBytes, cfg.FloorBytes)
	}
	// And the apply verb is the shared one, so the broker calls every resource
	// the same way
	if profile.Apply.Verb != "capacity" {
		t.Fatalf("apply verb = %q, want capacity", profile.Apply.Verb)
	}
	// And ollama satisfies the shared observer contract
	var _ companion.Observer = h
}

// Scenario: an environment override changes the poll cadence.
func TestConfigHonoursTheIntervalOverride(t *testing.T) {
	// Given an operator-set interval
	h := handlersFor(fakePS{})
	h.GetEnv = func(key string) string {
		if key == intervalEnv {
			return "90s"
		}
		return ""
	}

	// When the declaration is built
	// Then the override wins over the default
	if got := h.config().Interval; got != 90*time.Second {
		t.Fatalf("Interval = %s, want 90s", got)
	}
}
