package vector

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type capacityLeaseFake struct {
	receipt   string
	completed int
	err       error
}

func (l *capacityLeaseFake) ReceiptID() string { return l.receipt }
func (l *capacityLeaseFake) Complete(context.Context) error {
	if l.err != nil {
		return l.err
	}
	l.completed++
	return nil
}

func TestCapacityGenerationDiskAdmissionRetainsUncertainLeaseAndReusesOperation(t *testing.T) {
	ctx := context.Background()
	lease := &capacityLeaseFake{receipt: "capacity-receipt-1"}
	reserveCalls := 0
	adapter := &CapacityGenerationDiskAdmission{
		OwnerID: "source-ledger", Path: "./data/source-ledger.db", FloorBytes: 1 << 30,
		Reserve: func(_ context.Context, request GenerationDiskJobRequest) (GenerationDiskJobLease, error) {
			reserveCalls++
			require.Equal(t, "source-ledger", request.OwnerID)
			require.Equal(t, "source-ledger-vector-generation:op-1", request.JobID)
			require.Equal(t, "sqlite-vector-generation", request.Role)
			require.Greater(t, request.PeakBytes, int64(0))
			require.NotEmpty(t, request.Path)
			return lease, nil
		},
	}

	receipt, err := adapter.Admit(ctx, "op-1", "gen-1", "recipe-1", 4096)
	require.NoError(t, err)
	require.Equal(t, "capacity-receipt-1", receipt)
	require.NoError(t, adapter.Complete(ctx, "op-1", receipt, false))
	require.Zero(t, lease.completed, "uncertain writes must keep their capacity reservation")

	retriedReceipt, err := adapter.Admit(ctx, "op-1", "gen-1", "recipe-1", 4096)
	require.NoError(t, err)
	require.Equal(t, receipt, retriedReceipt)
	require.Equal(t, 1, reserveCalls, "retry of a stable operation must not reserve a second lease")
	require.NoError(t, adapter.Complete(ctx, "op-1", receipt, false))
	require.Zero(t, lease.completed)
}

func TestCapacityGenerationDiskAdmissionReleasesOnlyAfterSuccessfulCompletion(t *testing.T) {
	ctx := context.Background()
	lease := &capacityLeaseFake{receipt: "capacity-receipt-2"}
	reserveCalls := 0
	adapter := &CapacityGenerationDiskAdmission{
		OwnerID: "source-ledger", Path: "/data/source-ledger.db",
		Reserve: func(context.Context, GenerationDiskJobRequest) (GenerationDiskJobLease, error) {
			reserveCalls++
			return lease, nil
		},
	}
	receipt, err := adapter.Admit(ctx, "op-2", "gen-2", "recipe-2", 8192)
	require.NoError(t, err)
	require.Error(t, adapter.Complete(ctx, "op-2", "wrong-receipt", true))
	require.Zero(t, lease.completed)

	lease.err = errors.New("capacity owner temporarily unavailable")
	require.Error(t, adapter.Complete(ctx, "op-2", receipt, true))
	lease.err = nil
	require.NoError(t, adapter.Complete(ctx, "op-2", receipt, true))
	require.Equal(t, 1, lease.completed)
	require.NoError(t, adapter.Complete(ctx, "op-2", receipt, true), "completion retry is idempotent")
	require.Equal(t, 1, lease.completed, "completed lease must not be released twice")

	again, err := adapter.Admit(ctx, "op-2", "gen-2", "recipe-2", 8192)
	require.NoError(t, err)
	require.Equal(t, receipt, again)
	require.Equal(t, 1, reserveCalls, "completed stable operation must not create a replacement lease")
	require.NoError(t, adapter.Complete(ctx, "op-2", receipt, true))
}

func TestCapacityGenerationDiskAdmissionRejectsStableIdentityReuse(t *testing.T) {
	adapter := &CapacityGenerationDiskAdmission{
		OwnerID: "source-ledger", Path: "/data/source-ledger.db",
		Reserve: func(context.Context, GenerationDiskJobRequest) (GenerationDiskJobLease, error) {
			return &capacityLeaseFake{receipt: "receipt"}, nil
		},
	}
	_, err := adapter.Admit(context.Background(), "op-3", "gen-3", "recipe-3", 100)
	require.NoError(t, err)
	_, err = adapter.Admit(context.Background(), "op-3", "gen-3", "recipe-other", 100)
	require.ErrorIs(t, err, ErrGenerationOperationConflict)
}

