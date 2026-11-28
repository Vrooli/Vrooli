package domain

import (
	"testing"
	"time"
)

func TestCloudOperationHasLiveLease(t *testing.T) {
	now := time.Date(2026, time.November, 22, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Second)
	equal := now
	past := now.Add(-time.Second)

	tests := []struct {
		name string
		op   CloudOperation
		want bool
	}{
		{name: "no owner", op: CloudOperation{LeaseExpiresAt: &future}},
		{name: "no expiry", op: CloudOperation{WorkerID: "worker-a"}},
		{name: "future expiry", op: CloudOperation{WorkerID: "worker-a", LeaseExpiresAt: &future}, want: true},
		{name: "expiry at now is expired", op: CloudOperation{WorkerID: "worker-a", LeaseExpiresAt: &equal}},
		{name: "past expiry", op: CloudOperation{WorkerID: "worker-a", LeaseExpiresAt: &past}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.op.HasLiveLease(now); got != test.want {
				t.Fatalf("HasLiveLease() = %t, want %t", got, test.want)
			}
		})
	}
}
