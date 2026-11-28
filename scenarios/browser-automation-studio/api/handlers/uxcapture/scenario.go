package uxcapture

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	bundle "github.com/vrooli/browser-automation-studio/services/uxcapture"
)

type ScenarioCaptureClient interface {
	CaptureScenarioPage(ctx context.Context, request *driver.ScenarioCaptureRequest) (json.RawMessage, error)
}

type ScenarioURLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, slug string) (string, error)
}

type scenarioPortResolver interface {
	ResolveScenarioURL(ctx context.Context, slug, portKey string) (string, error)
}

var captureScenarioSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// MountScenarioHTTP adds the one-session capture entry for running local scenarios.
// The API resolves the scenario slug through the control plane; the browser never
// accepts an arbitrary local URL from the CLI.
func MountScenarioHTTP(r chi.Router, client ScenarioCaptureClient, resolver ScenarioURLResolver, capturesRoot string) {
	r.Post("/ux-capture/scenario", func(w http.ResponseWriter, req *http.Request) {
		if client == nil || resolver == nil {
			writeError(w, http.StatusServiceUnavailable, "BAS scenario capture is unavailable")
			return
		}
		var input struct {
			Scenario           string   `json:"scenario"`
			Route              string   `json:"route,omitempty"`
			Steps              []string `json:"steps,omitempty"`
			WaitSelector       string   `json:"wait_selector,omitempty"`
			WaitHiddenSelector string   `json:"wait_hidden_selector,omitempty"`
			Theme              string   `json:"theme,omitempty"`
			StrictTokens       bool     `json:"strict_tokens,omitempty"`
			Width              int      `json:"width,omitempty"`
			Height             int      `json:"height,omitempty"`
			EvidenceProfile    string   `json:"evidence_profile,omitempty"`
			RegionSelector     string   `json:"region_selector,omitempty"`
			FullPagePixels     bool     `json:"full_page_pixels,omitempty"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, 16*1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "Scenario capture request must be one valid JSON object.")
			return
		}
		if err := ensureRequestConsumed(decoder); err != nil {
			writeError(w, http.StatusBadRequest, "Capture request must be one JSON object.")
			return
		}
		if !captureScenarioSlug.MatchString(input.Scenario) {
			writeError(w, http.StatusBadRequest, "scenario must be a valid scenario slug")
			return
		}
		route, err := cleanCaptureRoute(input.Route)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !validCaptureDimension(input.Width, 320, 2560) || !validCaptureDimension(input.Height, 240, 1600) {
			writeError(w, http.StatusBadRequest, "viewport dimensions are outside the supported range")
			return
		}
		if len(input.Steps) > 32 {
			writeError(w, http.StatusBadRequest, "at most 32 state steps are allowed")
			return
		}
		for _, step := range input.Steps {
			if len(step) > 256 || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(step)), "click:") || strings.TrimSpace(strings.TrimPrefix(step, "click:")) == "" {
				writeError(w, http.StatusBadRequest, "state steps must use click:<accessible label>")
				return
			}
		}
		if len(input.WaitSelector) > 512 {
			writeError(w, http.StatusBadRequest, "wait_selector must be at most 512 characters")
			return
		}
		if len(input.WaitHiddenSelector) > 512 {
			writeError(w, http.StatusBadRequest, "wait_hidden_selector must be at most 512 characters")
			return
		}
		if !validCaptureTheme(input.Theme) {
			writeError(w, http.StatusBadRequest, "theme must be light, dark, or no-preference")
			return
		}
		if input.EvidenceProfile != "" && input.EvidenceProfile != "componenttests" {
			writeError(w, http.StatusBadRequest, "evidence_profile must be componenttests")
			return
		}
		if len(input.RegionSelector) > 256 || strings.HasPrefix(strings.TrimSpace(input.RegionSelector), "//") {
			writeError(w, http.StatusBadRequest, "region_selector must be a CSS selector up to 256 characters")
			return
		}
		base, err := resolveScenarioCaptureURL(req.Context(), resolver, input.Scenario)
		if err != nil {
			writeError(w, http.StatusFailedDependency, "Could not resolve the running scenario UI: "+err.Error())
			return
		}
		parsed, err := url.Parse(base)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || !isCaptureLoopback(parsed.Hostname()) {
			writeError(w, http.StatusFailedDependency, "scenario UI resolver returned a non-loopback URL")
			return
		}
		allowedOrigins := []string{parsed.Scheme + "://" + parsed.Host}
		if portResolver, ok := resolver.(scenarioPortResolver); ok {
			if apiBase, apiErr := portResolver.ResolveScenarioURL(req.Context(), input.Scenario, "API_PORT"); apiErr == nil {
				if apiURL, parseErr := url.Parse(apiBase); parseErr == nil && (apiURL.Scheme == "http" || apiURL.Scheme == "https") && isCaptureLoopback(apiURL.Hostname()) {
					origin := apiURL.Scheme + "://" + apiURL.Host
					if origin != allowedOrigins[0] {
						allowedOrigins = append(allowedOrigins, origin)
					}
				}
			}
		}
		raw, err := client.CaptureScenarioPage(req.Context(), &driver.ScenarioCaptureRequest{
			URL: strings.TrimRight(base, "/") + route, Scenario: input.Scenario, Route: redactRouteQuery(route),
			Width: input.Width, Height: input.Height, Steps: input.Steps, WaitSelector: input.WaitSelector, WaitHiddenSelector: input.WaitHiddenSelector, Theme: input.Theme, StrictTokens: input.StrictTokens, AllowedOrigins: allowedOrigins, EvidenceProfile: input.EvidenceProfile, RegionSelector: input.RegionSelector, FullPagePixels: input.FullPagePixels,
		})
		if err != nil {
			writeError(w, http.StatusBadGateway, "Scenario page capture failed: "+err.Error())
			return
		}
		manifest, payloads, err := DecodeCapture(raw)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Driver returned an invalid UX capture bundle: "+err.Error())
			return
		}
		if manifest.Source.Kind != bundle.SourceScenario || manifest.Source.Scenario != input.Scenario || manifest.Source.Route != redactRouteQuery(route) {
			writeError(w, http.StatusBadGateway, "Driver returned mismatched scenario source metadata")
			return
		}
		if err := persistCaptureBundle(capturesRoot, manifest, payloads); err != nil {
			respondCapturePersistenceError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		var driverMetadata struct {
			Artifacts struct {
				FullPagePixelsStatus bundle.Lens `json:"full_page_pixels_status"`
			} `json:"artifacts"`
		}
		_ = json.Unmarshal(raw, &driverMetadata)
		var fullPageStatus *bundle.Lens
		if driverMetadata.Artifacts.FullPagePixelsStatus.Status != "" {
			fullPageStatus = &driverMetadata.Artifacts.FullPagePixelsStatus
		}
		_ = json.NewEncoder(w).Encode(struct {
			Manifest             bundle.Bundle `json:"manifest"`
			FullPagePixelsStatus *bundle.Lens  `json:"full_page_pixels_status,omitempty"`
		}{Manifest: manifest, FullPagePixelsStatus: fullPageStatus})
	})
}

func redactRouteQuery(route string) string {
	parsed, err := url.ParseRequestURI(route)
	if err != nil || parsed.RawQuery == "" {
		return route
	}
	query := parsed.Query()
	for key := range query {
		query.Set(key, "[REDACTED]")
	}
	parsed.RawQuery = query.Encode()
	return parsed.Path + "?" + parsed.RawQuery
}

func resolveScenarioCaptureURL(ctx context.Context, resolver ScenarioURLResolver, slug string) (string, error) {
	if ports, ok := resolver.(scenarioPortResolver); ok {
		if value, err := ports.ResolveScenarioURL(ctx, slug, "UI_PORT"); err == nil && strings.TrimSpace(value) != "" {
			return value, nil
		}
	}
	return resolver.ResolveScenarioURLDefault(ctx, slug)
}

func cleanCaptureRoute(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/", nil
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, `\`) {
		return "", errors.New("route must be a local path beginning with one slash")
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.Contains(parsed.Path, "..") {
		return "", errors.New("route must be a safe local path")
	}
	return value, nil
}

func validCaptureDimension(value, min, max int) bool {
	return value == 0 || (value >= min && value <= max)
}

func isCaptureLoopback(host string) bool {
	return strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "::1"
}
