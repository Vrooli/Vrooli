package components

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type closureReader struct {
	byID      map[string]Component
	byLibrary map[string]Component
	versions  map[string]ComponentVersion
}

func (r closureReader) Get(_ context.Context, id string) (Component, error) {
	asset, ok := r.byID[id]
	if !ok {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: id}
	}
	return asset, nil
}

func (r closureReader) GetByLibraryID(_ context.Context, libraryID string) (Component, error) {
	asset, ok := r.byLibrary[libraryID]
	if !ok {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: libraryID}
	}
	return asset, nil
}

func (r closureReader) GetVersion(_ context.Context, id, version string) (ComponentVersion, error) {
	asset, ok := r.versions[id+"@"+version]
	if !ok {
		return ComponentVersion{}, errors.New("version not found")
	}
	return asset, nil
}

func TestResolveDependencyClosureOrdersDependenciesAndDeduplicatesPins(t *testing.T) {
	hook := Component{ID: "hook", LibraryID: "rcl:useFocusTrap", AssetKind: AssetKindHook, LatestVersion: "1.0.0"}
	utility := Component{ID: "utility", LibraryID: "rcl:useEscapeKey", AssetKind: AssetKindHook, LatestVersion: "1.0.0"}
	root := Component{ID: "drawer", LibraryID: "rcl:DrawerShell", AssetKind: AssetKindComponent, LatestVersion: "2.0.0", Dependencies: []AssetDependency{
		{LibraryID: utility.LibraryID, Version: "1.0.0"},
		{LibraryID: hook.LibraryID, Version: "1.0.0"},
		{LibraryID: hook.LibraryID, Version: "1.0.0"},
	}}
	reader := closureReader{
		byID:      map[string]Component{root.ID: root},
		byLibrary: map[string]Component{hook.LibraryID: hook, utility.LibraryID: utility},
		versions: map[string]ComponentVersion{
			"hook@1.0.0":    {ComponentID: hook.ID, Version: "1.0.0"},
			"utility@1.0.0": {ComponentID: utility.ID, Version: "1.0.0"},
			"drawer@2.0.0":  {ComponentID: root.ID, Version: "2.0.0"},
		},
	}

	closure, err := ResolveDependencyClosure(context.Background(), reader, root.ID, "")
	require.NoError(t, err)
	require.Equal(t, []string{"rcl:useEscapeKey", "rcl:useFocusTrap", "rcl:DrawerShell"}, []string{
		closure[0].Asset.LibraryID, closure[1].Asset.LibraryID, closure[2].Asset.LibraryID,
	})
}

func TestResolveDependencyClosureReportsMissingPinAndCycle(t *testing.T) {
	root := Component{ID: "root", LibraryID: "rcl:Root", LatestVersion: "1.0.0", Dependencies: []AssetDependency{{LibraryID: "rcl:Missing", Version: "1.0.0"}}}
	reader := closureReader{byID: map[string]Component{root.ID: root}, byLibrary: map[string]Component{}, versions: map[string]ComponentVersion{}}
	_, err := ResolveDependencyClosure(context.Background(), reader, root.ID, "")
	var missing ErrAssetDependency
	require.ErrorAs(t, err, &missing)
	require.Equal(t, "rcl:Missing", missing.LibraryID)

	a := Component{ID: "a", LibraryID: "rcl:A", LatestVersion: "1.0.0", Dependencies: []AssetDependency{{LibraryID: "rcl:B", Version: "1.0.0"}}}
	b := Component{ID: "b", LibraryID: "rcl:B", LatestVersion: "1.0.0", Dependencies: []AssetDependency{{LibraryID: "rcl:A", Version: "1.0.0"}}}
	reader = closureReader{
		byID:      map[string]Component{a.ID: a},
		byLibrary: map[string]Component{a.LibraryID: a, b.LibraryID: b},
		versions:  map[string]ComponentVersion{},
	}
	_, err = ResolveDependencyClosure(context.Background(), reader, a.ID, "")
	var cycle ErrAssetDependencyCycle
	require.ErrorAs(t, err, &cycle)
	require.Equal(t, []string{"rcl:A@1.0.0", "rcl:B@1.0.0", "rcl:A@1.0.0"}, cycle.Path)
}
