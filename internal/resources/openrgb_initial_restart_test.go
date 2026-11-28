package resources

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/vrooli/vrooli/internal/config"
	"os"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/lightingruntime"
)

// Reproduce the normal disable -> enable transition: canonical serving metadata
// still describes the retired launch when the next wrapper resolves selection.
func newOpenRGBRetiredRestartFixture(t *testing.T, prepareActual bool) (*openRGBPublicFixture, *openRGBInitialStore, *managedUpdateExecStore, ManagedServiceState) {
	t.Helper()
	f := newOpenRGBPublicFixture(t)
	store, err := newOpenRGBInitialStore()
	if err != nil {
		t.Fatal(err)
	}
	fences, err := newManagedInitialExecStore(f.initial.ID)
	if err != nil {
		t.Fatal(err)
	}
	ref := f.original.InitialLaunch.NativeRef
	err = retireOpenRGBInitialWith(t.Context(), store, f.initial, openRGBInitialCleanupOwner{
		revoke: fences.Revoke,
		lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return ref, nil },
		stop: func(context.Context, platform.ServiceRef) (platform.OperationStopReceipt, error) {
			return platform.OperationStopReceipt{NativeQuiescent: true}, nil
		},
		released: func(context.Context, platform.ServiceRef) (bool, error) { return true, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	stopped := f.original
	now := time.Now().UTC()
	stopped.StoppedAt = &now
	if err := f.supervisor.writeState(stopped); err != nil {
		t.Fatal(err)
	}
	var next openRGBInitialAttempt
	if prepareActual {
		launch, e := NewOpenRGBHeadless(f.c, nil).inspectSelection(t.Context(), lightingruntime.Selection{Version: stopped.ArtifactVersion, SHA256: stopped.ArtifactSHA256, DevicePolicyID: f.initial.Policy.ID})
		if e != nil {
			t.Fatal(e)
		}
		next, err = f.c.prepareOpenRGBInitialAttempt(t.Context(), launch)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		next = f.initial
		next.ID = "ms-restart-next"
		next.State = openRGBInitialPrepared
		next.Intent, err = platform.NewServiceIntent("resource-initial", next.ID, next.wrapperSpec())
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = store.prepare(t.Context(), next); err != nil {
		t.Fatal(err)
	}
	if _, err = store.submit(t.Context(), next); err != nil {
		t.Fatal(err)
	}
	return f, store, fences, stopped
}

func TestOpenRGBInitialRestartRetainsStoppedProfileAfterNextAdmission(t *testing.T) {
	f, store, _, stopped := newOpenRGBRetiredRestartFixture(t, true)
	launch, err := NewOpenRGBHeadless(f.c, nil).inspectSelection(t.Context(), lightingruntime.Selection{Version: stopped.ArtifactVersion, SHA256: stopped.ArtifactSHA256, DevicePolicyID: f.initial.Policy.ID})
	if err != nil {
		t.Fatalf("normal next wrapper cannot resolve retired selected profile: %v", err)
	}
	if launch.retained == nil || launch.policy.Artifact != f.initial.Contract.Artifact || string(launch.retained.Catalog) != string(f.initial.Catalog) {
		t.Fatal("restart did not retain exact original artifact and catalog")
	}
	// Retained metadata never supplies execution authority: the new admission
	// must consume its own distinct native claim, and cannot consume it twice.
	id, err := store.current()
	if err != nil {
		t.Fatal(err)
	}
	next, err := store.read(id)
	if err != nil {
		t.Fatal(err)
	}
	nextFence, err := newManagedInitialExecStore(next.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = nextFence.Prepare(t.Context(), next.fence()); err != nil {
		t.Fatal(err)
	}
	ref := f.original.InitialLaunch.NativeRef
	ref.Intent = platform.OperationIntent(next.Intent)
	ref.Unit, _ = next.Intent.UnitName()
	ref.Scope.Path = "/user.slice/" + platform.ServicesSlice + "/" + ref.Unit
	witness := openRGBExecWitness{DeviceGenerationSHA256: updateTestDigest("new device generation"), Policy: next.Policy, Incarnation: "boot:" + ref.RootStartTime, ClaimSHA256: next.claimDigest(ref), ContextSHA256: updateTestDigest("new confinement"), SelectedNodes: []string{"/dev/hidraw9"}}
	if _, err = nextFence.ClaimOpenRGB(t.Context(), next.fence(), ref, witness); err != nil {
		t.Fatal(err)
	}
	if _, err = nextFence.ClaimOpenRGB(t.Context(), next.fence(), ref, witness); !errors.Is(err, ErrManagedUpdateExecClaimed) {
		t.Fatalf("new admission consumed twice: %v", err)
	}
	if _, _, err := readOpenRGBInitialServing(f.original); err == nil {
		t.Fatal("superseded live publication accepted")
	}
}

func TestOpenRGBInitialRestartRejectsUncertainOrChangedHistoricalIdentity(t *testing.T) {
	cases := map[string]func(*ManagedServiceState, *openRGBInitialAttempt, *managedUpdateExecFence){
		"canonical still live": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) { s.StoppedAt = nil },
		"canonical zero stop": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			s.StoppedAt = &time.Time{}
		},
		"admission not retired": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			a.State = openRGBInitialServing
			a.StoppedAt = nil
			a.QuiescentAt = nil
		},
		"no admission stop":             func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) { a.StoppedAt = nil },
		"no quiescence":                 func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) { a.QuiescentAt = nil },
		"no native retirement identity": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) { a.NativeRef = nil },
		"different retired invocation": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			a.NativeRef.RootPID++
		},
		"claim not revoked": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			f.State = managedUpdateExecFenceClaimed
			f.RevokedAt = nil
		},
		"revocation not durable": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) { f.RevokedAt = nil },
		"missing claim":          func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) { f.Claim = nil },
		"missing witness": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			f.OpenRGBWitness = nil
		},
		"different incarnation": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			f.OpenRGBWitness.Incarnation = "other:" + f.Claim.RootStartTime
		},
		"different canonical artifact": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			s.ArtifactSHA256 = updateTestDigest("other artifact")
		},
		"different canonical token": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			s.OwnershipTokenHash = updateTestDigest("other token")
		},
		"different claimed start": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			s.StartedAt = s.StartedAt.Add(time.Second)
		},
		"internally valid different fence policy": func(s *ManagedServiceState, a *openRGBInitialAttempt, f *managedUpdateExecFence) {
			f.OpenRGBPolicy.ID = updateTestDigest("other policy")
			f.OpenRGBWitness.Policy = *f.OpenRGBPolicy
			f.OpenRGBWitness.ClaimSHA256 = f.openRGBClaimDigest(*f.Claim)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			_, store, fences, state := newOpenRGBRetiredRestartFixture(t, false)
			a, err := store.read(state.InitialLaunch.AttemptID)
			if err != nil {
				t.Fatal(err)
			}
			fence, err := fences.read()
			if err != nil {
				t.Fatal(err)
			}
			change(&state, &a, &fence)
			for path, value := range map[string]any{store.attemptPath(a.ID): a, fences.path: fence} {
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				if err = config.WriteOwnedFileDurable(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := readOpenRGBInitialServing(state); err == nil {
				t.Fatal("uncertain or changed retired publication accepted")
			}
		})
	}
	t.Run("missing current pointer", func(t *testing.T) {
		_, store, _, state := newOpenRGBRetiredRestartFixture(t, false)
		if err := os.Remove(store.currentPath()); err != nil {
			t.Fatal(err)
		}
		if _, _, err := readOpenRGBInitialServing(state); err == nil {
			t.Fatal("missing current pointer accepted")
		}
	})
}
