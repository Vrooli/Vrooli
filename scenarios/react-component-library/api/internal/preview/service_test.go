package preview

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
)

// fakeComponentsService is a minimal components.Service fake. The
// preview service only uses GetContent; the rest panics so we notice
// if the surface widens unintentionally.
type fakeComponentsService struct {
	getContentFn func(ctx context.Context, id string) (components.Content, error)
}

func (f *fakeComponentsService) GetContent(ctx context.Context, id string) (components.Content, error) {
	return f.getContentFn(ctx, id)
}

func (f *fakeComponentsService) Upsert(context.Context, components.UpsertInput) (components.Component, error) {
	panic("not called")
}

func (f *fakeComponentsService) Get(context.Context, string) (components.Component, error) {
	panic("not called")
}

func (f *fakeComponentsService) GetByLibraryID(context.Context, string) (components.Component, error) {
	panic("not called")
}

func (f *fakeComponentsService) List(context.Context, components.SearchQuery) ([]components.Component, error) {
	panic("not called")
}

func (f *fakeComponentsService) UpdateContent(context.Context, string, components.WriteContentInput) (components.Content, error) {
	panic("not called")
}

func (f *fakeComponentsService) ListVersions(context.Context, string, int) ([]components.ComponentVersion, error) {
	panic("not called")
}

func (f *fakeComponentsService) GetVersion(context.Context, string, string) (components.ComponentVersion, error) {
	panic("not called")
}

func (f *fakeComponentsService) GetVersionContent(context.Context, string, string) (components.Content, error) {
	panic("not called")
}

func (f *fakeComponentsService) InitializeComponent(context.Context, components.InitializeComponentInput) (components.InitializeComponentResult, error) {
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
