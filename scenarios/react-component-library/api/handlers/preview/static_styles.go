package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"react-component-library/internal/librarywalk"
	"react-component-library/internal/themes"
)

const defaultPreviewKit = "vrooli-default"

var previewCSSCache struct {
	sync.Mutex
	key      string
	modified string
	css      string
}

func discoverRepoRoot(explicit string) string {
	candidates := []string{explicit, ".", "../..", "../../..", "../../../..", "../../../../.."}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		path := filepath.Join(candidate, "templates", "design", defaultPreviewKit, "adapters", "react-vite-tailwind", "tokens.css")
		if _, err := os.Stat(path); err == nil {
			return candidate
		}
	}
	return explicit
}

func previewDesignSystemCSS(repoRoot, kit string) (string, error) {
	if kit == "none" {
		return "", nil
	}
	if kit == "" || strings.Contains(kit, ".") || strings.ContainsAny(kit, `/\\`) {
		return "", fmt.Errorf("invalid design kit %q", kit)
	}
	adapter := filepath.Join(repoRoot, "templates", "design", kit, "adapters", "react-vite-tailwind")
	tokensPath := filepath.Join(adapter, "tokens.css")
	baseTokensPath := filepath.Join(repoRoot, "templates", "design", "_base", "tokens.css")
	utilitiesPath := filepath.Join(adapter, "preview-utilities.css")
	baseTokensInfo, err := os.Stat(baseTokensPath)
	if err != nil {
		return "", fmt.Errorf("shared token vocabulary missing: %w", err)
	}
	tokensInfo, err := os.Stat(tokensPath)
	if err != nil {
		return "", fmt.Errorf("tokens stylesheet missing: %w", err)
	}
	utilitiesInfo, err := os.Stat(utilitiesPath)
	if err != nil {
		return "", fmt.Errorf("compiled utility stylesheet missing for %q: %w", kit, err)
	}
	key := kit
	modified := fmt.Sprintf("%d:%d:%d", baseTokensInfo.ModTime().UnixNano(), tokensInfo.ModTime().UnixNano(), utilitiesInfo.ModTime().UnixNano())
	previewCSSCache.Lock()
	defer previewCSSCache.Unlock()
	if previewCSSCache.key == key && previewCSSCache.modified == modified && previewCSSCache.css != "" {
		return previewCSSCache.css, nil
	}
	tokens, err := themes.ComposeKitCSS(repoRoot, kit)
	if err != nil {
		return "", fmt.Errorf("compose token stylesheets: %w", err)
	}
	utilities, err := os.ReadFile(utilitiesPath)
	if err != nil {
		return "", fmt.Errorf("read compiled utility stylesheet: %w", err)
	}
	if len(utilities) == 0 {
		return "", fmt.Errorf("compiled utility stylesheet for %q is empty", kit)
	}
	previewCSSCache.key = key
	previewCSSCache.modified = modified
	previewCSSCache.css = tokens + "\n" + string(utilities)
	return previewCSSCache.css, nil
}

// ServeBaseStylesFixture renders the exact published foundation CSS without
// loading any design kit. It is a browser-measurable proof of both the
// canonical floor and unlayered scenario precedence.
func (h *HarnessHandler) ServeBaseStylesFixture(w http.ResponseWriter, r *http.Request) {
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	if version == "" || filepath.Base(version) != version || strings.HasPrefix(version, ".") || strings.ContainsAny(version, `/\\`) {
		http.Error(w, "preview: a safe BaseStyles version is required", http.StatusBadRequest)
		return
	}
	path := filepath.Join(h.repoRoot, "scenarios", "react-component-library", "library", "foundations", "BaseStyles", "versions", version, "BaseStyles.ts")
	raw, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "preview: BaseStyles version is unavailable", http.StatusNotFound)
		return
	}
	css, err := extractBaseStylesCSS(string(raw))
	if err != nil {
		http.Error(w, "preview: BaseStyles source is malformed", http.StatusUnprocessableEntity)
		return
	}
	css = strings.ReplaceAll(css, "</style", "<\\/style")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><style>%s</style></head><body><main><div id="canonical" style="padding:var(--space-md);border-radius:var(--radius-panel)">Canonical layered default</div><div id="override" style="--space-md:7px;padding:var(--space-md);border-radius:var(--radius-panel)">Unlayered scenario override</div><output id="measurement" data-testid="base-styles-measurement"></output></main><script>const canonical=getComputedStyle(document.querySelector('#canonical'));const override=getComputedStyle(document.querySelector('#override'));const value='canonical padding='+canonical.paddingTop+', radius='+canonical.borderTopLeftRadius+'; override padding='+override.paddingTop;const output=document.querySelector('#measurement');output.textContent=value;output.dataset.canonicalPadding=canonical.paddingTop;output.dataset.canonicalRadius=canonical.borderTopLeftRadius;output.dataset.overridePadding=override.paddingTop;</script></body></html>`, css)
}

func extractBaseStylesCSS(source string) (string, error) {
	const marker = "export const baseStyles = `"
	start := strings.Index(source, marker)
	if start < 0 {
		return "", fmt.Errorf("baseStyles template literal not found")
	}
	rest := source[start+len(marker):]
	end := strings.Index(rest, "`;")
	if end < 0 {
		return "", fmt.Errorf("baseStyles template literal is unterminated")
	}
	return rest[:end], nil
}

