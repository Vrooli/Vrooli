// Package ogmeta provides Open Graph metadata fetching for link previews.
package ogmeta

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Handlers provides HTTP handlers for OG metadata operations.
type Handlers struct {
	client *http.Client
	cache  *metaCache
}

// metaCache provides simple in-memory caching for OG metadata.
type metaCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	data      Response
	expiresAt time.Time
}

// NewHandlers creates a new OG metadata handler.
func NewHandlers() *Handlers {
	return &Handlers{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 5 redirects
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		cache: &metaCache{
			entries: make(map[string]cacheEntry),
		},
	}
}

// Get handles GET /og-metadata?url=<encoded-url>
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "url parameter is required", http.StatusBadRequest)
		return
	}

	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		http.Error(w, "invalid URL: must be http or https", http.StatusBadRequest)
		return
	}

	// Check cache
	if cached, ok := h.cache.get(targetURL); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Fetch and parse OG metadata
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	meta, err := h.fetchOGMeta(ctx, targetURL)
	if err != nil {
		http.Error(w, "failed to fetch metadata: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Cache the result for 15 minutes
	h.cache.set(targetURL, meta, 15*time.Minute)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	json.NewEncoder(w).Encode(meta)
}

// fetchOGMeta fetches and parses OG metadata from a URL.
func (h *Handlers) fetchOGMeta(ctx context.Context, targetURL string) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return Response{}, err
	}

	// Set a realistic User-Agent to avoid blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LinkPreviewBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := h.client.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	// Limit reading to 1MB to avoid huge pages
	limitedReader := io.LimitReader(resp.Body, 1024*1024)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return Response{}, err
	}

	html := string(body)

	// Parse OG metadata
	meta := Response{
		URL: targetURL,
	}

	// Extract OG tags
	meta.Title = extractMeta(html, "og:title")
	meta.Description = extractMeta(html, "og:description")
	meta.Image = extractMeta(html, "og:image")
	meta.SiteName = extractMeta(html, "og:site_name")
	meta.Type = extractMeta(html, "og:type")

	// Fallback to regular title if no OG title
	if meta.Title == "" {
		meta.Title = extractTitle(html)
	}

	// Fallback to meta description if no OG description
	if meta.Description == "" {
		meta.Description = extractMeta(html, "description")
	}

	// Extract favicon
	meta.Favicon = extractFavicon(html, targetURL)

	return meta, nil
}

// extractMeta extracts content from a meta tag by property or name.
func extractMeta(html, name string) string {
	// Try property attribute (og: tags)
	pattern := regexp.MustCompile(`<meta[^>]+(?:property|name)=["']` + regexp.QuoteMeta(name) + `["'][^>]+content=["']([^"']+)["']`)
	matches := pattern.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try with content first
	pattern = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+(?:property|name)=["']` + regexp.QuoteMeta(name) + `["']`)
	matches = pattern.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// extractTitle extracts the page title from <title> tag.
func extractTitle(html string) string {
	pattern := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	matches := pattern.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractFavicon extracts the favicon URL.
func extractFavicon(html, baseURL string) string {
	// Try link rel="icon"
	pattern := regexp.MustCompile(`<link[^>]+rel=["'](?:shortcut )?icon["'][^>]+href=["']([^"']+)["']`)
	matches := pattern.FindStringSubmatch(html)
	if len(matches) > 1 {
		return resolveURL(baseURL, matches[1])
	}

	// Try with href first
	pattern = regexp.MustCompile(`<link[^>]+href=["']([^"']+)["'][^>]+rel=["'](?:shortcut )?icon["']`)
	matches = pattern.FindStringSubmatch(html)
	if len(matches) > 1 {
		return resolveURL(baseURL, matches[1])
	}

	// Default to /favicon.ico
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host + "/favicon.ico"
}

// resolveURL resolves a potentially relative URL against a base URL.
func resolveURL(baseURL, ref string) string {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return ref
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}

	return base.ResolveReference(refURL).String()
}

// Cache methods
func (c *metaCache) get(key string) (Response, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return Response{}, false
	}
	return entry.data, true
}

func (c *metaCache) set(key string, data Response, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}

	// Clean up expired entries periodically (simple approach)
	if len(c.entries) > 100 {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expiresAt) {
				delete(c.entries, k)
			}
		}
	}
}
