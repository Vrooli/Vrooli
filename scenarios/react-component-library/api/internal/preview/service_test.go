package preview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
)

// fakeComponentsService is a minimal components.Service fake. The
// preview service only uses GetContent; the rest panics so we notice
// if the surface widens unintentionally.
type fakeComponentsService struct {
	getFn               func(ctx context.Context, id string) (components.Component, error)
	listFn              func(ctx context.Context, q components.SearchQuery) ([]components.Component, error)
	getContentFn        func(ctx context.Context, id string) (components.Content, error)
	getVersionContentFn func(ctx context.Context, id, version string) (components.Content, error)
}

func (f *fakeComponentsService) GetContent(ctx context.Context, id string) (components.Content, error) {
	return f.getContentFn(ctx, id)
}

func (f *fakeComponentsService) GetContentAt(ctx context.Context, id, _ string) (components.Content, error) {
	return f.GetContent(ctx, id)
}

func (f *fakeComponentsService) Upsert(context.Context, components.UpsertInput) (components.Component, error) {
	panic("not called")
}

func (f *fakeComponentsService) Get(ctx context.Context, id string) (components.Component, error) {
	if f.getFn != nil {
		return f.getFn(ctx, id)
	}
	return components.Component{ID: id, LibraryID: "react-component-library:PreviewFixture", AssetKind: components.AssetKindComponent, LatestVersion: "1.0.0"}, nil
}

func (f *fakeComponentsService) GetByLibraryID(context.Context, string) (components.Component, error) {
	panic("not called")
}

func (f *fakeComponentsService) List(ctx context.Context, q components.SearchQuery) ([]components.Component, error) {
	if f.listFn != nil {
		return f.listFn(ctx, q)
	}
	panic("not called")
}

func (f *fakeComponentsService) UpdateContent(context.Context, string, components.WriteContentInput) (components.Content, error) {
	panic("not called")
}

func (f *fakeComponentsService) UpdateContentAt(context.Context, string, string, components.WriteContentInput) (components.Content, error) {
	panic("not called")
}

func (f *fakeComponentsService) ListVersions(context.Context, string, int) ([]components.ComponentVersion, error) {
	panic("not called")
}

func (f *fakeComponentsService) GetVersion(context.Context, string, string) (components.ComponentVersion, error) {
	panic("not called")
}

func (f *fakeComponentsService) GetVersionContent(ctx context.Context, id, version string) (components.Content, error) {
	if f.getVersionContentFn == nil {
		panic("not called")
	}
	return f.getVersionContentFn(ctx, id, version)
}

func (f *fakeComponentsService) ListStories(context.Context, components.StoryQuery) ([]components.ComponentStory, error) {
	panic("not called")
}

func (f *fakeComponentsService) ListDesignStyles(context.Context) ([]components.DesignStyle, error) {
	panic("not called")
}

func (f *fakeComponentsService) ValidateDesignStyle(context.Context, string) error {
	panic("not called")
}

func (f *fakeComponentsService) ValidateStyleFit(context.Context, string, string, string) (components.StyleFitVerdict, error) {
	panic("not called")
}

func (f *fakeComponentsService) InitializeComponent(context.Context, components.InitializeComponentInput) (components.InitializeComponentResult, error) {
	panic("not called")
}

func (f *fakeComponentsService) IngestComponent(context.Context, components.IngestComponentInput) (components.IngestComponentResult, error) {
	panic("not called")
}

func (f *fakeComponentsService) CreateComponentVersion(context.Context, components.CreateComponentVersionInput) (components.CreateComponentVersionResult, error) {
	panic("not called")
}

func (f *fakeComponentsService) UpdateComponentManifest(context.Context, components.UpdateComponentManifestInput) (components.Component, error) {
	panic("not called")
}

