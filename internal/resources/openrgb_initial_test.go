package resources

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	platform "github.com/vrooli/platform-go"
)

type initialExecFixture struct {
	a        openRGBInitialAttempt
	o        openRGBInitialExecOwner
	store    *openRGBInitialStore
	fence    *managedUpdateExecStore
	ref      platform.ServiceRef
	replaced int
}

func newInitialExecFixture(t *testing.T) *initialExecFixture {
	t.Helper()
	x := newOpenRGBExecFixture(t)
	i := x.fence.Intent
	a := openRGBInitialAttempt{SchemaVersion: openRGBInitialSchemaVersion, ID: i.InstanceID, State: openRGBInitialPrepared, Contract: x.update.Target, Runtime: i.Runtime, Policy: *x.fence.OpenRGBPolicy}
	a.Intent, _ = platform.NewServiceIntent("resource-initial", a.ID, a.wrapperSpec())
	token := "initial-test-token"
	a.TokenHash = managedServiceTokenHash(token)
	if err := os.MkdirAll(a.root(), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.root(), "ownership-token"), []byte(token), 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.validate(); err != nil {
		t.Fatal(err)
	}
	ref := x.ref
	ref.Intent = platform.OperationIntent(a.Intent)
	ref.Unit, _ = a.Intent.UnitName()
	ref.Scope.Path = "/user.slice/" + platform.ServicesSlice + "/" + ref.Unit
	f := execFenceTestStore(filepath.Join(t.TempDir(), "initial-fence.json"), a.fence())
	f.admissionKind = managedExecAdmissionInitial
	if _, err := f.Prepare(t.Context(), a.fence()); err != nil {
		t.Fatal(err)
	}
	z := &initialExecFixture{a: a, ref: ref, fence: f, store: &openRGBInitialStore{dir: a.Runtime.StateDir, write: writeResourceOwnedFileDurable}}
	z.o = openRGBInitialExecOwner{lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return ref, nil }, selfImage: x.owner.selfImage, incarnation: x.owner.incarnation, confine: func() (func(), error) { return func() {}, nil }, image: x.owner.openRGBImage, claim: f.ClaimOpenRGB, chdir: func(string) error { return nil }, replace: func(*os.File, []string, []string) error { z.replaced++; return nil }}
	z.o.witness = func(ctx context.Context, a openRGBInitialAttempt, base managedUpdateInvocation, ref platform.ServiceRef, inc string) (openRGBExecWitness, error) {
		return x.owner.openRGBObserve(ctx, a.Contract, base, a.fence(), ref, inc)
	}
	return z
}
func TestOpenRGBInitialExecUsesOneClaimWithoutUpdateLedger(t *testing.T) {
	x := newInitialExecFixture(t)
	x.a.State = openRGBInitialSubmitted
	failed := errors.New("injected exec failure")
	x.o.replace = func(*os.File, []string, []string) error { x.replaced++; return failed }
	if err := executeOpenRGBInitial(t.Context(), x.a, x.o); !errors.Is(err, failed) || x.replaced != 1 {
		t.Fatalf("exec=%v count=%d", err, x.replaced)
	}
	// A replay must fail at the durable claim, before a second payload exec.
	x.o.witness = func(_ context.Context, a openRGBInitialAttempt, _ managedUpdateInvocation, ref platform.ServiceRef, inc string) (openRGBExecWitness, error) {
		f, e := x.fence.read()
		if e != nil {
			return openRGBExecWitness{}, e
		}
		return *f.OpenRGBWitness, nil
	}
	if err := executeOpenRGBInitial(t.Context(), x.a, x.o); !errors.Is(err, ErrManagedUpdateExecClaimed) || x.replaced != 1 {
		t.Fatalf("replay=%v count=%d", err, x.replaced)
	}
	f, e := x.fence.read()
	if e != nil || f.Claim == nil || f.OpenRGBWitness == nil || f.AdmissionKind != managedExecAdmissionInitial {
		t.Fatalf("lost witness: %+v %v", f, e)
	}
	update := f
	update.AdmissionKind = ""
	if update.openRGBClaimDigest(x.ref) == f.openRGBClaimDigest(x.ref) {
		t.Fatal("cross-admission digest collision")
	}
}
func TestOpenRGBInitialExecWitnessAndNativeDriftFailBeforeClaim(t *testing.T) {
	for _, mode := range []string{"access", "native", "launcher"} {
		t.Run(mode, func(t *testing.T) {
			x := newInitialExecFixture(t)
			x.a.State = openRGBInitialSubmitted
			switch mode {
			case "access":
				x.o.witness = func(context.Context, openRGBInitialAttempt, managedUpdateInvocation, platform.ServiceRef, string) (openRGBExecWitness, error) {
					return openRGBExecWitness{}, errors.New("denied")
				}
			case "native":
				count := 0
				x.o.lookup = func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) {
					count++
					ref := x.ref
					if count > 1 {
						ref.NativeInvocationID = strings.Repeat("f", 32)
					}
					return ref, nil
				}
			case "launcher":
				x.a.Runtime.LauncherSHA256 = updateTestDigest("changed")
			}
			if err := executeOpenRGBInitial(t.Context(), x.a, x.o); err == nil || x.replaced != 0 {
				t.Fatalf("unqualified exec=%v count=%d", err, x.replaced)
			}
			f, e := x.fence.read()
			if e != nil || f.State != managedUpdateExecFencePrepared {
				t.Fatalf("premature claim: %+v %v", f, e)
			}
		})
	}
}
func TestOpenRGBInitialAdmissionLostSubmitAcknowledgementNeverRedispatches(t *testing.T) {
	x := newInitialExecFixture(t)
	if err := x.store.prepare(t.Context(), x.a); err != nil {
		t.Fatal(err)
	}
	write := x.store.write
	x.store.write = func(path string, data []byte, mode os.FileMode) error {
		if err := write(path, data, mode); err != nil {
			return err
		}
		return errors.New("lost write acknowledgement")
	}
	if _, err := x.store.submit(t.Context(), x.a); err == nil {
		t.Fatal("lost acknowledgement reported success")
	}
	x.store.write = write
	current, e := x.store.read(x.a.ID)
	if e != nil || current.State != openRGBInitialSubmitted {
		t.Fatalf("submission lost: %+v %v", current, e)
	}
	if _, err := x.store.submit(t.Context(), current); err == nil {
		t.Fatal("uncertain submission replay admitted")
	}
	if err := x.store.prepare(t.Context(), x.a); err == nil {
		t.Fatal("pending attempt overwritten")
	}
}
func TestOpenRGBInitialCleanupRetainsQuiescenceBeforeReleaseAndResumes(t *testing.T) {
	x := newInitialExecFixture(t)
	if err := x.store.prepare(t.Context(), x.a); err != nil {
		t.Fatal(err)
	}
	x.a, _ = x.store.submit(t.Context(), x.a)
	stopped, released := 0, false
	o := openRGBInitialCleanupOwner{revoke: x.fence.Revoke, lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return x.ref, nil }, stop: func(context.Context, platform.ServiceRef) (platform.OperationStopReceipt, error) {
		stopped++
		return platform.OperationStopReceipt{NativeQuiescent: true}, nil
	}, released: func(context.Context, platform.ServiceRef) (bool, error) { return released, nil }, release: func(context.Context, platform.ServiceRef) error {
		a, e := x.store.read(x.a.ID)
		if e != nil || a.QuiescentAt == nil || a.NativeRef == nil || *a.NativeRef != x.ref {
			t.Fatal("release before durable exact quiescence")
		}
		released = true
		return errors.New("release reply lost")
	}}
	if err := retireOpenRGBInitialWith(t.Context(), x.store, x.a, o); err == nil {
		t.Fatal("lost release reply reported completion")
	}
	a, e := x.store.read(x.a.ID)
	if e != nil || a.State != openRGBInitialQuiescent {
		t.Fatalf("quiescence lost %+v %v", a, e)
	}
	o.lookup = func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) {
		return platform.ServiceRef{}, errors.New("unit released")
	}
	if err := retireOpenRGBInitialWith(t.Context(), x.store, a, o); err != nil {
		t.Fatal(err)
	}
	a, e = x.store.read(x.a.ID)
	if e != nil || a.State != openRGBInitialStopped || stopped != 1 {
		t.Fatalf("cleanup replay %+v %v stop=%d", a, e, stopped)
	}
	if _, err := x.fence.ClaimOpenRGB(t.Context(), x.a.fence(), x.ref, openRGBExecWitness{}); err == nil {
		t.Fatal("late delivery after cleanup executed")
	}
}

