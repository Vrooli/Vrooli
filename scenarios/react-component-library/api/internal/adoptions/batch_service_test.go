package adoptions_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"

	"react-component-library/internal/adoptions"
	adoptmocks "react-component-library/internal/adoptions/mocks"
	"react-component-library/internal/components"
	"react-component-library/internal/deps"
)

func batchAsset(id, libraryID string, dependencies ...components.AssetDependency) components.Component {
	return components.Component{ID: id, LibraryID: libraryID, DisplayName: id, AssetKind: components.AssetKindComponent, LatestVersion: "1.0.0", Dependencies: dependencies}
}

func batchVersion(componentID, libraryID, version, file, body string) components.ComponentVersion {
	return components.ComponentVersion{
		ComponentID:   componentID,
		LibraryID:     libraryID,
		Version:       version,
		Status:        components.VersionStatusReleased,
		SourcePath:    file,
		Content:       body,
		ContentSHA256: sha(body),
		Files: []components.ComponentVersionFile{{
			Path: file, Content: body, ContentSHA256: sha(body), IsEntry: true,
		}},
	}
}

func TestBatchApplyWritesSharedDependencyOnceAndRecordsBothOwners(t *testing.T) {
	dep := batchAsset("dep", "rcl:Shared")
	first := batchAsset("first", "rcl:First", components.AssetDependency{LibraryID: dep.LibraryID, Version: "1.0.0"})
	second := batchAsset("second", "rcl:Second", components.AssetDependency{LibraryID: dep.LibraryID, Version: "1.0.0"})
	lib := &fakeLibrary{
		byID: map[string]components.Component{dep.ID: dep, first.ID: first, second.ID: second},
		versions: map[string]components.ComponentVersion{
			"dep@1.0.0":    batchVersion(dep.ID, dep.LibraryID, "1.0.0", "Shared.tsx", "export const Shared = 1;"),
			"first@1.0.0":  batchVersion(first.ID, first.LibraryID, "1.0.0", "First.tsx", "export const First = 1;"),
			"second@1.0.0": batchVersion(second.ID, second.LibraryID, "1.0.0", "Second.tsx", "export const Second = 1;"),
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{}}
	repo := adoptmocks.NewFakeRepository()
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))

	result, err := svc.BatchApply(context.Background(), adoptions.BatchApplyInput{Items: []adoptions.BatchApplyItem{
		{ComponentID: first.ID, Scenario: "target", AdoptedPath: "ui/src/components/First.tsx"},
		{ComponentID: second.ID, Scenario: "target", AdoptedPath: "ui/src/components/Second.tsx"},
	}})
	require.NoError(t, err)
	require.Len(t, result.Results, 2)
	require.Equal(t, []string{"rcl:Shared"}, result.SharedDependencies)
	require.Len(t, files.bytes, 3, "the shared dependency must be written once")
	require.Len(t, result.Results[0].Adoption.Files, 2)
	require.Len(t, result.Results[1].Adoption.Files, 2)
	require.Len(t, mustListBatchRows(t, repo, "target"), 2)
}

func TestBatchApplyRejectsConflictingDependencyPinsWithoutWriting(t *testing.T) {
	dep := batchAsset("dep", "rcl:Shared")
	first := batchAsset("first", "rcl:First", components.AssetDependency{LibraryID: dep.LibraryID, Version: "1.0.0"})
	second := batchAsset("second", "rcl:Second", components.AssetDependency{LibraryID: dep.LibraryID, Version: "2.0.0"})
	lib := &fakeLibrary{
		byID: map[string]components.Component{dep.ID: dep, first.ID: first, second.ID: second},
		versions: map[string]components.ComponentVersion{
			"dep@1.0.0":    batchVersion(dep.ID, dep.LibraryID, "1.0.0", "Shared.tsx", "one"),
			"dep@2.0.0":    batchVersion(dep.ID, dep.LibraryID, "2.0.0", "Shared.tsx", "two"),
			"first@1.0.0":  batchVersion(first.ID, first.LibraryID, "1.0.0", "First.tsx", "first"),
			"second@1.0.0": batchVersion(second.ID, second.LibraryID, "1.0.0", "Second.tsx", "second"),
		},
	}
	files := &fakeFiles{bytes: map[string][]byte{}}
	repo := adoptmocks.NewFakeRepository()
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))

	_, err := svc.BatchApply(context.Background(), adoptions.BatchApplyInput{Items: []adoptions.BatchApplyItem{
		{ComponentID: first.ID, Scenario: "target", AdoptedPath: "ui/src/components/First.tsx"},
		{ComponentID: second.ID, Scenario: "target", AdoptedPath: "ui/src/components/Second.tsx"},
	}})
	var conflict adoptions.ErrBatchDependencyConflict
	require.ErrorAs(t, err, &conflict)
	require.Contains(t, err.Error(), "first")
	require.Contains(t, err.Error(), "second")
	require.Contains(t, err.Error(), "1.0.0")
	require.Contains(t, err.Error(), "2.0.0")
	require.Empty(t, files.bytes)
	require.Empty(t, mustListBatchRows(t, repo, "target"))
}

