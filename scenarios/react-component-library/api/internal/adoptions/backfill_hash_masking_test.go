package adoptions_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	"react-component-library/internal/testutil/mocks"
)

// These tests pin the fix for the adoptions backfill hash-masking defect
// (knw-1784076189438860926): a reconcile/backfill over a pre-existing local file
// recorded the LOCAL file's bytes as the pristine snapshot, so a locally-modified
// copy read CLEAN and reconverge silently overwrote the local edits.

// provHeader builds a minimal @vrooliComponentSource provenance header block so a
// scanned/on-disk file mirrors a real vendored copy (header + body).
func provHeader(libraryID, version, adoptionID string) string {
	return "/**\n * @vrooliComponentSource " + libraryID +
		"\n * @vrooliComponentVersion " + version +
		"\n * @vrooliComponentAdoption " + adoptionID + "\n */\n"
}

// dataTableLibrary builds rcl:DataTable with a behind release (1.1.1) and the
// current release (1.1.2), both carrying real per-file bodies as production
// GetVersion does. Drift is then decided by comparing the local body to the
// library body rather than trusting a snapshot.
func dataTableLibrary(body111, body112 string) *fakeLibrary {
	return &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-dt": {ID: "cmp-dt", LibraryID: "rcl:DataTable", Version: "1.1.2", LatestVersion: "1.1.2"},
		},
		versions: map[string]components.ComponentVersion{
			"cmp-dt@1.1.1": {
				ComponentID: "cmp-dt", LibraryID: "rcl:DataTable", Version: "1.1.1", Status: components.VersionStatusReleased,
				Content: body111, ContentSHA256: sha(body111),
				Files: []components.ComponentVersionFile{{Path: "DataTable.tsx", Content: body111, ContentSHA256: sha(body111), IsEntry: true}},
			},
			"cmp-dt@1.1.2": {
				ComponentID: "cmp-dt", LibraryID: "rcl:DataTable", Version: "1.1.2", Status: components.VersionStatusReleased,
				Content: body112, ContentSHA256: sha(body112),
				Files: []components.ComponentVersionFile{{Path: "DataTable.tsx", Content: body112, ContentSHA256: sha(body112), IsEntry: true}},
			},
		},
	}
}

const emDataTablePath = "ui/src/components/ui/data-table.tsx"

func emDataTableKey() string { return "experience-manager::" + emDataTablePath }

// TestService_Reconcile_BackfillOverModifiedCopyReadsModified is the incident
// repro: reconciling a header-tagged file whose body already diverged from the
// library version records it as MODIFIED — never CLEAN — and reconverge refuses
// to overwrite it. The recorded source hash is the library's, not the local file.
func TestService_Reconcile_BackfillOverModifiedCopyReadsModified(t *testing.T) {
	ctx := context.Background()
	repo := adoptmocks.NewFakeRepository()
	pristine := "export const DataTable = () => null;\n"
	lib := dataTableLibrary(pristine, pristine)
	onDisk := provHeader("rcl:DataTable", "1.1.1", "adopt-dt") + "export const DataTable = () => 'LOCAL MOBILE FLOOR FIX';\n"
	files := &fakeFiles{
		bytes: map[string][]byte{emDataTableKey(): []byte(onDisk)},
		provenance: []adoptions.ProvenanceFile{{
			Scenario: "experience-manager", AdoptedPath: emDataTablePath,
			LibraryID: "rcl:DataTable", Version: "1.1.1", AdoptionID: "adopt-dt",
			Content: []byte(onDisk),
		}},
	}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))

	applied, err := svc.Reconcile(ctx, adoptions.ReconcileInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, applied.Created)

	rows, err := svc.List(ctx, adoptions.ListQuery{Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rows[0]
	require.Equal(t, sha(pristine), row.SourceSHA256, "source hash must be the library body, never the local file")
	require.NotEqual(t, sha(onDisk), row.AdoptedSnapshotSHA256, "snapshot must not be the poisoned local-file hash")
	require.Equal(t, adoptions.LocalStatusModified, row.LocalStatus, "a locally-modified copy must read MODIFIED at backfill")

	out, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 0, out.Reapplied)
	require.Equal(t, 1, out.Flagged)
	require.Equal(t, adoptions.ReconvergeActionFlaggedModified, out.Outcomes[0].Action)
	require.Equal(t, []byte(onDisk), files.bytes[emDataTableKey()], "a modified copy must never be overwritten")
}

