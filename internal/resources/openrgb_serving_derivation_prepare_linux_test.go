//go:build linux

package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func windowsHIDPreparationFixture(t *testing.T, adopted bool) (ManagedServiceArtifact, string, openRGBReleaseProfile, openRGBStagePreparation, *int) {
	t.Helper()
	profile, source, archive := windowsHIDServingSourceFixture(t)
	return openRGBReleasePreparationFixture(t, profile, source, archive, adopted)
}

func openRGBReleasePreparationFixture(t *testing.T, profile openRGBReleaseProfile, source, archive string, adopted bool) (ManagedServiceArtifact, string, openRGBReleaseProfile, openRGBStagePreparation, *int) {
	t.Helper()
	artifact := ManagedServiceArtifact{Resource: openRGBResourceName, Version: openRGBAppImageVersion, Path: archive, SHA256: profile.EnvelopeSHA256}
	if adopted {
		artifact.Path = filepath.Join(source, profile.Payload)
		artifact.SHA256 = profile.PayloadSHA256
	}
	owner, err := openRGBStagePreparationOwner()
	if err != nil {
		t.Fatal(err)
	}
	owner.profile = &profile
	owner.payloadRelative = profile.Payload
	owner.bundleDirectory = profile.bundleDirectory()
	owner.verifySeal = func(ctx context.Context, root string) error {
		return verifyOpenRGBAppDirEntries(ctx, root, uint32(os.Geteuid()))
	}
	extractions := 0
	owner.extract = func(ctx context.Context, file *os.File, destination string) error {
		extractions++
		if adopted {
			return copyOpenRGBAdoptedBundle(ctx, source, destination, profile)
		}
		return extractOpenRGBZip(ctx, file, destination, profile)
	}
	final := filepath.Join(t.TempDir(), "published")
	t.Cleanup(func() {
		_ = filepath.WalkDir(filepath.Dir(final), func(path string, d os.DirEntry, e error) error {
			if e == nil && d.IsDir() {
				return os.Chmod(path, 0o700)
			}
			return e
		})
	})
	return artifact, final, profile, owner, &extractions
}

func TestWindowsHIDStageSourceModesRecoveryAndRecipeIdentity(t *testing.T) {
	for _, adopted := range []bool{false, true} {
		t.Run(map[bool]string{false: "managed", true: "adopted"}[adopted], func(t *testing.T) {
			source, final, p, owner, extractions := windowsHIDPreparationFixture(t, adopted)
			sync := owner.syncDir
			lost := true
			owner.syncDir = func(path string) error {
				if path == filepath.Dir(final) && lost {
					lost = false
					return errors.New("injected lost durable-publication acknowledgement")
				}
				return sync(path)
			}
			if _, err := prepareOpenRGBAppImageLocked(t.Context(), source, final, p.servingTreeSHA256(), p.PayloadSHA256, owner); !errors.Is(err, ErrOpenRGBStageCommitUnknown) {
				t.Fatalf("lost acknowledgement not retained: %v", err)
			}
			stage, err := prepareOpenRGBAppImageLocked(t.Context(), source, final, p.servingTreeSHA256(), p.PayloadSHA256, owner)
			if err != nil {
				t.Fatal(err)
			}
			if *extractions != 1 {
				t.Fatalf("recovery re-extracted verified serving closure: %d", *extractions)
			}
			if err = binaryfetch.VerifyArtifactContext(t.Context(), stage, "dir", p.ServingTreeSHA256); err != nil {
				t.Fatal(err)
			}
			if err = p.verifyServingModulesAbsent(t.Context(), stage); err != nil {
				t.Fatal(err)
			}
			if err = p.verifyServingProfile(t.Context(), final, source); err != nil {
				t.Fatal(err)
			}
			retained := source
			retained.Path = filepath.Join(final, openRGBSealedEnvelope)
			if err = verifyOpenRGBStageEnvelope(t.Context(), retained); err != nil {
				t.Fatal(err)
			}
			// A source or recipe change cannot reinterpret an existing sealed root.
			other := p
			other.ServingRecipeSHA256 = updateTestDigest("different-recipe")
			owner.profile = &other
			if _, err = prepareOpenRGBAppImageLocked(t.Context(), source, final, p.servingTreeSHA256(), p.PayloadSHA256, owner); err == nil {
				t.Fatal("recovery accepted a different recipe identity")
			}
		})
	}
}