func TestBatchApplyRejectsCrossItemTargetCollisionWithoutWriting(t *testing.T) {
	first := batchAsset("first", "rcl:First")
	second := batchAsset("second", "rcl:Second")
	lib := &fakeLibrary{byID: map[string]components.Component{first.ID: first, second.ID: second}, versions: map[string]components.ComponentVersion{
		"first@1.0.0":  batchVersion(first.ID, first.LibraryID, "1.0.0", "First.tsx", "first"),
		"second@1.0.0": batchVersion(second.ID, second.LibraryID, "1.0.0", "Second.tsx", "second"),
	}}
	files := &fakeFiles{bytes: map[string][]byte{}}
	repo := adoptmocks.NewFakeRepository()
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))

	_, err := svc.BatchApply(context.Background(), adoptions.BatchApplyInput{Items: []adoptions.BatchApplyItem{
		{ComponentID: first.ID, Scenario: "target", AdoptedPath: "ui/src/components/Same.tsx"},
		{ComponentID: second.ID, Scenario: "target", AdoptedPath: "ui/src/components/Same.tsx"},
	}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ui/src/components/Same.tsx")
	require.Contains(t, err.Error(), first.ID)
	require.Contains(t, err.Error(), second.ID)
	require.Empty(t, files.bytes)
	require.Empty(t, mustListBatchRows(t, repo, "target"))
}

func TestBatchApplyBlockedItemRefusesWholeBatch(t *testing.T) {
	first := batchAsset("first", "rcl:First")
	second := batchAsset("second", "rcl:Second")
	lib := &fakeLibrary{byID: map[string]components.Component{first.ID: first, second.ID: second}, versions: map[string]components.ComponentVersion{
		"first@1.0.0":  batchVersion(first.ID, first.LibraryID, "1.0.0", "First.tsx", "first"),
		"second@1.0.0": batchVersion(second.ID, second.LibraryID, "1.0.0", "Second.tsx", "second"),
	}}
	files := &fakeFiles{bytes: map[string][]byte{}}
	repo := adoptmocks.NewFakeRepository()
	svc := adoptions.NewService(repo, lib, files, scheduletest.New(time.Unix(0, 0)))
	adoptions.SetValidationGates(svc, &validationDeps{verdict: deps.Verdict{Kind: deps.VerdictBlock}}, &validationStyles{})

	_, err := svc.BatchApply(context.Background(), adoptions.BatchApplyInput{Items: []adoptions.BatchApplyItem{
		{ComponentID: first.ID, Scenario: "target", AdoptedPath: "ui/src/components/First.tsx"},
		{ComponentID: second.ID, Scenario: "target", AdoptedPath: "ui/src/components/Second.tsx"},
	}})
	var blocked adoptions.ErrAdoptionValidationBlocked
	require.ErrorAs(t, err, &blocked)
	require.Empty(t, files.bytes)
	require.Empty(t, mustListBatchRows(t, repo, "target"))
}

func mustListBatchRows(t *testing.T, repo *adoptmocks.FakeRepository, scenario string) []adoptions.Adoption {
	t.Helper()
	rows, err := repo.List(context.Background(), adoptions.ListQuery{Scenario: scenario, Limit: 20})
	require.NoError(t, err)
	return rows
}
