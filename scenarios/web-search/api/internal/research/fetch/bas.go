package fetch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/discovery"

	"web-search/internal/evidence"
	internalresearch "web-search/internal/research"
)

// DefaultBrowserTimeout bounds one browser-leg fetch. Public capture includes
// navigation, artifact export, and one bounded local artifact read.
const DefaultBrowserTimeout = 60 * time.Second

// BundleResolver resolves BAS and submits one public-page capture request.
// Tests inject an HTTP server; production uses scenario discovery.
type BundleResolver func(ctx context.Context) (string, error)

type bundleArtifact struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Size   int64  `json:"size_bytes"`
	SHA256 string `json:"sha256"`
}

type captureBundle struct {
	SchemaVersion string `json:"schema_version"`
	ID            string `json:"id"`
	Source        struct {
		Kind string `json:"kind"`
		URL  string `json:"url"`
	} `json:"source"`
	Session struct {
		ID string `json:"id"`
	} `json:"session"`
	Lenses map[string]struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"lenses"`
	Privacy struct {
		RedactionStatus string `json:"redaction_status"`
		External        bool   `json:"external"`
		Ephemeral       bool   `json:"ephemeral"`
	} `json:"privacy"`
	Artifacts []bundleArtifact `json:"artifacts"`
}

type captureResponse struct {
	Manifest captureBundle `json:"manifest"`
}

// BASFetcher is the browser escalation leg. It asks BAS for its canonical,
// redaction-first ux-capture.v1 public-page bundle and extracts readable text
// from the frozen-page artifact, preserving the HTTP leg's text contract.
type BASFetcher struct {
	Resolve BundleResolver
	Timeout time.Duration
	Client  *http.Client

	mu  sync.Mutex
	url string
}

func NewBASFetcher() *BASFetcher {
	return &BASFetcher{
		Resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
		},
		Timeout: DefaultBrowserTimeout,
		Client:  &http.Client{Timeout: DefaultBrowserTimeout},
	}
}

func (f *BASFetcher) baseURL(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.url != "" {
		return f.url, nil
	}
	if f.Resolve == nil {
		return "", fmt.Errorf("research: BAS fetcher has no URL resolver")
	}
	u, err := f.Resolve(ctx)
	if err != nil {
		return "", fmt.Errorf("research: browser-automation-studio not available: %w", err)
	}
	f.url = strings.TrimRight(u, "/")
	return f.url, nil
}

func (f *BASFetcher) Fetch(ctx context.Context, url string) (string, error) {
	text, _, err := f.fetchBundle(ctx, url)
	return text, err
}

func (f *BASFetcher) fetchBundle(ctx context.Context, pageURL string) (text, executionID string, retErr error) {
	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" {
		return "", "", fmt.Errorf("research: fetch: empty url")
	}
	timeout := f.Timeout
	if timeout <= 0 {
		timeout = DefaultBrowserTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	base, err := f.baseURL(ctx)
	if err != nil {
		return "", "", err
	}
	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	body, _ := json.Marshal(map[string]any{"url": pageURL, "width": 1280, "height": 720, "theme": "light"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/ux-capture/public", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		f.mu.Lock()
		f.url = ""
		f.mu.Unlock()
		return "", "", fmt.Errorf("research: browser capture: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", "", fmt.Errorf("research: browser capture returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	var result captureResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&result); err != nil {
		return "", "", fmt.Errorf("research: decode UX capture bundle: %w", err)
	}
	m := result.Manifest
	if validCaptureBundleID(m.ID) {
		// The BAS owner alone removes an external ephemeral bundle. Detach cleanup
		// from caller cancellation so a canceled fetch does not leak its capture.
		defer func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cleanupCancel()
			if err := f.deleteEphemeralBundle(cleanupCtx, client, base, m.ID); err != nil && retErr == nil {
				retErr = fmt.Errorf("research: BAS ephemeral bundle cleanup: %w", err)
			}
		}()
	}
	if m.SchemaVersion != "ux-capture.v1" || m.Source.Kind != "url" || m.Session.ID == "" || m.ID == "" {
		return "", "", fmt.Errorf("research: BAS returned an incompatible public-page bundle")
	}
	if m.Privacy.RedactionStatus != "complete" || !m.Privacy.External || !m.Privacy.Ephemeral {
		return "", "", fmt.Errorf("research: refusing UX bundle without complete redaction and external ephemeral status")
	}
	if l := m.Lenses["L1"]; l.Status != "present" {
		return "", "", fmt.Errorf("research: UX bundle has no frozen page (L1 %s: %s)", l.Status, l.Reason)
	}
	var artifact *bundleArtifact
	for i := range m.Artifacts {
		if m.Artifacts[i].Kind == "frozen-page" {
			artifact = &m.Artifacts[i]
			break
		}
	}
	if artifact == nil {
		return "", "", fmt.Errorf("research: UX bundle has no frozen-page artifact")
	}
	artifactURL, err := captureArtifactURL(base, m.ID, artifact.Path)
	if err != nil {
		return "", "", err
	}
	data, contentType, err := f.fetchArtifact(ctx, client, artifactURL, artifact.Size)
	if err != nil {
		return "", "", fmt.Errorf("research: read BAS-owned frozen-page artifact: %w", err)
	}
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType != "text/html" {
		return "", "", fmt.Errorf("research: frozen-page artifact has unexpected content type %q", contentType)
	}
	sum := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), artifact.SHA256) {
		return "", "", fmt.Errorf("research: frozen-page artifact failed integrity check")
	}
	text = internalresearch.ExtractReadableText(string(data))
	if strings.TrimSpace(text) == "" {
		return "", "", fmt.Errorf("research: browser capture returned no readable page text for %q", pageURL)
	}
	return text, m.Session.ID, nil
}

func captureArtifactURL(baseURL, bundleID, artifactPath string) (string, error) {
	if !validCaptureBundleID(bundleID) || artifactPath == "" || strings.ContainsRune(artifactPath, '\\') || strings.HasPrefix(artifactPath, "/") {
		return "", fmt.Errorf("research: unsafe BAS bundle artifact identity")
	}
	segments := strings.Split(artifactPath, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("research: unsafe BAS bundle artifact path")
		}
	}
	escaped := make([]string, len(segments))
	for i, segment := range segments {
		escaped[i] = url.PathEscape(segment)
	}
	return strings.TrimRight(baseURL, "/") + "/api/v1/ux-capture/bundles/" + url.PathEscape(bundleID) + "/artifacts/" + strings.Join(escaped, "/"), nil
}

func validCaptureBundleID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}

func (f *BASFetcher) fetchArtifact(ctx context.Context, client *http.Client, artifactURL string, expectedSize int64) ([]byte, string, error) {
	if expectedSize < 0 || expectedSize > 5*1024*1024 {
		return nil, "", fmt.Errorf("frozen-page artifact size is outside the 5 MiB bound")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("BAS artifact owner returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) != expectedSize || len(data) > 5*1024*1024 {
		return nil, "", fmt.Errorf("frozen-page artifact size does not match manifest")
	}
	return data, resp.Header.Get("Content-Type"), nil
}

func (f *BASFetcher) deleteEphemeralBundle(ctx context.Context, client *http.Client, baseURL, bundleID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, strings.TrimRight(baseURL, "/")+"/api/v1/ux-capture/bundles/"+url.PathEscape(bundleID), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("BAS artifact owner returned HTTP %d for ephemeral cleanup", resp.StatusCode)
	}
	return nil
}

func (f *BASFetcher) FetchObservation(ctx context.Context, url string) (evidence.NewObservation, error) {
	text, executionID, err := f.fetchBundle(ctx, url)
	if err != nil {
		return evidence.NewObservation{}, err
	}
	return evidence.NewObservation{URL: url, RetrievedAt: time.Now().UTC(), ProducerExecutionID: executionID, Content: []byte(text), ExtractionRevision: "bas-readable-text-v1", Retention: "metadata-and-extracted-content"}, nil
}

var (
	_ Fetcher                             = (*BASFetcher)(nil)
	_ internalresearch.ObservationFetcher = (*BASFetcher)(nil)
)
