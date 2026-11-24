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
)

func TestService_ReconcileReportsUnresolvableProvenance(t *testing.T) {
	tests := []struct {
		name      string
		libraryID string
		version   string
		library   *fakeLibrary
		want      string
	}{
		{
			name:      "unknown library component",
			libraryID: "rcl:retired", version: "1.0.0",
			library: &fakeLibrary{}, want: "library component is not indexed",
		},
		{
			name:      "unknown version",
			libraryID: "rcl:Button", version: "9.9.9",
			library: &fakeLibrary{byID: map[string]components.Component{
				"button": {ID: "button", LibraryID: "rcl:Button"},
			}}, want: "library version is not indexed",
		},
		{
			name:      "no matching file",
			libraryID: "rcl:Button", version: "1.0.0",
			library: &fakeLibrary{
				byID: map[string]components.Component{"button": {ID: "button", LibraryID: "rcl:Button"}},
				versions: map[string]components.ComponentVersion{"button@1.0.0": {
					ComponentID: "button", LibraryID: "rcl:Button", Version: "1.0.0",
					Files: []components.ComponentVersionFile{
						{Path: "Button.tsx", Content: "export const Button = 1;", IsEntry: true},
						{Path: "button-state.ts", Content: "export const state = 1;"},
					},
				}},
			}, want: "no library file matches ui/src/Missing.css",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			files := &fakeFiles{provenance: []adoptions.ProvenanceFile{{
				Scenario: "consumer", AdoptedPath: "ui/src/Missing.css",
				LibraryID: tc.libraryID, Version: tc.version, Content: []byte("copy"),
			}}}
			repo := adoptmocks.NewFakeRepository()
			svc := adoptions.NewService(repo, tc.library, files, scheduletest.New(time.Unix(0, 0)))

			result, err := svc.Reconcile(context.Background(), adoptions.ReconcileInput{Apply: true})

			require.NoError(t, err)
			require.Equal(t, 1, result.Scanned)
			require.Zero(t, result.Created)
			require.Len(t, result.Findings, 1)
			require.Equal(t, "consumer", result.Findings[0].Scenario)
			require.Equal(t, "ui/src/Missing.css", result.Findings[0].AdoptedPath)
			require.Equal(t, tc.libraryID, result.Findings[0].LibraryID)
			require.Equal(t, tc.version, result.Findings[0].Version)
			require.Equal(t, tc.want, result.Findings[0].Detail)
			rows, listErr := repo.List(context.Background(), adoptions.ListQuery{Limit: 10})
			require.NoError(t, listErr)
			require.Empty(t, rows)
		})
	}
}
