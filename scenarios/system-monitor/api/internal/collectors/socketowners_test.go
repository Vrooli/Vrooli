package collectors

import (
	"testing"
)

func TestTopSocketOwnersRanksByCountThenPID(t *testing.T) {
	counts := map[int]int{10: 5, 11: 53474, 12: 5, 13: 158}
	names := map[int]string{10: "a", 11: "agent-manager-a", 12: "c", 13: "git-control-tow"}

	owners := topSocketOwners(counts, names, 3)
	if len(owners) != 3 {
		t.Fatalf("len=%d, want 3 (limit honored)", len(owners))
	}
	if owners[0].PID != 11 || owners[0].Count != 53474 || owners[0].Comm != "agent-manager-a" {
		t.Fatalf("top owner = %+v", owners[0])
	}
	if owners[1].PID != 13 {
		t.Fatalf("second owner = %+v, want pid 13", owners[1])
	}
	// Ties break on PID so repeated cycles do not reshuffle the table.
	if owners[2].PID != 10 {
		t.Fatalf("tie broken unstably: %+v", owners[2])
	}
}

func TestTopSocketOwnersUnlimited(t *testing.T) {
	owners := topSocketOwners(map[int]int{1: 1, 2: 2}, map[int]string{}, 0)
	if len(owners) != 2 {
		t.Fatalf("len=%d, want 2 when limit is 0", len(owners))
	}
}

func TestShouldAttributeSocketsHonorsThreshold(t *testing.T) {
	t.Setenv(socketAttributionThresholdEnv, "100")
	if shouldAttributeSockets(99) {
		t.Fatal("attributed below threshold; the /proc walk is not free")
	}
	if !shouldAttributeSockets(100) {
		t.Fatal("did not attribute at the threshold")
	}
}

func TestShouldAttributeSocketsDisabledByZero(t *testing.T) {
	t.Setenv(socketAttributionThresholdEnv, "0")
	if shouldAttributeSockets(1_000_000) {
		t.Fatal("threshold 0 must disable attribution entirely")
	}
}

func TestSocketAttributionThresholdFallsBackOnGarbage(t *testing.T) {
	t.Setenv(socketAttributionThresholdEnv, "not-a-number")
	if got := socketAttributionThreshold(); got != defaultSocketAttributionThreshold {
		t.Fatalf("threshold=%d, want default %d", got, defaultSocketAttributionThreshold)
	}
	t.Setenv(socketAttributionThresholdEnv, "-5")
	if got := socketAttributionThreshold(); got != defaultSocketAttributionThreshold {
		t.Fatalf("negative threshold=%d, want default %d", got, defaultSocketAttributionThreshold)
	}
}