func TestCapacityGenerationDiskAdmissionReattachesUncertainLeaseAfterRestart(t *testing.T) {
	ctx := context.Background()
	lease := &capacityLeaseFake{receipt: "capacity-receipt-restarted"}
	var request GenerationDiskJobRequest
	firstReserveCalls := 0
	first := &CapacityGenerationDiskAdmission{
		OwnerID: "source-ledger", Path: "/data/source-ledger.db",
		Reserve: func(_ context.Context, got GenerationDiskJobRequest) (GenerationDiskJobLease, error) {
			firstReserveCalls++
			request = got
			return lease, nil
		},
	}
	receipt, err := first.Admit(ctx, "restart-op", "generation-7", "recipe-7", 8192)
	require.NoError(t, err)
	require.NoError(t, first.Complete(ctx, "restart-op", receipt, false), "uncertain effects retain the active lease")
	require.Zero(t, lease.completed)

	findCalls, reattachCalls, secondReserveCalls := 0, 0, 0
	second := &CapacityGenerationDiskAdmission{
		OwnerID: "source-ledger", Path: "/data/source-ledger.db",
		Find: func(_ context.Context, ownerID, jobID string) (GenerationDiskJobReceipt, bool, error) {
			findCalls++
			require.Equal(t, "source-ledger", ownerID)
			require.Equal(t, request.JobID, jobID)
			return GenerationDiskJobReceipt{
				ID: receipt, Owner: request.OwnerID, JobID: request.JobID, Status: "active",
				Path: request.Path, Role: request.Role, Peak: request.PeakBytes,
			}, true, nil
		},
		Reattach: func(_ context.Context, ownerID, jobID, leaseID string) (GenerationDiskJobLease, error) {
			reattachCalls++
			require.Equal(t, "source-ledger", ownerID)
			require.Equal(t, request.JobID, jobID)
			require.Equal(t, receipt, leaseID)
			return lease, nil
		},
		Reserve: func(context.Context, GenerationDiskJobRequest) (GenerationDiskJobLease, error) {
			secondReserveCalls++
			return nil, errors.New("must reattach the active owner lease")
		},
	}
	restartedReceipt, err := second.Admit(ctx, "restart-op", "generation-7", "recipe-7", 8192)
	require.NoError(t, err)
	require.Equal(t, receipt, restartedReceipt)
	require.Equal(t, 1, findCalls)
	require.Equal(t, 1, reattachCalls)
	require.Zero(t, secondReserveCalls, "restart must not create a second reservation")
	require.NoError(t, second.Complete(ctx, "restart-op", restartedReceipt, true))
	require.Equal(t, 1, lease.completed)
}

func TestCapacityGenerationDiskAdmissionRejectsMismatchedRecoveredLease(t *testing.T) {
	adapter := &CapacityGenerationDiskAdmission{
		OwnerID: "source-ledger", Path: "/data/source-ledger.db",
		Find: func(context.Context, string, string) (GenerationDiskJobReceipt, bool, error) {
			return GenerationDiskJobReceipt{ID: "receipt", Owner: "other-owner", JobID: "wrong-job", Status: "active"}, true, nil
		},
		Reattach: func(context.Context, string, string, string) (GenerationDiskJobLease, error) {
			t.Fatal("must reject mismatched receipt before reattachment")
			return nil, nil
		},
		Reserve: func(context.Context, GenerationDiskJobRequest) (GenerationDiskJobLease, error) {
			t.Fatal("must fail closed rather than reserve over a mismatched active lease")
			return nil, nil
		},
	}
	_, err := adapter.Admit(context.Background(), "restart-op", "generation-7", "recipe-7", 8192)
	require.Error(t, err)
}

func TestCapacityGenerationDiskAdmissionFailsClosed(t *testing.T) {
	lease := func(receipt string) GenerationDiskJobReserve {
		return func(context.Context, GenerationDiskJobRequest) (GenerationDiskJobLease, error) {
			return &capacityLeaseFake{receipt: receipt}, nil
		}
	}
	findNone := func(context.Context, string, string) (GenerationDiskJobReceipt, bool, error) {
		return GenerationDiskJobReceipt{}, false, nil
	}
	findActive := func(context.Context, string, string) (GenerationDiskJobReceipt, bool, error) {
		return GenerationDiskJobReceipt{
			ID: "surviving", Owner: "source-ledger", JobID: "source-ledger-vector-generation:op", Status: "active",
			Path: "/data", Role: "sqlite-vector-generation", Peak: 4096,
		}, true, nil
	}
	reattach := func(receipt string) GenerationDiskJobReattach {
		return func(context.Context, string, string, string) (GenerationDiskJobLease, error) {
			return &capacityLeaseFake{receipt: receipt}, nil
		}
	}
	adapter := func(mutate func(*CapacityGenerationDiskAdmission)) *CapacityGenerationDiskAdmission {
		a := &CapacityGenerationDiskAdmission{OwnerID: "source-ledger", Path: "/data/source-ledger.db", Reserve: lease("reserved")}
		mutate(a)
		return a
	}
	cases := []struct {
		name    string
		adapter *CapacityGenerationDiskAdmission
		peak    int64
		want    string
	}{
		{"reserves when no job survived", adapter(func(a *CapacityGenerationDiskAdmission) { a.Find, a.Reattach = findNone, reattach("unused") }), 4096, "reserved"},
		{"reattaches the surviving job", adapter(func(a *CapacityGenerationDiskAdmission) { a.Find, a.Reattach = findActive, reattach("surviving") }), 4096, "surviving"},
		{"nil adapter", nil, 4096, ""},
		{"no reserve callback", adapter(func(a *CapacityGenerationDiskAdmission) { a.Reserve = nil }), 4096, ""},
		{"blank owner", adapter(func(a *CapacityGenerationDiskAdmission) { a.OwnerID = " " }), 4096, ""},
		{"negative floor", adapter(func(a *CapacityGenerationDiskAdmission) { a.FloorBytes = -1 }), 4096, ""},
		{"non-positive peak", adapter(func(*CapacityGenerationDiskAdmission) {}), 0, ""},
		{"lookup without reattachment", adapter(func(a *CapacityGenerationDiskAdmission) { a.Find = findNone }), 4096, ""},
		{"reattachment without lookup", adapter(func(a *CapacityGenerationDiskAdmission) { a.Reattach = reattach("x") }), 4096, ""},
		{"surviving job with another peak", adapter(func(a *CapacityGenerationDiskAdmission) { a.Find, a.Reattach = findActive, reattach("surviving") }), 8192, ""},
		{"reattached under another receipt", adapter(func(a *CapacityGenerationDiskAdmission) { a.Find, a.Reattach = findActive, reattach("other") }), 4096, ""},
		{"blank reserved receipt", adapter(func(a *CapacityGenerationDiskAdmission) { a.Reserve = lease(" ") }), 4096, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			receipt, err := tc.adapter.Admit(context.Background(), "op", "generation", "recipe", tc.peak)
			if tc.want == "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, receipt)
		})
	}
}
