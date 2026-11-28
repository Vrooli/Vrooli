package preview

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"react-component-library/internal/components"
	domainpreview "react-component-library/internal/preview"
)

type imageComponentService struct {
	components.Service
	component components.Component
	err       error
}

func testPreviewRepoRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	adapter := filepath.Join(root, "templates", "design", "vrooli-default", "adapters", "react-vite-tailwind")
	require.NoError(t, os.MkdirAll(adapter, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(adapter, "tokens.css"), []byte(":root{}"), 0o600))
	return root
}

func (s imageComponentService) Get(_ context.Context, _ string) (components.Component, error) {
	return s.component, s.err
}

func (s imageComponentService) GetByLibraryID(_ context.Context, _ string) (components.Component, error) {
	return s.component, s.err
}

func TestServePreviewImageUsesComponentVersionAndMapsFailures(t *testing.T) {
	repo := testPreviewRepoRoot(t)
	h := NewHarnessHandlerWithStoriesAtRoot(nil, imageComponentService{component: components.Component{
		ID: "component-id", Slug: "Button", LatestVersion: "2.0.0", ManifestPath: "components/Button/component.json",
	}}, log.New(os.Stderr, "", 0), repo)
	image := []byte("published-preview-image")
	imagePath := filepath.Join(repo, "scenarios", "react-component-library", "library", "components", "Button", "versions", "2.0.0", "preview.png")
	require.NoError(t, os.MkdirAll(filepath.Dir(imagePath), 0o700))
	require.NoError(t, os.WriteFile(imagePath, image, 0o600))

	request := httptest.NewRequest(http.MethodGet, "/preview/component-id/preview.png", nil)
	request = mux.SetURLVars(request, map[string]string{"id": "component-id"})
	response := httptest.NewRecorder()
	h.ServePreviewImage(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, image, response.Body.Bytes())
	require.Equal(t, "public, max-age=60", response.Header().Get("Cache-Control"))

	request = httptest.NewRequest(http.MethodGet, "/preview/component-id/preview.png?version=1.0.0", nil)
	request = mux.SetURLVars(request, map[string]string{"id": "component-id"})
	response = httptest.NewRecorder()
	h.ServePreviewImage(response, request)
	require.Equal(t, http.StatusNotFound, response.Code, "an absent explicit version must not fall back to latest")

	for _, version := range []string{"../outside", "1.0.0-draft.1"} {
		request = httptest.NewRequest(http.MethodGet, "/preview/component-id/preview.png?version="+url.QueryEscape(version), nil)
		request = mux.SetURLVars(request, map[string]string{"id": "component-id"})
		response = httptest.NewRecorder()
		h.ServePreviewImage(response, request)
		require.Equal(t, http.StatusBadRequest, response.Code, version)
		require.Contains(t, response.Body.String(), "invalid preview version")
	}

	request = httptest.NewRequest(http.MethodGet, "/preview/component-id/preview.png", nil)
	response = httptest.NewRecorder()
	h.ServePreviewImage(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code, "the handler requires the router's component id")
}

func TestServePreviewImageUsesManifestPathAndRejectsEscapingPath(t *testing.T) {
	repo := testPreviewRepoRoot(t)
	component := components.Component{ID: "text", Slug: "Text", LatestVersion: "1.0.0", ManifestPath: "primitives/Text/component.json"}
	h := NewHarnessHandlerWithStoriesAtRoot(nil, imageComponentService{component: component}, nil, repo)
	path := filepath.Join(repo, "scenarios", "react-component-library", "library", "primitives", "Text", "versions", "1.0.0", "preview.png")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("primitive-preview"), 0o600))
	request := httptest.NewRequest(http.MethodGet, "/preview/text/preview.png", nil)
	request = mux.SetURLVars(request, map[string]string{"id": "text"})
	response := httptest.NewRecorder()
	h.ServePreviewImage(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, "primitive-preview", response.Body.String())

	component.ManifestPath = "../outside/component.json"
	h = NewHarnessHandlerWithStoriesAtRoot(nil, imageComponentService{component: component}, nil, repo)
	request = httptest.NewRequest(http.MethodGet, "/preview/text/preview.png", nil)
	request = mux.SetURLVars(request, map[string]string{"id": "text"})
	response = httptest.NewRecorder()
	h.ServePreviewImage(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "preview path escapes library root")
}

