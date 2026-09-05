package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	sharedenv "scenario-to-desktop-api/shared/env"
)

// desktopRampBuilder adapts the scenario's existing artifact index to the
// spine Builder seam. The build pipeline remains the owner of electron-builder
// invocation; validation receives only an immutable artifact description.
type desktopRampBuilder struct{ finder validationArtifactFinder }

func (b desktopRampBuilder) Build(_ context.Context, request deliveryramp.BuildRequest) (deliveryramp.Artifact, error) {
	if b.finder == nil {
		return deliveryramp.Artifact{}, fmt.Errorf("desktop artifact builder is unavailable")
	}
	scenario := strings.TrimSpace(request.SourceRef)
	if scenario == "" {
		return deliveryramp.Artifact{}, fmt.Errorf("desktop build source reference is required")
	}
	path, err := b.finder.FindArtifact(scenario)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("resolve built desktop artifact: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("stat built desktop artifact: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("hash built desktop artifact: %w", err)
	}
	digest := sha256.Sum256(data)
	metadata := map[string]string{"source_ref": scenario, "artifact_filename": filepath.Base(path)}
	metadata["platform"] = strings.TrimSpace(os.Getenv("S2D_ARTIFACT_PLATFORM"))
	if metadata["platform"] == "" {
		metadata["platform"] = runtime.GOOS
	}
	metadata["release_version"] = strings.TrimSpace(os.Getenv("S2D_RELEASE_VERSION"))
	root, _ := os.Getwd()
	if configuredRoot := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); configuredRoot != "" {
		root = configuredRoot
	}
	manifestPath := filepath.Join(root, "scenarios", scenario, ".vrooli", "monetization.json")
	if manifestBytes, readErr := os.ReadFile(manifestPath); readErr == nil {
		var manifest struct {
			BundleKey string `json:"bundle_key"`
			AppKey    string `json:"app_key"`
		}
		if json.Unmarshal(manifestBytes, &manifest) == nil {
			metadata["bundle_key"], metadata["app_key"] = strings.TrimSpace(manifest.BundleKey), strings.TrimSpace(manifest.AppKey)
		}
	}
	return deliveryramp.Artifact{
		ImmutableRef: "artifact:" + hex.EncodeToString(digest[:]),
		LocalPath:    path,
		Kind:         "desktop-installer",
		Checksum:     "sha256:" + hex.EncodeToString(digest[:]),
		SizeBytes:    info.Size(),
		CreatedAt:    info.ModTime().UTC(),
		Metadata:     metadata,
	}, nil
}

// desktopRampDistributor registers the immutable artifact with LPBS after the
// build pipeline has produced it. Uploading the bytes is intentionally outside
// this adapter: the operator supplies the durable artifact URL, while this
// call records the exact checksum and entitlement metadata that was built.
type desktopRampDistributor struct{ client *http.Client }

func (d desktopRampDistributor) Distribute(ctx context.Context, request deliveryramp.DistributionRequest) (deliveryramp.DistributionResult, error) {
	if strings.TrimSpace(request.Artifact.ImmutableRef) == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "desktop artifact has no immutable identity"}, nil
	}
	metadata := request.Artifact.Metadata
	bundleKey, appKey := strings.TrimSpace(metadata["bundle_key"]), strings.TrimSpace(metadata["app_key"])
	if bundleKey == "" || appKey == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "monetization artifact metadata is incomplete"}, nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("S2D_LPBS_CATALOG_URL")), "/")
	artifactURL := strings.TrimSpace(metadata["artifact_url"])
	if artifactURL == "" {
		artifactURL = strings.TrimSpace(os.Getenv("S2D_DESKTOP_ARTIFACT_URL"))
	}
	if baseURL == "" || artifactURL == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "LPBS catalog URL and durable artifact URL are required"}, nil
	}
	parsedURL, err := url.Parse(artifactURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "artifact URL must be an absolute HTTP(S) URL"}, nil
	}
	token := strings.TrimSpace(sharedenv.ResolveSecret("LPBS_SERVICE_SECRET"))
	if token == "" {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: "LPBS catalog authorization is unavailable"}, nil
	}
	payload := map[string]any{
		"app_key": appKey,
		"name":    appKey,
		"platforms": []map[string]any{{
			"platform":             metadata["platform"],
			"artifact_url":         artifactURL,
			"artifact_source":      "direct",
			"release_version":      metadata["release_version"],
			"checksum":             request.Artifact.Checksum,
			"requires_entitlement": true,
			"metadata":             map[string]string{"immutable_ref": request.Artifact.ImmutableRef, "bundle_key": bundleKey},
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return deliveryramp.DistributionResult{}, fmt.Errorf("encode LPBS catalog registration: %w", err)
	}
	endpoint := baseURL + "/api/v1/admin/download-apps/" + url.PathEscape(appKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return deliveryramp.DistributionResult{}, fmt.Errorf("create LPBS catalog registration: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := d.client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	response, err := client.Do(req) // #nosec G704 -- endpoint is operator-configured LPBS.
	if err != nil {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: fmt.Sprintf("register LPBS catalog: %v", err)}, nil
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionUnavailable, Reason: fmt.Sprintf("LPBS catalog returned HTTP %d", response.StatusCode)}, nil
	}
	return deliveryramp.DistributionResult{Disposition: deliveryramp.DispositionPass, Targets: []deliveryramp.DistributionTarget{{ID: "lpbs-download-catalog", Kind: "catalog", Available: true}}}, nil
}

var (
	_ deliveryramp.Builder     = desktopRampBuilder{}
	_ deliveryramp.Distributor = desktopRampDistributor{}
)
