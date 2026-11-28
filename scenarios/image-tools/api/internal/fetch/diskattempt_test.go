package fetch

import (
	"context"
	"strings"
	"testing"

	"image-tools/internal/disktest"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

func attemptRequest(job string) sharedcapacity.DiskJobRequest {
	return sharedcapacity.DiskJobRequest{OwnerID: "image-tools:model-install", JobID: job, Paths: []sharedcapacity.DiskJobPath{{Path: "/data", Role: "model", PeakBytes: 1}}}
}

// The ledger keeps one receipt per (owner, job key): reserving the durable key
// twice is refused. ReserveAttempt is what lets a requeued job reserve again.
func TestReserveAttemptSurvivesEveryEarlierRowOfTheSameDurableJob(t *testing.T) {
	ctx := context.Background()
	const owner, job = "image-tools:model-install", "job-1"

	var l disktest.Ledger
	if _, err := l.ReserveDiskJob(ctx, attemptRequest(job)); err != nil {
		t.Fatal(err)
	}
	if _, err := l.ReserveDiskJob(ctx, attemptRequest(job)); err == nil {
		t.Fatal("the ledger accepted a reused (owner, job) key; the fake no longer models the contract")
	}

	// A row an earlier process left under the durable key itself, and one left
	// by an earlier attempt of this durable job whose producer is gone.
	l.Seed(owner, job, "completed", false)
	dead := l.Seed(owner, AttemptPrefix(job)+"old-dead", "active", false)
	// A live producer's lease, another job's lease and another owner's lease
	// under the same prefix are not this attempt's to close.
	live := l.Seed(owner, AttemptPrefix(job)+"old-live", "active", true)
	other := l.Seed(owner, AttemptPrefix("job-2")+"x", "active", false)
	foreign := l.Seed("someone-else", AttemptPrefix(job)+"x", "active", false)

	lease, err := ReserveAttempt(ctx, &l, attemptRequest(job))
	if err != nil {
		t.Fatalf("resume with earlier rows present: %v", err)
	}
	if got := lease.Reservation().JobID; !strings.HasPrefix(got, AttemptPrefix(job)) || got == dead.JobID {
		t.Fatalf("attempt key %q is not a new key under %q", got, AttemptPrefix(job))
	}
	status := map[string]string{}
	for _, r := range l.Rows() {
		status[r.ID] = r.Status
	}
	if status[dead.ID] != "completed" {
		t.Errorf("dead predecessor still %q: its reservation would be counted twice", status[dead.ID])
	}
	for name, id := range map[string]string{"live producer": live.ID, "other job": other.ID, "other owner": foreign.ID} {
		if status[id] != "active" {
			t.Errorf("%s lease was closed (%q)", name, status[id])
		}
	}
	// Retrying again, in the same process, also reserves.
	if _, err := ReserveAttempt(ctx, &l, attemptRequest(job)); err != nil {
		t.Fatalf("second attempt: %v", err)
	}
}

// An admitter that cannot supersede (no receipts to close) still reserves.
type plainAdmitter struct{ keys []string }

func (p *plainAdmitter) ReserveDiskJob(_ context.Context, r sharedcapacity.DiskJobRequest) (sharedcapacity.DiskJobLease, error) {
	p.keys = append(p.keys, r.JobID)
	var l disktest.Ledger
	return l.ReserveDiskJob(context.Background(), r)
}

func TestReserveAttemptWithoutSupersederAndWithoutAdmitter(t *testing.T) {
	p := &plainAdmitter{}
	if _, err := ReserveAttempt(context.Background(), p, attemptRequest("job-1")); err != nil {
		t.Fatal(err)
	}
	if len(p.keys) != 1 || !strings.HasPrefix(p.keys[0], AttemptPrefix("job-1")) {
		t.Fatalf("keys = %v", p.keys)
	}
	if _, err := ReserveAttempt(context.Background(), nil, attemptRequest("job-1")); err == nil {
		t.Fatal("a nil admitter must refuse")
	}
}

func TestIsAttemptOf(t *testing.T) {
	for _, c := range []struct {
		durable, key string
		want         bool
	}{
		{"job-1", "job-1", true},
		{"job-1", "job-1/attempt/abc", true},
		{"job-1", "job-1/companions/attempt/abc", false},
		{"job-1", "job-10", false},
		{"job-1", "job-10/attempt/abc", false},
		{"", "", false},
	} {
		if got := IsAttemptOf(c.durable, c.key); got != c.want {
			t.Errorf("IsAttemptOf(%q, %q) = %v, want %v", c.durable, c.key, got, c.want)
		}
	}
}