func TestOpenRGBInitialCleanupCannotTreatUnknownNativeAsAbsent(t *testing.T) {
	x := newInitialExecFixture(t)
	if err := x.store.prepare(t.Context(), x.a); err != nil {
		t.Fatal(err)
	}
	stops := 0
	o := openRGBInitialCleanupOwner{revoke: x.fence.Revoke, lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) {
		return platform.ServiceRef{}, errors.New("native observer unavailable")
	}, absent: func(context.Context, platform.ServiceIntent) (bool, error) { return false, nil }, stop: func(context.Context, platform.ServiceRef) (platform.OperationStopReceipt, error) {
		stops++
		return platform.OperationStopReceipt{}, nil
	}}
	if err := retireOpenRGBInitialWith(t.Context(), x.store, x.a, o); err == nil {
		t.Fatal("unknown native state was treated as absent")
	}
	a, e := x.store.read(x.a.ID)
	if e != nil || a.State == openRGBInitialStopped || stops != 0 {
		t.Fatalf("unsafe cleanup %+v %v stops=%d", a, e, stops)
	}
	f, e := x.fence.read()
	if e != nil || f.State != "revoked" {
		t.Fatalf("unknown cleanup did not fence late execution %+v %v", f, e)
	}
}
func TestOpenRGBInitialSettingsPreserveOwnedPreferencesWithoutAmbientRead(t *testing.T) {
	c, m, _ := openRGBHeadlessFixture(t)
	paths, e := resourceStoragePaths(openRGBResourceName)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(paths.ConfigDir, filepath.FromSlash(m.ManagedService.Config.Path))
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	original := []byte(`{"UserInterface":{"language":"retain"},"Detectors":{"detectors":{"unqualified":true}}}`)
	if e = os.WriteFile(path, original, 0600); e != nil {
		t.Fatal(e)
	}
	catalog, e := os.ReadFile(filepath.Join(c.Root, "resources", openRGBResourceName, m.ManagedService.Config.ContentFile))
	if e != nil {
		t.Fatal(e)
	}
	settings, e := c.freezeOpenRGBInitialSettings(m, string(catalog), []string{"ASUS Aura Motherboard"})
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(settings, "retain") || strings.Contains(settings, `"unqualified": true`) {
		t.Fatal("resource preference or detector isolation lost")
	}
	after, e := os.ReadFile(path)
	if e != nil || string(after) != string(original) {
		t.Fatal("preparation changed resource preferences")
	}
	if e = os.Remove(path); e != nil {
		t.Fatal(e)
	}
	if e = os.Symlink(filepath.Join(c.Home, "desktop.json"), path); e != nil {
		t.Fatal(e)
	}
	if _, e = c.freezeOpenRGBInitialSettings(m, string(catalog), nil); e == nil {
		t.Fatal("symlinked preferences were accepted")
	}
}

