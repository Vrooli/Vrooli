package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	repocontract "github.com/vrooli/repo-contract-go"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

const rehearsalDiskFloorBytes int64 = 120 * 1024 * 1024 * 1024

// admitTreasuryRehearsalDisk holds a renewable reservation in the shared host
// ledger for the complete lifecycle/rehearsal job and writes its admission and
// release receipts next to that run's evidence.
func admitTreasuryRehearsalDisk(t *testing.T, reservationPath, evidenceDir, ownerID, job string, peakBytes int64) {
	t.Helper()
	if !filepath.IsAbs(reservationPath) || !filepath.IsAbs(evidenceDir) {
		t.Fatalf("rehearsal disk reservation and evidence paths must be absolute: reserve=%q evidence=%q", reservationPath, evidenceDir)
	}
	ledgerPath, err := treasuryRehearsalCapacityLedgerPath()
	require.NoError(t, err)
	broker, err := sharedcapacity.NewBroker(context.Background(), ledgerPath)
	require.NoError(t, err)
	lease, err := broker.ReserveDisk(context.Background(), ownerID, reservationPath, peakBytes, rehearsalDiskFloorBytes)
	require.NoError(t, err, "rehearsal job must be admitted by the shared host disk ledger")
	reservation := lease.Reservation()
	write := func(name string, value any) error {
		raw, marshalErr := json.MarshalIndent(value, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		return os.WriteFile(filepath.Join(evidenceDir, name), raw, 0o600)
	}
	require.NoError(t, write("disk-admission.json", map[string]any{
		"job": job, "owner": reservation.OwnerID, "reservation_id": reservation.ID,
		"filesystem_id": reservation.FilesystemID, "ledger_path": ledgerPath,
		"path": reservation.Path, "peak_bytes": reservation.PeakBytes,
		"floor_bytes": reservation.FloorBytes, "available_bytes_at_admission": reservation.AvailableBytes,
		"other_reserved_bytes": reservation.OtherReservedBytes, "expires_at": reservation.ExpiresAt,
	}))
	beatCtx, stopBeats := context.WithCancel(context.Background())
	beatDone := make(chan struct{})
	beatErrors := make(chan error, 1)
	go func() {
		defer close(beatDone)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-beatCtx.Done():
				return
			case <-ticker.C:
				if heartbeatErr := lease.Heartbeat(beatCtx); heartbeatErr != nil {
					select {
					case beatErrors <- heartbeatErr:
					default:
					}
					return
				}
			}
		}
	}()
	t.Cleanup(func() {
		stopBeats()
		<-beatDone
		releaseErr := lease.Release(context.Background())
		require.NoError(t, write("disk-release.json", map[string]any{
			"reservation_id": reservation.ID, "released_at": time.Now().UTC(),
			"release_error": fmt.Sprint(releaseErr),
		}))
		_ = broker.Close()
		require.NoError(t, releaseErr, "release rehearsal shared disk lease")
		select {
		case heartbeatErr := <-beatErrors:
			t.Errorf("shared disk lease heartbeat failed during %s: %v", job, heartbeatErr)
		default:
		}
	})
}

func treasuryRehearsalCapacityLedgerPath() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("CAPACITY_ADMISSION_DB")); configured != "" {
		return configured, nil
	}
	// Refuse to guess a sudo/root home. The host owner can pass its resolved
	// shared path explicitly when the test is launched from a root context.
	if os.Geteuid() == 0 {
		return "", fmt.Errorf("cannot infer the operator capacity ledger from a root test process; set CAPACITY_ADMISSION_DB to the owner-resolved shared path")
	}
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		return "", fmt.Errorf("HOME is unset; pass CAPACITY_ADMISSION_DB from the host admission owner")
	}
	stateDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", err
	}
	return filepath.Join(stateDir, "capacity.db"), nil
}
