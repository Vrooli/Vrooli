package support

import (
	"encoding/json"
	"net/url"
	"testing"
)

// [REQ:REQ-P0-007] CLI support helpers unwrap API envelopes and preserve direct JSON shapes.
func TestDecodeEnvelopeAndRawBody(t *testing.T) {
	var data FoliageData
	body := []byte(`{"status":"success","data":{"region_id":3,"peak_status":"near_peak"}}`)
	if err := Decode(body, &data); err != nil {
		t.Fatalf("Decode envelope error: %v", err)
	}
	if data.RegionID != 3 || data.PeakStatus != "near_peak" {
		t.Fatalf("decoded envelope = %#v", data)
	}

	var raw map[string]string
	if err := Decode([]byte(`{"name":"White Mountains"}`), &raw); err != nil {
		t.Fatalf("Decode raw body error: %v", err)
	}
	if raw["name"] != "White Mountains" {
		t.Fatalf("decoded raw body = %#v", raw)
	}
}

func TestBuildQueryTrimsAndDropsEmptyValues(t *testing.T) {
	got := BuildQuery(map[string]string{
		"region_id": " 8 ",
		"date":      "",
	})
	want := url.Values{"region_id": []string{"8"}}
	if got.Encode() != want.Encode() {
		t.Fatalf("BuildQuery() = %q, want %q", got.Encode(), want.Encode())
	}
}

func TestRenderValue(t *testing.T) {
	values := map[interface{}]string{
		nil:        "null",
		"peak":     "peak",
		true:       "true",
		float64(7): "7",
	}
	for input, want := range values {
		if got := RenderValue(input); got != want {
			t.Fatalf("RenderValue(%#v) = %q, want %q", input, got, want)
		}
	}

	raw := json.RawMessage(`{"nested":true}`)
	if got := RenderValue(raw); got != `{"nested":true}` {
		t.Fatalf("RenderValue(raw JSON) = %q", got)
	}
}
