package lightingruntime

import (
	"context"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestResourceUpdateConvergesVerifiedSelectionDespiteLostAcknowledgement(t *testing.T) {
	_, controller, fixture := newLightingFixture(t)
	old := selectedLighting()
	if err := controller.Enable(t.Context(), old); err != nil {
		t.Fatal(err)
	}
	preparation := &Preparation{Source: PreparationManaged, Selection: old, Qualified: true}
	if err := controller.write(Intent{SchemaVersion: lightingStateVersion, Phase: Enabled, Selection: &old, Preparation: preparation}); err != nil {
		t.Fatal(err)
	}
	next := old
	next.Version = "1.2"
	publicationErr := errors.New("publication acknowledgement lost")
	if err := controller.WithResourceUpdate(t.Context(), func(snapshot Intent) (*Selection, error) {
		if *snapshot.Selection != old {
			t.Fatalf("unexpected snapshot: %+v", snapshot)
		}
		// Neither pointer belongs to the transaction's retained snapshot.
		snapshot.Selection.Version = "callback-mutation"
		snapshot.Preparation.Selection.Version = "callback-mutation"
		return &next, publicationErr
	}); !errors.Is(err, publicationErr) {
		t.Fatalf("lost acknowledgement error = %v", err)
	}
	state, err := controller.Current()
	if err != nil || *state.Selection != next || state.Preparation != nil {
		t.Fatalf("verified publication did not converge: %+v, %v", state, err)
	}
	// The resource can return the same authoritative receipt on retry.
	if err := controller.WithResourceUpdate(t.Context(), func(snapshot Intent) (*Selection, error) {
		if *snapshot.Selection != next {
			t.Fatalf("retry snapshot = %+v", snapshot)
		}
		return &next, nil
	}); err != nil {
		t.Fatal(err)
	}
	fixture.backend.Incarnation = "boot-a:updated-process"
	if err := controller.Reconcile(t.Context()); err != nil {
		t.Fatal(err)
	}
	if fixture.selection != next || fixture.bound.Incarnation != "boot-a:updated-process" {
		t.Fatalf("updated selection did not reach reconciler: %+v, %+v", fixture.selection, fixture.bound)
	}
}

func TestResourceUpdatePreservesDisabledIntentAndMatchingPreparation(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			_, controller, _ := newLightingFixture(t)
			selection := selectedLighting()
			before := Intent{SchemaVersion: lightingStateVersion, Phase: Disabled,
				Preparation: &Preparation{Source: PreparationExisting, Selection: selection, Qualified: true}}
			if enabled {
				before.Phase, before.Selection = Enabled, &selection
			}
			if err := controller.write(before); err != nil {
				t.Fatal(err)
			}
			bytesBefore, err := os.ReadFile(controller.path)
			if err != nil {
				t.Fatal(err)
			}
			if err := controller.WithResourceUpdate(t.Context(), func(snapshot Intent) (*Selection, error) {
				snapshot.Phase = Disabling
				snapshot.Preparation.Qualified = false
				return &selection, nil
			}); err != nil {
				t.Fatal(err)
			}
			state, err := controller.Current()
			if err != nil || !reflect.DeepEqual(state, before) {
				t.Fatalf("unchanged selection changed intent: %+v, %v", state, err)
			}
			bytesAfter, err := os.ReadFile(controller.path)
			if err != nil || string(bytesBefore) != string(bytesAfter) {
				t.Fatalf("unchanged intent contents changed: %s, %v", bytesAfter, err)
			}
		})
	}
}

func TestResourceUpdateRejectsPendingDisableAndInvalidReceipts(t *testing.T) {
	_, controller, _ := newLightingFixture(t)
	old := selectedLighting()
	if err := controller.write(Intent{SchemaVersion: lightingStateVersion, Phase: Disabling, Selection: &old}); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := controller.WithResourceUpdate(t.Context(), func(Intent) (*Selection, error) {
		called = true
		return nil, nil
	}); err == nil || called {
		t.Fatalf("pending disable admitted update: called=%t, err=%v", called, err)
	}
	if err := controller.write(Intent{SchemaVersion: lightingStateVersion, Phase: Enabled, Selection: &old}); err != nil {
		t.Fatal(err)
	}
	cause := errors.New("resource failed")
	for _, result := range []*Selection{nil, {Version: "invalid"}} {
		err := controller.WithResourceUpdate(t.Context(), func(snapshot Intent) (*Selection, error) {
			snapshot.Selection.Version = "mutated"
			return result, cause
		})
		state, readErr := controller.Current()
		if !errors.Is(err, cause) || readErr != nil || *state.Selection != old {
			t.Fatalf("invalid/unverified receipt changed selection: %+v, %v, %v", state, err, readErr)
		}
	}
}

