package handlers

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LinkPreviewResponse contains OpenGraph metadata for a URL.
type LinkPreviewResponse struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
	SiteName    string `json:"site_name,omitempty"`
}

// linkPreviewCache stores fetched previews with TTL.
type linkPreviewCache struct {
	mu      sync.RWMutex
	entries map[string]linkPreviewCacheEntry
}

type linkPreviewCacheEntry struct {
	preview   *LinkPreviewResponse
	expiresAt time.Time
}

var previewCache = &linkPreviewCache{
	entries: make(map[string]linkPreviewCacheEntry),
}

const (
	linkPreviewCacheTTL    = 24 * time.Hour // Cache for 24 hours since OG data rarely changes
	linkPreviewFetchTimeout = 5 * time.Second
	linkPreviewMaxBodySize  = 512 * 1024 // 512KB max HTML to parse
)

// GetLinkPreview fetches OpenGraph metadata for a URL.
// GET /api/v1/link-preview?url=<encoded-url>
func (h *Handler) GetLinkPreview(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	// Validate URL format
	parsedURL, err := url.Parse(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	// Check cache first
	if cached := previewCache.get(rawURL); cached != nil {
		h.respondSuccess(w, http.StatusOK, cached)
		return
	}

	// Fetch and parse the URL
	preview, err := fetchLinkPreviewData(r.Context(), rawURL)
	if err != nil {
		// Return 204 No Content if we can't fetch metadata
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Cache the result
	previewCache.set(rawURL, preview)

	h.respondSuccess(w, http.StatusOK, preview)
}

func (c *linkPreviewCache) get(url string) *LinkPreviewResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[url]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.preview
}

func (c *linkPreviewCache) set(url string, preview *LinkPreviewResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[url] = linkPreviewCacheEntry{
		preview:   preview,
		expiresAt: time.Now().Add(linkPreviewCacheTTL),
	}
}

func fetchLinkPreviewData(ctx context.Context, targetURL string) (*LinkPreviewResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, linkPreviewFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	// Set a reasonable User-Agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LinkPreviewBot/1.0)")
	req.Header.Set("Accept", "text/html")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	// Read limited body
	body, err := io.ReadAll(io.LimitReader(resp.Body, linkPreviewMaxBodySize))
	if err != nil {
		return nil, err
	}

	html := string(body)
	preview := parseOpenGraphTags(html)

	// Extract favicon if not in OG tags
	if preview.Favicon == "" {
		preview.Favicon = extractFaviconFromHTML(html, targetURL)
	}

	// Use page title if no OG title
	if preview.Title == "" {
		preview.Title = extractPageTitle(html)
	}

	return preview, nil
}

// OpenGraph meta tag patterns
var (
	ogTitleRegex       = regexp.MustCompile(`(?i)<meta\s+(?:property|name)=["']og:title["']\s+content=["']([^"']+)["']`)
	ogDescRegex        = regexp.MustCompile(`(?i)<meta\s+(?:property|name)=["']og:description["']\s+content=["']([^"']+)["']`)
	ogImageRegex       = regexp.MustCompile(`(?i)<meta\s+(?:property|name)=["']og:image["']\s+content=["']([^"']+)["']`)
	ogSiteNameRegex    = regexp.MustCompile(`(?i)<meta\s+(?:property|name)=["']og:site_name["']\s+content=["']([^"']+)["']`)
	pageTitleRegex     = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	faviconLinkRegex   = regexp.MustCompile(`(?i)<link[^>]+rel=["'](?:icon|shortcut icon)["'][^>]+href=["']([^"']+)["']`)
	descMetaRegex      = regexp.MustCompile(`(?i)<meta\s+name=["']description["']\s+content=["']([^"']+)["']`)

	// Alternative patterns with content first
	ogTitleRegexAlt    = regexp.MustCompile(`(?i)<meta\s+content=["']([^"']+)["']\s+(?:property|name)=["']og:title["']`)
	ogDescRegexAlt     = regexp.MustCompile(`(?i)<meta\s+content=["']([^"']+)["']\s+(?:property|name)=["']og:description["']`)
	ogImageRegexAlt    = regexp.MustCompile(`(?i)<meta\s+content=["']([^"']+)["']\s+(?:property|name)=["']og:image["']`)
	ogSiteNameRegexAlt = regexp.MustCompile(`(?i)<meta\s+content=["']([^"']+)["']\s+(?:property|name)=["']og:site_name["']`)
	descMetaRegexAlt   = regexp.MustCompile(`(?i)<meta\s+content=["']([^"']+)["']\s+name=["']description["']`)
)

func parseOpenGraphTags(html string) *LinkPreviewResponse {
	preview := &LinkPreviewResponse{}

	// Try primary patterns first, then alternatives
	if match := ogTitleRegex.FindStringSubmatch(html); len(match) > 1 {
		preview.Title = strings.TrimSpace(match[1])
	} else if match := ogTitleRegexAlt.FindStringSubmatch(html); len(match) > 1 {
		preview.Title = strings.TrimSpace(match[1])
	}

	if match := ogDescRegex.FindStringSubmatch(html); len(match) > 1 {
		preview.Description = strings.TrimSpace(match[1])
	} else if match := ogDescRegexAlt.FindStringSubmatch(html); len(match) > 1 {
		preview.Description = strings.TrimSpace(match[1])
	} else if match := descMetaRegex.FindStringSubmatch(html); len(match) > 1 {
		preview.Description = strings.TrimSpace(match[1])
	} else if match := descMetaRegexAlt.FindStringSubmatch(html); len(match) > 1 {
		preview.Description = strings.TrimSpace(match[1])
	}

	if match := ogImageRegex.FindStringSubmatch(html); len(match) > 1 {
		preview.Image = strings.TrimSpace(match[1])
	} else if match := ogImageRegexAlt.FindStringSubmatch(html); len(match) > 1 {
		preview.Image = strings.TrimSpace(match[1])
	}

	if match := ogSiteNameRegex.FindStringSubmatch(html); len(match) > 1 {
		preview.SiteName = strings.TrimSpace(match[1])
	} else if match := ogSiteNameRegexAlt.FindStringSubmatch(html); len(match) > 1 {
		preview.SiteName = strings.TrimSpace(match[1])
	}

	return preview
}

func extractPageTitle(html string) string {
	if match := pageTitleRegex.FindStringSubmatch(html); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func extractFaviconFromHTML(html, baseURL string) string {
	if match := faviconLinkRegex.FindStringSubmatch(html); len(match) > 1 {
		favicon := strings.TrimSpace(match[1])
		// Make relative URLs absolute
		if !strings.HasPrefix(favicon, "http") {
			parsed, err := url.Parse(baseURL)
			if err == nil {
				if strings.HasPrefix(favicon, "//") {
					return parsed.Scheme + ":" + favicon
				} else if strings.HasPrefix(favicon, "/") {
					return parsed.Scheme + "://" + parsed.Host + favicon
				} else {
					return parsed.Scheme + "://" + parsed.Host + "/" + favicon
				}
			}
		}
		return favicon
	}

	// Default to /favicon.ico
	parsed, err := url.Parse(baseURL)
	if err == nil {
		return parsed.Scheme + "://" + parsed.Host + "/favicon.ico"
	}
	return ""
}