// previewConsumerCSS reads a scenario's current build output and source-owned
// token bridge directly. It never copies either artifact into RCL: the named
// scenario remains the sole source of truth for the cascade under test.
func previewConsumerCSS(repoRoot, consumer string) (string, error) {
	if consumer == "" || strings.Contains(consumer, ".") || strings.ContainsAny(consumer, `/\\`) {
		return "", fmt.Errorf("invalid consumer %q", consumer)
	}
	uiRoot := filepath.Join(repoRoot, "scenarios", consumer, "ui")
	tokensPath := filepath.Join(uiRoot, "src", "design-tokens.css")
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		return "", fmt.Errorf("read source token bridge: %w", err)
	}
	bundlePaths, err := filepath.Glob(filepath.Join(uiRoot, "dist", "assets", "*.css"))
	if err != nil || len(bundlePaths) == 0 {
		return "", fmt.Errorf("compiled CSS bundle is missing; build scenarios/%s first", consumer)
	}
	var newestBundle int64
	var css strings.Builder
	for _, path := range bundlePaths {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return "", fmt.Errorf("stat compiled CSS %s: %w", filepath.Base(path), statErr)
		}
		if info.ModTime().UnixNano() > newestBundle {
			newestBundle = info.ModTime().UnixNano()
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", fmt.Errorf("read compiled CSS %s: %w", filepath.Base(path), readErr)
		}
		css.Write(data)
		css.WriteByte('\n')
	}
	var newestSource int64
	err = librarywalk.WalkInfo(context.Background(), filepath.Join(uiRoot, "src"), func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() && info.ModTime().UnixNano() > newestSource {
			newestSource = info.ModTime().UnixNano()
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("inspect consumer sources: %w", err)
	}
	if newestSource > newestBundle {
		return "", fmt.Errorf("compiled CSS is older than consumer source; rebuild scenarios/%s", consumer)
	}
	return string(tokens) + "\n" + css.String(), nil
}

// previewConsumerContrastFloor keeps contrast acceptance coupled to the
// consumer-owned token contract. Browser workflows read the emitted meta value
// instead of duplicating a threshold that could drift from token-map.json.
func previewConsumerContrastFloor(repoRoot, consumer string) (string, error) {
	if consumer == "" || strings.Contains(consumer, ".") || strings.ContainsAny(consumer, `/\\`) {
		return "", fmt.Errorf("invalid consumer %q", consumer)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", consumer, "ui", "token-map.json"))
	if err != nil {
		return "", fmt.Errorf("read consumer token map: %w", err)
	}
	var tokenMap struct {
		ContrastFloor float64 `json:"contrast_floor"`
	}
	if err := json.Unmarshal(data, &tokenMap); err != nil {
		return "", fmt.Errorf("parse consumer token map: %w", err)
	}
	if tokenMap.ContrastFloor <= 0 {
		return "", fmt.Errorf("consumer token map has no positive contrast_floor")
	}
	return strconv.FormatFloat(tokenMap.ContrastFloor, 'f', -1, 64), nil
}
