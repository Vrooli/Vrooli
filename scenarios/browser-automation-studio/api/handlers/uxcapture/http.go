package uxcapture

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	bundle "github.com/vrooli/browser-automation-studio/services/uxcapture"
)

type PublicCaptureClient interface {
	CapturePublicPage(ctx context.Context, request *driver.PublicCaptureRequest) (json.RawMessage, error)
	CaptureHTMLFile(ctx context.Context, request *driver.FileCaptureRequest) (json.RawMessage, error)
}

// MountHTTP exposes the first EX9 command surface behind BAS. The public URL
// browser and all network policy remain owned by the Playwright driver.
func MountHTTP(r chi.Router, client PublicCaptureClient, capturesRoot string) {
	MountArtifactHTTP(r, capturesRoot)
	r.Post("/ux-capture/public", func(w http.ResponseWriter, req *http.Request) {
		if client == nil {
			writeError(w, http.StatusServiceUnavailable, "BAS public capture driver is unavailable")
			return
		}
		var input driver.PublicCaptureRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, 16*1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "Capture request must contain a public URL and optional viewport dimensions.")
			return
		}
		if strings.TrimSpace(input.URL) == "" {
			writeError(w, http.StatusBadRequest, "url is required")
			return
		}
		parsed, parseErr := url.ParseRequestURI(input.URL)
		if parseErr != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
			writeError(w, http.StatusBadRequest, "url must be an absolute HTTP(S) URL without embedded credentials")
			return
		}
		if (input.Width != 0 && (input.Width < 320 || input.Width > 2560)) || (input.Height != 0 && (input.Height < 240 || input.Height > 1600)) {
			writeError(w, http.StatusBadRequest, "viewport dimensions are outside the supported range")
			return
		}
		if !validCaptureTheme(input.Theme) {
			writeError(w, http.StatusBadRequest, "theme must be light, dark, or no-preference")
			return
		}
		if err := ensureRequestConsumed(decoder); err != nil {
			writeError(w, http.StatusBadRequest, "Capture request must be one JSON object.")
			return
		}
		raw, err := client.CapturePublicPage(req.Context(), &input)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Public page capture failed: "+err.Error())
			return
		}
		manifest, payloads, err := DecodeCapture(raw)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Driver returned an invalid UX capture bundle: "+err.Error())
			return
		}
		if strings.TrimSpace(capturesRoot) == "" {
			writeError(w, http.StatusInternalServerError, "BAS private capture storage is unavailable")
			return
		}
		bundleDir, err := bundleDirectory(capturesRoot, manifest.ID)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Driver returned an invalid UX capture bundle ID")
			return
		}
		if err := bundle.WriteBundleWithArtifacts(bundleDir, manifest, payloads); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not persist the private capture bundle: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(struct {
			Manifest bundle.Bundle `json:"manifest"`
		}{Manifest: manifest})
	})
	r.Post("/ux-capture/file", func(w http.ResponseWriter, req *http.Request) {
		if client == nil {
			writeError(w, http.StatusServiceUnavailable, "BAS capture driver is unavailable")
			return
		}
		var input driver.FileCaptureRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, 4*1024*1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "HTML file must be UTF-8 JSON with a bounded html field.")
			return
		}
		if len(input.HTML) == 0 || len(input.HTML) > 2*1024*1024 {
			writeError(w, http.StatusRequestEntityTooLarge, "HTML file must be between 1 byte and 2 MiB.")
			return
		}
		input.Filename = filepath.Base(strings.ReplaceAll(input.Filename, "\\", "/"))
		if input.Filename == "." || input.Filename == "/" || strings.TrimSpace(input.Filename) == "" {
			writeError(w, http.StatusBadRequest, "filename is required")
			return
		}
		if strings.Contains(input.Filename, "\x00") {
			writeError(w, http.StatusBadRequest, "filename is invalid")
			return
		}
		if (input.Width != 0 && (input.Width < 320 || input.Width > 2560)) || (input.Height != 0 && (input.Height < 240 || input.Height > 1600)) {
			writeError(w, http.StatusBadRequest, "viewport dimensions are outside the supported range")
			return
		}
		if !validCaptureTheme(input.Theme) {
			writeError(w, http.StatusBadRequest, "theme must be light, dark, or no-preference")
			return
		}
		if err := ensureRequestConsumed(decoder); err != nil {
			writeError(w, http.StatusBadRequest, "Capture request must be one JSON object.")
			return
		}
		raw, err := client.CaptureHTMLFile(req.Context(), &input)
		if err != nil {
			writeError(w, http.StatusBadGateway, "HTML file capture failed: "+err.Error())
			return
		}
		manifest, payloads, err := DecodeCapture(raw)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Driver returned an invalid UX capture bundle: "+err.Error())
			return
		}
		if manifest.Source.Kind != bundle.SourceFile || manifest.Source.Filename != input.Filename {
			writeError(w, http.StatusBadGateway, "Driver returned mismatched file source metadata")
			return
		}
		if strings.TrimSpace(capturesRoot) == "" {
			writeError(w, http.StatusInternalServerError, "BAS private capture storage is unavailable")
			return
		}
		bundleDir, err := bundleDirectory(capturesRoot, manifest.ID)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Driver returned an invalid UX capture bundle ID")
			return
		}
		if err := bundle.WriteBundleWithArtifacts(bundleDir, manifest, payloads); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not persist the private capture bundle: "+err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(struct {
			Manifest bundle.Bundle `json:"manifest"`
		}{manifest})
	})
}

