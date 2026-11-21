package resources

import (
	"bytes"
	"encoding/json"
	platform "github.com/vrooli/platform-go"
	"path/filepath"
	"testing"
)

// This proves selection from the exact durable role and consumed claim. Native
// sealed-image opening and new submission have separate platform proofs.
func TestOpenRGBRetainedNativeLauncherUsesCurrentClaimAndNamespace(t *testing.T) {
	for _, target := range []string{openRGBPlatformDarwin, openRGBPlatformWindows} {
		t.Run(target, func(t *testing.T) {
			f := newOpenRGBPublicFixtureFor(t, target)
			selected, err := f.c.openRGBRetainedNativeLauncherRuntime(t.Context(), f.original)
			if err != nil || selected != f.initial.Runtime {
				t.Fatalf("exact retained role runtime=%+v err=%v", selected, err)
			}
			foreign := NewController(f.c.Root, filepath.Join(f.c.Home, "another-operator"))
			if _, err = foreign.openRGBRetainedNativeLauncherRuntime(t.Context(), f.original); err == nil {
				t.Fatal("foreign operator selected retained launcher")
			}
			changed := f.original
			initial := *changed.InitialLaunch
			changed.InitialLaunch = &initial
			initial.NativeRef.RootPID++
			if _, err = f.c.openRGBRetainedNativeLauncherRuntime(t.Context(), changed); err == nil {
				t.Fatal("changed native owner selected retained launcher")
			}
			// Cleanup revokes execution but retains its exact consumed ancestry. That
			// ancestry may select code for a separately admitted successor; it never
			// permits replay of this revoked invocation.
			store, err := newManagedInitialExecStore(f.initial.ID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.Revoke(t.Context(), f.initial.fence()); err != nil {
				t.Fatal(err)
			}
			after, err := f.c.openRGBRetainedNativeLauncherRuntime(t.Context(), f.original)
			if err != nil || after != selected {
				t.Fatalf("exact retired ancestry lost: %+v %v", after, err)
			}
		})
	}
}

func TestOpenRGBNativeLauncherProofOnlyChangesNewSuccessorSpec(t *testing.T) {
	f := newOpenRGBPublicFixtureFor(t, openRGBPlatformDarwin)
	old := f.initial
	before, err := json.Marshal(old.Runtime)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(before, []byte("native_launcher_profile")) || old.wrapperSpec().NativeLauncher != nil {
		t.Fatal("legacy runtime gained a publication proof")
	}
	if err = old.Intent.ValidateSpec(old.wrapperSpec()); err != nil {
		t.Fatal(err)
	}
	successor := old
	successor.Runtime.NativeLauncherProfile = platform.NativeLauncherProfileOpenRGB
	proof := successor.wrapperSpec().NativeLauncher
	if proof == nil || proof.Validate() != nil || proof.SHA256 != old.Runtime.LauncherSHA256 {
		t.Fatal("new successor lost exact code proof")
	}
	if err = old.Intent.ValidateSpec(successor.wrapperSpec()); err == nil {
		t.Fatal("new code proof did not change frozen invocation digest")
	}
	after, err := json.Marshal(old.Runtime)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("successor proof changed retained runtime bytes")
	}
	invalid := successor.Runtime
	invalid.NativeAccount = ""
	if invalid.validate() == nil {
		t.Fatal("Linux runtime accepted native publication authority")
	}
	invalid = successor.Runtime
	invalid.NativeLauncherProfile = "another-publisher"
	if invalid.validate() == nil {
		t.Fatal("unknown publisher admitted")
	}
}
