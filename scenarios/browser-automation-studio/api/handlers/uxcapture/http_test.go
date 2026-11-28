package uxcapture

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	bundle "github.com/vrooli/browser-automation-studio/services/uxcapture"
)

type publicCaptureStub struct{ raw json.RawMessage }

func (s publicCaptureStub) CapturePublicPage(context.Context, *driver.PublicCaptureRequest) (json.RawMessage, error) {
	return s.raw, nil
}
func (s publicCaptureStub) CaptureHTMLFile(context.Context, *driver.FileCaptureRequest) (json.RawMessage, error) {
	return s.raw, nil
}

func TestMountHTTPPersistsPrivateBundle(t *testing.T) {
	raw := `{"schema_version":"ux-capture.v1","id":"11111111-1111-4111-8111-111111111111","captured_at":"2026-11-14T05:00:00Z","source":{"kind":"url","url":"https://example.org/"},"session":{"id":"session-test","browser":"chromium"},"viewport":{"width":1280,"height":720},"lenses":{"L0":{"status":"present"},"L1":{"status":"absent","reason":"fixture"},"L2":{"status":"absent","reason":"fixture"},"L3":{"status":"absent","reason":"fixture"},"L4":{"status":"present"},"L5":{"status":"absent","reason":"fixture"},"L6":{"status":"absent","reason":"fixture"}},"privacy":{"redaction_status":"complete","redacted_classes":[],"external":true,"ephemeral":true},"artifacts":{"screenshot_png_base64":"aGVsbG8=","component_identity":{"schema":"bas-component-identity/v1","element_count":1,"stamped_count":1,"components":[{"selector":"#action","asset":"controls.button","version":"2.1.0","stamp":"vite"}]}}}`
	root := t.TempDir()
	router := chi.NewRouter()
	MountHTTP(router, publicCaptureStub{raw: json.RawMessage(raw)}, root)
	req := httptest.NewRequest("POST", "/ux-capture/public", strings.NewReader(`{"url":"https://example.org/"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != 201 {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "bundle_dir") {
		t.Fatal("BAS response leaked its private bundle directory")
	}
	var result struct {
		Manifest struct {
			ID string `json:"id"`
		} `json:"manifest"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"manifest.json", "README.md", "pixels/light-1280x720.png", "identity/components.json"} {
		info, err := os.Stat(filepath.Join(root, "ux", result.Manifest.ID, path))
		if err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("%s permissions = %o, want 600", path, info.Mode().Perm())
		}
	}
}

func TestFileCapturePersistsSanitizedFilenameAndPayload(t *testing.T) {
	raw := `{"schema_version":"ux-capture.v1","id":"22222222-2222-4222-8222-222222222222","captured_at":"2026-11-14T05:00:00Z","source":{"kind":"file","filename":"mockup.html"},"session":{"id":"session-file","browser":"chromium"},"viewport":{"width":1280,"height":720},"lenses":{"L0":{"status":"present"},"L1":{"status":"present"},"L2":{"status":"absent","reason":"fixture"},"L3":{"status":"absent","reason":"fixture"},"L4":{"status":"absent","reason":"fixture"},"L5":{"status":"absent","reason":"network disabled"},"L6":{"status":"absent","reason":"fixture"}},"privacy":{"redaction_status":"complete"},"artifacts":{"screenshot_png_base64":"cG5n","frozen_page_html":"<html>safe</html>"}}`
	router := chi.NewRouter()
	root := t.TempDir()
	MountHTTP(router, publicCaptureStub{raw: json.RawMessage(raw)}, root)
	req := httptest.NewRequest("POST", "/ux-capture/file", strings.NewReader(`{"filename":"/private/user/mockup.html","html":"<html>secret</html>"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != 201 {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "bundle_dir") {
		t.Fatal("BAS response leaked its private bundle directory")
	}
	var result struct {
		Manifest struct {
			ID     string `json:"id"`
			Source struct {
				Filename string `json:"filename"`
			} `json:"source"`
		} `json:"manifest"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Manifest.Source.Filename != "mockup.html" {
		t.Fatalf("filename leaked path or mismatched: %q", result.Manifest.Source.Filename)
	}
	for _, file := range []string{"manifest.json", "README.md", "pixels/light-1280x720.png", "page/frozen.html"} {
		if _, err := os.Stat(filepath.Join(root, "ux", result.Manifest.ID, file)); err != nil {
			t.Errorf("missing bundle file %s: %v", file, err)
		}
	}
}

func TestFileCaptureRejectsOversizeBeforeDriver(t *testing.T) {
	router := chi.NewRouter()
	MountHTTP(router, publicCaptureStub{}, t.TempDir())
	body := `{"filename":"x.html","html":"` + strings.Repeat("x", 2*1024*1024+1) + `"}`
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("POST", "/ux-capture/file", strings.NewReader(body)))
	if response.Code != 413 {
		t.Fatalf("status = %d, want 413", response.Code)
	}
}

func TestMountHTTPRejectsCredentialedURLAndUnknownFields(t *testing.T) {
	router := chi.NewRouter()
	MountHTTP(router, publicCaptureStub{}, t.TempDir())
	for _, body := range []string{`{"url":"https://user:secret@example.org"}`, `{"url":"https://example.org","auth":"secret"}`} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("POST", "/ux-capture/public", strings.NewReader(body)))
		if response.Code != 400 {
			t.Errorf("body %s: status = %d, want 400", body, response.Code)
		}
	}
}

