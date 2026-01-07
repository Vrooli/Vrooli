package bundles

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"deployment-manager/shared"
)

// analyzerBundleResponse represents the response structure from the analyzer's bundle endpoint.
// The analyzer may return the manifest in two formats:
//   - Wrapped: {"manifest": {"skeleton": {...actual manifest...}}}
//   - Direct:  {"manifest": {...actual manifest...}}
type analyzerBundleResponse struct {
	Manifest json.RawMessage `json:"manifest"`
}

// extractManifestBytes extracts the actual manifest bytes from the analyzer response.
// It handles both wrapped (with "skeleton" key) and direct manifest formats.
func extractManifestBytes(manifestField json.RawMessage) ([]byte, error) {
	if len(manifestField) == 0 {
		return nil, fmt.Errorf("analyzer response missing manifest")
	}

	// Try to extract from nested "skeleton" field (wrapped format)
	var wrapped struct {
		Skeleton json.RawMessage `json:"skeleton"`
	}
	if json.Unmarshal(manifestField, &wrapped) == nil && len(wrapped.Skeleton) > 0 {
		return wrapped.Skeleton, nil
	}

	// Use manifest directly (it's already the final form)
	return manifestField, nil
}

// FetchSkeletonBundle retrieves the analyzer-emitted desktop bundle skeleton for a scenario.
func FetchSkeletonBundle(ctx context.Context, scenario string) (*Manifest, error) {
	baseURL, err := shared.GetConfigResolver().ResolveAnalyzerURL()
	if err != nil {
		return nil, err
	}

	target, err := url.Parse(fmt.Sprintf("%s/api/v1/scenarios/%s/bundle/manifest", baseURL, url.PathEscape(scenario)))
	if err != nil {
		return nil, fmt.Errorf("build analyzer url: %w", err)
	}
	q := target.Query()
	q.Set("refresh", "true")
	target.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create analyzer request: %w", err)
	}

	res, err := shared.GetHTTPClient(ctx).Do(req)
	if err != nil {
		return nil, fmt.Errorf("call analyzer: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<15))
		return nil, fmt.Errorf("analyzer returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var response analyzerBundleResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode analyzer response: %w", err)
	}

	manifestBytes, err := extractManifestBytes(response.Manifest)
	if err != nil {
		return nil, err
	}

	if err := ValidateManifestBytes(manifestBytes); err != nil {
		return nil, fmt.Errorf("analyzer manifest failed validation: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.Ports != nil && manifest.Ports.Reserved == nil {
		manifest.Ports.Reserved = []int{}
	}
	applyOnDemandCLIMetadata(&manifest)
	return &manifest, nil
}

func applyOnDemandCLIMetadata(manifest *Manifest) {
	if manifest == nil {
		return
	}
	for i := range manifest.Services {
		svc := &manifest.Services[i]
		if !isCLIService(*svc) {
			continue
		}
		if svc.Metadata == nil {
			svc.Metadata = map[string]interface{}{}
		}
		if _, ok := svc.Metadata["run_mode"]; !ok {
			svc.Metadata["run_mode"] = "on_demand"
		}
		if _, ok := svc.Metadata["skip_reason"]; !ok {
			svc.Metadata["skip_reason"] = "CLI is invoked on demand"
		}
	}
}

func isCLIService(svc ServiceEntry) bool {
	id := strings.ToLower(svc.ID)
	if strings.Contains(id, "cli") {
		return true
	}
	if svc.Build != nil {
		base := strings.ToLower(filepath.Base(svc.Build.SourceDir))
		if base == "cli" {
			return true
		}
	}
	return false
}
