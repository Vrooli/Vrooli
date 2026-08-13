package adoptions_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	"react-component-library/internal/deps"

	"github.com/vrooli/api-core/scheduletest"
)

// reconvergeLibrary builds a Button with a BEHIND release (1.0.0) and the
// current release (1.1.0). GetVersion returns released status for both so
// computeStatus classifies an adopted 1.0.0 copy as BEHIND, not unknown.
func reconvergeLibrary(bodyV10, bodyV11 string) *fakeLibrary {
	return &fakeLibrary{
		byID: map[string]components.Component{
			"cmp-btn": {ID: "cmp-btn", LibraryID: "rcl:Button", Version: "1.1.0", LatestVersion: "1.1.0"},
		},
		versions: map[string]components.ComponentVersion{
			"cmp-btn@1.0.0": {
				ComponentID: "cmp-btn", LibraryID: "rcl:Button", Version: "1.0.0", Status: components.VersionStatusReleased,
				Content: bodyV10, ContentSHA256: sha(bodyV10),
				Files: []components.ComponentVersionFile{{Path: "Button.tsx", Content: bodyV10, ContentSHA256: sha(bodyV10), IsEntry: true}},
			},
			"cmp-btn@1.1.0": {
				ComponentID: "cmp-btn", LibraryID: "rcl:Button", Version: "1.1.0", Status: components.VersionStatusReleased,
				Content: bodyV11, ContentSHA256: sha(bodyV11),
				Files: []components.ComponentVersionFile{{Path: "Button.tsx", Content: bodyV11, ContentSHA256: sha(bodyV11), IsEntry: true}},
			},
		},
	}
}

const tmButtonPath = "ui/src/components/ui/button.tsx"

func tmButtonKey() string { return "template-manager::" + tmButtonPath }

// TestService_Reconverge_ReappliesBehindCleanAndRunsValidation proves the
// happy path: a BEHIND + CLEAN copy is left untouched in dry-run, re-applied to
// the current version under --apply, and both validation gates fire on the
// re-apply (the batch flow does not bypass them). A second pass finds nothing.
func TestService_Reconverge_ReappliesBehindCleanAndRunsValidation(t *testing.T) {
	ctx := context.Background()
	repo := adoptmocks.NewFakeRepository()
	bodyV10, bodyV11 := "BODY-V10", "BODY-V11"
	lib := reconvergeLibrary(bodyV10, bodyV11)
	files := &fakeFiles{bytes: map[string][]byte{tmButtonKey(): []byte(bodyV10)}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Now()))
	depsGate := &validationDeps{verdict: deps.Verdict{Kind: deps.VerdictWarn}}
	styleGate := &validationStyles{}
	adoptions.SetValidationGates(svc, depsGate, styleGate)

	repo.Seed(adoptions.Adoption{
		ID: "row-1", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
		Scenario: "template-manager", AdoptedPath: tmButtonPath,
		AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10),
		Files: []adoptions.AdoptionFile{{LibraryPath: "Button.tsx", AdoptedPath: tmButtonPath, AdoptedSnapshotSHA256: sha(bodyV10)}},
	})

	// Dry-run: reports would_reapply, targets the current version, writes nothing.
	dry, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{})
	require.NoError(t, err)
	require.Equal(t, 1, dry.Scanned)
	require.Equal(t, 1, dry.Behind)
	require.Equal(t, 0, dry.Reapplied)
	require.Len(t, dry.Outcomes, 1)
	require.Equal(t, adoptions.ReconvergeActionWouldReapply, dry.Outcomes[0].Action)
	require.Equal(t, "1.1.0", dry.Outcomes[0].TargetVersion)
	require.Equal(t, adoptions.LocalStatusClean, dry.Outcomes[0].Files[0].LocalStatus)
	require.Equal(t, []byte(bodyV10), files.bytes[tmButtonKey()], "dry-run must never write")
	require.Zero(t, depsGate.calls, "dry-run must not re-apply, so gates do not fire")

	// Apply: re-applies to the current library version and fires both gates.
	applied, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, applied.Reapplied)
	require.Equal(t, 0, applied.Flagged)
	require.Equal(t, 0, applied.Errored)
	require.Equal(t, adoptions.ReconvergeActionReapplied, applied.Outcomes[0].Action)
	require.Equal(t, "1.1.0", applied.Outcomes[0].TargetVersion)
	require.Greater(t, depsGate.calls, 0, "dependency validation must run on the batch re-apply")
	require.Greater(t, styleGate.calls, 0, "style-fit validation must run on the batch re-apply")
	require.Contains(t, string(files.bytes[tmButtonKey()]), bodyV11, "re-applied file carries the current body")

	// The row is now current — a follow-up reconverge finds nothing behind.
	after, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{})
	require.NoError(t, err)
	require.Equal(t, 0, after.Behind)
	require.Empty(t, after.Outcomes)
}

