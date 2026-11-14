package uiruntime

import "testing"

func TestAbsoluteAssetURLResolvesBASArtifactReference(t *testing.T) {
	got := absoluteAssetURL("http://127.0.0.1:19893/api/v1/", "/api/v1/artifacts/screenshot.png")
	if got != "http://127.0.0.1:19893/api/v1/artifacts/screenshot.png" {
		t.Fatalf("absolute screenshot URL = %q", got)
	}
	if got := absoluteAssetURL("http://127.0.0.1:19893/", "http://bas.local/image.png"); got != "http://bas.local/image.png" {
		t.Fatalf("existing absolute URL changed: %q", got)
	}
}