func TestDecodeCapturePersistsStrictTokenDiagnostic(t *testing.T) {
	raw := `{"schema_version":"ux-capture.v1","id":"33333333-3333-4333-8333-333333333333","captured_at":"2026-11-14T05:00:00Z","source":{"kind":"scenario","scenario":"git-control-tower","route":"/"},"session":{"id":"session-strict","browser":"chromium"},"viewport":{"width":1280,"height":720},"lenses":{"L0":{"status":"absent","reason":"fixture"},"L1":{"status":"absent","reason":"fixture"},"L2":{"status":"absent","reason":"fixture"},"L3":{"status":"absent","reason":"fixture"},"L4":{"status":"absent","reason":"fixture"},"L5":{"status":"absent","reason":"fixture"},"L6":{"status":"absent","reason":"fixture"}},"privacy":{"redaction_status":"complete"},"artifacts":{"strict_tokens":{"schema":"bas-strict-token-diagnostic/v1","token_count":12,"sentinel_element_count":4,"non_sentinel_color_count":3}}}`
	manifest, payloads, err := DecodeCapture([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Artifacts) != 1 || manifest.Artifacts[0].Path != "diagnostics/strict-tokens.json" {
		t.Fatalf("strict diagnostic artifacts = %+v", manifest.Artifacts)
	}
	if len(payloads) != 1 || !json.Valid(payloads[0].Content) || !strings.Contains(string(payloads[0].Content), `"token_count":12`) {
		t.Fatalf("strict diagnostic payload = %+v", payloads)
	}
}

func TestDecodeCaptureDeclaresBoundedComponentEvidenceWithHashes(t *testing.T) {
	raw := `{"schema_version":"ux-capture.v1","id":"44444444-4444-4444-8444-444444444444","captured_at":"2026-11-14T05:00:00Z","source":{"kind":"scenario","scenario":"git-control-tower","route":"/"},"session":{"id":"session-evidence","browser":"chromium"},"viewport":{"width":1280,"height":720},"lenses":{"L0":{"status":"present"},"L1":{"status":"absent","reason":"fixture"},"L2":{"status":"absent","reason":"fixture"},"L3":{"status":"absent","reason":"fixture"},"L4":{"status":"absent","reason":"fixture"},"L5":{"status":"absent","reason":"fixture"},"L6":{"status":"absent","reason":"fixture"}},"privacy":{"redaction_status":"complete"},"artifacts":{"screenshot_png_base64":"cG5n","region_png_base64":"cmVnaW9u","region_style_sample":{"schema":"bas-region-style-sample/v1","computed_style":{"color":"rgb(0, 0, 0)","background_color":"rgb(255, 255, 255)","border_color":"rgb(0, 0, 0)","border_style":"solid","border_width":"1px","visibility":"visible","display":"block","opacity":"1"}},"accessibility_snapshot":{"schema":"bas-accessibility-snapshot/v1","snapshot":"main"},"console_summary":{"schema":"bas-console-summary/v1","error_count":1,"warning_count":0},"performance_metrics":{"schema":"bas-performance-metrics/v1","dom_content_loaded_ms":42,"load_ms":50,"resource_count":3}}}`
	manifest, payloads, err := DecodeCapture([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"pixels/region.png": 1, "style/region-sample.json": 1, "accessibility/snapshot.json": 1, "console/summary.json": 1, "performance/metrics.json": 1}
	for _, artifact := range manifest.Artifacts {
		if _, ok := want[artifact.Path]; ok {
			if len(artifact.SHA256) != 64 || artifact.SizeBytes <= 0 {
				t.Errorf("incomplete declaration: %+v", artifact)
			}
			delete(want, artifact.Path)
		}
	}
	if len(want) != 0 || len(payloads) != 6 {
		t.Fatalf("missing evidence artifacts %v; payloads=%d", want, len(payloads))
	}
}

func TestDecodeCaptureDeclaresFullPagePixelArtifact(t *testing.T) {
	raw := `{"schema_version":"ux-capture.v1","id":"55555555-5555-4555-8555-555555555555","captured_at":"2026-11-14T05:00:00Z","source":{"kind":"scenario","scenario":"git-control-tower","route":"/"},"session":{"id":"session-full-page","browser":"chromium"},"viewport":{"width":1440,"height":900},"lenses":{"L0":{"status":"present"},"L1":{"status":"absent","reason":"fixture"},"L2":{"status":"absent","reason":"fixture"},"L3":{"status":"absent","reason":"fixture"},"L4":{"status":"absent","reason":"fixture"},"L5":{"status":"absent","reason":"fixture"},"L6":{"status":"absent","reason":"fixture"}},"privacy":{"redaction_status":"complete"},"artifacts":{"screenshot_png_base64":"cG5n","full_page_pixels_png_base64":"ZnVsbA=="}}`
	manifest, payloads, err := DecodeCapture([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(payloads) != 2 {
		t.Fatalf("payload count = %d, want viewport and full-page pixels", len(payloads))
	}
	var fullPage *bundle.Artifact
	for index := range manifest.Artifacts {
		artifact := &manifest.Artifacts[index]
		if artifact.Kind == "full-page-pixels" {
			fullPage = artifact
		}
	}
	if fullPage == nil || fullPage.Path != "pixels/full-page-light-1440x900.png" || fullPage.SizeBytes != 4 || len(fullPage.SHA256) != 64 {
		t.Fatalf("full-page declaration = %+v", fullPage)
	}
}
