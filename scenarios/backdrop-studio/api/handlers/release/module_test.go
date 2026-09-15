package release

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"testing"

	"backdrop-studio/internal/catalog"
	internal "backdrop-studio/internal/release"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type candidateSource map[string]internal.CandidateEvidence

func (s candidateSource) CandidateEvidence(id string) (internal.CandidateEvidence, bool) {
	candidate, ok := s[id]
	return candidate, ok
}

type provenanceSource struct{}

func (provenanceSource) CandidateProvenance(string) (internal.Provenance, bool) {
	return internal.Provenance{Strategy: "procedural", ModelBacked: false}, true
}

func releaseTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.Black)
		}
	}
	var out bytes.Buffer
	require.NoError(t, png.Encode(&out, img))
	return out.Bytes()
}

func TestAssetEndpointReturnsQualifiedCandidateBytes(t *testing.T) {
	pngBytes := releaseTestPNG(t)
	store := internal.NewStoreWithPublisher(nil, provenanceSource{}, candidateSource{"candidate-1": {
		ID: "candidate-1", ImagePNG: pngBytes, Width: 8, Height: 8,
		StyleID: "style-1", Strategy: "procedural", SurfaceID: "web.hero", Placement: "full_bleed",
		Regions: []catalog.Region{{X: 0, Y: 0, Width: 1, Height: 1, Kind: "overlay", TextColor: "#FFFFFF"}}, ContrastThreshold: 4.5,
	}})
	released, err := store.Release(internal.Request{CandidateID: "candidate-1", StyleID: "style-1", Strategy: "procedural", SurfaceID: "web.hero", AltText: "A black ambient backdrop"})
	require.NoError(t, err)

	router := mux.NewRouter()
	Module(store).Mount(router)
	req := httptest.NewRequest("GET", "/api/v1/backdrops/"+released.ID+"/asset", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, 200, res.Code)
	require.Equal(t, pngBytes, res.Body.Bytes())
	require.Equal(t, "image/png", res.Header().Get("Content-Type"))
	require.Equal(t, released.ContentHash, res.Header().Get("X-Content-SHA256"))
	require.NotEmpty(t, res.Header().Get("ETag"))
}
