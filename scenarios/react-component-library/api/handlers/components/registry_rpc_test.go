package components_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	authoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componentauthoring/componentauthoring_v1connect"
	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"
)

// These checks exercise the mounted registry RPCs at the transport boundary:
// request fields must reach the service and domain results/errors must come
// back in the declared Connect response shape.
func TestRegistryConnectReadsComponentAndVersionContent(t *testing.T) {
	r, _ := setupModule(t)
	initialized := callConnect(r, authoringconnect.ComponentAuthoringServiceInitializeComponentProcedure, `{
		"slug":"rpc-card",
		"libraryId":"react-component-library:rpc-card",
		"displayName":"RPC Card",
		"initialVersion":"1.0.0",
		"initialSource":"export function Card() { return <article>Card body</article>; }"
	}`)
	require.Equal(t, http.StatusOK, initialized.Code, initialized.Body.String())

	byLibraryID := callConnect(r, componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		`{"libraryId":"react-component-library:rpc-card"}`)
	require.Equal(t, http.StatusOK, byLibraryID.Code, byLibraryID.Body.String())
	require.Contains(t, byLibraryID.Body.String(), `"displayName":"RPC Card"`)
	require.Contains(t, byLibraryID.Body.String(), `"version":"1.0.0"`)

	versions := callConnect(r, componentsconnect.ComponentsServiceListComponentVersionsProcedure,
		`{"componentId":"react-component-library:rpc-card","limit":5}`)
	require.Equal(t, http.StatusOK, versions.Code, versions.Body.String())
	require.Contains(t, versions.Body.String(), `"version":"1.0.0"`)
	require.Contains(t, versions.Body.String(), `"status":"COMPONENT_VERSION_STATUS_RELEASED"`)

	content := callConnect(r, componentsconnect.ComponentsServiceGetComponentVersionContentProcedure,
		`{"componentId":"react-component-library:rpc-card","version":"1.0.0","path":"rpc-card.tsx"}`)
	require.Equal(t, http.StatusOK, content.Code, content.Body.String())
	require.Contains(t, content.Body.String(), `Card body`)
	require.Contains(t, content.Body.String(), `"contentSha256"`)

	styles := callConnect(r, componentsconnect.ComponentsServiceListDesignStylesProcedure, `{}`)
	require.Equal(t, http.StatusOK, styles.Code, styles.Body.String())
	require.Contains(t, styles.Body.String(), `"styles"`)

	stories := callConnect(r, componentsconnect.ComponentsServiceListComponentStoriesProcedure,
		`{"componentId":"react-component-library:rpc-card","version":"1.0.0","limit":10}`)
	require.Equal(t, http.StatusOK, stories.Code, stories.Body.String())
	require.Contains(t, stories.Body.String(), `"stories"`)
}

func TestRegistryConnectMapsMissingAssetsToNotFound(t *testing.T) {
	r, _ := setupModule(t)
	initialized := callConnect(r, authoringconnect.ComponentAuthoringServiceInitializeComponentProcedure, `{
		"slug":"rpc-card",
		"libraryId":"react-component-library:rpc-card",
		"displayName":"RPC Card",
		"initialVersion":"1.0.0"
	}`)
	require.Equal(t, http.StatusOK, initialized.Code, initialized.Body.String())

	cases := []struct {
		name, procedure, body string
	}{
		{
			name:      "library id lookup",
			procedure: componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
			body:      `{"libraryId":"react-component-library:missing"}`,
		},
		{
			name:      "version content lookup",
			procedure: componentsconnect.ComponentsServiceGetComponentVersionContentProcedure,
			body:      `{"componentId":"missing","version":"1.0.0"}`,
		},
		{
			name:      "missing version file",
			procedure: componentsconnect.ComponentsServiceGetComponentVersionContentProcedure,
			body:      `{"componentId":"react-component-library:rpc-card","version":"1.0.0","path":"missing.tsx"}`,
		},
		{
			name:      "style fit lookup",
			procedure: componentsconnect.ComponentsServiceValidateStyleFitProcedure,
			body:      `{"componentId":"missing","version":"1.0.0","scenario":"web-console"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rw := callConnect(r, tc.procedure, tc.body)
			require.Equal(t, http.StatusNotFound, rw.Code, rw.Body.String())
			require.Contains(t, rw.Body.String(), `"code":"not_found"`)
		})
	}
}