type driverCaptureResult struct {
	SchemaVersion string `json:"schema_version"`
	ID            string `json:"id"`
	CapturedAt    string `json:"captured_at"`
	Source        struct {
		Kind     string `json:"kind"`
		URL      string `json:"url"`
		Filename string `json:"filename"`
		Scenario string `json:"scenario"`
		Route    string `json:"route"`
	} `json:"source"`
	Session struct {
		ID      string `json:"id"`
		Browser string `json:"browser"`
	} `json:"session"`
	Viewport struct {
		Width  int32 `json:"width"`
		Height int32 `json:"height"`
	} `json:"viewport"`
	Theme   string                          `json:"theme"`
	Lenses  map[bundle.LensName]bundle.Lens `json:"lenses"`
	Privacy struct {
		RedactionStatus string   `json:"redaction_status"`
		RedactionReason string   `json:"redaction_reason"`
		RedactedClasses []string `json:"redacted_classes"`
		External        bool     `json:"external"`
		Ephemeral       bool     `json:"ephemeral"`
	} `json:"privacy"`
	Artifacts struct {
		ScreenshotPNGBase64     string          `json:"screenshot_png_base64"`
		FullPagePixelsPNGBase64 string          `json:"full_page_pixels_png_base64"`
		FullPagePixelsStatus    json.RawMessage `json:"full_page_pixels_status"`
		FrozenPageHTML          string          `json:"frozen_page_html"`
		Outline                 json.RawMessage `json:"outline"`
		StyleCensus             json.RawMessage `json:"style_census"`
		Network                 json.RawMessage `json:"network"`
		ComponentIdentity       json.RawMessage `json:"component_identity"`
		StrictTokens            json.RawMessage `json:"strict_tokens"`
		RegionPNGBase64         string          `json:"region_png_base64"`
		AccessibilitySnapshot   json.RawMessage `json:"accessibility_snapshot"`
		ConsoleSummary          json.RawMessage `json:"console_summary"`
		PerformanceMetrics      json.RawMessage `json:"performance_metrics"`
		RegionStyleSample       json.RawMessage `json:"region_style_sample"`
	} `json:"artifacts"`
}

func DecodePublicCapture(raw []byte) (bundle.Bundle, []bundle.ArtifactPayload, error) {
	return DecodeCapture(raw)
}

