// Package capacitysync is ollama's half of the capacity companion contract:
// observe how much VRAM the currently loaded models hold.
//
// Everything after the observation — claim, resize, release, heartbeat, flags,
// signals and the exit contract — lives in packages/capacity/companion, shared
// with every other accelerated resource. This file holds only what is specific
// to ollama: the /api/ps footprint and the model-policy-derived degrade ladder.
package capacitysync

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/capacity/companion"
)

var (
	resourceName    = "ollama"
	defaultInterval = tuning.CompanionCapacitySyncInterval()
	// intervalEnv lets an operator slow the companion down without a rebuild.
	intervalEnv     = "OLLAMA_CAPACITY_SYNC_INTERVAL"
	bytesPerGiB     = int64(1024 * 1024 * 1024)
	defaultIdleWait = tuning.ResourceLongHTTPTimeout()
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
	preferred, floor, profile := h.claimProfile()
	return companion.Config{
		Resource:       resourceName,
		Observer:       h,
		Exec:           h.Exec,
		Interval:       interval,
		Priority:       "service",
		PreferredBytes: preferred,
		FloorBytes:     floor,
		Profile:        profile,
		YieldWhenIdle:  true,
		IdleGrace:      defaultIdleWait,
		Log:            h.Stderr,
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

type profileStep struct {
	Label       string `json:"label"`
	AmountBytes int64  `json:"amount_bytes"`
}

type degradeProfile struct {
	Steps []profileStep `json:"steps"`
	Apply struct {
		Verb string   `json:"verb"`
		Argv []string `json:"argv"`
	} `json:"apply"`
	Upshift bool `json:"upshift"`
}

// claimProfile derives the broker reservation and ladder from the same model
// policy the resource's capacity planner uses. When policy is unavailable the
// conservative manifest-equivalent defaults preserve an honest claim rather
// than silently falling back to the currently loaded footprint.
func (h *Handlers) claimProfile() (int64, int64, string) {
	refs := []string{"qwen3.5:9b", "qwen3.5:4b", "qwen3:4b", "qwen3:1.7b"}
	profile := degradeProfile{}
	profile.Apply.Verb = "capacity"
	profile.Apply.Argv = []string{"degrade", "--to", "{label}"}
	profile.Upshift = true
	getenv := h.GetEnv
	if getenv == nil {
		getenv = os.Getenv
	}
	if p, _, err := policy.LoadDefaultFile(getenv); err == nil {
		for _, ref := range refs {
			model, ok := p.Models[ref]
			if !ok || model.VRAMGBEstimate <= 0 {
				continue
			}
			profile.Steps = append(profile.Steps, profileStep{Label: ref, AmountBytes: int64(model.VRAMGBEstimate * float64(bytesPerGiB))})
		}
	}
	if len(profile.Steps) == 0 {
		profile.Steps = []profileStep{
			{Label: "qwen3.5:9b", AmountBytes: 11 * bytesPerGiB},
			{Label: "qwen3.5:4b", AmountBytes: 6 * bytesPerGiB},
			{Label: "qwen3:4b", AmountBytes: 5 * bytesPerGiB},
			{Label: "qwen3:1.7b", AmountBytes: 3 * bytesPerGiB},
		}
	}
	return profile.Steps[0].AmountBytes, profile.Steps[len(profile.Steps)-1].AmountBytes, mustJSON(profile)
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return `{"steps":[],"apply":{"verb":"capacity","argv":["degrade","--to","{label}"]}}`
	}
	return string(data)
}

func (h *Handlers) client() psClient {
	if h.NewClient != nil {
		return h.NewClient()
	}
	return ensure.NewClient()
}
