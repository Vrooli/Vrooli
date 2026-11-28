package resources

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

func TestManagedUpdateRejectedAdmissionCleanupKeepsAdmittedAndUnknownInputs(t *testing.T) {
	for _, mode := range []string{"rejected", "admitted", "unreadable"} {
		t.Run(mode, func(t *testing.T) {
			u := updateTestInitial(t)
			s := updateTestStore(t, u)
			dir := filepath.Dir(s.path)
			budget := managedUpdateCapacityBudget(u)
			budget.LedgerPath = capacity.LedgerReference(filepath.Join(dir, "capacity.db"))
			u.Capacity = &budget
			claim, err := capacity.ReserveOperationRAM(context.Background(), capacity.OperationReservationOptions{OwnerID: budget.OwnerID, InstanceID: u.ID, AmountBytes: budget.AmountBytes, TTL: 10 * time.Minute, Admission: capacity.AdmitOptions{Source: capacity.StaticSource{Inventory: hostinventory.Snapshot{Memory: hostinventory.Memory{TotalBytes: 4 << 30, AvailableBytes: 4 << 30}}}, OpenStore: func(ctx context.Context) (capacity.AdmitStore, error) {
				return capacity.NewSQLiteStore(ctx, capacity.Config{DBPath: string(budget.LedgerPath)})
			}}})
			if err != nil {
				t.Fatal(err)
			}
			root := filepath.Join(dir, "managed-update-exec", u.ID, "credentials")
			if err := os.MkdirAll(root, 0o700); err != nil {
				t.Fatal(err)
			}
			secret := filepath.Join(root, "private.value")
			if err := os.WriteFile(secret, []byte("fake fixture credential"), 0o600); err != nil {
				t.Fatal(err)
			}
			if mode == "admitted" {
				if _, _, err := s.Create(context.Background(), u, false); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "unreadable" {
				if err := os.WriteFile(s.path, []byte("not a ledger"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err = cleanupManagedUpdateRejectedAdmission(context.Background(), s, u, dir)
			if (err != nil) != (mode == "unreadable") {
				t.Fatalf("cleanup outcome for %s: %v", mode, err)
			}
			_, statErr := os.Stat(secret)
			if mode == "rejected" {
				if !os.IsNotExist(statErr) {
					t.Fatal("certain rejection retained private credentials")
				}
			} else if statErr != nil {
				t.Fatalf("admitted/unknown cleanup removed reconciliation inputs: %v", statErr)
			}
			store, err := capacity.NewSQLiteStore(context.Background(), capacity.Config{DBPath: string(budget.LedgerPath)})
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			current, err := store.GetClaim(context.Background(), claim.ClaimID)
			if err != nil {
				t.Fatal(err)
			}
			want := capacity.StatusReserved
			if mode == "rejected" {
				want = capacity.StatusReleased
			}
			if current.Status != want {
				t.Fatalf("capacity ownership changed incorrectly for %s: %s", mode, current.Status)
			}
		})
	}
}