func TestWindowsHIDStageRefusesOriginalUntransformedClosure(t *testing.T) {
	source, final, p, owner, extractions := windowsHIDPreparationFixture(t, false)
	// Simulate a retained legacy stage at the new path. Current preparation must
	// refuse it rather than mutate or execute it while treating source=serving.
	if err := os.MkdirAll(final, 0o700); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(final, openRGBSealedEnvelope)
	if err := copyOpenRGBStageEnvelope(t.Context(), source, snapshot); err != nil {
		t.Fatal(err)
	}
	archive, err := os.Open(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(final, owner.bundlePath())
	err = extractOpenRGBZip(t.Context(), archive, bundle, p)
	archive.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err = owner.seal(t.Context(), final); err != nil {
		t.Fatal(err)
	}
	if _, err = prepareOpenRGBAppImageLocked(t.Context(), source, final, p.servingTreeSHA256(), p.PayloadSHA256, owner); err == nil {
		t.Fatal("untransformed official tree was accepted as HID serving tree")
	}
	if *extractions != 0 {
		t.Fatal("unqualified existing root was rewritten in place")
	}
	if err = binaryfetch.VerifyArtifactContext(t.Context(), bundle, "dir", p.TreeSHA256); err != nil {
		t.Fatal("refusal changed original retained closure")
	}
}

// The artifact owner does not switch lifecycle roles. It must preserve a
// verified serving/rollback stage while a separate candidate is prepared,
// including cancellation after derivation but before publication.
func TestWindowsHIDCandidateCancellationPreservesRollbackClosure(t *testing.T) {
	source, active, p, owner, _ := windowsHIDPreparationFixture(t, false)
	activeBundle, err := prepareOpenRGBAppImageLocked(t.Context(), source, active, p.servingTreeSHA256(), p.PayloadSHA256, owner)
	if err != nil {
		t.Fatal(err)
	}
	candidate := active + "-candidate"
	seal := owner.seal
	interrupted := owner
	interrupted.seal = func(ctx context.Context, path string) error {
		if err := p.verifyServingModulesAbsent(ctx, filepath.Join(path, owner.bundlePath())); err != nil {
			t.Fatalf("candidate was not derived before simulated cancellation: %v", err)
		}
		return context.Canceled
	}
	if _, err = prepareOpenRGBAppImageLocked(t.Context(), source, candidate, p.servingTreeSHA256(), p.PayloadSHA256, interrupted); !errors.Is(err, context.Canceled) {
		t.Fatalf("candidate cancellation lost: %v", err)
	}
	if _, err = os.Lstat(candidate); !os.IsNotExist(err) {
		t.Fatalf("canceled candidate became visible: %v", err)
	}
	verify := func(root, bundle string) {
		t.Helper()
		if _, err := verifyOpenRGBReleaseDir(t.Context(), bundle, p.servingTreeSHA256(), p.PayloadSHA256, p.Payload, owner.verifySeal); err != nil {
			t.Fatal(err)
		}
		if err := p.verifyServingModulesAbsent(t.Context(), bundle); err != nil {
			t.Fatal(err)
		}
		if err := p.verifyServingProfile(t.Context(), root, source); err != nil {
			t.Fatal(err)
		}
	}
	verify(active, activeBundle)
	owner.seal = seal
	candidateBundle, err := prepareOpenRGBAppImageLocked(t.Context(), source, candidate, p.servingTreeSHA256(), p.PayloadSHA256, owner)
	if err != nil {
		t.Fatal(err)
	}
	verify(candidate, candidateBundle)
	// Retrying or retaining the old role neither transforms it in place nor
	// reinterprets its original source identity as the derived tree identity.
	rollback, err := prepareOpenRGBAppImageLocked(t.Context(), source, active, p.servingTreeSHA256(), p.PayloadSHA256, owner)
	if err != nil || rollback != activeBundle {
		t.Fatalf("verified rollback stage was not reusable: %q %v", rollback, err)
	}
	verify(active, rollback)
}
