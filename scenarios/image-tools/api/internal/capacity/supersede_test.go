package capacity

import (
	"context"
	"strings"
	"testing"

	"image-tools/internal/disktest"
)

func TestSupersedeDiskJobsClosesOnlyTheOwnersExitedAttempts(t *testing.T) {
	ctx := context.Background()
	const owner, prefix = "image-tools:model-install", "job-1/attempt/"
	var l disktest.Ledger
	dead := l.Seed(owner, prefix+"a", "active", false)
	live := l.Seed(owner, prefix+"b", "active", true)
	done := l.Seed(owner, prefix+"c", "completed", false)
	other := l.Seed(owner, "job-2/attempt/a", "active", false)
	foreign := l.Seed("someone-else", prefix+"a", "active", false)

	closed, err := supersedeDiskJobs(ctx, &l, owner, func(key string) bool { return strings.HasPrefix(key, prefix) })
	if closed != 1 {
		t.Fatalf("closed %d leases, want 1", closed)
	}
	if err == nil {
		t.Fatal("the lease whose producer still runs must be reported, not silently skipped")
	}
	status := map[string]string{}
	for _, r := range l.Rows() {
		status[r.ID] = r.Status
	}
	want := map[string]string{dead.ID: "completed", live.ID: "active", done.ID: "completed", other.ID: "active", foreign.ID: "active"}
	for id, w := range want {
		if status[id] != w {
			t.Errorf("lease %s = %q, want %q", id, status[id], w)
		}
	}
}
