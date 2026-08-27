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

func TestResolveDependencyClosureIncludesVersionedFoundationDependencies(t *testing.T) {
	foundation := Component{ID: "tokens", LibraryID: "rcl:Tokens", AssetKind: AssetKindFoundation, LatestVersion: "1.0.0"}
	root := Component{ID: "text", LibraryID: "rcl:Text", AssetKind: AssetKindComponent, LatestVersion: "1.0.0", Dependencies: []AssetDependency{{LibraryID: foundation.LibraryID, Version: "1.0.0"}}}
	reader := closureReader{
		byID:      map[string]Component{root.ID: root},
		byLibrary: map[string]Component{foundation.LibraryID: foundation},
		versions: map[string]ComponentVersion{
			"tokens@1.0.0": {ComponentID: foundation.ID, LibraryID: foundation.LibraryID, Version: "1.0.0"},
			"text@1.0.0":   {ComponentID: root.ID, LibraryID: root.LibraryID, Version: "1.0.0"},
		},
	}

	closure, err := ResolveDependencyClosure(context.Background(), reader, root.ID, "")
	require.NoError(t, err)
	require.Equal(t, []string{"rcl:Tokens", "rcl:Text"}, []string{closure[0].Asset.LibraryID, closure[1].Asset.LibraryID})
	require.Equal(t, AssetKindFoundation, closure[0].Asset.AssetKind)
}

func TestResolveDependencyClosureReportsMissingPinAndCycle(t *testing.T) {
	root := Component{ID: "root", LibraryID: "rcl:Root", LatestVersion: "1.0.0", Dependencies: []AssetDependency{{LibraryID: "rcl:Missing", Version: "1.0.0"}}}
	reader := closureReader{byID: map[string]Component{root.ID: root}, byLibrary: map[string]Component{}, versions: map[string]ComponentVersion{"root@1.0.0": {ComponentID: root.ID, Version: "1.0.0"}}}
	_, err := ResolveDependencyClosure(context.Background(), reader, root.ID, "")
	var missing ErrAssetDependency
	require.ErrorAs(t, err, &missing)
	require.Equal(t, "rcl:Missing", missing.LibraryID)

	a := Component{ID: "a", LibraryID: "rcl:A", LatestVersion: "1.0.0", Dependencies: []AssetDependency{{LibraryID: "rcl:B", Version: "1.0.0"}}}
	b := Component{ID: "b", LibraryID: "rcl:B", LatestVersion: "1.0.0", Dependencies: []AssetDependency{{LibraryID: "rcl:A", Version: "1.0.0"}}}
	reader = closureReader{
		byID:      map[string]Component{a.ID: a},
		byLibrary: map[string]Component{a.LibraryID: a, b.LibraryID: b},
		versions: map[string]ComponentVersion{
			"a@1.0.0": {ComponentID: a.ID, Version: "1.0.0"},
			"b@1.0.0": {ComponentID: b.ID, Version: "1.0.0"},
		},
	}
	_, err = ResolveDependencyClosure(context.Background(), reader, a.ID, "")
	var cycle ErrAssetDependencyCycle
	require.ErrorAs(t, err, &cycle)
	require.Equal(t, []string{"rcl:A@1.0.0", "rcl:B@1.0.0", "rcl:A@1.0.0"}, cycle.Path)
}

func TestResolveDependencyClosureExcludesSuggestionsUnlessOptedIn(t *testing.T) {
	root := Component{ID: "root", LibraryID: "rcl:Root", LatestVersion: "1.0.0", Dependencies: []AssetDependency{{LibraryID: "rcl:Required", Version: "1.0.0", Kind: DependencyRequires}, {LibraryID: "rcl:Suggested", Version: "1.0.0", Kind: DependencySuggests}}}
	required := Component{ID: "required", LibraryID: "rcl:Required", LatestVersion: "1.0.0"}
	suggested := Component{ID: "suggested", LibraryID: "rcl:Suggested", LatestVersion: "1.0.0"}
	reader := closureReader{byID: map[string]Component{"root": root}, byLibrary: map[string]Component{"rcl:Root": root, "rcl:Required": required, "rcl:Suggested": suggested}, versions: map[string]ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0"}, "required@1.0.0": {ComponentID: "required", Version: "1.0.0"}, "suggested@1.0.0": {ComponentID: "suggested", Version: "1.0.0"}}}
	closure, err := ResolveDependencyClosure(context.Background(), reader, "root", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(closure) != 2 {
		t.Fatalf("default closure length = %d, want root + requires", len(closure))
	}
	closure, err = ResolveDependencyClosureWithOptions(context.Background(), reader, "root", "1.0.0", []string{"rcl:Suggested"})
	if err != nil {
		t.Fatal(err)
	}
	if len(closure) != 3 {
		t.Fatalf("opted-in closure length = %d, want root + requires + suggests", len(closure))
	}
}

func TestResolveDependencyClosureReportSeparatesSuggestionsAndTemplatePorts(t *testing.T) {
	root := Component{ID: "root", LibraryID: "rcl:Root", LatestVersion: "1.0.0", Expects: []string{"design-tokens", "ui-provider"}, Dependencies: []AssetDependency{{LibraryID: "rcl:Suggested", Version: "1.0.0", Kind: DependencySuggests}}}
	suggested := Component{ID: "suggested", LibraryID: "rcl:Suggested", LatestVersion: "1.0.0"}
	reader := closureReader{byID: map[string]Component{"root": root}, byLibrary: map[string]Component{"rcl:Suggested": suggested}, versions: map[string]ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0"}}}
	report, err := ResolveDependencyClosureReport(context.Background(), reader, "root", "1.0.0", nil, []string{"design-tokens", "ui-provider"}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"design-tokens", "ui-provider"}, report.SatisfiedPorts)
	require.Equal(t, []string{"rcl:Suggested"}, report.AvailableSuggestions)
	require.Len(t, report.Assets, 1)
}

func TestResolvePortPrecedenceTemplateAndRankRules(t *testing.T) {
	got, err := ResolvePortPrecedence([]string{"design-tokens", "router"}, []string{"design-tokens"}, []PortCandidate{{Port: "router", Source: "adopted-high", Rank: 3}, {Port: "router", Source: "adopted-low", Rank: 1}})
	require.NoError(t, err)
	require.Equal(t, map[string]string{"design-tokens": "template", "router": "adopted-low"}, got.Sources)
	_, err = ResolvePortPrecedence([]string{"router"}, nil, []PortCandidate{{Port: "router", Source: "a", Rank: 1}, {Port: "router", Source: "b", Rank: 1}})
	require.Error(t, err)
}
