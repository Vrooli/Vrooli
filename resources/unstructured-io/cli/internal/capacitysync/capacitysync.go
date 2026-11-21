// Package capacitysync keeps Unstructured's measured RAM claim alive.
// The shared companion owns claim lifecycle; this package only observes the
// resource's current resident footprint.
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

const (
	resourceName     = "unstructured-io"
	defaultPreferred = int64(1536 << 20)
	intervalEnv      = "UNSTRUCTURED_IO_CAPACITY_SYNC_INTERVAL"
)

var defaultInterval = tuning.CompanionCapacitySyncInterval()

// Handlers carries the seams used by the companion command.
type Handlers struct {
	Stderr   io.Writer
	Exec     companion.Exec
	GetEnv   func(string) string
	Interval time.Duration
}

// Default returns handlers wired to the host control plane.
func Default() *Handlers {
	return &Handlers{Stderr: os.Stderr, Exec: companion.DefaultExec, GetEnv: os.Getenv}
}

// Command returns the managed-service capacity companion command.
func Command(h *Handlers) func([]string) error {
	if h == nil {
		h = Default()
	}
	return func(args []string) error {
		return companion.Run(companion.CommandOptions{Config: h.config(), Stderr: h.Stderr}, args)
	}
}

func (h *Handlers) config() companion.Config {
	interval := h.Interval
	if interval <= 0 {
		interval = companion.PollInterval(h.GetEnv, intervalEnv, defaultInterval)
	}
	return companion.Config{
		Resource:       resourceName,
		ResourceKind:   "ram",
		Observer:       h,
		Exec:           h.Exec,
		Interval:       interval,
		Priority:       "service",
		PreferredBytes: defaultPreferred,
		FloorBytes:     defaultPreferred,
		Log:            h.Stderr,
	}
}

type capacityList struct {
	Claims []struct {
		OwnerID       string `json:"owner_id"`
		OwnerKind     string `json:"owner_kind"`
		ResourceKind  string `json:"resource_kind"`
		AmountBytes   int64  `json:"amount_bytes"`
		ObservedBytes int64  `json:"observed_bytes"`
	} `json:"claims"`
}

// Observe reads the current process footprint recorded by the capacity owner.
// It never starts the resource or invents a claim when no active claim exists.
func (h *Handlers) Observe(ctx context.Context) (companion.Footprint, error) {
	out, err := h.Exec(ctx, "vrooli", "capacity", "list", "--owner", resourceName, "--active", "--json")
	if err != nil {
		return companion.Footprint{}, err
	}
	var payload capacityList
	if err := json.Unmarshal(out, &payload); err != nil {
		return companion.Footprint{}, err
	}
	for _, claim := range payload.Claims {
		if claim.OwnerID != resourceName || claim.OwnerKind != "resource" || claim.ResourceKind != "ram" {
			continue
		}
		if claim.ObservedBytes > 0 {
			return companion.Footprint{Bytes: claim.ObservedBytes}, nil
		}
		return companion.Footprint{Bytes: claim.AmountBytes}, nil
	}
	return companion.Footprint{}, nil
}