func TestOpenRGBNativePolicyPinsCanonicalGenerationAroundLiveObservation(t *testing.T) {
	x := newInitialExecFixture(t)
	state := ManagedServiceState{InitialLaunch: &ManagedInitialServingLaunch{AttemptID: x.a.ID, NativeRef: x.ref}, InstanceID: x.a.ID, PID: x.ref.RootPID, Incarnation: "boot:" + x.ref.RootStartTime, ArtifactPath: x.a.Contract.Artifact.Path, ArtifactVersion: x.a.Contract.Artifact.Version, ArtifactSHA256: x.a.Contract.Artifact.SHA256}
	check := OpenRGBDevicePolicyCheck{DeviceGenerationSHA256: updateTestDigest("fixture device generation"), PID: state.PID, Incarnation: state.Incarnation, Artifact: x.a.Contract.Artifact, PolicyID: x.a.Policy.ID, DetectorSHA256: x.a.Policy.DetectorCatalogSHA256}
	for _, mode := range []string{"ready", "changed", "witness-failed", "borrowed", "policy"} {
		t.Run(mode, func(t *testing.T) {
			reads, observed := 0, 0
			read := func() (ManagedServiceState, error) {
				reads++
				s := state
				if mode == "changed" && reads > 1 {
					s.InstanceID = "replacement"
				}
				if mode == "borrowed" {
					s.InitialLaunch = nil
				}
				return s, nil
			}
			observe := func(context.Context, ManagedServiceState) (openRGBExecWitness, error) {
				observed++
				if mode == "witness-failed" {
					return openRGBExecWitness{}, errors.New("native root changed")
				}
				w := openRGBExecWitness{DeviceGenerationSHA256: updateTestDigest("fixture device generation"), Policy: x.a.Policy}
				if mode == "policy" {
					w.Policy.ID = updateTestDigest("other policy")
				}
				return w, nil
			}
			err := verifyOpenRGBNativePolicy(t.Context(), check, read, observe)
			if mode == "ready" {
				if err != nil || reads != 2 || observed != 1 {
					t.Fatalf("proof %v reads=%d observations=%d", err, reads, observed)
				}
			} else if err == nil {
				t.Fatal("unqualified proof succeeded")
			}
			if mode == "borrowed" && observed != 0 {
				t.Fatal("borrowed process accepted live owner observation")
			}
		})
	}
}

