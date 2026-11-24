//go:build linux

package capacity

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// exitedProcess starts a child, records its identity, and returns it after it
// has exited and been reaped.
func exitedProcess(t *testing.T) (pid int, start string) {
	t.Helper()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid = cmd.Process.Pid
	start, err := processStartIdentity(pid)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	return pid, start
}

func TestExpireStaleClaimsReleasesProcessBoundClaimsOfExitedOwnersOnly(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLiteStore(ctx, Config{DBPath: t.TempDir() + "/capacity.db"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	deadPID, deadStart := exitedProcess(t)
	livePID := os.Getpid()
	liveStart, err := processStartIdentity(livePID)
	if err != nil {
		t.Fatal(err)
	}
	claim := func(owner string) CapacityClaim {
		c, err := store.CreateClaim(ctx, CapacityClaim{OwnerKind: OwnerKindOp, OwnerID: owner, ResourceKind: ResourceKindRAM, AmountBytes: 1 << 20, PreferredBytes: 1 << 20, FloorBytes: 1 << 20, Status: StatusGranted}, 15*time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	cases := map[string]struct {
		owner       string
		wantExpired bool
	}{
		"exited owner with start identity":     {fmt.Sprintf("cli-install:pid-%d-%s:resource-a-1", deadPID, deadStart), true},
		"exited owner, pid-only identity":      {fmt.Sprintf("cli-install:pid-%d:resource-b-1", deadPID), true},
		"live owner":                           {fmt.Sprintf("cli-install:pid-%d-%s:resource-c-1", livePID, liveStart), false},
		"live pid-only owner":                  {fmt.Sprintf("cli-install:pid-%d:resource-d-1", livePID), false},
		"reused pid is a different process":    {fmt.Sprintf("cli-install:pid-%d-%s:resource-e-1", livePID, "1"), true},
		"unrelated op claim naming a dead pid": {fmt.Sprintf("other-scenario:pid-%d:phase", deadPID), false},
		"unparseable pid":                      {"cli-install:pid-notanumber:resource-f-1", false},
	}
	claims := map[string]CapacityClaim{}
	for name, tc := range cases {
		claims[name] = claim(tc.owner)
	}
	expired, err := store.ExpireStaleClaims(ctx, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range expired {
		got[c.ClaimID] = true
	}
	for name, tc := range cases {
		stored, err := store.GetClaim(ctx, claims[name].ClaimID)
		if err != nil {
			t.Fatal(err)
		}
		if got[claims[name].ClaimID] != tc.wantExpired || (stored.Status == StatusExpired) != tc.wantExpired {
			t.Errorf("%s: reported expired=%t, stored status=%s, want expired=%t", name, got[claims[name].ClaimID], stored.Status, tc.wantExpired)
		}
	}
}