func DecodeCapture(raw []byte) (bundle.Bundle, []bundle.ArtifactPayload, error) {
	var result driverCaptureResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return bundle.Bundle{}, nil, fmt.Errorf("decode driver result: %w", err)
	}
	if result.SchemaVersion != bundle.SchemaVersion {
		return bundle.Bundle{}, nil, fmt.Errorf("unexpected schema version %q", result.SchemaVersion)
	}
	if (result.Source.Kind != "url" && result.Source.Kind != "file" && result.Source.Kind != "scenario") || result.Session.ID == "" {
		return bundle.Bundle{}, nil, errors.New("driver result is missing its source or session identity")
	}
	theme := result.Theme
	if theme == "" {
		theme = "light"
	}
	if theme != "light" && theme != "dark" && theme != "no-preference" {
		return bundle.Bundle{}, nil, fmt.Errorf("driver returned unsupported theme %q", theme)
	}
	source := bundle.Source{Kind: bundle.SourceKind(result.Source.Kind), URL: result.Source.URL, Filename: result.Source.Filename, Scenario: result.Source.Scenario, Route: result.Source.Route}
	manifest := bundle.Bundle{
		SchemaVersion: result.SchemaVersion,
		ID:            result.ID,
		CapturedAt:    result.CapturedAt,
		Source:        source,
		Session:       bundle.Session{ID: result.Session.ID, Browser: result.Session.Browser},
		Viewport:      bundle.Viewport{Width: result.Viewport.Width, Height: result.Viewport.Height},
		Theme:         theme,
		Lenses:        result.Lenses,
		Privacy: bundle.Privacy{
			RedactionStatus: result.Privacy.RedactionStatus,
			RedactionReason: result.Privacy.RedactionReason,
			RedactedClasses: result.Privacy.RedactedClasses,
			External:        result.Privacy.External,
			Ephemeral:       result.Privacy.Ephemeral,
		},
		NextCommands: []string{
			"Inspect manifest.json and README.md for available lenses and capture limits.",
			"Use the L2 outline to select a page region before proposing a revision.",
		},
	}
	payloads := make([]bundle.ArtifactPayload, 0, 10)
	add := func(kind, path, mediaType string, content []byte) {
		manifest.Artifacts = append(manifest.Artifacts, bundle.Artifact{
			Kind: kind, Path: path, MediaType: mediaType, SizeBytes: int64(len(content)), SHA256: bundle.Digest(content),
		})
		payloads = append(payloads, bundle.ArtifactPayload{Path: path, Content: content})
	}
	if result.Artifacts.ScreenshotPNGBase64 != "" {
		content, err := base64.StdEncoding.Strict().DecodeString(result.Artifacts.ScreenshotPNGBase64)
		if err != nil {
			return bundle.Bundle{}, nil, errors.New("screenshot artifact is not valid base64")
		}
		if len(content) > 20*1024*1024 {
			return bundle.Bundle{}, nil, errors.New("screenshot artifact exceeds the 20 MiB capture limit")
		}
		add("pixels", fmt.Sprintf("pixels/%s-%dx%d.png", theme, result.Viewport.Width, result.Viewport.Height), "image/png", content)
	}
	if result.Artifacts.FullPagePixelsPNGBase64 != "" {
		content, err := base64.StdEncoding.Strict().DecodeString(result.Artifacts.FullPagePixelsPNGBase64)
		if err != nil {
			return bundle.Bundle{}, nil, errors.New("full-page screenshot artifact is not valid base64")
		}
		if len(content) > 20*1024*1024 {
			return bundle.Bundle{}, nil, errors.New("full-page screenshot artifact exceeds the 20 MiB capture limit")
		}
		add("full-page-pixels", fmt.Sprintf("pixels/full-page-%s-%dx%d.png", theme, result.Viewport.Width, result.Viewport.Height), "image/png", content)
	}
	if result.Artifacts.FrozenPageHTML != "" {
		content := []byte(result.Artifacts.FrozenPageHTML)
		if len(content) > 5*1024*1024 {
			return bundle.Bundle{}, nil, errors.New("frozen page exceeds the 5 MiB capture limit")
		}
		add("frozen-page", "page/frozen.html", "text/html; charset=utf-8", content)
	}
	for _, item := range []struct {
		kind string
		path string
		mime string
		raw  json.RawMessage
	}{
		{kind: "outline", path: "outline/outline.json", mime: "application/json", raw: result.Artifacts.Outline},
		{kind: "style-census", path: "style/census.json", mime: "application/json", raw: result.Artifacts.StyleCensus},
		{kind: "redacted-network", path: "network/summary.json", mime: "application/json", raw: result.Artifacts.Network},
		{kind: "component-identity", path: "identity/components.json", mime: "application/json", raw: result.Artifacts.ComponentIdentity},
		{kind: "strict-token-diagnostic", path: "diagnostics/strict-tokens.json", mime: "application/json", raw: result.Artifacts.StrictTokens},
		{kind: "accessibility-snapshot", path: "accessibility/snapshot.json", mime: "application/json", raw: result.Artifacts.AccessibilitySnapshot},
		{kind: "console-summary", path: "console/summary.json", mime: "application/json", raw: result.Artifacts.ConsoleSummary},
		{kind: "performance-metrics", path: "performance/metrics.json", mime: "application/json", raw: result.Artifacts.PerformanceMetrics},
		{kind: "region-style-sample", path: "style/region-sample.json", mime: "application/json", raw: result.Artifacts.RegionStyleSample},
	} {
		if len(item.raw) == 0 {
			continue
		}
		limit := 1024 * 1024
		if item.kind == "accessibility-snapshot" {
			limit = 40 * 1024
		}
		if item.kind == "region-style-sample" {
			limit = 8 * 1024
		}
		if !json.Valid(item.raw) || len(item.raw) > limit {
			return bundle.Bundle{}, nil, fmt.Errorf("%s artifact is invalid JSON or exceeds its %d-byte limit", item.kind, limit)
		}
		add(item.kind, item.path, item.mime, append([]byte(nil), item.raw...))
	}
	if result.Artifacts.RegionPNGBase64 != "" {
		content, err := base64.StdEncoding.Strict().DecodeString(result.Artifacts.RegionPNGBase64)
		if err != nil || len(content) > 5*1024*1024 {
			return bundle.Bundle{}, nil, errors.New("region screenshot is invalid base64 or exceeds the 5 MiB limit")
		}
		add("region-pixels", "pixels/region.png", "image/png", content)
	}
	if err := manifest.Validate(); err != nil {
		return bundle.Bundle{}, nil, fmt.Errorf("validate driver bundle: %w", err)
	}
	for lens, artifact := range map[bundle.LensName]string{
		bundle.LensPixels:     "pixels",
		bundle.LensFrozenPage: "frozen-page",
		bundle.LensOutline:    "outline",
		bundle.LensStyle:      "style-census",
		bundle.LensIdentity:   "component-identity",
		bundle.LensNetwork:    "redacted-network",
	} {
		if manifest.Lenses[lens].Status == bundle.LensPresent && !hasArtifact(manifest.Artifacts, artifact) {
			return bundle.Bundle{}, nil, fmt.Errorf("lens %s is present but artifact %q is missing", lens, artifact)
		}
	}
	return manifest, payloads, nil
}

func validCaptureTheme(theme string) bool {
	return theme == "" || theme == "light" || theme == "dark" || theme == "no-preference"
}

func hasArtifact(artifacts []bundle.Artifact, kind string) bool {
	for _, artifact := range artifacts {
		if artifact.Kind == kind {
			return true
		}
	}
	return false
}

func ensureRequestConsumed(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request has trailing JSON")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
