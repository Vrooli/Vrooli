//go:build !windows

package platform

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDarwinMaintainedLaunchRequiresApprovedBytesAndImmutablePath(t *testing.T) {
	for _, kind := range []string{"staged-copy", "wrong-image", "mutable-path", "missing-approved", "bad-command"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			approved := filepath.Join(root, "installed-vrooli")
			candidate := filepath.Join(root, "staged-vrooli")
			if e := os.WriteFile(approved, []byte("approved-maintained-host"), 0555); e != nil {
				t.Fatal(e)
			}
			bytes := []byte("approved-maintained-host")
			if kind == "wrong-image" {
				bytes = []byte("arbitrary-other-program")
			}
			if e := os.WriteFile(candidate, bytes, 0555); e != nil {
				t.Fatal(e)
			}
			i := OperationIntent{Owner: "resource-initial", AttemptID: "fixture-1"}
			s := OperationSpec{Path: candidate, Dir: root, Args: []string{"__vrooli-openrgb-native-capsule", "/Library/Application Support/Vrooli/resources/openrgb/launch/fixture-1/capsule.json"}, Env: []string{}}
			approvals := 0
			g := darwinMaintainedLaunchGuard{approved: func(context.Context) (*os.File, error) {
				approvals++
				if kind == "missing-approved" {
					return nil, ErrOperationNotFound
				}
				return os.Open(approved)
			}, verifyPath: func(context.Context, string, bool) error {
				if kind == "mutable-path" {
					return ErrOperationIdentity
				}
				return nil
			}}
			if kind == "bad-command" {
				s.Args[0] = "arbitrary-exec"
			}
			err := g.validate(context.Background(), i, s)
			if kind == "staged-copy" {
				if err != nil {
					t.Fatal("byte-identical separate inode refused", err)
				}
				a, _ := os.Stat(approved)
				b, _ := os.Stat(candidate)
				if os.SameFile(a, b) {
					t.Fatal("fixture did not test separate staged inode")
				}
			} else if err == nil {
				t.Fatal("unsupported native launcher admitted")
			}
			if (kind == "bad-command" || kind == "mutable-path") && approvals != 0 {
				t.Fatal("refused entrypoint reached approved image acquisition")
			}
			if kind == "missing-approved" && !errors.Is(err, ErrOperationNotFound) {
				t.Fatal(err)
			}
		})
	}
}
func TestDarwinMaintainedLaunchSupportsCanonicalCustomCapsuleRoot(t *testing.T) {
	i := OperationIntent{Owner: "resource-initial", AttemptID: "fixture-1"}
	s := OperationSpec{Path: "/immutable/vrooli", Dir: "/immutable", Args: []string{"__vrooli-openrgb-native-capsule", "/Volumes/Operator Data/native/openrgb/launch/fixture-1/capsule.json"}}
	if err := validateDarwinMaintainedLaunchShape(i, s); err != nil {
		t.Fatal("canonical custom namespace refused", err)
	}
	s.Args[1] = "/Volumes/Operator Data/native/openrgb/launch/another-attempt/capsule.json"
	if err := validateDarwinMaintainedLaunchShape(i, s); err == nil {
		t.Fatal("foreign attempt path admitted")
	}
}