func TestService_GetBundle_RoundTrip(t *testing.T) {
	comp := &fakeComponentsService{
		getContentFn: func(_ context.Context, id string) (components.Content, error) {
			require.Equal(t, "cmp-1", id)
			return components.Content{
				Body:       "export const Hello = () => <div>hi</div>;\n",
				SourcePath: "components/Hello.tsx",
				SHA256:     "src-sha",
			}, nil
		},
	}
	svc := NewService(comp, NewEsbuilder())

	bundle, err := svc.GetBundle(context.Background(), "cmp-1")
	require.NoError(t, err)
	require.Equal(t, "components/Hello.tsx", bundle.SourcePath)
	require.NotEmpty(t, bundle.SHA256)
	// JSX transformed to React calls + bare specifier preserved.
	require.True(t, strings.Contains(bundle.JS, "export"), "expected `export` token in %q", bundle.JS)
	require.True(t, strings.Contains(bundle.JS, "react/jsx-runtime") || strings.Contains(bundle.JS, "jsx("),
		"expected automatic JSX runtime in %q", bundle.JS)
}

func TestEsbuilder_BundlesVersionedComponentLibraryImports(t *testing.T) {
	root := t.TempDir()
	classMergeDir := filepath.Join(root, "library", "foundations", "ClassMerge", "versions", "1.0.1")
	componentDir := filepath.Join(root, "library", "components", "Root", "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(classMergeDir, 0o755))
	require.NoError(t, os.MkdirAll(componentDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(classMergeDir, "ClassMerge.ts"), []byte("export const helper = (value: string) => value;\n"), 0o644))
	source := `import { helper } from "@vrooli/react-component-library/ClassMerge/1.0.1";
export const Root = () => helper("root");
`
	sourcePath := filepath.Join(componentDir, "Root.tsx")
	require.NoError(t, os.WriteFile(sourcePath, []byte(source), 0o644))

	bundle, _, err := NewEsbuilder().BuildBundle(context.Background(), source, sourcePath)
	require.NoError(t, err)
	require.Contains(t, bundle, "root")
	require.NotContains(t, bundle, "@vrooli/react-component-library/ClassMerge/1.0.1")
}

func TestService_GetBundle_PropagatesComponentsError(t *testing.T) {
	wantErr := errors.New("registry blew up")
	comp := &fakeComponentsService{
		getContentFn: func(context.Context, string) (components.Content, error) {
			return components.Content{}, wantErr
		},
	}
	svc := NewService(comp, NewEsbuilder())
	_, err := svc.GetBundle(context.Background(), "missing")
	require.ErrorIs(t, err, wantErr)
}

func TestService_GetBundle_BundlerSyntaxError(t *testing.T) {
	comp := &fakeComponentsService{
		getContentFn: func(context.Context, string) (components.Content, error) {
			return components.Content{
				Body:       "export const Broken = () => <div", // unclosed JSX
				SourcePath: "components/Broken.tsx",
			}, nil
		},
	}
	svc := NewService(comp, NewEsbuilder())
	_, err := svc.GetBundle(context.Background(), "cmp-broken")
	require.Error(t, err)
	var ee ErrBundle
	require.True(t, errors.As(err, &ee), "expected ErrBundle, got %T (%v)", err, err)
	require.Equal(t, "components/Broken.tsx", ee.SourcePath)
	require.NotEmpty(t, ee.Messages)
}

func TestService_GetBundle_DigestStable(t *testing.T) {
	body := "export const A = () => <span>a</span>;\n"
	comp := &fakeComponentsService{
		getContentFn: func(context.Context, string) (components.Content, error) {
			return components.Content{Body: body, SourcePath: "components/A.tsx"}, nil
		},
	}
	svc := NewService(comp, NewEsbuilder())
	first, err := svc.GetBundle(context.Background(), "cmp-a")
	require.NoError(t, err)
	second, err := svc.GetBundle(context.Background(), "cmp-a")
	require.NoError(t, err)
	require.Equal(t, first.SHA256, second.SHA256)
	require.Equal(t, first.JS, second.JS)
}

func TestService_GetBundleVersion_UsesImmutableVersionContent(t *testing.T) {
	comp := &fakeComponentsService{
		getContentFn: func(context.Context, string) (components.Content, error) {
			return components.Content{Body: "export const Current = () => null", SourcePath: "current.tsx"}, nil
		},
		getVersionContentFn: func(_ context.Context, id, version string) (components.Content, error) {
			require.Equal(t, "cmp-1", id)
			require.Equal(t, "1.0.0", version)
			return components.Content{Body: "export const Historical = () => <div>v1</div>", SourcePath: "versions/1.0.0/Historical.tsx"}, nil
		},
	}
	bundle, err := NewService(comp, NewEsbuilder()).GetBundleVersion(context.Background(), "cmp-1", "1.0.0")
	require.NoError(t, err)
	require.Equal(t, "versions/1.0.0/Historical.tsx", bundle.SourcePath)
	require.Contains(t, bundle.JS, "Historical")
}

func TestService_GetBundleVersionWithFrameBundlesSubjectAndCatalogFrame(t *testing.T) {
	comp := &fakeComponentsService{
		getFn: func(_ context.Context, id string) (components.Component, error) {
			if id == "frame-id" {
				return components.Component{ID: id, CatalogID: "navigation.page", LibraryID: "react-component-library:PageFrame", LatestVersion: "1.0.0"}, nil
			}
			return components.Component{ID: id, LibraryID: "react-component-library:SidebarShell", LatestVersion: "1.2.0"}, nil
		},
		listFn: func(context.Context, components.SearchQuery) ([]components.Component, error) {
			return []components.Component{{ID: "frame-id"}}, nil
		},
		getContentFn: func(context.Context, string) (components.Content, error) {
			return components.Content{Body: "export default function Subject() { return <aside>subject</aside> }", SourcePath: "components/Subject.tsx"}, nil
		},
		getVersionContentFn: func(_ context.Context, id, version string) (components.Content, error) {
			if id == "frame-id" {
				return components.Content{Body: "export function PageFrame({ regions }) { return <main data-frame>{regions.navigation}{regions.content}</main> }", SourcePath: "components/PageFrame/versions/1.0.0/PageFrame.tsx"}, nil
			}
			return components.Content{Body: "export default function Subject() { return <aside>subject</aside> }", SourcePath: "components/Subject/versions/1.2.0/Subject.tsx"}, nil
		},
	}
	repoRoot := filepath.Join("..", "..", "..", "..", "..")
	svc := NewServiceWithDepsAtRoot(comp, NewEsbuilder(), nil, repoRoot)
	bundle, err := svc.GetBundleVersionWithFrame(context.Background(), "subject-id", "1.2.0", &components.StoryFrame{Asset: "navigation.page", Version: "1.0.0", Region: "navigation", Capability: "navigation", Fixture: "fixtures.resource-collection"})
	// The production service needs only the repository root to load the catalog;
	// use the concrete root directly above so this remains a focused bundler
	// assertion rather than a handler integration test.
	require.NoError(t, err)
	require.Contains(t, bundle.JS, "Subject")
	require.NotEmpty(t, bundle.FrameJS)
	require.Contains(t, bundle.FrameJS, "data-frame")
	require.Equal(t, "navigation.page", bundle.FrameAsset)
	require.Equal(t, "navigation", bundle.FrameRegion)
	require.Contains(t, bundle.FixtureJSON, "fixtures.resource-collection")
}

func TestService_GetBundleVersionWithFrameRejectsAmbiguousImplementations(t *testing.T) {
	comp := &fakeComponentsService{
		getFn: func(_ context.Context, id string) (components.Component, error) {
			if id == "frame-a" || id == "frame-b" {
				return components.Component{ID: id, CatalogID: "navigation.page", LibraryID: "react-component-library:PageFrame", LatestVersion: "1.0.0"}, nil
			}
			return components.Component{ID: id, LibraryID: "react-component-library:Subject", LatestVersion: "1.0.0"}, nil
		},
		listFn: func(context.Context, components.SearchQuery) ([]components.Component, error) {
			return []components.Component{{ID: "frame-a"}, {ID: "frame-b"}}, nil
		},
		getContentFn: func(context.Context, string) (components.Content, error) {
			return components.Content{Body: "export default function Subject() { return <aside>subject</aside> }", SourcePath: "components/Subject.tsx"}, nil
		},
		getVersionContentFn: func(context.Context, string, string) (components.Content, error) {
			return components.Content{Body: "export function PageFrame() { return <main /> }", SourcePath: "components/PageFrame.tsx"}, nil
		},
	}
	repoRoot := filepath.Join("..", "..", "..", "..", "..")
	svc := NewServiceWithDepsAtRoot(comp, NewEsbuilder(), nil, repoRoot)
	_, err := svc.GetBundleVersionWithFrame(context.Background(), "subject-id", "1.0.0", &components.StoryFrame{Asset: "navigation.page", Version: "1.0.0", Region: "navigation", Capability: "navigation"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ambiguous react-vite implementations")
}

func TestService_GetBundleVersionWithCompositionHarnessInjectsGenericPreviewAsset(t *testing.T) {
	comp := &fakeComponentsService{
		getFn: func(_ context.Context, id string) (components.Component, error) {
			return components.Component{ID: id, LibraryID: "react-component-library:Skeleton", LatestVersion: "1.0.0"}, nil
		},
		getVersionContentFn: func(_ context.Context, _, _ string) (components.Content, error) {
			return components.Content{Body: "export default function Subject() { return <div>subject</div> }", SourcePath: "primitives/Skeleton/versions/1.0.0/Skeleton.tsx"}, nil
		},
	}
	repoRoot := filepath.Join("..", "..", "..", "..", "..")
	svc := NewServiceWithDepsAtRoot(comp, NewEsbuilder(), nil, repoRoot)
	bundle, err := svc.(*service).GetBundleVersionWithFrameAndHarness(context.Background(), "subject-id", "1.0.0", nil, &components.StoryHarnessRef{Asset: "preview.showcase", Version: "1.0.0", Export: "Showcase"})
	require.NoError(t, err)
	require.Contains(t, bundle.CompositionHarnessJS, "data-preview-harness")
	require.NotContains(t, bundle.CompositionHarnessJS, "library/components/")
}

func TestValidateSpecimenSourceRejectsNonDeterministicBrowserAccess(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		want   string
	}{
		{name: "network", source: "export const Story = () => fetch('/api');", want: "network fetch"},
		{name: "storage", source: "export const Story = () => localStorage.getItem('x');", want: "localStorage"},
		{name: "process", source: "export const Story = () => process.env.MODE;", want: "process access"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateSpecimenSource(test.source)
			require.Error(t, err)
			require.Contains(t, err.Error(), test.want)
		})
	}
}

func TestValidateSpecimenSourceAllowsDeterministicRecipe(t *testing.T) {
	require.NoError(t, validateSpecimenSource(`import { useState } from "react"; export function Story() { const [open] = useState(false); return <button>{String(open)}</button>; }`))
}

func TestResolveDeterministicFixtureIsStableAndIncludesAdversarialState(t *testing.T) {
	first, err := ResolveDeterministicFixture("fixtures.resource-collection", "1.0.0", "overflow")
	require.NoError(t, err)
	second, err := ResolveDeterministicFixture("fixtures.resource-collection", "1.0.0", "overflow")
	require.NoError(t, err)
	require.Equal(t, "rcl-fixture-v1", first.Seed)
	require.Equal(t, first.Clock, second.Clock)
	require.Len(t, first.Records, 4)
	require.NotEmpty(t, first.Records[3]["name"])
}

func TestResolveDeterministicFixtureRejectsUnknownFamilyAndMissingVersion(t *testing.T) {
	_, err := ResolveDeterministicFixture("fixtures.unknown", "1.0.0", "typical")
	require.Error(t, err)
	_, err = ResolveDeterministicFixture("fixtures.resource-collection", "", "typical")
	require.Error(t, err)
}

func TestResolveDeterministicFixtureSupportsNavigationHealthAndRecoveryFamilies(t *testing.T) {
	for _, asset := range []string{"fixtures.navigation-tree", "fixtures.status-health", "fixtures.error-recovery"} {
		fixture, err := ResolveDeterministicFixture(asset, "1.0.0", "failure")
		require.NoError(t, err, asset)
		require.NotEmpty(t, fixture.DataShapes, asset)
		require.NotEmpty(t, fixture.Error, asset)
	}
}

func TestService_GetBundleVersionWithCompositionHarnessBundlesEveryRegisteredFamily(t *testing.T) {
	comp := &fakeComponentsService{
		getFn: func(_ context.Context, id string) (components.Component, error) {
			return components.Component{ID: id, LibraryID: "react-component-library:PreviewSubject", LatestVersion: "1.0.0"}, nil
		},
		getVersionContentFn: func(_ context.Context, _, _ string) (components.Content, error) {
			return components.Content{Body: "export default function Subject() { return <button>subject</button> }", SourcePath: "components/PreviewSubject/versions/1.0.0/PreviewSubject.tsx"}, nil
		},
	}
	repoRoot := filepath.Join("..", "..", "..", "..", "..")
	svc := NewServiceWithDepsAtRoot(comp, NewEsbuilder(), nil, repoRoot).(*service)
	families := []struct {
		asset  string
		export string
	}{
		{asset: "preview.showcase", export: "Showcase"},
		{asset: "preview.controlled-state", export: "ControlledState"},
		{asset: "preview.state-transition", export: "StateTransition"},
		{asset: "preview.async-state", export: "AsyncState"},
		{asset: "preview.recovery", export: "Recovery"},
		{asset: "preview.data-state", export: "DataState"},
		{asset: "preview.overlay-interaction", export: "OverlayInteraction"},
		{asset: "preview.responsive-mode", export: "ResponsiveMode"},
		{asset: "preview.hook-contract", export: "HookContract"},
	}
	for _, family := range families {
		t.Run(family.asset, func(t *testing.T) {
			bundle, err := svc.GetBundleVersionWithFrameAndHarness(context.Background(), "subject-id", "1.0.0", nil, &components.StoryHarnessRef{Asset: family.asset, Version: "1.0.0", Export: family.export})
			require.NoError(t, err)
			require.Contains(t, bundle.CompositionHarnessJS, "data-preview-harness")
		})
	}
}

func TestService_GetBundleVersionWithCompositionHarnessRejectsUndeclaredConfig(t *testing.T) {
	comp := &fakeComponentsService{
		getFn: func(_ context.Context, id string) (components.Component, error) {
			return components.Component{ID: id, LibraryID: "react-component-library:Button", LatestVersion: "1.0.0"}, nil
		},
		getVersionContentFn: func(_ context.Context, _, _ string) (components.Content, error) {
			return components.Content{Body: "export default function Subject() { return <button>subject</button> }", SourcePath: "components/Button/versions/1.0.0/Button.tsx"}, nil
		},
	}
	repoRoot := filepath.Join("..", "..", "..", "..", "..")
	svc := NewServiceWithDepsAtRoot(comp, NewEsbuilder(), nil, repoRoot).(*service)
	_, err := svc.GetBundleVersionWithFrameAndHarness(context.Background(), "subject-id", "1.0.0", nil, &components.StoryHarnessRef{
		Asset: "preview.showcase", Version: "1.0.0", Export: "Showcase",
		Config: []byte(`{"title":"Button","notRegistered":true}`),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `config key "notRegistered" is not declared`)
}

func TestService_GetBundleVersionWithCompositionDigestIncludesReferences(t *testing.T) {
	comp := &fakeComponentsService{
		getContentFn: func(context.Context, string) (components.Content, error) {
			return components.Content{Body: "export default function Subject() { return <button>subject</button> }", SourcePath: "components/Subject.tsx"}, nil
		},
		getVersionContentFn: func(context.Context, string, string) (components.Content, error) {
			return components.Content{Body: "export default function Subject() { return <button>subject</button> }", SourcePath: "components/Subject/versions/1.0.0/Subject.tsx"}, nil
		},
	}
	repoRoot := filepath.Join("..", "..", "..", "..", "..")
	svc := NewServiceWithDepsAtRoot(comp, NewEsbuilder(), nil, repoRoot).(*service)
	first, err := svc.GetBundleVersionWithFrameAndHarness(context.Background(), "subject-id", "1.0.0", nil, &components.StoryHarnessRef{
		Asset: "preview.showcase", Version: "1.0.0", Export: "Showcase", Config: []byte(`{"title":"One"}`),
	})
	require.NoError(t, err)
	second, err := svc.GetBundleVersionWithFrameAndHarness(context.Background(), "subject-id", "1.0.0", nil, &components.StoryHarnessRef{
		Asset: "preview.showcase", Version: "1.0.0", Export: "Showcase", Config: []byte(`{"title":"Two"}`),
	})
	require.NoError(t, err)
	require.NotEqual(t, first.SHA256, second.SHA256)
}

func TestService_GetBundle_AllowsHookFixtures(t *testing.T) {
	comp := &fakeComponentsService{
		getFn: func(_ context.Context, id string) (components.Component, error) {
			return components.Component{ID: id, LibraryID: "react-component-library:useFocusTrap", AssetKind: components.AssetKindHook}, nil
		},
		getContentFn: func(context.Context, string) (components.Content, error) {
			return components.Content{Body: "export const useFocusTrap = () => {}", SourcePath: "hooks/useFocusTrap.ts"}, nil
		},
	}
	bundle, err := NewService(comp, NewEsbuilder()).GetBundle(context.Background(), "hook-1")
	require.NoError(t, err)
	require.Contains(t, bundle.JS, "useFocusTrap")
}

func TestEsbuilderBundlesRelativeImports(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "components", "Relative"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Relative", "label.ts"), []byte(`export const label = "Relative import works";`), 0o644))
	tsx := `import { label } from "./label";
export default function RelativeDemo() {
  return <button>{label}</button>;
}`
	bundler := NewEsbuilderWithRoot(root)

	js, warnings, err := bundler.BuildBundle(context.Background(), tsx, "components/Relative/RelativeDemo.tsx")
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Contains(t, js, "Relative import works")
	require.NotContains(t, js, `from "./label"`)
	require.Contains(t, js, "react/jsx-runtime")
}

func TestEsbuilderBundlesCSSImportsIntoOneJavaScriptModule(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "components", "Styled")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "styles.css"), []byte("[data-styled] { color: red; }"), 0o644))

	bundler := NewEsbuilderWithRoot(root)
	js, warnings, err := bundler.BuildBundle(context.Background(), `import "./styles.css";
export const Styled = () => <div data-styled>styled</div>;`, "components/Styled/Styled.tsx")
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Contains(t, js, "data-rcl-asset-style")
	require.Contains(t, js, "[data-styled] { color: red; }")
	require.Contains(t, js, "export")
}

func TestEsbuilderBundlesLocalVrooliPackageImports(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "packages", "audio-capture-browser", "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "packages", "audio-capture-browser", "src", "index.ts"), []byte(`export const localVoice = "local package works";`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "components", "Voice"), 0o755))

	ts := `import { localVoice } from "@vrooli/audio-capture-browser";
export const useVoiceInput = () => localVoice;`
	bundler := NewEsbuilderWithRoot(root)

	js, warnings, err := bundler.BuildBundle(context.Background(), ts, "components/Voice/useVoiceInput.ts")
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Contains(t, js, "local package works")
	require.NotContains(t, js, `from "@vrooli/audio-capture-browser"`)
}
