// Package capacitysync is Ollaya's half of the capacity companion
// contract: observe how much VRAM the Ollaya daemon currently holds.
//
// Everything after the observation — claim, resize, release, heartbeat, flags,
// signals and the exit contract — lives in packages/capacity/companion, shared
// with every other accelerated resource. This file holds only what is specific
// to Ollaya.
package capacitysync

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/packages/capacity/companion"
)

var (
	resourceName    = "ollaya"
	defaultInterval = tuning.CompanionCapacitySyncInterval()
	// intervalEnv lets an operator slow the companion down without a rebuild.
	intervalEnv = "OLLAYA_CAPACITY_SYNC_INTERVAL"
)

// Handlers carries the injectable seams. Tests provide fakes; production takes
// the defaults.
type Handlers struct {
	Stdout   io.Writer
	Stderr   io.Writer
	Exec     companion.Exec
	GetEnv   func(string) string
	Interval time.Duration
}

// Default returns Handlers wired to the real shell and environment.
func Default() *Handlers {
	return &Handlers{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Exec:   companion.DefaultExec,
		GetEnv: os.Getenv,
	}
}

// Command returns the `capacity-sync` command for registration.
func Command(h *Handlers) func([]string) error {
	if h == nil {
		h = Default()
	}
	return func(args []string) error {
		return companion.Run(companion.CommandOptions{Config: h.config(), Stderr: h.Stderr}, args)
	}
}

// config declares Ollaya's companion.
func (h *Handlers) config() companion.Config {
	interval := h.Interval
	if interval <= 0 {
		interval = companion.PollInterval(h.GetEnv, intervalEnv, defaultInterval)
	}
	return companion.Config{
		Resource: resourceName,
		Observer: h,
		Exec:     h.Exec,
		Interval: interval,
		Priority: "service",
		Log:      h.Stderr,
	}
}

// residentFootprint is the slice of `vrooli capacity list --json` the observer
// reads to learn what Ollaya currently holds.
type residentFootprint struct {
	Claims []struct {
		OwnerID       string `json:"owner_id"`
		AmountBytes   int64  `json:"amount_bytes"`
		ObservedBytes int64  `json:"observed_bytes"`
		Status        string `json:"status"`
	} `json:"claims"`
}

// Observe reports Ollaya's resident footprint.
//
// The daemon loads decision models on demand and keeps them for keep_alive, so
// its footprint is whatever the host observes it using. Reading the observed
// figure rather than a declared constant is what lets the right-sizing
// advisory work on a claim this companion maintains.
//
// Observation is read-only. An earlier version re-admitted the resource from
// this method (`vrooli resource start`), which created a claim and then returned
// a zero footprint; the caller immediately released that new claim. That loop
// produced one create/release pair every poll — 8,764 released Ollaya claims on
// this host — and the resulting write contention is what surfaced as
// "ledger unavailable" and blocked the broker's upshift. Admission belongs to
// the lifecycle start; this companion only maintains a claim that exists.
//
// A degraded claim at the CPU rung is marked Parked: the daemon is alive but
// deliberately holds no VRAM, so its claim records the mode and must survive
// until the broker upshifts it.
func (h *Handlers) Observe(ctx context.Context) (companion.Footprint, error) {
	out, err := h.Exec(ctx, "vrooli", "capacity", "list", "--owner", resourceName, "--active", "--json")
	if err != nil {
		return companion.Footprint{}, err
	}
	var payload residentFootprint
	if err := json.Unmarshal(out, &payload); err != nil {
		return companion.Footprint{}, err
	}
	for _, claim := range payload.Claims {
		if claim.OwnerID != resourceName {
			continue
		}
		parked := claim.Status == "degraded" && claim.AmountBytes <= 0
		// An observed figure is the truth when the host has produced one;
		// otherwise the claim's own amount keeps the reservation alive without
		// inventing a measurement.
		if claim.ObservedBytes > 0 {
			return companion.Footprint{Bytes: claim.ObservedBytes, Parked: parked}, nil
		}
		return companion.Footprint{Bytes: claim.AmountBytes, Parked: parked}, nil
	}
	// No active claim. The lifecycle admission path owns the manifest-derived
	// values; re-admitting here would mutate the ledger from a read and churn a
	// claim per poll. A zero footprint with no claim is a no-op.
	return companion.Footprint{}, nil
}
