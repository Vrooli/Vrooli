package uxcapture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDownloadDeclaredArtifactUsesBASOwnerAndVerifiesManifest(t *testing.T) {
	content := []byte("verified-pixels")
	digest := sha256.Sum256(content)
	const id = "81dbd4b4-53ba-4e54-98a4-f98790da56da"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/v1/ux-capture/bundles/"+id+"/artifacts/pixels/light%20capture.png" {
			t.Errorf("artifact request path = %q", r.URL.EscapedPath())
		}
		_, _ = w.Write(content)
	}))
	defer server.Close()
	response, _ := json.Marshal(map[string]any{"manifest": map[string]any{
		"schema_version": "ux-capture.v1", "id": id,
		"privacy":   map[string]any{"redaction_status": "complete"},
		"artifacts": []any{map[string]any{"kind": "pixels", "path": "pixels/light capture.png", "size_bytes": len(content), "sha256": hex.EncodeToString(digest[:])}},
	}})
	destination := filepath.Join(t.TempDir(), "out", "capture.png")
	if err := downloadDeclaredArtifact(context.Background(), server.URL, response, "pixels", destination); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(destination)
	if err != nil || string(actual) != string(content) {
		t.Fatalf("artifact=%q err=%v", actual, err)
	}
}

func TestDownloadDeclaredArtifactRejectsDigestMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("wrong")) }))
	defer server.Close()
	response, _ := json.Marshal(map[string]any{"manifest": map[string]any{
		"schema_version": "ux-capture.v1", "id": "81dbd4b4-53ba-4e54-98a4-f98790da56da",
		"privacy":   map[string]any{"redaction_status": "complete"},
		"artifacts": []any{map[string]any{"kind": "pixels", "path": "pixels/capture.png", "size_bytes": 5, "sha256": strings.Repeat("0", 64)}},
	}})
	destination := filepath.Join(t.TempDir(), "capture.png")
	if err := downloadDeclaredArtifact(context.Background(), server.URL, response, "pixels", destination); err == nil {
		t.Fatal("accepted a mismatched artifact digest")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("invalid artifact was written: %v", err)
	}
}

func TestValidateWaitSelectorIsScenarioOnlyAndBounded(t *testing.T) {
	if got, err := validateWaitSelector("web-console", ` [data-testid="pane-grid"] `); err != nil || got != `[data-testid="pane-grid"]` {
		t.Fatalf("validate scenario selector = %q, %v", got, err)
	}
	if _, err := validateWaitSelector("", `[data-testid="pane-grid"]`); err == nil {
		t.Fatal("expected a public capture to reject a scenario wait selector")
	}
	if _, err := validateWaitSelector("web-console", strings.Repeat("x", 513)); err == nil {
		t.Fatal("expected an overlong selector to be rejected")
	}
	if got, err := validateWaitSelector("web-console", "   "); err != nil || got != "" {
		t.Fatalf("blank selector = %q, %v", got, err)
	}
}

func TestBuildScenarioRequestCarriesWaitSelector(t *testing.T) {
	got := buildScenarioRequest("web-console", "/", nil, 1280, 720, "light", `[data-testid="pane-grid"]`, false, "", "")
	want := map[string]any{
		"scenario":      "web-console",
		"route":         "/",
		"steps":         []string(nil),
		"width":         1280,
		"height":        720,
		"theme":         "light",
		"wait_selector": `[data-testid="pane-grid"]`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scenario request = %#v, want %#v", got, want)
	}
}

func TestValidateRegionSelectorRequiresComponentTestScenarioAndIsBounded(t *testing.T) {
	if got, err := validateRegionSelector("react-component-library", "componenttests", `[data-preview-capture-boundary]`); err != nil || got != `[data-preview-capture-boundary]` {
		t.Fatalf("validate region selector = %q, %v", got, err)
	}
	if _, err := validateRegionSelector("react-component-library", "", `[data-preview-capture-boundary]`); err == nil {
		t.Fatal("expected a profile error")
	}
	if _, err := validateRegionSelector("", "componenttests", `[data-preview-capture-boundary]`); err == nil {
		t.Fatal("expected a scenario-only error")
	}
	if _, err := validateRegionSelector("react-component-library", "componenttests", strings.Repeat("x", 257)); err == nil {
		t.Fatal("expected an overlong selector to be rejected")
	}
}

func TestBuildScenarioRequestCarriesComponentEvidenceProfile(t *testing.T) {
	got := buildScenarioRequest("react-component-library", "/preview/button/harness.html?version=1.2.3", nil, 1280, 900, "light", `[data-preview-capture-boundary]`, false, "componenttests", `[data-preview-capture-boundary]`)
	if got["evidence_profile"] != "componenttests" || got["region_selector"] != `[data-preview-capture-boundary]` {
		t.Fatalf("component evidence request = %#v", got)
	}
}