// TestService_Reconverge_FlagsModifiedNeverOverwrites proves the safety
// contract: a BEHIND copy with local edits is flagged for human review and its
// bytes are left exactly as they are, even under --apply.
func TestService_Reconverge_FlagsModifiedNeverOverwrites(t *testing.T) {
	ctx := context.Background()
	repo := adoptmocks.NewFakeRepository()
	bodyV10, bodyV11 := "BODY-V10", "BODY-V11"
	lib := reconvergeLibrary(bodyV10, bodyV11)
	edited := "BODY-V10-WITH-LOCAL-EDIT"
	files := &fakeFiles{bytes: map[string][]byte{tmButtonKey(): []byte(edited)}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Now()))
	adoptions.SetValidationGates(svc, &validationDeps{verdict: deps.Verdict{Kind: deps.VerdictWarn}}, &validationStyles{})

	repo.Seed(adoptions.Adoption{
		ID: "row-mod", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
		Scenario: "template-manager", AdoptedPath: tmButtonPath,
		AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10), // disk diverges from snapshot
		Files: []adoptions.AdoptionFile{{LibraryPath: "Button.tsx", AdoptedPath: tmButtonPath, AdoptedSnapshotSHA256: sha(bodyV10)}},
	})

	out, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, out.Behind)
	require.Equal(t, 1, out.Flagged)
	require.Equal(t, 0, out.Reapplied)
	require.Len(t, out.Outcomes, 1)
	require.Equal(t, adoptions.ReconvergeActionFlaggedModified, out.Outcomes[0].Action)
	require.Equal(t, adoptions.LocalStatusModified, out.Outcomes[0].LocalStatus)
	require.Equal(t, adoptions.LocalStatusModified, out.Outcomes[0].Files[0].LocalStatus)
	require.Equal(t, []byte(edited), files.bytes[tmButtonKey()], "a modified copy must never be overwritten")
}

// TestService_Reconverge_ScopesByScenarioAndSkipsCurrent proves the scenario
// filter restricts the batch to one adopter tree and that CURRENT adoptions are
// not reconverge candidates (only BEHIND rows appear in the outcomes).
func TestService_Reconverge_ScopesByScenarioAndSkipsCurrent(t *testing.T) {
	ctx := context.Background()
	repo := adoptmocks.NewFakeRepository()
	bodyV10, bodyV11 := "BODY-V10", "BODY-V11"
	lib := reconvergeLibrary(bodyV10, bodyV11)
	cleanupKey := "storage-manager::" + tmButtonPath
	files := &fakeFiles{bytes: map[string][]byte{
		tmButtonKey(): []byte(bodyV10),
		cleanupKey:    []byte(bodyV11), // current copy in a different scenario
	}}
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Now()))
	adoptions.SetValidationGates(svc, &validationDeps{verdict: deps.Verdict{Kind: deps.VerdictWarn}}, &validationStyles{})

	repo.Seed(adoptions.Adoption{
		ID: "row-tm", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
		Scenario: "template-manager", AdoptedPath: tmButtonPath,
		AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10),
		Files: []adoptions.AdoptionFile{{LibraryPath: "Button.tsx", AdoptedPath: tmButtonPath, AdoptedSnapshotSHA256: sha(bodyV10)}},
	})
	repo.Seed(adoptions.Adoption{
		ID: "row-cm-current", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
		Scenario: "storage-manager", AdoptedPath: tmButtonPath,
		AdoptedVersion: "1.1.0", AdoptedSnapshotSHA256: sha(bodyV11), // already current
		Files: []adoptions.AdoptionFile{{LibraryPath: "Button.tsx", AdoptedPath: tmButtonPath, AdoptedSnapshotSHA256: sha(bodyV11)}},
	})

	// Filtered to template-manager: only the behind TM row is a candidate.
	scoped, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{Scenario: "template-manager"})
	require.NoError(t, err)
	require.Equal(t, 1, scoped.Scanned, "scenario filter restricts the scan to template-manager")
	require.Equal(t, 1, scoped.Behind)
	require.Equal(t, "template-manager", scoped.Outcomes[0].Scenario)

	// Unfiltered: the current storage-manager row is scanned but not a candidate.
	all, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{})
	require.NoError(t, err)
	require.Equal(t, 2, all.Scanned)
	require.Equal(t, 1, all.Behind, "the current copy is not a reconverge candidate")
	require.Len(t, all.Outcomes, 1)
	require.Equal(t, "template-manager", all.Outcomes[0].Scenario)
}

// TestService_Reconverge_MissingCopyIsSkippedNotReapplied proves a BEHIND row
// whose file is gone from disk is surfaced as skipped_unresolved rather than
// silently re-created.
func TestService_Reconverge_MissingCopyIsSkippedNotReapplied(t *testing.T) {
	ctx := context.Background()
	repo := adoptmocks.NewFakeRepository()
	bodyV10, bodyV11 := "BODY-V10", "BODY-V11"
	lib := reconvergeLibrary(bodyV10, bodyV11)
	files := &fakeFiles{bytes: map[string][]byte{}} // nothing on disk
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Now()))

	repo.Seed(adoptions.Adoption{
		ID: "row-missing", ComponentID: "cmp-btn", LibraryID: "rcl:Button",
		Scenario: "template-manager", AdoptedPath: tmButtonPath,
		AdoptedVersion: "1.0.0", AdoptedSnapshotSHA256: sha(bodyV10),
		Files: []adoptions.AdoptionFile{{LibraryPath: "Button.tsx", AdoptedPath: tmButtonPath, AdoptedSnapshotSHA256: sha(bodyV10)}},
	})

	out, err := svc.Reconverge(ctx, adoptions.ReconvergeInput{Apply: true})
	require.NoError(t, err)
	require.Equal(t, 1, out.Behind)
	require.Equal(t, 1, out.Skipped)
	require.Equal(t, 0, out.Reapplied)
	require.Equal(t, adoptions.ReconvergeActionSkippedUnresolved, out.Outcomes[0].Action)
	require.Equal(t, adoptions.LocalStatusMissing, out.Outcomes[0].Files[0].LocalStatus)
}
