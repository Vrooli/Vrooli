package resources

import (
	"context"
	"errors"
	platform "github.com/vrooli/platform-go"
	"testing"
	"time"
)

func TestOpenRGBPermissionMutationExcludesConcurrentNativeLifecycle(t *testing.T) {
	c, _, _ := openRGBHeadlessFixture(t)
	changes := 0
	err := c.withOpenRGBDevicePermissionFence(t.Context(), func() error {
		changes++
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()
		release, err := acquireManagedServiceLifecycleLock(ctx, openRGBResourceName)
		if err == nil {
			release()
			t.Fatal("native start/restart acquired lifecycle during permission mutation")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected lock failure: %v", err)
		}
		return nil
	})
	if err != nil || changes != 1 {
		t.Fatalf("permission fence=%v changes=%d", err, changes)
	}
	release, err := acquireManagedServiceLifecycleLock(t.Context(), openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	release()
}
func TestOpenRGBPermissionMutationRejectsEnabledResourceBeforeEffects(t *testing.T) {
	c, _, _ := openRGBHeadlessFixture(t)
	if _, err := c.operatorStateService().Apply(t.Context(), []byte(`{"resources":{"openrgb":{"enabled":true}}}`)); err != nil {
		t.Fatal(err)
	}
	changes := 0
	if err := c.withOpenRGBDevicePermissionFence(t.Context(), func() error { changes++; return nil }); err == nil || changes != 0 {
		t.Fatalf("enabled permission mutation=%v changes=%d", err, changes)
	}
}

func TestOpenRGBPermissionMutationRequiresInitialAdmissionRetirement(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending", true: "stopped-without-revoked-fence"}[stopped], func(t *testing.T) {
			c, _, _ := openRGBHeadlessFixture(t)
			x := newInitialExecFixture(t)
			store, err := newOpenRGBInitialStore()
			if err != nil {
				t.Fatal(err)
			}
			a := x.a
			a.Runtime.StateDir = store.dir
			a.Runtime.Root = c.Root
			a.Runtime.Home = c.Home
			a.Intent, err = platform.NewServiceIntent("resource-initial", a.ID, a.wrapperSpec())
			if err != nil {
				t.Fatal(err)
			}
			if err = store.prepare(t.Context(), a); err != nil {
				t.Fatal(err)
			}
			if stopped {
				_, err = store.change(t.Context(), a, func(a *openRGBInitialAttempt) error {
					now := time.Now().UTC()
					a.State = openRGBInitialStopped
					a.StoppedAt = &now
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			effects := 0
			if err = c.withOpenRGBDevicePermissionFence(t.Context(), func() error { effects++; return nil }); err == nil || effects != 0 {
				t.Fatalf("unretired admission authorized permission change: %v effects=%d", err, effects)
			}
		})
	}
}
