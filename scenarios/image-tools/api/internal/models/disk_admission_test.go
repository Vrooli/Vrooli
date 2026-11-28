package models

import (
	"context"
	"errors"
	"testing"

	"image-tools/internal/disktest"

	"github.com/vrooli/vrooli/packages/capacity"
)

// refusingDiskAdmitter fails the test if an install reserves disk.
type refusingDiskAdmitter struct{ t *testing.T }

func (a refusingDiskAdmitter) ReserveDiskJob(_ context.Context, r capacity.DiskJobRequest) (capacity.DiskJobLease, error) {
	a.t.Errorf("revalidation reserved a disk peak: %+v", r.Paths)
	return nil, errors.New("unexpected disk reservation")
}

func TestRevalidatingValidWeightsReservesNoDiskPeak(t *testing.T) {
	f := newInstallFixture(t)
	ctx := context.Background()
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatal(err)
	}
	downloads := f.downloads
	// A second install of intact weights (the `models install` / auto
	// revalidation path for a pending smoke) writes nothing, so it must not
	// plan or reserve an acquisition peak.
	f.in.DiskAdmission = refusingDiskAdmitter{t: t}
	if _, err := f.in.InstallForJob(ctx, installTestModelID, "revalidate-job", nil); err != nil {
		t.Fatalf("revalidate: %v", err)
	}
	if f.downloads != downloads {
		t.Fatalf("revalidation downloaded again: %d -> %d", downloads, f.downloads)
	}
}

// TestInstallResumesWithItsInterruptedLeaseRowsStillInTheLedger is the live
// restart failure: a drain interrupts a download, the durable job is requeued,
// and the resumed attempt reserves its disk peak while the ledger still holds
// the row the first attempt left. Every shape that row can take must resume.
func TestInstallResumesWithItsInterruptedLeaseRowsStillInTheLedger(t *testing.T) {
	const durableJob = "e8402392"
	const owner = "image-tools:model-install"
	cases := []struct {
		name   string
		seed   func(l *disktest.Ledger)
		active int // leases still holding a reservation after the install
	}{
		{"active lease of the exited producer under the durable key", func(l *disktest.Ledger) { l.Seed(owner, durableJob, "active", false) }, 0},
		{"completed lease under the durable key", func(l *disktest.Ledger) { l.Seed(owner, durableJob, "completed", false) }, 0},
		{"active lease of an exited earlier attempt", func(l *disktest.Ledger) { l.Seed(owner, durableJob+"/attempt/earlier", "active", false) }, 0},
		{"lease of an unrelated live job", func(l *disktest.Ledger) { l.Seed(owner, "other-job/attempt/x", "active", true) }, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newInstallFixture(t)
			tc.seed(f.ledger)
			if _, err := f.in.InstallForJob(context.Background(), installTestModelID, durableJob, nil); err != nil {
				t.Fatalf("resumed install failed with the old lease row present: %v", err)
			}
			if !f.in.Installed(context.Background(), installTestModelID) {
				t.Fatal("resumed install finished but the model is not installed")
			}
			if got := f.ledger.Active(); got != tc.active {
				t.Fatalf("%d leases still hold a reservation after the install, want %d: %+v", got, tc.active, f.ledger.Rows())
			}
		})
	}
}

func TestSecondInstallAttemptOfTheSameJobReserves(t *testing.T) {
	f := newInstallFixture(t)
	ctx := context.Background()
	if _, err := f.in.InstallForJob(ctx, installTestModelID, "retry-job", nil); err != nil {
		t.Fatal(err)
	}
	if err := f.in.Remove(ctx, installTestModelID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.in.InstallForJob(ctx, installTestModelID, "retry-job", nil); err != nil {
		t.Fatalf("requeued job could not reserve again: %v", err)
	}
}
