package agentaccess

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var chromeExtensionIDPattern = regexp.MustCompile(`^[a-p]{32}$`)

const extensionPermissionProbeChromeGetAllV1 = "chrome.permissions.getAll.v1"

type ScenarioToExtensionArtifactVerifier struct {
	BaseURL string
	Client  *http.Client
}

type scenarioToExtensionArtifactResponse struct {
	ArtifactRef     string   `json:"artifact_ref"`
	ArtifactSHA256  string   `json:"artifact_sha256"`
	ManifestSHA256  string   `json:"manifest_sha256"`
	ExtensionID     string   `json:"extension_id"`
	Version         string   `json:"version"`
	Permissions     []string `json:"permissions"`
	HostPermissions []string `json:"host_permissions"`
	PermissionProbe string   `json:"permission_probe,omitempty"`
}

func NewScenarioToExtensionArtifactVerifier(baseURL string, client *http.Client) (*ScenarioToExtensionArtifactVerifier, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("scenario-to-extension artifact API URL must be an absolute http(s) origin")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return nil, errors.New("scenario-to-extension artifact API URL must not include a path prefix")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	} else {
		copy := *client
		copy.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
		client = &copy
	}
	return &ScenarioToExtensionArtifactVerifier{BaseURL: strings.TrimRight(parsed.String(), "/"), Client: client}, nil
}

func (v *ScenarioToExtensionArtifactVerifier) VerifyBrowserExtensionArtifact(ctx context.Context, artifactRef, expectedDigest string) (ExtensionArtifact, error) {
	if v == nil || v.Client == nil || v.BaseURL == "" {
		return ExtensionArtifact{}, errors.New("scenario-to-extension artifact owner is unavailable")
	}
	if strings.TrimSpace(artifactRef) == "" || !isSHA256Hex(expectedDigest) {
		return ExtensionArtifact{}, errors.New("artifact_ref and exact SHA-256 digest are required")
	}
	endpoint := v.BaseURL + "/api/v1/extension/artifact/" + url.PathEscape(artifactRef) + "/metadata"
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return ExtensionArtifact{}, errors.New("scenario-to-extension artifact URL is invalid")
	}
	query := parsed.Query()
	query.Set("expected_digest", expectedDigest)
	parsed.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return ExtensionArtifact{}, fmt.Errorf("create artifact metadata request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-store")
	response, err := v.Client.Do(request)
	if err != nil {
		return ExtensionArtifact{}, fmt.Errorf("resolve extension artifact metadata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return ExtensionArtifact{}, fmt.Errorf("scenario-to-extension artifact owner returned HTTP %d", response.StatusCode)
	}
	var metadata scenarioToExtensionArtifactResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return ExtensionArtifact{}, fmt.Errorf("decode artifact owner metadata: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ExtensionArtifact{}, errors.New("artifact owner returned trailing metadata")
	}
	if metadata.ArtifactRef != artifactRef || metadata.ArtifactSHA256 != expectedDigest || !isSHA256Hex(metadata.ManifestSHA256) || !chromeExtensionIDPattern.MatchString(metadata.ExtensionID) || strings.TrimSpace(metadata.Version) == "" {
		return ExtensionArtifact{}, errors.New("artifact owner metadata is not bound to the requested build and digest")
	}
	if !uniqueNonblank(metadata.Permissions) || !uniqueNonblank(metadata.HostPermissions) {
		return ExtensionArtifact{}, errors.New("artifact owner returned malformed or duplicate manifest permissions")
	}
	if metadata.PermissionProbe != "" && metadata.PermissionProbe != extensionPermissionProbeChromeGetAllV1 {
		return ExtensionArtifact{}, errors.New("artifact owner returned an unsupported runtime permission source")
	}
	return ExtensionArtifact{
		ArtifactRef: metadata.ArtifactRef, ExtensionID: metadata.ExtensionID,
		Digest: metadata.ArtifactSHA256, ManifestSHA256: metadata.ManifestSHA256,
		Version: metadata.Version, Permissions: metadata.Permissions, HostPermissions: metadata.HostPermissions,
		PermissionProbe: metadata.PermissionProbe,
	}, nil
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func uniqueNonblank(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}
