package components_test

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	authoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componentauthoring/componentauthoring_v1connect"
)

// These requests exercise the mounted authoring Connect service's error paths,
// including the conversion from domain errors to Connect HTTP responses.
func TestAuthoringConnectRejectsInvalidInitialization(t *testing.T) {
	r, root := setupModule(t)

	rw := callConnect(r, authoringconnect.ComponentAuthoringServiceInitializeComponentProcedure, `{
		"slug":"invalid",
		"libraryId":"react-component-library:invalid",
		"displayName":"Invalid",
		"initialVersion":"not-semver"
	}`)

	require.Equal(t, http.StatusBadRequest, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"code":"invalid_argument"`)
	require.NoDirExists(t, filepath.Join(root, "components", "invalid"))
}

func TestAuthoringConnectMapsMissingContentToNotFound(t *testing.T) {
	r, _ := setupModule(t)

	rw := callConnect(r, authoringconnect.ComponentAuthoringServiceGetComponentContentProcedure,
		`{"id":"missing-component"}`)

	require.Equal(t, http.StatusNotFound, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"code":"not_found"`)
}

func TestAuthoringConnectMapsMissingVersionComponentToNotFound(t *testing.T) {
	r, _ := setupModule(t)

	rw := callConnect(r, authoringconnect.ComponentAuthoringServiceCreateComponentVersionProcedure, `{
		"componentId":"missing-component",
		"version":"1.0.0-draft.1",
		"fromVersion":"1.0.0",
		"intent":"COMPONENT_VERSION_INTENT_DRAFT",
		"source":"export function Missing() { return null; }"
	}`)

	require.Equal(t, http.StatusNotFound, rw.Code, rw.Body.String())
	require.Contains(t, rw.Body.String(), `"code":"not_found"`)
}