func TestResourceUpdateSerializesNewUserChoiceAndResourceLock(t *testing.T) {
	for _, disable := range []bool{false, true} {
		t.Run(map[bool]string{false: "new-selection", true: "disable"}[disable], func(t *testing.T) {
			_, controller, fixture := newLightingFixture(t)
			old := selectedLighting()
			if err := controller.Enable(t.Context(), old); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
			defer cancel()
			var resource sync.Mutex
			entered, finish := make(chan struct{}), make(chan struct{})
			updated := make(chan error, 1)
			next := old
			next.Version = "resource-update"
			go func() {
				updated <- controller.WithResourceUpdate(ctx, func(Intent) (*Selection, error) {
					resource.Lock()
					defer resource.Unlock()
					close(entered)
					select {
					case <-finish:
						return &next, nil
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				})
			}()
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			// A competing intent operation cannot enter its resource owner while
			// this callback holds the fence. Cancellation bounds that assertion.
			blocked, cancelBlocked := context.WithCancel(ctx)
			cancelBlocked()
			if err := controller.Enable(blocked, old); !errors.Is(err, context.Canceled) {
				t.Fatalf("competing intent bypassed update fence: %v", err)
			}
			choice := old
			choice.Version = "newer-user-choice"
			chosen := make(chan error, 1)
			go func() {
				if disable {
					chosen <- controller.Disable(ctx)
				} else {
					chosen <- controller.Enable(ctx, choice)
				}
			}()
			fixture.ensure = func(context.Context) error {
				resource.Lock()
				defer resource.Unlock()
				return nil
			}
			reconciled := make(chan error, 1)
			go func() { reconciled <- controller.Reconcile(ctx) }()
			close(finish)
			for _, done := range []chan error{updated, chosen, reconciled} {
				select {
				case err := <-done:
					if err != nil {
						t.Fatal(err)
					}
				case <-ctx.Done():
					t.Fatal("intent/resource lock order deadlocked")
				}
			}
			if err := controller.Reconcile(ctx); err != nil {
				t.Fatal(err)
			}
			state, err := controller.Current()
			if err != nil || (disable && state.Phase != Disabled) || (!disable && *state.Selection != choice) {
				t.Fatalf("newer user choice was lost: %+v, %v", state, err)
			}
		})
	}
}

func TestResourceUpdateWaitsForEarlierUserIntentBeforeCallingResource(t *testing.T) {
	_, controller, _ := newLightingFixture(t)
	old := selectedLighting()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	entered, finish := make(chan struct{}), make(chan struct{})
	enabled := make(chan error, 1)
	go func() {
		enabled <- controller.EnablePrepared(ctx, old, func() error {
			close(entered)
			select {
			case <-finish:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	// An update that cannot acquire the intent fence must not acquire its
	// resource lock or perform any lifecycle action through the callback.
	canceled, cancelUpdate := context.WithCancel(ctx)
	cancelUpdate()
	called := false
	if err := controller.WithResourceUpdate(canceled, func(Intent) (*Selection, error) {
		called = true
		return nil, nil
	}); !errors.Is(err, context.Canceled) || called {
		t.Fatalf("resource callback bypassed user intent fence: called=%t, err=%v", called, err)
	}
	close(finish)
	select {
	case err := <-enabled:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if err := controller.WithResourceUpdate(ctx, func(snapshot Intent) (*Selection, error) {
		if snapshot.Phase != Enabled || *snapshot.Selection != old {
			t.Fatalf("update observed superseded user intent: %+v", snapshot)
		}
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
}
