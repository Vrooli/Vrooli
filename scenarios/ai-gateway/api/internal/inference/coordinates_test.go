package inference

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestNormalizeLocateVisualJSONConvergesAcrossProviderConventions(t *testing.T) {
	attachment := visualAttachment(1000, 500)
	want := []float64{0.1, 0.2, 0.4, 0.8}

	for _, test := range []struct {
		name       string
		convention string
		bounds     []float64
	}{
		{name: "normalized 0 to 1000", convention: CoordinateConventionNormalized1000, bounds: []float64{100, 200, 400, 800}},
		{name: "absolute pixels", convention: CoordinateConventionAbsolutePixels, bounds: []float64{100, 100, 400, 400}},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{"found": true, "bounds": test.bounds, "confidence": 0.73})
			require.NoError(t, err)
			if test.convention == CoordinateConventionAbsolutePixels {
				raw = []byte(`{"found":true,"bounds":[100,100,400,400],"confidence":0.73}`)
			}
			gotRaw, err := NormalizeLocateVisualJSON(string(raw), test.convention, []*sharedv1.Attachment{attachment})
			require.NoError(t, err)
			var got visualResult
			require.NoError(t, json.Unmarshal([]byte(gotRaw), &got))
			require.Equal(t, want, got.Bounds)
			require.Equal(t, 0.73, got.Confidence)
		})
	}
}

func TestNormalizeLocateVisualJSONRequiresDeclaredConventionAndImageBounds(t *testing.T) {
	attachment := visualAttachment(1000, 500)
	raw := `{"found":true,"bounds":[100,100,400,400],"confidence":0.5}`
	_, err := NormalizeLocateVisualJSON(raw, "", []*sharedv1.Attachment{attachment})
	require.ErrorContains(t, err, "unsupported coordinate convention")

	_, err = NormalizeLocateVisualJSON(`{"found":true,"bounds":[-1,100,400,400],"confidence":0.5}`, CoordinateConventionNormalized1000, []*sharedv1.Attachment{attachment})
	require.ErrorContains(t, err, "outside the submitted image")

	_, err = NormalizeLocateVisualJSON(raw, CoordinateConventionAbsolutePixels, nil)
	require.ErrorContains(t, err, "requires an image attachment")
}

func TestServiceNormalizesLocateVisualBeforeCanonicalSchemaValidation(t *testing.T) {
	repository := fakeRepository{result: ProviderResult{
		ValueJSON:            `{"found":true,"bounds":[100,200,400,800],"confidence":0.61}`,
		Provider:             "ollama",
		Model:                "qwen3-vl:4b",
		CoordinateConvention: CoordinateConventionNormalized1000,
	}}
	service := NewService(repository)
	response := service.Run(t.Context(), ProviderRequest{
		SchemaJSON:  `{"type":"object","required":["found","bounds","confidence"],"properties":{"found":{"type":"boolean"},"bounds":{"type":"array","items":{"type":"number"}},"confidence":{"type":"number","minimum":0,"maximum":1}}}`,
		Instruction: "Find the target.",
		Role:        "locate.visual",
		Attachments: []*sharedv1.Attachment{visualAttachment(1000, 500)},
	})
	require.True(t, response.GetValidated(), response.GetError())
	require.Contains(t, response.GetValueJson(), `0.1`)
	require.Contains(t, response.GetValueJson(), `0.2`)
	require.Equal(t, "qwen3-vl:4b", response.GetModel())
}

func TestCatalogDeclaresLocateVisualRole(t *testing.T) {
	catalog, err := LoadCatalog("../../../config/inference-role-catalog.json")
	require.NoError(t, err)
	role, ok := catalog.Roles["locate.visual"]
	require.True(t, ok)
	require.Len(t, role.Candidates, 3)
	require.Equal(t, "vision.default", role.Candidates[0].ResourceRole)
	require.Equal(t, "vision.default", role.Candidates[1].ResourceRole)
}

func visualAttachment(width, height uint32) *sharedv1.Attachment {
	return &sharedv1.Attachment{
		Modality:  sharedv1.Modality_MODALITY_IMAGE,
		MediaType: "image/png",
		Width:     width,
		Height:    height,
		Bytes:     4,
		Payload:   &sharedv1.Attachment_InlineBytes{InlineBytes: []byte("png")},
	}
}
