package runtimesupervisorsafeguard

import (
	"context"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

func TestServingVerificationRequiresTheNewImageAndAFreshRenewedLease(t *testing.T) {
	sha, build := strings.Repeat("a", 64), strings.Repeat("b", 64)
	now := time.Now()
	healthy := func() (platform.CoreServiceState, string, supervisorLease) {
		return platform.CoreServiceState{LoadState: "loaded", ActiveState: "active", SubState: "running", PID: 42, NeedDaemonReload: "no"},
			sha,
			supervisorLease{PID: 42, Build: build, Incarnation: "inc", Started: now.Add(-time.Second), Heartbeat: now, Deadline: now.Add(time.Minute), FleetRenewed: true}
	}
	ops := func(state platform.CoreServiceState, image string, lease supervisorLease) activationOps {
		return activationOps{
			inspect:     func(context.Context) (platform.CoreServiceState, error) { return state, nil },
			image:       func(context.Context, int) (string, error) { return image, nil },
			incarnation: func(int) (string, error) { return "inc", nil },
			lease:       func(context.Context, string) (supervisorLease, error) { return lease, nil },
		}
	}
	state, image, lease := healthy()
	if err := observeServing(context.Background(), ops(state, image, lease), "/home", sha, build); err != nil {
		t.Fatalf("healthy serving image was refused: %v", err)
	}
	for name, mutate := range map[string]func(*platform.CoreServiceState, *string, *supervisorLease){
		"inactive":        func(s *platform.CoreServiceState, _ *string, _ *supervisorLease) { s.ActiveState = "activating" },
		"prior image":     func(_ *platform.CoreServiceState, i *string, _ *supervisorLease) { *i = strings.Repeat("c", 64) },
		"stale build":     func(_ *platform.CoreServiceState, _ *string, l *supervisorLease) { l.Build = strings.Repeat("d", 64) },
		"other process":   func(_ *platform.CoreServiceState, _ *string, l *supervisorLease) { l.PID = 7 },
		"expired lease":   func(_ *platform.CoreServiceState, _ *string, l *supervisorLease) { l.Deadline = now.Add(-time.Second) },
		"fleet unrenewed": func(_ *platform.CoreServiceState, _ *string, l *supervisorLease) { l.FleetRenewed = false },
	} {
		t.Run(name, func(t *testing.T) {
			state, image, lease := healthy()
			mutate(&state, &image, &lease)
			if err := observeServing(context.Background(), ops(state, image, lease), "/home", sha, build); err == nil {
				t.Fatal("unverified serving state was accepted")
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	state.ActiveState = "failed"
	if err := waitServing(ctx, ops(state, image, lease), "/home", sha, build); err == nil || !strings.Contains(err.Error(), "did not verify") {
		t.Fatalf("waiting did not end with the last observation: %v", err)
	}
}
