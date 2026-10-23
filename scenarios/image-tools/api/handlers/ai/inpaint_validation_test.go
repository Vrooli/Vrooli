package ai

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContextualInpaintRejectsInvalidControlsBeforeInputOrModelWork(t *testing.T) {
	for _, tc := range []struct{ name, operation, params string }{
		{"wrong-operation", "text_to_image", `{"inpaint":{"cropToMask":true,"maxEdgePx":1024}}`},
		{"missing-explicit-crop", "inpaint", `{"inpaint":{"contextPixels":64}}`},
		{"neutralize-requires-explicit-crop", "inpaint", `{"inpaint":{"neutralizeMaskedPixels":true}}`},
		{"excessive-context", "inpaint", `{"inpaint":{"cropToMask":true,"contextPixels":513}}`},
		{"unsupported-raster", "inpaint", `{"inpaint":{"cropToMask":true,"maxEdgePx":8192}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router, jobs := newTestServer(t, true)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, newMultipartSubmit(t, tc.operation, tc.params))
			if response.Code != http.StatusBadRequest || jobs.calls != 0 {
				t.Fatalf("invalid crop admitted: %d %s calls=%d", response.Code, response.Body, jobs.calls)
			}
		})
	}
}