// TestService_Reconcile_CleanCopyReadsCleanAndReconverges is the calibration
// case: a genuinely pristine copy still reads CLEAN and a behind-but-clean copy
// fast-forwards to the current library body under reconverge.
func TestService_Reconcile_CleanCopyReadsCleanAndReconverges(t *testing.T) {
	ctx := context.Background()
	repo := adoptmocks.NewFakeRepository()
	body111 := "export const DataTable = () => null;\n"
	body112 := "export const DataTable = () => 'v2';\n"
	lib := dataTableLibrary(body111, body112)
	onDisk := provHeader("rcl:DataTable", "1.1.1", "adopt-dt") + body111
	files := &fakeFiles{
		bytes: map[string][]byte{emDataTableKey(): []byte(onDisk)},
		provenance: []adoptions.ProvenanceFile{{
			Scenario: "experience-manager", AdoptedPath: emDataTablePath,
			LibraryID: "rcl:DataTable", Version: "1.1.1", AdoptionID: "adopt-dt",
			Content: []byte(onDisk),
		}},
	}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))

	applied, err := svc.Reconcile(ctx, adoptions.ReconcileInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, applied.Created)

	rows, err := svc.List(ctx, adoptions.ListQuery{Limit: 10})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, adoptions.LocalStatusClean, rows[0].LocalStatus)
	require.Equal(t, adoptions.LibraryVersionStatusBehind, rows[0].LibraryVersionStatus)

	out, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, out.Reapplied, "a genuinely clean behind copy fast-forwards")
	require.Equal(t, 0, out.Flagged)
	require.Contains(t, string(files.bytes[emDataTableKey()]), "'v2'")
}

// TestService_Reconverge_RefusesPoisonedCleanSnapshot proves the durable safety
// guard: even a row whose snapshot was captured from an already-modified copy
// (so snapshot-based drift reads CLEAN) is verified against the library body
// before overwrite and flagged, never overwritten.
func TestService_Reconverge_RefusesPoisonedCleanSnapshot(t *testing.T) {
	ctx := context.Background()
	repo := adoptmocks.NewFakeRepository()
	body111 := "export const DataTable = () => null;\n"
	body112 := "export const DataTable = () => 'v2';\n"
	lib := dataTableLibrary(body111, body112)
	modified := provHeader("rcl:DataTable", "1.1.1", "adopt-dt") + "export const DataTable = () => 'LOCAL FIX';\n"
	files := &fakeFiles{bytes: map[string][]byte{emDataTableKey(): []byte(modified)}}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))

	// Poisoned row: snapshot equals the already-modified copy, so a snapshot-only
	// drift check would call it CLEAN.
	repo.Seed(adoptions.Adoption{
		ID: "poison", ComponentID: "cmp-dt", LibraryID: "rcl:DataTable",
		Scenario: "experience-manager", AdoptedPath: emDataTablePath,
		AdoptedVersion: "1.1.1", AdoptedSnapshotSHA256: sha(modified),
		Files: []adoptions.AdoptionFile{{LibraryPath: "DataTable.tsx", AdoptedPath: emDataTablePath, SourceSHA256: sha(body111), AdoptedSnapshotSHA256: sha(modified)}},
	})

	out, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, out.Behind)
	require.Equal(t, 0, out.Reapplied, "a poisoned CLEAN snapshot must not trigger an overwrite")
	require.Equal(t, 1, out.Flagged)
	require.Equal(t, adoptions.ReconvergeActionFlaggedModified, out.Outcomes[0].Action)
	require.Equal(t, []byte(modified), files.bytes[emDataTableKey()], "poisoned copy must never be overwritten")
}

// TestService_Reconcile_HealsPoisonedSnapshot proves reconcile --apply re-derives
// an honest baseline for an already-recorded row whose snapshot was poisoned, so
// it stops reading CLEAN.
func TestService_Reconcile_HealsPoisonedSnapshot(t *testing.T) {
	ctx := context.Background()
	repo := adoptmocks.NewFakeRepository()
	body111 := "export const DataTable = () => null;\n"
	lib := dataTableLibrary(body111, body111)
	modified := provHeader("rcl:DataTable", "1.1.1", "adopt-dt") + "export const DataTable = () => 'LOCAL FIX';\n"
	files := &fakeFiles{bytes: map[string][]byte{emDataTableKey(): []byte(modified)}}
	svc := adoptions.NewService(repo, lib, files, mocks.NewFakeClock(time.Now()))

	repo.Seed(adoptions.Adoption{
		ID: "poison", ComponentID: "cmp-dt", LibraryID: "rcl:DataTable",
		Scenario: "experience-manager", AdoptedPath: emDataTablePath,
		AdoptedVersion: "1.1.1", AdoptedSnapshotSHA256: sha(modified),
		Files:       []adoptions.AdoptionFile{{LibraryPath: "DataTable.tsx", AdoptedPath: emDataTablePath, SourceSHA256: sha(body111), AdoptedSnapshotSHA256: sha(modified)}},
		LocalStatus: adoptions.LocalStatusClean, LibraryVersionStatus: adoptions.LibraryVersionStatusBehind,
	})

	dry, err := svc.Reconcile(ctx, adoptions.ReconcileInput{})
	require.NoError(t, err)
	require.Equal(t, 0, dry.Healed, "dry-run must not heal")

	applied, err := svc.Reconcile(ctx, adoptions.ReconcileInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, applied.Healed)

	healed, err := svc.Get(ctx, "poison")
	require.NoError(t, err)
	require.Equal(t, adoptions.LocalStatusModified, healed.LocalStatus, "healed row must read MODIFIED honestly")
}
