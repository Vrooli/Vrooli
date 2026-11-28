package contracts

import (
	"encoding/json"
	"testing"
)

func TestVariantSEOConfigPreservesPublishedJSONContract(t *testing.T) {
	config := VariantSEOConfig{Title: "Launch", OGImageURL: "https://cdn.example.test/og.png", NoIndex: true, StructuredData: map[string]any{"@type": "WebSite"}}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["og_image_url"] != config.OGImageURL || decoded["noindex"] != true || decoded["structured_data"].(map[string]any)["@type"] != "WebSite" {
		t.Fatalf("decoded=%#v", decoded)
	}
	if _, found := decoded["description"]; found {
		t.Fatalf("empty optional description was serialized: %#v", decoded)
	}
}
