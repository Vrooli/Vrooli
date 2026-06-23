// Package capacitysync is ollama's footprint-tracking companion: a host-side
// poller that keeps the platform capacity ledger in sync with the models Ollama
// actually has loaded.
//
// Why a poller and not the static resident block (plan §Phase 5, contract C5):
// ollama's footprint is DYNAMIC. With OLLAMA_KEEP_ALIVE the runtime loads a model
// on demand and unloads it after idle. A static `service / 5 GiB` admission claim
// is wrong on both ends — when the model unloads there is no GPU process, so the
// sweep cannot presence-refresh it and it EXPIRES; when it reloads nothing
// re-claims, so ollama runs unclaimed. This poller instead reads `GET /api/ps`
// (the honest loaded-footprint source) on an interval and maintains a single
// ollama claim sized to the currently-loaded models: it claims when a model
// loads, resizes when the loaded set changes, and releases when everything
// unloads. It is FAIL-OPEN: any poll or ledger error leaves the ledger unchanged
// (never strand ollama).
package capacitysync

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/resources/ollama/cli/internal/ensure"
)

const (
	// resourceName is the capacity owner id the poller maintains a claim for.
	resourceName = "ollama"
	// defaultInterval is how often the poller reconciles the ledger against
	// /api/ps. Short enough to track a load/unload promptly, long enough not to
	// hammer the runtime.
	defaultInterval = 15 * time.Second
	// resizeThresholdBytes is the minimum footprint delta that triggers a
	// claim resize (release + reclaim). Small jitter in reported size_vram does not
	// churn the ledger.
	resizeThresholdBytes = 256 * 1024 * 1024 // 256 MiB

	envInterval = "OLLAMA_CAPACITY_SYNC_INTERVAL"
)

// psClient is the slice of the Ollama client the poller needs (the loaded-model
// list from /api/ps). Tests inject a fake.
type psClient interface {
	ListRunning(ctx context.Context) ([]ensure.RunningModel, error)
}

// Handlers owns the poller's dependencies; tests inject the seams so no real
// runtime, exec, or clock is needed.
type Handlers struct {
	Stdout io.Writer
	Stderr io.Writer
	GetEnv func(string) string
	// NewClient builds the /api/ps client (default: ensure.NewClient).
	NewClient func() psClient
	// Exec runs a `vrooli capacity …` call and returns its stdout. Tests inject a
	// fake; production shells the on-PATH vrooli binary.
	Exec func(ctx context.Context, name string, args ...string) ([]byte, error)
	// Interval overrides the poll interval (default defaultInterval / env).
	Interval time.Duration
}

// Default returns Handlers wired to the real runtime + shell.
func Default() *Handlers {
	return &Handlers{
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		GetEnv:    os.Getenv,
		NewClient: func() psClient { return ensure.NewClient() },
		Exec: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		},
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

// Run parses flags and polls until signaled (or once with --once).
func (h *Handlers) Run(args []string) error {
	fs := flag.NewFlagSet("capacity-sync", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	interval := fs.Duration("interval", h.interval(), "poll interval")
	once := fs.Bool("once", false, "run a single reconcile and exit (for tests/cron)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *once {
		h.syncOnce(context.Background())
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(h.Stdout, "ollama capacity-sync: tracking loaded footprint every %s\n", *interval)

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	h.syncOnce(ctx) // reconcile immediately on start
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			h.syncOnce(ctx)
		}
	}
}

// ledgerClaim is the slice of `vrooli capacity list --json` the poller reads.
type ledgerClaim struct {
	ClaimID     string `json:"claim_id"`
	OwnerID     string `json:"owner_id"`
	AmountBytes int64  `json:"amount_bytes"`
	Generation  int64  `json:"generation"`
}

// syncOnce reconciles the ollama claim against the live loaded footprint. It is
// fail-open at every step: a poll or ledger error leaves the ledger untouched.
func (h *Handlers) syncOnce(ctx context.Context) {
	running, err := h.client().ListRunning(ctx)
	if err != nil {
		return // /api/ps unavailable — leave the ledger as-is (never strand ollama)
	}
	var total int64
	for _, m := range running {
		total += m.SizeVRAM
	}

	active := h.activeClaim(ctx)

	switch {
	case total <= 0 && active == nil:
		return // nothing loaded, nothing claimed — steady state
	case total <= 0 && active != nil:
		// Everything unloaded → release the claim (no expire-churn).
		h.release(ctx, active.ClaimID)
	case total > 0 && active == nil:
		// A model loaded → claim sized to the footprint.
		h.claim(ctx, total)
	default: // total > 0 && active != nil
		if abs64(active.AmountBytes-total) > resizeThresholdBytes {
			// Loaded set changed materially → resize (release + reclaim).
			h.release(ctx, active.ClaimID)
			h.claim(ctx, total)
		} else {
			// Footprint steady → keep the claim alive.
			h.heartbeat(ctx, active.ClaimID, active.Generation)
		}
	}
}

// activeClaim returns the current active ollama claim, or nil. Fail-open: any
// error / parse failure yields nil (the poller will re-claim, which is idempotent
// against the broker's own reuse logic).
func (h *Handlers) activeClaim(ctx context.Context) *ledgerClaim {
	out, err := h.Exec(ctx, "vrooli", "capacity", "list", "--owner", resourceName, "--active", "--json")
	if err != nil {
		return nil
	}
	var payload struct {
		Claims []ledgerClaim `json:"claims"`
	}
	if json.Unmarshal(out, &payload) != nil {
		return nil
	}
	for i := range payload.Claims {
		if payload.Claims[i].OwnerID == resourceName && payload.Claims[i].ClaimID != "" {
			return &payload.Claims[i]
		}
	}
	return nil
}

// claim records ollama's current loaded footprint as a single-size service-tier
// reservation. It deliberately carries NO degrade profile: ollama is a
// self-managing resident — OLLAMA_KEEP_ALIVE unloads idle models on its own and
// the next poll then releases the claim — so the broker never needs to
// degrade/unload ollama itself. The claim's job is honest footprint accounting,
// not handing the broker a degrade-to-zero step (which, under a contended fleet,
// Decide would pick in advisory and the ledger would then record 0 bytes while
// the model is actually resident).
func (h *Handlers) claim(ctx context.Context, total int64) {
	_, _ = h.Exec(ctx, "vrooli", "capacity", "claim",
		"--owner-kind", "resource", "--owner-id", resourceName,
		"--resource-kind", "vram", "--gpu-index", "0",
		"--preferred", strconv.FormatInt(total, 10), "--floor", strconv.FormatInt(total, 10),
		"--priority", "service", "--json")
}

func (h *Handlers) release(ctx context.Context, claimID string) {
	_, _ = h.Exec(ctx, "vrooli", "capacity", "release", "--claim-id", claimID, "--json")
}

func (h *Handlers) heartbeat(ctx context.Context, claimID string, generation int64) {
	_, _ = h.Exec(ctx, "vrooli", "capacity", "heartbeat",
		"--claim-id", claimID, "--generation", strconv.FormatInt(generation, 10), "--json")
}

func (h *Handlers) client() psClient {
	if h.NewClient != nil {
		return h.NewClient()
	}
	return ensure.NewClient()
}

func (h *Handlers) interval() time.Duration {
	if h.Interval > 0 {
		return h.Interval
	}
	if h.GetEnv != nil {
		if v := strings.TrimSpace(h.GetEnv(envInterval)); v != "" {
			if d, err := time.ParseDuration(v); err == nil && d > 0 {
				return d
			}
		}
	}
	return defaultInterval
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
