package capacity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

type fakeResidentLedger struct {
	claims   []vroolicli.CapacityClaimInfo
	released []string
	failOn   string
}

func (f *fakeResidentLedger) ListActiveCapacityClaims(_ context.Context, owns func(string) bool) ([]vroolicli.CapacityClaimInfo, error) {
	var out []vroolicli.CapacityClaimInfo
	for _, c := range f.claims {
		if owns(c.OwnerID) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeResidentLedger) ReleaseCapacityClaim(_ context.Context, id string) error {
	if id == f.failOn {
		return errors.New("release timed out")
	}
	f.released = append(f.released, id)
	return nil
}

// deadToken is the token of a process that has exited.
func deadToken(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Skipf("cannot run a short process: %v", err)
	}
	return fmt.Sprintf("p%d-1", cmd.Process.Pid)
}

func TestResidentOwnerIDCarriesTheProcessTokenAndSuffix(t *testing.T) {
	ram := ResidentOwnerID("hyperswap-1c-256")
	gpu := ResidentOwnerID("hyperswap-1c-256", "GPU-uuid")
	if !strings.HasPrefix(ram, ResidentOwnerPrefix+"hyperswap-1c-256@"+ProcessToken()) {
		t.Fatalf("ram owner = %q", ram)
	}
	if !strings.HasSuffix(gpu, ":GPU-uuid") || !strings.Contains(gpu, ProcessToken()) {
		t.Fatalf("gpu owner = %q", gpu)
	}
	if IsOrphanedResident(ram) || IsOrphanedResident(gpu) {
		t.Fatal("this process's own residents were classed as orphans")
	}
}

func TestOrphanedResidentClassification(t *testing.T) {
	dead := deadToken(t)
	cases := []struct {
		owner string
		want  bool
	}{
		{ResidentOwnerPrefix + "m@" + dead, true},                       // owner process exited
		{ResidentOwnerPrefix + "m@" + dead + ":GPU-1", true},            // its GPU claim too
		{ResidentOwnerPrefix + "m", true},                               // before tokens existed: the previous process is gone
		{ResidentOwnerPrefix + "m:GPU-1", true},                         // same, GPU form
		{ResidentOwnerPrefix + "m@" + ProcessToken(), false},            // this process
		{ResidentOwnerPrefix + "m@p1-99999999999", true},                // recycled pid: start time differs
		{"image-tools:d56a5d6e-a75d-49c7-816f-125c21800bd2", false},     // a job claim is not a resident
		{"other-scenario:resident:m", false},                            // someone else's
		{ResidentOwnerPrefix + "m@p" + fmt.Sprint(os.Getppid()), false}, // a live process without a start time
	}
	for _, c := range cases {
		if got := IsOrphanedResident(c.owner); got != c.want {
			t.Errorf("IsOrphanedResident(%q) = %v, want %v", c.owner, got, c.want)
		}
	}
}

func TestBootReconciliationReleasesOnlyOrphanedResidentsAndKeepsGoingPastAFailure(t *testing.T) {
	dead := deadToken(t)
	ledger := &fakeResidentLedger{
		failOn: "stuck",
		claims: []vroolicli.CapacityClaimInfo{
			{ClaimID: "ram", OwnerID: ResidentOwnerPrefix + "hyperswap-1c-256", ResourceKind: "ram"},
			{ClaimID: "vram", OwnerID: ResidentOwnerPrefix + "hyperswap-1c-256@" + dead + ":GPU-1", ResourceKind: "vram"},
			{ClaimID: "stuck", OwnerID: ResidentOwnerPrefix + "florence-2@" + dead, ResourceKind: "ram"},
			{ClaimID: "live", OwnerID: ResidentOwnerID("florence-2"), ResourceKind: "ram"},
			{ClaimID: "job", OwnerID: "image-tools:5a5ce743-0000", ResourceKind: "vram"},
		},
	}
	released, err := releaseOrphanedResidents(context.Background(), ledger)
	if err == nil || !strings.Contains(err.Error(), "release timed out") {
		t.Fatalf("a failed release must be reported, got %v", err)
	}
	if released != 2 || strings.Join(ledger.released, ",") != "ram,vram" {
		t.Fatalf("released %d %v, want the two orphans (ram, vram) and nothing live or job-owned", released, ledger.released)
	}
}