func TestDarwinMaintainedPublishedLauncherStartsAfterBootDefinitionRemoved(t *testing.T) {
	root := t.TempDir()
	image := darwinBrokerMachFixture("/usr/lib/libSystem.B.dylib")
	digest := fmt.Sprintf("%x", sha256.Sum256(image))
	path := filepath.Join(root, digest, "vrooli")
	if err := os.Mkdir(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, image, 0555); err != nil {
		t.Fatal(err)
	}
	installedCalls, publishedCalls := 0, 0
	reader := darwinPublishedLauncherReader{path: func(context.Context, string, bool) error { return nil }, format: func(f *os.File) error { return validateDarwinBrokerMachO(f, "arm64") }}
	g := darwinMaintainedLaunchGuard{approved: func(context.Context) (*os.File, error) { installedCalls++; return nil, ErrOperationNotFound }, verifyPath: func(context.Context, string, bool) error { return nil }, published: func(ctx context.Context, path string, proof NativeLauncherProof) (*os.File, error) {
		publishedCalls++
		return openPublishedNativeLauncherWith(ctx, path, proof, "vrooli", func() (string, error) { return root, nil }, reader.open)
	}}
	i := OperationIntent{Owner: "resource-update", AttemptID: "successor-1"}
	s := OperationSpec{Path: path, Dir: filepath.Dir(path), Args: []string{"__vrooli-openrgb-native-capsule", "/custom/launch/successor-1/capsule.json"}, Env: []string{}, NativeLauncher: &NativeLauncherProof{SchemaVersion: NativeLauncherProofVersion, Profile: NativeLauncherProfileOpenRGB, SHA256: digest}}
	if err := g.validate(context.Background(), i, s); err != nil {
		t.Fatal("exact published current host required removed boot definition", err)
	}
	if installedCalls != 0 || publishedCalls != 1 {
		t.Fatal("published proof consulted installed selector", installedCalls, publishedCalls)
	}
	s.NativeLauncher = nil
	if err := g.validate(context.Background(), i, s); !errors.Is(err, ErrOperationNotFound) {
		t.Fatal("legacy nil proof stopped using installed-host authority", err)
	}
	if installedCalls != 1 || publishedCalls != 1 {
		t.Fatal("legacy route changed authority", installedCalls, publishedCalls)
	}
}

type darwinPublishedAdmissionFixture struct {
	*darwinFixtureNative
	guard darwinMaintainedLaunchGuard
}

func (f darwinPublishedAdmissionFixture) ValidateLaunch(ctx context.Context, i OperationIntent, s OperationSpec) error {
	return f.guard.validate(ctx, i, s)
}
func TestDarwinPublishedCurrentHostAdmitsNewOwnedRoleWithoutBootUnit(t *testing.T) {
	b, native, _, original := newDarwinFixture(t)
	collection := t.TempDir()
	image := darwinBrokerMachFixture("/usr/lib/libSystem.B.dylib")
	digest := fmt.Sprintf("%x", sha256.Sum256(image))
	path := filepath.Join(collection, digest, "vrooli")
	if err := os.Mkdir(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, image, 0555); err != nil {
		t.Fatal(err)
	}
	installedCalls := 0
	reader := darwinPublishedLauncherReader{path: func(context.Context, string, bool) error { return nil }, format: func(f *os.File) error { return validateDarwinBrokerMachO(f, "arm64") }}
	guard := darwinMaintainedLaunchGuard{approved: func(context.Context) (*os.File, error) { installedCalls++; return nil, ErrOperationNotFound }, published: func(ctx context.Context, path string, proof NativeLauncherProof) (*os.File, error) {
		return openPublishedNativeLauncherWith(ctx, path, proof, "vrooli", func() (string, error) { return collection, nil }, reader.open)
	}}
	b.native = darwinPublishedAdmissionFixture{native, guard}
	service := OwnedServiceSpec{Path: path, Dir: filepath.Dir(path), NativeAccount: original.NativeAccount, ObserverPrincipal: original.ObserverPrincipal, NativeLauncher: &NativeLauncherProof{SchemaVersion: NativeLauncherProofVersion, Profile: NativeLauncherProfileOpenRGB, SHA256: digest}, Args: original.Args, Env: original.Env, Limits: ServiceLimits{MemoryMaxBytes: original.Limits.MemoryMaxBytes, TasksMax: original.Limits.TasksMax, CPUWeight: original.Limits.CPUWeight, Enforcement: original.Limits.Enforcement}}
	intent, err := NewServiceIntent("resource-update", "fixture-1", service)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := b.start(context.Background(), OperationIntent(intent), service.processSpec())
	if err != nil {
		t.Fatal("new role rejected exact retained published host", err)
	}
	if installedCalls != 0 || operation.Ref.RootPID != 42 || len(native.calls) != 1 {
		t.Fatal("new submission used removed boot authority or lacked exact birth", installedCalls, operation.Ref, native.calls)
	}
	record, err := b.store.Read(OperationIntent(intent))
	if err != nil || record.Spec.NativeLauncher == nil || record.Spec.NativeLauncher.SHA256 != digest || !record.DispatchPrepared {
		t.Fatalf("published proof missing from frozen admitted spec: %+v %v", record, err)
	}
}
