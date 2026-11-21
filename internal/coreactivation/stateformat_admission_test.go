package coreactivation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/stateformat"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// stubStateFormats replaces the image probe for the duration of one test.
func stubStateFormats(t *testing.T, formats map[string]stateformat.Manifest, failures map[string]error) {
	t.Helper()
	previous := probeStateFormat
	t.Cleanup(func() { probeStateFormat = previous })
	probeStateFormat = func(_ context.Context, image string) (stateformat.Manifest, error) {
		if err, failed := failures[image]; failed {
			return stateformat.Manifest{}, err
		}
		manifest, ok := formats[image]
		if !ok {
			return stateformat.Manifest{}, errors.New("no state format")
		}
		return manifest, nil
	}
}

func sealedManifest(t *testing.T, stores ...stateformat.Store) stateformat.Manifest {
	t.Helper()
	if err := stateformat.SortStores(stores); err != nil {
		t.Fatal(err)
	}
	manifest := stateformat.Manifest{Format: stateformat.ManifestFormat, Stores: stores}
	if err := manifest.Seal(); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func registryStore(t *testing.T, write stateformat.Version, reads []stateformat.Version, schemas map[stateformat.Version]string) stateformat.Store {
	t.Helper()
	return stateformat.Store{Name: "runtime-registry", Write: write, Reads: reads, Schemas: schemas}
}

func additiveRegistry(t *testing.T, write stateformat.Version) stateformat.Store {
	t.Helper()
	const hexDigits = "0123456789abcdef"
	schemas := map[stateformat.Version]string{}
	reads := make([]stateformat.Version, 0, write)
	for version := stateformat.Version(1); version <= write; version++ {
		schemas[version] = strings.Repeat(string(hexDigits[int(version)%len(hexDigits)]), 64)
		reads = append(reads, version)
	}
	store := registryStore(t, write, reads, schemas)
	store.AdditiveFrom = 1
	if write > 1 {
		store.MigratesFrom = write - 1
	}
	return store
}

func rollbackIdentity() *cliv1.CliVersion {
	return &cliv1.CliVersion{
		BuildIdentity:                 strings.Repeat("a", 64),
		RuntimeHomeContractSha256:     strings.Repeat("b", 64),
		PersistentStateContractSha256: strings.Repeat("c", 64),
	}
}

// The directional manifests are the compatibility decision. The older identity
// digest is absent from both images because the new build no longer stamps it;
// that must not become a refusal.
func TestRollbackImagesCompatibilityDoesNotRequireTheIdentityDigest(t *testing.T) {
	priorImage, candidateImage := "prior", "candidate"
	stubStateFormats(t, map[string]stateformat.Manifest{
		priorImage:     sealedManifest(t, additiveRegistry(t, 16)),
		candidateImage: sealedManifest(t, additiveRegistry(t, 17)),
	}, nil)
	prior := rollbackIdentity()
	candidate := rollbackIdentity()
	prior.PersistentStateContractSha256 = ""
	candidate.PersistentStateContractSha256 = ""
	if err := QualifyRollbackImages(context.Background(), prior, candidate, priorImage, candidateImage); err != nil {
		t.Fatalf("an additive format change with manifests was refused: %v", err)
	}
}

// Equal identity digests cannot override a manifest that says the rollback
// image cannot read the candidate's writes.
func TestRollbackImagesCompatibilityIgnoresIdentityWhenManifestsDisagree(t *testing.T) {
	priorImage, candidateImage := "prior", "candidate"
	additive := additiveRegistry(t, 16)
	breaking := registryStore(t, 17, []stateformat.Version{17}, map[stateformat.Version]string{17: strings.Repeat("c", 64)})
	stubStateFormats(t, map[string]stateformat.Manifest{
		priorImage:     sealedManifest(t, additive),
		candidateImage: sealedManifest(t, breaking),
	}, nil)
	if err := QualifyRollbackImages(context.Background(), rollbackIdentity(), rollbackIdentity(), priorImage, candidateImage); !errors.Is(err, ErrStateContractChanged) {
		t.Fatalf("a breaking manifest change was admitted on identity equality: %v", err)
	}
}

// A shared version with a different schema digest is retroactive
// reinterpretation, even when both images claim the same version.
func TestRollbackImagesCompatibilityRefusesRewrittenSharedSchema(t *testing.T) {
	priorImage, candidateImage := "prior", "candidate"
	prior := additiveRegistry(t, 16)
	candidate := additiveRegistry(t, 16)
	candidate.Schemas[16] = strings.Repeat("f", 64)
	stubStateFormats(t, map[string]stateformat.Manifest{
		priorImage:     sealedManifest(t, prior),
		candidateImage: sealedManifest(t, candidate),
	}, nil)
	err := QualifyRollbackImages(context.Background(), rollbackIdentity(), rollbackIdentity(), priorImage, candidateImage)
	if !errors.Is(err, ErrStateContractChanged) || !strings.Contains(err.Error(), "schema digest changed") {
		t.Fatalf("a rewritten shared schema was admitted: %v", err)
	}
}

// The bootstrap case: the serving image predates the manifest. The identity
// rule still decides, and it still refuses a contract change.
func TestRollbackImagesCompatibilityFallsBackWhenTheServingImageHasNoManifest(t *testing.T) {
	priorImage, candidateImage := "prior", "candidate"
	stubStateFormats(t, map[string]stateformat.Manifest{
		candidateImage: sealedManifest(t, additiveRegistry(t, 17)),
	}, map[string]error{priorImage: errors.New("no state format")})

	if err := QualifyRollbackImages(context.Background(), rollbackIdentity(), rollbackIdentity(), priorImage, candidateImage); err != nil {
		t.Fatalf("equal identity digests were refused on the fallback path: %v", err)
	}
	candidate := rollbackIdentity()
	candidate.PersistentStateContractSha256 = strings.Repeat("d", 64)
	if err := QualifyRollbackImages(context.Background(), rollbackIdentity(), candidate, priorImage, candidateImage); !errors.Is(err, ErrStateContractChanged) {
		t.Fatalf("a contract change was admitted when only one image has a manifest: %v", err)
	}
	candidate.PersistentStateContractSha256 = ""
	if err := QualifyRollbackImages(context.Background(), rollbackIdentity(), candidate, priorImage, candidateImage); !errors.Is(err, ErrUnqualifiedStateContract) {
		t.Fatalf("an unstamped fallback image was admitted: %v", err)
	}
}

// Layout and build metadata govern more than persistent state and are required
// on every path.
func TestRollbackImagesCompatibilityStillRequiresBuildAndLayout(t *testing.T) {
	priorImage, candidateImage := "prior", "candidate"
	stubStateFormats(t, map[string]stateformat.Manifest{
		priorImage:     sealedManifest(t, additiveRegistry(t, 16)),
		candidateImage: sealedManifest(t, additiveRegistry(t, 17)),
	}, nil)
	candidate := rollbackIdentity()
	candidate.RuntimeHomeContractSha256 = strings.Repeat("e", 64)
	err := QualifyRollbackImages(context.Background(), rollbackIdentity(), candidate, priorImage, candidateImage)
	if !errors.Is(err, ErrConflict) || errors.Is(err, ErrStateContractChanged) {
		t.Fatalf("a changed runtime-home layout was misclassified: %v", err)
	}
	candidate = rollbackIdentity()
	candidate.BuildIdentity = "unknown"
	if err := QualifyRollbackImages(context.Background(), rollbackIdentity(), candidate, priorImage, candidateImage); !errors.Is(err, ErrConflict) {
		t.Fatalf("an unstamped candidate was admitted: %v", err)
	}
}

// An unrecognized manifest cannot prove compatibility, so it must refuse. The
// stale digest here is exactly what an older reader would see: a manifest it
// cannot validate is not evidence.
func TestRollbackImagesCompatibilityTreatsAnUnrecognizedManifestAsBreaking(t *testing.T) {
	priorImage, candidateImage := "prior", "candidate"
	candidate := sealedManifest(t, additiveRegistry(t, 17))
	candidate.Format = stateformat.ManifestFormat + 1
	stubStateFormats(t, map[string]stateformat.Manifest{
		priorImage:     sealedManifest(t, additiveRegistry(t, 16)),
		candidateImage: candidate,
	}, nil)
	err := QualifyRollbackImages(context.Background(), rollbackIdentity(), rollbackIdentity(), priorImage, candidateImage)
	if !errors.Is(err, ErrStateContractChanged) || !strings.Contains(err.Error(), "could not be compared") {
		t.Fatalf("an unknown manifest format was admitted: %v", err)
	}
}