func TestOpenRGBInitialStoppedPublicationDoesNotFollowReusedPID(t *testing.T) {
	x := newInitialExecFixture(t)
	now := time.Now().UTC()
	state := ManagedServiceState{InitialLaunch: &ManagedInitialServingLaunch{AttemptID: x.a.ID, NativeRef: x.ref}, InstanceID: x.a.ID, PID: x.ref.RootPID, Incarnation: "boot:" + x.ref.RootStartTime, ArtifactPath: x.a.Contract.Artifact.Path, ArtifactVersion: x.a.Contract.Artifact.Version, ArtifactSHA256: x.a.Contract.Artifact.SHA256, OwnershipTokenHash: x.a.TokenHash, StoppedAt: &now}
	supervisor := newManagedServiceSupervisor(filepath.Join(t.TempDir(), "state.json"), filepath.Join(t.TempDir(), "log"))
	if err := supervisor.writeState(state); err != nil {
		t.Fatal(err)
	}
	if _, running, err := supervisor.Status(); err != nil || running {
		t.Fatalf("stopped publication followed reused PID running=%v error=%v", running, err)
	}
}

func TestOpenRGBInitialCleanupDrainsLateNativeDeliveryAfterAbsentStop(t *testing.T) {
	x := newInitialExecFixture(t)
	if err := x.store.prepare(t.Context(), x.a); err != nil {
		t.Fatal(err)
	}
	delivered := false
	stops := 0
	o := openRGBInitialCleanupOwner{revoke: x.fence.Revoke, lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) {
		if !delivered {
			return platform.ServiceRef{}, errors.New("not submitted yet")
		}
		return x.ref, nil
	}, absent: func(context.Context, platform.ServiceIntent) (bool, error) { return true, nil }, stop: func(context.Context, platform.ServiceRef) (platform.OperationStopReceipt, error) {
		stops++
		return platform.OperationStopReceipt{NativeQuiescent: true}, nil
	}, released: func(context.Context, platform.ServiceRef) (bool, error) { return false, nil }, release: func(context.Context, platform.ServiceRef) error { return nil }}
	if err := retireOpenRGBInitialWith(t.Context(), x.store, x.a, o); err != nil {
		t.Fatal(err)
	}
	a, e := x.store.read(x.a.ID)
	if e != nil || a.State != openRGBInitialStopped || a.NativeRef != nil {
		t.Fatalf("absence receipt %+v %v", a, e)
	}
	// A delayed first native submission cannot execute the revoked payload, but
	// its exact unit must still be drained and released on repeated disable.
	delivered = true
	if err := retireOpenRGBInitialWith(t.Context(), x.store, a, o); err != nil {
		t.Fatal(err)
	}
	a, e = x.store.read(x.a.ID)
	if e != nil || a.State != openRGBInitialStopped || a.NativeRef == nil || stops != 1 {
		t.Fatalf("late native cleanup %+v %v stops=%d", a, e, stops)
	}
}

