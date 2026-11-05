package target

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CDPPage is the target-owned subset of Chrome's /json/list response needed
// to bind BAS to one Electron renderer without guessing from page order.
type CDPPage struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	Type                 string `json:"type"`
}

// RendererExpectation describes the stable identity used to select a page
// when a generated application has more than one Electron window.
type RendererExpectation struct {
	ID        string
	URLPrefix string
	Title     string
}

// DiscoverElectronRenderer reads the controlled CDP target list and returns
// the exact renderer requested by identity. It never falls back to the first
// page: a missing or ambiguous renderer is a target failure.
func DiscoverElectronRenderer(ctx context.Context, endpoint, rendererID string) (CDPPage, error) {
	if strings.TrimSpace(rendererID) == "" {
		return CDPPage{}, fmt.Errorf("renderer id is required")
	}
	pages, err := discoverElectronRenderers(ctx, endpoint)
	if err != nil {
		return CDPPage{}, err
	}
	return SelectElectronRenderer(pages, RendererExpectation{ID: rendererID})
}

// DiscoverElectronRenderers returns the controlled target list from the
// renderer's HTTP discovery endpoint. The endpoint itself must be numeric
// loopback; websocket URLs are returned only as attachment references.
func DiscoverElectronRenderers(ctx context.Context, endpoint string) ([]CDPPage, error) {
	return discoverElectronRenderers(ctx, endpoint)
}

func discoverElectronRenderers(ctx context.Context, endpoint string) ([]CDPPage, error) {
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("CDP discovery endpoint must be an HTTP URL")
	}
	if err := ValidateAppTargetEndpoint(u); err != nil {
		return nil, err
	}
	u.Path = "/json/list"
	u.RawQuery = ""
	u.Fragment = ""
	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create CDP discovery request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discover Electron renderers: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discover Electron renderers: HTTP %s", resp.Status)
	}
	var pages []CDPPage
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		return nil, fmt.Errorf("decode Electron renderer list: %w", err)
	}
	return pages, nil
}

// SelectElectronRenderer applies the target identity rules to a discovered
// page list. It never silently chooses the first page when the identity is
// ambiguous.
func SelectElectronRenderer(pages []CDPPage, expectation RendererExpectation) (CDPPage, error) {
	if strings.TrimSpace(expectation.ID) == "" && strings.TrimSpace(expectation.URLPrefix) == "" && strings.TrimSpace(expectation.Title) == "" {
		return CDPPage{}, fmt.Errorf("renderer identity is required; refusing to select an attachable page by order")
	}

	var matches []CDPPage
	for _, page := range pages {
		if page.Type != "page" || strings.TrimSpace(page.WebSocketDebuggerURL) == "" {
			continue
		}
		if expectation.ID != "" && page.ID != expectation.ID {
			continue
		}
		if expectation.URLPrefix != "" && !strings.HasPrefix(page.URL, expectation.URLPrefix) {
			continue
		}
		if expectation.Title != "" && page.Title != expectation.Title {
			continue
		}
		matches = append(matches, page)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if expectation.ID != "" && len(matches) == 0 {
		return CDPPage{}, fmt.Errorf("renderer %q was not found or attachable", expectation.ID)
	}
	return CDPPage{}, fmt.Errorf("renderer identity matched %d attachable pages (expectation url_prefix=%q title=%q pages=%s)", len(matches), expectation.URLPrefix, expectation.Title, summarizePageCriteria(pages, expectation))
}

func summarizePageCriteria(pages []CDPPage, expectation RendererExpectation) string {
	if len(pages) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(pages))
	for _, page := range pages {
		parts = append(parts, fmt.Sprintf("{type=%q,url=%q,ws=%t,id=%q,url_match=%t,title_match=%t}", page.Type, page.URL, strings.TrimSpace(page.WebSocketDebuggerURL) != "", page.ID, expectation.URLPrefix == "" || strings.HasPrefix(page.URL, expectation.URLPrefix), expectation.Title == "" || page.Title == expectation.Title))
	}
	return strings.Join(parts, ",")
}
