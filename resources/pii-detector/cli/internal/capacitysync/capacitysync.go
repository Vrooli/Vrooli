// Package capacitysync is the pii-detector's half of the capacity companion
// contract: observe how much the detector currently holds.
//
// Everything after the observation — claim, resize, release, heartbeat, flags,
// signals and the exit contract — lives in packages/capacity/companion, shared
// with every other resource. This file holds only what is specific to the
// pii-detector.
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
	resourceName    = "pii-detector"
	defaultInterval = tuning.CompanionCapacitySyncInterval()
	// intervalEnv lets an operator slow the companion down without a rebuild.
	intervalEnv = "PII_DETECTOR_CAPACITY_SYNC_INTERVAL"
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

// config declares the pii-detector's companion.
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
// reads to learn what the detector currently holds.
type residentFootprint struct {
	Claims []struct {
		OwnerID       string `json:"owner_id"`
		AmountBytes   int64  `json:"amount_bytes"`
		ObservedBytes int64  `json:"observed_bytes"`
	} `json:"claims"`
}

// Observe reports the detector's resident footprint. The detector loads one
// token-classification model at start and holds it for its lifetime, so its
// footprint is whatever the host observes it using; reading the observed figure
// rather than a declared constant is what lets the right-sizing advisory work.
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
		if claim.ObservedBytes > 0 {
			return companion.Footprint{Bytes: claim.ObservedBytes}, nil
		}
		return companion.Footprint{Bytes: claim.AmountBytes}, nil
	}
	// No active claim: report a zero footprint and nothing more. Observe is an
	// observation, not a lifecycle action — starting/admitting the resource is
	// the control plane's job, so this method must never shell `resource start`
	// (that conflated observation with admission and could restart-loop).
	return companion.Footprint{}, nil
}