func TestOpenRGBInitialDisableWinsAgainstWitnessInFlight(t *testing.T) {
	x := newInitialExecFixture(t)
	if err := x.store.prepare(t.Context(), x.a); err != nil {
		t.Fatal(err)
	}
	var err error
	x.a, err = x.store.submit(t.Context(), x.a)
	if err != nil {
		t.Fatal(err)
	}
	witnessed, continueExec := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-continueExec:
		default:
			close(continueExec)
		}
	})
	witness := x.o.witness
	x.o.witness = func(ctx context.Context, a openRGBInitialAttempt, base managedUpdateInvocation, ref platform.ServiceRef, inc string) (openRGBExecWitness, error) {
		w, e := witness(ctx, a, base, ref, inc)
		close(witnessed)
		<-continueExec
		return w, e
	}
	result := make(chan error, 1)
	go func() { result <- executeOpenRGBInitial(t.Context(), x.a, x.o) }()
	select {
	case <-witnessed:
	case err := <-result:
		t.Fatalf("wrapper failed before witness barrier: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("wrapper did not reach witness barrier")
	}
	stopped, released := false, false
	o := openRGBInitialCleanupOwner{revoke: x.fence.Revoke, lookup: func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return x.ref, nil }, stop: func(context.Context, platform.ServiceRef) (platform.OperationStopReceipt, error) {
		stopped = true
		return platform.OperationStopReceipt{NativeQuiescent: true}, nil
	}, released: func(context.Context, platform.ServiceRef) (bool, error) { return released, nil }, release: func(context.Context, platform.ServiceRef) error { released = true; return nil }}
	if err := retireOpenRGBInitialWith(t.Context(), x.store, x.a, o); err != nil {
		t.Fatal(err)
	}
	close(continueExec)
	if err := <-result; !errors.Is(err, ErrManagedUpdateExecRevoked) {
		t.Fatalf("late witness bypassed disabled fence: %v", err)
	}
	if !stopped || !released || x.replaced != 0 {
		t.Fatalf("disable effects stop=%v release=%v payloadExecs=%d", stopped, released, x.replaced)
	}
	a, e := x.store.read(x.a.ID)
	if e != nil || a.State != openRGBInitialStopped {
		t.Fatalf("disabled receipt %+v %v", a, e)
	}
	base, e := renderManagedServiceInvocation(x.a.Contract, x.a.Runtime, x.a.root(), "initial-test-token")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(base.ConfigPath); !os.IsNotExist(e) {
		t.Fatalf("late wrapper mutated config after disable: %v", e)
	}
}

func TestOpenRGBUpdateClaimDigestPreservesLegacyWireFraming(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	legacy, err := json.Marshal(struct {
		Resource    string
		OperationID string
		Intent      ManagedUpdateLaunchIntent
		Ref         platform.ServiceRef
	}{x.fence.Resource, x.fence.OperationID, x.fence.Intent, x.ref})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := x.fence.openRGBClaimDigest(x.ref), updateTestDigest(string(legacy)); got != want {
		t.Fatalf("default update digest changed: got %s want %s", got, want)
	}
}
