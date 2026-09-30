// Package sessionlease is the runtime-registry implementation of cli-core's
// EditorLeaseRecorder: the coding-agent launcher binaries register it so
// every session is visible in `vrooli agent list` with its tree, scope and
// claims for as long as the process lives. cli-core cannot import the
// registry (244 modules replace cli-core), so the seam lives there and the
// store here.
package sessionlease

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// Recorder adapts the runtime registry to cli-core's seam.
type Recorder struct {
	// Open returns the registry store; tests inject a temporary one.
	Open func(ctx context.Context) (*scenarioruntime.SQLiteStore, error)
}

// Register installs the recorder as cli-core's default.
func Register() { cliutil.RegisterEditorLeaseRecorder(Recorder{Open: openRegistry}) }

func openRegistry(ctx context.Context) (*scenarioruntime.SQLiteStore, error) {
	home, err := config.HomeDir()
	if err != nil {
		return nil, err
	}
	return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
}

func (r Recorder) open(ctx context.Context) (*scenarioruntime.SQLiteStore, error) {
	if r.Open == nil {
		return nil, fmt.Errorf("session lease: no registry opener")
	}
	return r.Open(ctx)
}

// Create writes the session's lease; the boot id makes a reboot provable.
func (r Recorder) Create(ctx context.Context, record cliutil.EditorLeaseRecord) error {
	store, err := r.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	lease := scenarioruntime.EditorLease{
		SessionID: record.SessionID, Harness: record.Harness, Agent: record.Agent, PID: record.PID,
		WorkingDir: record.WorkingDir, Scope: record.Scope, ContainmentMethod: record.ContainmentMethod, Claims: record.Claims,
	}
	if host, err := (hostsession.DefaultProvider{}).Current(ctx, ""); err == nil {
		lease.HostBootID = host.BootID
	}
	_, err = store.CreateEditorLease(ctx, lease, cliutil.EditorLeaseTTL)
	return err
}

// Heartbeat renews the lease deadline.
func (r Recorder) Heartbeat(ctx context.Context, sessionID string) error {
	store, err := r.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	_, err = store.HeartbeatEditorLease(ctx, sessionID, cliutil.EditorLeaseTTL)
	return err
}

// Stop ends the lease with a reason.
func (r Recorder) Stop(ctx context.Context, sessionID, reason string) error {
	store, err := r.open(ctx)
	if err != nil {
		return err
	}
	defer store.Close()
	_, err = store.StopEditorLease(ctx, sessionID, reason)
	return err
}

// Overlaps names live sessions whose claims overlap the requested paths.
func (r Recorder) Overlaps(ctx context.Context, paths []string) ([]cliutil.ClaimHolder, error) {
	store, err := r.open(ctx)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	leases, err := store.ListEditorLeases(ctx, false)
	if err != nil {
		return nil, err
	}
	// Only a holder that is provably alive is named; a dead session's claim
	// is not a claim.
	guard := scenarioruntime.StartingLeaseGuard{PIDRunning: platform.IsPIDRunning}
	if host, err := (hostsession.DefaultProvider{}).Current(ctx, ""); err == nil {
		guard.CurrentBootID = host.BootID
	}
	leases = scenarioruntime.LiveEditorLeases(leases, guard)
	now := time.Now().UTC()
	var out []cliutil.ClaimHolder
	for _, overlap := range scenarioruntime.ClaimOverlaps(leases, paths) {
		out = append(out, cliutil.ClaimHolder{
			SessionID: overlap.Holder.SessionID, Agent: overlap.Holder.Agent, PID: overlap.Holder.PID, WorkingDir: overlap.Holder.WorkingDir,
			HeldPath: overlap.HeldPath, Overlap: string(overlap.Overlap), Age: now.Sub(overlap.Holder.CreatedAt),
		})
	}
	return out, nil
}
