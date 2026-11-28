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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	broker "github.com/vrooli/vrooli/internal/capacity"
	hostconfig "github.com/vrooli/vrooli/internal/config"
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
	Stdout          io.Writer
	Stderr          io.Writer
	GetEnv          func(string) string
	NewClient       func() psClient
	Exec            companion.Exec
	Interval        time.Duration
	ReservationFile string
	Now             func() time.Time
	AdmitModel      func(context.Context, int64, int64, int64) (string, int64, bool, error)
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

// LockSync serializes the shared companion's complete observe/reconcile pass
// with reservation confirmation and release.
func (h *Handlers) LockSync(ctx context.Context) (func(), error) {
	return h.reservationStore().LockSync(ctx)
}

func (h *Handlers) admitModelReservation(ctx context.Context, claimBytes, admissionBytes, observedResidentBytes int64) (string, int64, bool, error) {
	if h.AdmitModel != nil {
		return h.AdmitModel(ctx, claimBytes, admissionBytes, observedResidentBytes)
	}
	root, err := buildinfo.ResolveSourceRoot()
	if err != nil {
		return "", 0, false, fmt.Errorf("resolve Ollama capacity manifest root: %w", err)
	}
	result, err := broker.AdmitResource(ctx, broker.AdmitOptions{
		Root: root, ResourceName: resourceName, NoReclaim: true,
		PreferredBytesOverride: claimBytes, AdmissionBytesOverride: admissionBytes,
		ObservedResidentBytes: observedResidentBytes, Log: h.Stderr,
	})
	if err != nil {
		return "", 0, false, err
	}
	if result.Skipped || result.ClaimID == "" || !result.Verdict.Granted() || result.ClaimBytes < claimBytes {
		return "", 0, false, fmt.Errorf("Ollama model reservation admission did not establish %d bytes without reclaim", claimBytes)
	}
	return result.ClaimID, result.ClaimBytes, result.ClaimCreated, nil
}

// config declares ollama's companion.
func (h *Handlers) config() companion.Config {
	interval := h.Interval
	if interval <= 0 {
		interval = companion.PollInterval(h.GetEnv, intervalEnv, defaultInterval)
	}
	cfg := companion.Config{
		Resource: resourceName,
		Observer: h,
		Exec:     h.Exec,
		Interval: interval,
		Log:      h.Stderr,
	}
	return cfg
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
	var observed int64
	for _, model := range running {
		observed, err = addVRAM(observed, model.SizeVRAM)
		if err != nil {
			return companion.Footprint{}, err
		}
	}
	total := observed
	reservation, err := h.reservationStore().Read()
	if err != nil {
		return companion.Footprint{}, err
	}
	activity := ""
	admissionRequired := false
	claimPreferred := int64(0)
	claimFloor := int64(0)
	if reservation != nil {
		activity = companion.ActivityActive
		var targetResident int64
		for _, model := range running {
			if sameModel(model.Name, reservation.Model) && model.SizeVRAM > 0 {
				targetResident, err = addVRAM(targetResident, model.SizeVRAM)
				if err != nil {
					return companion.Footprint{}, err
				}
			}
		}
		held, err := addVRAM(total, reservation.RequestedBytes)
		if err != nil {
			return companion.Footprint{}, err
		}
		if targetResident > 0 {
			otherResident := total - targetResident
			if reservation.RequestedBytes > targetResident {
				held, err = addVRAM(otherResident, reservation.RequestedBytes)
				if err != nil {
					return companion.Footprint{}, err
				}
			} else {
				held = total
			}
		}
		if reservation.CapacityBytes > 0 {
			if reservation.CapacityBytes > held {
				held = reservation.CapacityBytes
			}
		}
		admissionRequired = true
		if total < held {
			total = held
		}
		claimPreferred = total
		claimFloor = total
	}
	return companion.Footprint{
		Bytes: total, Activity: activity, AdmissionRequired: admissionRequired,
		ClaimPreferredBytes: claimPreferred, ClaimFloorBytes: claimFloor,
		ObservedBytes: observed,
	}, nil
}

func sameModel(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	return a == b || a == b+":latest" || b == a+":latest"
}

func (h *Handlers) client() psClient {
	if h.NewClient != nil {
		return h.NewClient()
	}
	return ensure.NewClient()
}

func (h *Handlers) reservationStore() ReservationStore {
	path := strings.TrimSpace(h.ReservationFile)
	if path == "" {
		home, err := hostconfig.HomeDir()
		if err == nil {
			if dataRoot, pathErr := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyData); pathErr == nil {
				path = filepath.Join(dataRoot, "resources", resourceName, "model-capacity-reservation.json")
			}
		}
	}
	return ReservationStore{Path: path, Now: h.Now}
}
