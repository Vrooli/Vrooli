package pipeline

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

type ociRegistry struct {
	base       string
	repository string
	client     *http.Client
	username   string
	token      string
}

func newOCIRegistry(raw string) (*ociRegistry, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("PLUGIN_REGISTRY is empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || strings.Trim(u.Path, "/") == "" {
		return nil, fmt.Errorf("PLUGIN_REGISTRY must be a registry host and repository, got %q", raw)
	}
	return &ociRegistry{
		base:       u.Scheme + "://" + u.Host,
		repository: strings.Trim(u.Path, "/"),
		client:     http.DefaultClient,
		username:   strings.TrimSpace(getenv("PLUGIN_REGISTRY_USERNAME")),
		token:      strings.TrimSpace(getenv("PLUGIN_REGISTRY_TOKEN")),
	}, nil
}

func getenv(name string) string {
	// Kept behind a tiny seam so registry tests can provide credentials without
	// changing process-global state.
	return lookupEnv(name)
}

var lookupEnv = func(name string) string {
	return strings.TrimSpace(envValue(name))
}

// envValue is declared in module.go's process package seam through the normal
// environment lookup. Keeping the lookup in one function makes this file easy
// to exercise with an httptest registry.
func envValue(name string) string {
	return os.Getenv(name)
}

func (r *ociRegistry) request(ctx context.Context, method, endpoint string, body []byte, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if r.username != "" || r.token != "" {
		req.SetBasicAuth(r.username, r.token)
	}
	return r.client.Do(req)
}

func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (r *ociRegistry) pushBlob(ctx context.Context, blob []byte) (string, error) {
	digest := digestBytes(blob)
	endpoint := r.base + path.Join("/v2", r.repository, "blobs/uploads/")
	resp, err := r.request(ctx, http.MethodPost, endpoint, nil, "")
	if err != nil {
		return "", err
	}
	location := resp.Header.Get("Location")
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted || location == "" {
		return "", fmt.Errorf("blob upload start returned %s", resp.Status)
	}
	if strings.HasPrefix(location, "/") {
		location = r.base + location
	}
	separator := "?"
	if strings.Contains(location, "?") {
		separator = "&"
	}
	resp, err = r.request(ctx, http.MethodPut, location+separator+"digest="+url.QueryEscape(digest), blob, "application/octet-stream")
	if err != nil {
		return "", err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("blob upload returned %s", resp.Status)
	}
	return digest, nil
}

func (r *ociRegistry) putManifest(ctx context.Context, reference string, manifest []byte) error {
	endpoint := r.base + path.Join("/v2", r.repository, "manifests", reference)
	resp, err := r.request(ctx, http.MethodPut, endpoint, manifest, "application/vnd.oci.image.manifest.v1+json")
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("manifest upload returned %s", resp.Status)
	}
	return nil
}

func (r *ociRegistry) getManifest(ctx context.Context, reference string) ([]byte, error) {
	endpoint := r.base + path.Join("/v2", r.repository, "manifests", reference)
	resp, err := r.request(ctx, http.MethodGet, endpoint, nil, "application/vnd.oci.image.manifest.v1+json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("manifest retrieval returned %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func descriptor(mediaType, digest string, size int64) map[string]any {
	return map[string]any{"mediaType": mediaType, "digest": digest, "size": size}
}

func (r *ociRegistry) pushPackage(ctx context.Context, tag string, artifact, signature, provenance, sbom []byte) (string, error) {
	artifactDigest, err := r.pushBlob(ctx, artifact)
	if err != nil {
		return "", err
	}
	empty := []byte("{}")
	emptyDigest, err := r.pushBlob(ctx, empty)
	if err != nil {
		return "", err
	}
	manifest := map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.manifest.v1+json",
		"artifactType":  "application/vnd.agent-plugin+tar",
		"config":        descriptor("application/vnd.oci.empty.v1+json", emptyDigest, int64(len(empty))),
		"layers":        []any{descriptor("application/vnd.agent-plugin.layer.v1+tar+gzip", artifactDigest, int64(len(artifact)))},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	if err := r.putManifest(ctx, tag, manifestBytes); err != nil {
		return "", err
	}
	confirmed, err := r.getManifest(ctx, tag)
	if err != nil {
		return "", fmt.Errorf("PLG-DIST-CONFIRM: %w", err)
	}
	var fetched map[string]any
	if err := json.Unmarshal(confirmed, &fetched); err != nil {
		return "", fmt.Errorf("PLG-DIST-CONFIRM: invalid retrieved manifest: %w", err)
	}
	layers, ok := fetched["layers"].([]any)
	firstLayer, layerOK := map[string]any(nil), false
	if ok && len(layers) > 0 {
		firstLayer, layerOK = layers[0].(map[string]any)
	}
	if !ok || !layerOK || firstLayer["digest"] != artifactDigest {
		return "", fmt.Errorf("PLG-DIST-CONFIRM: retrieved layer digest does not match uploaded artifact")
	}

	for _, evidence := range []struct {
		kind string
		body []byte
	}{{"application/vnd.dev.cosign.signature", signature}, {"application/vnd.in-toto+json", provenance}, {"application/vnd.cyclonedx+json", sbom}} {
		evidenceDigest, pushErr := r.pushBlob(ctx, evidence.body)
		if pushErr != nil {
			return "", pushErr
		}
		referrer := map[string]any{
			"schemaVersion": 2,
			"mediaType":     "application/vnd.oci.image.manifest.v1+json",
			"artifactType":  evidence.kind,
			"subject":       descriptor("application/vnd.agent-plugin+tar", artifactDigest, int64(len(artifact))),
			"config":        descriptor("application/vnd.oci.empty.v1+json", emptyDigest, int64(len(empty))),
			"layers":        []any{descriptor(evidence.kind, evidenceDigest, int64(len(evidence.body)))},
		}
		refBytes, marshalErr := json.Marshal(referrer)
		if marshalErr != nil {
			return "", marshalErr
		}
		if err := r.putManifest(ctx, digestBytes(refBytes), refBytes); err != nil {
			return "", err
		}
	}
	referrers, err := r.getReferrers(ctx, artifactDigest)
	if err != nil {
		return "", fmt.Errorf("PLG-DIST-CONFIRM: referrer index retrieval failed: %w", err)
	}
	if len(referrers) < 3 {
		return "", fmt.Errorf("PLG-DIST-CONFIRM: retrieved referrer index contains %d evidence manifests, want at least 3", len(referrers))
	}
	return r.base + "/v2/" + r.repository + "/manifests/" + tag, nil
}

func (r *ociRegistry) getReferrers(ctx context.Context, subject string) ([]any, error) {
	endpoint := r.base + path.Join("/v2", r.repository, "referrers", subject)
	resp, err := r.request(ctx, http.MethodGet, endpoint, nil, "application/vnd.oci.image.index.v1+json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("referrer index returned %s", resp.Status)
	}
	var index struct {
		Manifests []any `json:"manifests"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, err
	}
	return index.Manifests, nil
}
