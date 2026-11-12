package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
)

// This composes the real update-role fence and native capsule executor. Native
// manager/image/HID/SDK edges are simulated; no native runtime is qualified.
func TestNativeOpenRGBUpdateAndRestoreCapsulesKeepSingleUseAdmission(t *testing.T) {
	for _, target := range []string{openRGBPlatformDarwin, openRGBPlatformWindows} {
		for _, role := range []string{"qualification", "target", "restore"} {
			t.Run(target+"/"+role, func(t *testing.T) {
				var f *nativeExecFixture
				var child *nativePreparedChildFixture
				if target == openRGBPlatformWindows {
					f, child = nativeWindowsCapsuleExecFixture(t)
				} else {
					f = newNativeCapsuleExecFixture(t)
				}
				m := f.session.material
				instance := "native-" + role
				operation := "native-update"
				runtime := m.Runtime
				invocation, err := renderManagedUpdateInvocation(m.Contract, runtime, operation, instance, "private-token")
				if err != nil {
					t.Fatal(err)
				}
				intent := ManagedUpdateLaunchIntent{InstanceID: instance, OwnershipTokenHash: m.TokenHash, ContractDigest: m.Contract.Digest, Runtime: runtime}
				// Native updates dispatch the maintained capsule, with no source command,
				// mutable environment, alternate executable or caller-chosen process.
				capsulePath, err := openRGBNativeCapsulePath(instance)
				if err != nil {
					t.Fatal(err)
				}
				spec := managedUpdateWrapperSpec(ManagedServiceUpdate{Resource: openRGBResourceName, ID: operation}, intent)
				if spec.Path != runtime.Launcher || spec.NativeAccount != runtime.NativeAccount || len(spec.Env) != 0 || len(spec.Args) != 2 || spec.Args[0] != "__vrooli-openrgb-native-capsule" || spec.Args[1] != capsulePath {
					t.Fatal("update role lost its closed maintained entrypoint", spec)
				}
				intent.NativeIntent, err = platform.NewServiceIntent("resource-update", instance, spec)
				if err != nil {
					t.Fatal(err)
				}
				ref := f.session.ref
				native := *ref.Native
				ref.Native = &native
				ref.Intent = platform.OperationIntent(intent.NativeIntent)
				ref.Unit, _ = intent.NativeIntent.UnitName()
				ref.NativeInvocationID = intent.NativeIntent.InvocationID
				native.InvocationID = intent.NativeIntent.InvocationID
				if target == openRGBPlatformWindows {
					native.Name = "vrooli-owned-" + intent.NativeIntent.Owner
					native.ContainmentID = `Global\VrooliOwned-` + native.InvocationID
					child.id.ContainmentID = native.ContainmentID
				} else {
					native.Name = "com.vrooli.owned." + strings.TrimSuffix(ref.Unit, ".service")
					native.ContainmentID = native.Domain + "/" + native.Name
				}
				ref.Scope.Path = native.Domain + "/" + native.Name
				if err = ref.ValidateIdentity(); err != nil {
					t.Fatal(err)
				}
				expected := managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: openRGBResourceName, OperationID: operation, Intent: intent, State: managedUpdateExecFencePrepared, OpenRGBPolicy: &m.Policy}
				store := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), expected)
				if _, err = store.Prepare(t.Context(), expected); err != nil {
					t.Fatal(err)
				}
				m.Invocation = invocation
				m.ClaimSHA256 = expected.openRGBClaimDigest(ref)
				f.capsule.ID = instance
				f.capsule.Intent = intent.NativeIntent
				f.capsule.RuntimeRoot = invocation.RuntimeRoot
				f.capsule.MaterialSHA256 = m.digest()
				session := &nativeLifecycleFenceSession{material: m, ref: ref, store: store, expected: expected}
				f.owner.lookup = func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error) { return ref, nil }
				f.owner.incarnation = func(int) (string, error) { return native.RootIncarnation, nil }
				f.owner.connect = func(context.Context, openRGBNativeCapsule) (openRGBNativeCapsuleSession, error) { return session, nil }
				prepareImage := f.owner.prepareImage
				f.owner.prepareImage = func(ctx context.Context, m openRGBNativeMaterial) (managedUpdateInvocation, *os.File, error) {
					_, file, err := prepareImage(ctx, m)
					return invocation, file, err
				}
				observe := f.owner.observe
				f.owner.observe = func(ctx context.Context, m openRGBNativeMaterial, r platform.ServiceRef, inc string) (openRGBExecWitness, error) {
					w, err := observe(ctx, m, r, inc)
					w.Policy = m.Policy
					w.ClaimSHA256 = m.ClaimSHA256
					w.Incarnation = inc
					w.ContextSHA256 = updateTestDigest("exact native context")
					return w, err
				}
				materializations := 0
				f.owner.materialize = func(openRGBNativeMaterial, managedUpdateInvocation) error {
					saved, err := store.read()
					if err != nil {
						return err
					}
					if saved.State != managedUpdateExecFenceClaimed || saved.Claim == nil || !saved.Claim.Equal(ref) || saved.OpenRGBWitness == nil {
						return errors.New("update role materialized before durable exact claim")
					}
					if target == openRGBPlatformWindows && (saved.OpenRGBWitness.ExecutionChild == nil || *saved.OpenRGBWitness.ExecutionChild != child.id) {
						return errors.New("update child not recorded before resume")
					}
					materializations++
					return nil
				}
				executions := 0
				f.owner.replace = func(*os.File, []string, []string) error { executions++; return nil }
				if err = executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); err != nil {
					t.Fatal(err)
				}
				claimed, err := store.read()
				if err != nil {
					t.Fatal(err)
				}
				if _, err = verifyOpenRGBConsumedClaim(claimed, intent, m.Contract, ref, native.RootIncarnation); err != nil {
					t.Fatal(err)
				}
				if materializations != 1 || target == openRGBPlatformDarwin && executions != 1 || target == openRGBPlatformWindows && !child.resumed {
					t.Fatal("native role did not execute once after claim")
				}
				// Use a new client/capsule invocation against the same durable fence.
				// A replacement wrapper cannot obtain a second execution admission.
				var replayChildren []*nativePreparedChildFixture
				if target == openRGBPlatformWindows {
					f.owner.prepareChild = func(*os.File, managedUpdateInvocation) (openRGBNativePreparedChild, error) {
						candidate := *child
						candidate.resumed = false
						candidate.id.PID += len(replayChildren) + 1
						candidate.id.Incarnation = fmt.Sprintf("windows:%d:88800", candidate.id.PID)
						replayChildren = append(replayChildren, &candidate)
						return &candidate, nil
					}
				}
				if err = executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); !errors.Is(err, ErrManagedUpdateExecClaimed) {
					t.Fatalf("replayed role: %v", err)
				}
				after, err := store.read()
				if err != nil {
					t.Fatal(err)
				}
				if materializations != 1 || executions > 1 || !after.Claim.Equal(*claimed.Claim) || !after.ClaimedAt.Equal(*claimed.ClaimedAt) {
					t.Fatal("replay changed exact consumed admission")
				}
				for _, candidate := range replayChildren {
					if candidate.resumed {
						t.Fatal("replayed role resumed an unadmitted child")
					}
				}
				if target == openRGBPlatformWindows && !child.resumed {
					t.Fatal("replay disturbed the admitted serving child")
				}
				if _, err = store.Revoke(t.Context(), expected); err != nil {
					t.Fatal(err)
				}
				if err = executeOpenRGBNativeCapsule(t.Context(), f.capsule, f.owner); !errors.Is(err, ErrManagedUpdateExecRevoked) {
					t.Fatalf("late native role after revoke: %v", err)
				}
				if materializations != 1 || executions > 1 {
					t.Fatal("revoked update/restore role executed")
				}
			})
		}
	}
}
