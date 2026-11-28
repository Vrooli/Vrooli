package serviceofferprototype

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestProposalPreviewExposesOneOfferAndItsDerivedSurfaces(t *testing.T) {
	module := Module()
	router := mux.NewRouter()
	module.Mount(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, previewPath, nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))

	var body previewResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	require.Equal(t, "company-market-scan", body.Offer.ID)
	require.Equal(t, body.Offer.ID, body.Surfaces.Page.Slug)
	require.Equal(t, body.Offer.ID, body.Surfaces.Checkout.OfferID)
	require.Equal(t, body.Offer.ID, body.Surfaces.MPP.OfferID)
	require.Equal(t, body.Offer.MCPToolName, body.Surfaces.MCP.Name)
	require.Equal(t, "pending", string(body.Offer.OperatorApproval))
	require.Equal(t, "open", body.Offer.Legal.Status)
	require.NotEmpty(t, body.Sample)
}

func TestProposalSampleEndpointServesTheSamePublicArtifact(t *testing.T) {
	module := Module()
	router := mux.NewRouter()
	module.Mount(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, previewPath+"/sample", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "Apple FY2026 Form 10-K")
	require.Contains(t, response.Body.String(), "$478.8 billion")
	require.Contains(t, response.Body.String(), "Not checked:")
	require.Contains(t, response.Body.String(), "Securities and Exchange Commission")
}