func TestServePreviewImageMapsComponentLookupError(t *testing.T) {
	repo := testPreviewRepoRoot(t)
	h := NewHarnessHandlerWithStoriesAtRoot(nil, imageComponentService{err: components.ErrComponentNotFound{IDOrLibraryID: "missing"}}, nil, repo)
	request := httptest.NewRequest(http.MethodGet, "/preview/missing/preview.png", nil)
	request = mux.SetURLVars(request, map[string]string{"id": "missing"})
	response := httptest.NewRecorder()
	h.ServePreviewImage(response, request)
	require.Equal(t, http.StatusNotFound, response.Code)
}

func TestServeBaseStylesFixtureReturnsExactPublishedCSSAndValidatesInput(t *testing.T) {
	repo := testPreviewRepoRoot(t)
	h := NewHarnessHandlerWithStoriesAtRoot(nil, nil, nil, repo)
	versionDir := filepath.Join(repo, "scenarios", "react-component-library", "library", "foundations", "BaseStyles", "versions", "1.2.3")
	require.NoError(t, os.MkdirAll(versionDir, 0o700))
	sourcePath := filepath.Join(versionDir, "BaseStyles.ts")
	require.NoError(t, os.WriteFile(sourcePath, []byte("export const baseStyles = `:root { --space-md: 24px; }`;"), 0o600))

	request := httptest.NewRequest(http.MethodGet, "/preview/foundations/base-styles-fixture.html?version=1.2.3", nil)
	response := httptest.NewRecorder()
	h.ServeBaseStylesFixture(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, "text/html; charset=utf-8", response.Header().Get("Content-Type"))
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Contains(t, response.Body.String(), "--space-md: 24px")
	require.Contains(t, response.Body.String(), `id="canonical"`)
	require.Contains(t, response.Body.String(), `id="override"`)
	require.Contains(t, response.Body.String(), `data-testid="base-styles-measurement"`)

	for _, version := range []string{"", "../outside", ".hidden", `1.2.3\evil`} {
		request = httptest.NewRequest(http.MethodGet, "/fixture?version="+url.QueryEscape(version), nil)
		response = httptest.NewRecorder()
		h.ServeBaseStylesFixture(response, request)
		require.Equal(t, http.StatusBadRequest, response.Code, version)
	}

	request = httptest.NewRequest(http.MethodGet, "/fixture?version=9.9.9", nil)
	response = httptest.NewRecorder()
	h.ServeBaseStylesFixture(response, request)
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Contains(t, response.Body.String(), "BaseStyles version is unavailable")

	require.NoError(t, os.WriteFile(sourcePath, []byte("export const baseStyles = `unterminated"), 0o600))
	request = httptest.NewRequest(http.MethodGet, "/fixture?version=1.2.3", nil)
	response = httptest.NewRecorder()
	h.ServeBaseStylesFixture(response, request)
	require.Equal(t, http.StatusUnprocessableEntity, response.Code)
	require.Contains(t, response.Body.String(), "BaseStyles source is malformed")
}

func TestToConnectErrorPreservesPreviewAndComponentDomainCodes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code connect.Code
	}{
		{name: "nil", err: nil, code: connect.CodeUnknown},
		{name: "invalid bundle", err: domainpreview.ErrBundle{SourcePath: "Button.tsx", Messages: []string{"syntax error"}}, code: connect.CodeInvalidArgument},
		{name: "evicted dependency", err: domainpreview.ErrEvictedDependency{LibraryID: "rcl:Child", Version: "1.2.3"}, code: connect.CodeFailedPrecondition},
		{name: "component not found", err: components.ErrComponentNotFound{IDOrLibraryID: "missing"}, code: connect.CodeNotFound},
		{name: "path escape", err: components.ErrPathEscape{SourcePath: "../outside.tsx", Root: "/library"}, code: connect.CodeInvalidArgument},
		{name: "unknown failure", err: errors.New("storage unavailable"), code: connect.CodeInternal},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toConnectError(tc.err)
			if tc.err == nil {
				require.NoError(t, got)
				return
			}
			require.Equal(t, tc.code, connect.CodeOf(got))
			require.Contains(t, got.Error(), tc.err.Error())
		})
	}
}
