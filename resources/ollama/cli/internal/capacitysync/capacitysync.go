// Package capacitysync is ollama's half of the capacity companion contract:
// observe how much VRAM the currently loaded models hold.
//
// Everything after the observation — claim, resize, release, heartbeat, flags,
// signals and the exit contract — lives in packages/capacity/companion, shared
// with every other accelerated resource. This file holds only what is specific
// to ollama: the /api/ps footprint. The claim itself (tier, reservation ladder,
// idle-yield, idle grace and idle-unload TTL) is the resource.json
// acceleration.claim block, applied by the shared companion.
package capacitysync

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

var (
	resourceName    = "ollama"
	defaultInterval = tuning.CompanionCapacitySyncInterval()
	// intervalEnv lets an operator slow the companion down without a rebuild.
	intervalEnv = "OLLAMA_CAPACITY_SYNC_INTERVAL"
)

// psClient is the slice of the Ollama client the observer needs: the
// loaded-model list from /api/ps. Tests inject a fake.
type psClient interface {
	ListRunning(ctx context.Context) ([]ensure.RunningModel, error)
}

// Handlers carries the injectable seams. Tests provide fakes; production takes
// the defaults.
type Handlers struct {
	Stdout    io.Writer
	Stderr    io.Writer
	GetEnv    func(string) string
	NewClient func() psClient
	Exec      companion.Exec
	Interval  time.Duration
}

// Default returns Handlers wired to the real runtime and shell.
func Default() *Handlers {
	return &Handlers{
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		GetEnv:    os.Getenv,
		NewClient: func() psClient { return ensure.NewClient() },
		Exec:      companion.DefaultExec,
	}
}

// Command returns the `capacity-sync` command for registration.
func Command(h *Handlers) cliapp.Command {
	if h == nil {
		h = Default()
	}
	return cliapp.Command{
		Name:        "capacity-sync",
		Description: "Run ollama's footprint-tracking companion: poll /api/ps and keep the capacity ledger sized to the loaded models",
		Usage:       "ollama capacity-sync [--interval 15s] [--once]",
		Run:         h.Run,
	}
}

// Run drives the shared companion loop.
func (h *Handlers) Run(args []string) error {
	return companion.Run(companion.CommandOptions{Config: h.config(), Stderr: h.Stderr}, args)
}

// config declares ollama's companion.
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
		Log:      h.Stderr,
	}
}

// Observe reports the VRAM the currently loaded models hold.
//
// A poll failure is an error rather than a zero footprint: ollama unloading
// every model and ollama being unreachable look identical from a zero, and only
// one of them should release the reservation.
func (h *Handlers) Observe(ctx context.Context) (companion.Footprint, error) {
	running, err := h.client().ListRunning(ctx)
	if err != nil {
		return companion.Footprint{}, err
	}
	var total int64
	for _, model := range running {
		total += model.SizeVRAM
	}
	return companion.Footprint{Bytes: total}, nil
}

func (h *Handlers) client() psClient {
	if h.NewClient != nil {
		return h.NewClient()
	}
	return ensure.NewClient()
}
