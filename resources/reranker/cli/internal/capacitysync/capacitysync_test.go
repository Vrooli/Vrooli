package capacitysync

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/capacity/companion"
)

// Feature: the reranker reports what it holds, and nothing else
//
//	As the reranker CLI
//	I want to implement only the observation
//	So that the claim, resize, release and heartbeat logic is the same code
//	every other accelerated resource runs.

func handlersWith(output string, err error) (*Handlers, *[]string) {
	var calls []string
	h := &Handlers{
		Exec: func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, name+" "+strings.Join(args, " "))
			if err != nil {
				return nil, err
			}
			return []byte(output), nil
		},
	}
	return h, &calls
}

// Scenario: the observed figure wins over the claimed one.
func TestObserveReportsTheObservedFootprint(t *testing.T) {
	// Given a claim the host has measured
	h, _ := handlersWith(`{"claims":[{"owner_id":"reranker","amount_bytes":1447034880,"observed_bytes":1380000000}]}`, nil)

	// When the observer looks
	footprint, err := h.Observe(context.Background())
	// Then it reports what the host measured, not what was reserved: a
	// reservation is a request, a measurement is a fact
	if err != nil {
		t.Fatalf("Observe() = %v, want nil", err)
	}
	if footprint.Bytes != 1380000000 {
		t.Fatalf("Bytes = %d, want the observed 1380000000", footprint.Bytes)
	}
}

// Scenario: with no measurement yet, the claim keeps its own size.
func TestObserveFallsBackToTheClaimedAmount(t *testing.T) {
	// Given a claim the host has not measured yet
	h, _ := handlersWith(`{"claims":[{"owner_id":"reranker","amount_bytes":1447034880,"observed_bytes":0}]}`, nil)

	// When the observer looks
	footprint, err := h.Observe(context.Background())

	// Then the reservation is carried through unchanged rather than a
	// measurement being invented
	if err != nil || footprint.Bytes != 1447034880 {
		t.Fatalf("Observe() = %+v, %v; want the claimed amount", footprint, err)
	}
}

// Scenario: a missing claim re-admits through the control plane.
//
// The lifecycle admission path owns the manifest-derived reservation ladder.
// Reconstructing it here would make the companion a second source of policy.
func TestObserveReadmitsAMissingClaimThroughTheControlPlane(t *testing.T) {
	// Given no active claim
	h, calls := handlersWith(`{"claims":[]}`, nil)

	// When the observer looks
	footprint, err := h.Observe(context.Background())

	// Then it reports nothing held and asks the control plane to re-admit
	if err != nil || footprint.Bytes != 0 {
		t.Fatalf("Observe() = %+v, %v; want an empty footprint", footprint, err)
	}
	joined := strings.Join(*calls, " | ")
	if !strings.Contains(joined, "resource start reranker --json") {
		t.Fatalf("calls = %v, want control-plane re-admission", *calls)
	}
}

// Scenario: an unreadable ledger is an error, never an empty footprint.
func TestObserveSurfacesALedgerError(t *testing.T) {
	// Given a ledger that cannot be read
	h, _ := handlersWith("", errors.New("broker unavailable"))

	// When the observer looks
	_, err := h.Observe(context.Background())

	// Then the error surfaces, so the shared loop leaves the ledger alone
	// rather than releasing a live reservation
	if err == nil {
		t.Fatal("Observe() = nil error; an unreadable ledger must not read as zero held")
	}
}

// Scenario: the companion declaration names the reranker and its tier.
func TestConfigDeclaresTheResource(t *testing.T) {
	h := Default()
	cfg := h.config()

	if cfg.Resource != resourceName {
		t.Fatalf("Resource = %q, want %q", cfg.Resource, resourceName)
	}
	if cfg.Priority != "service" {
		t.Fatalf("Priority = %q, want service", cfg.Priority)
	}
	if cfg.Observer == nil {
		t.Fatal("Observer is nil; the config must carry the reranker's own observation")
	}
	// And the interval is the reranker's, not the shared default
	if cfg.Interval != defaultInterval {
		t.Fatalf("Interval = %s, want %s", cfg.Interval, defaultInterval)
	}
	var _ companion.Observer = h
}
