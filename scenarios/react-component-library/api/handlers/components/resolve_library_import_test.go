package components_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"
	componentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components"
	handlers "react-component-library/handlers/components"
	internalcomponents "react-component-library/internal/components"
)

type importResolutionService struct {
	internalcomponents.Service
	components               []internalcomponents.Component
	version                  internalcomponents.ComponentVersion
	listErr, versionErr      error
	listed                   internalcomponents.SearchQuery
	gotComponent, gotVersion string
}

func (s *importResolutionService) List(_ context.Context, q internalcomponents.SearchQuery) ([]internalcomponents.Component, error) {
	s.listed = q
	return s.components, s.listErr
}

func (s *importResolutionService) GetVersion(_ context.Context, componentID, version string) (internalcomponents.ComponentVersion, error) {
	s.gotComponent, s.gotVersion = componentID, version
	return s.version, s.versionErr
}

func TestResolveLibraryImportContract(t *testing.T) {
	component := internalcomponents.Component{ID: "button-id", LibraryID: "react-component-library:button", Slug: "button", CatalogID: "forms.button", LatestVersion: "2.0.0", Description: "Manifest summary"}
	tests := []struct {
		name                                    string
		specifier                               string
		service                                 *importResolutionService
		wantResolved                            bool
		wantDiagnostic                          string
		wantVersion, wantDescription, wantMatch string
		wantCode                                connect.Code
		wantError                               bool
	}{
		{name: "foreign specifier stays a normal unresolved result", specifier: "react", service: &importResolutionService{}, wantDiagnostic: "not a React Component Library"},
		{name: "relative import stays unresolved", specifier: "../Button", service: &importResolutionService{}, wantDiagnostic: "local to the current file"},
		{name: "missing asset", specifier: "@vrooli/react-component-library/Ghost", service: &importResolutionService{}, wantDiagnostic: "does not resolve to a catalog asset"},
		{name: "list failure maps to connect error", specifier: "@vrooli/react-component-library/button", service: &importResolutionService{listErr: errors.New("catalog unavailable")}, wantCode: connect.CodeInternal, wantError: true},
		{name: "version failure is diagnostic", specifier: "@vrooli/react-component-library/button/1.2.3", service: &importResolutionService{components: []internalcomponents.Component{component}, versionErr: errors.New("version missing")}, wantDiagnostic: "does not resolve to react-component-library:button@1.2.3", wantVersion: "1.2.3"},
		{name: "latest resolves by library id with manifest description", specifier: " @vrooli/react-component-library/button ", service: &importResolutionService{components: []internalcomponents.Component{component}, version: internalcomponents.ComponentVersion{Version: "2.0.0", SourcePath: "components/button.tsx"}}, wantResolved: true, wantVersion: "2.0.0", wantDescription: "Manifest summary", wantMatch: "button"},
		{name: "slug fallback and artifact description", specifier: "@vrooli/react-component-library/legacy/0.9.0", service: &importResolutionService{components: []internalcomponents.Component{{ID: "legacy-id", Slug: "legacy", LibraryID: "legacy:Widget", LatestVersion: "1.0.0"}}, version: internalcomponents.ComponentVersion{Version: "0.9.0", Headers: map[string]string{"description": "Artifact summary"}}}, wantResolved: true, wantVersion: "0.9.0", wantDescription: "Artifact summary", wantMatch: "legacy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handlers.NewConnectHandler(handlers.Deps{Service: tt.service})
			resp, err := h.ResolveLibraryImport(t.Context(), connect.NewRequest(&componentsv1.ResolveLibraryImportRequest{Specifier: tt.specifier}))
			if tt.wantError {
				require.Error(t, err)
				require.Equal(t, tt.wantCode, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)
			got := resp.Msg
			require.Equal(t, tt.wantResolved, got.Resolved)
			if tt.wantDiagnostic != "" {
				require.Contains(t, got.Diagnostic, tt.wantDiagnostic)
			}
			if tt.wantResolved {
				require.Equal(t, tt.wantVersion, got.Version)
				require.Equal(t, tt.wantDescription, got.Description)
				require.Equal(t, tt.wantMatch, tt.service.listed.Match)
				require.Equal(t, 2000, tt.service.listed.Limit)
				require.Equal(t, tt.wantVersion, tt.service.gotVersion)
			}
		})
	}
}
