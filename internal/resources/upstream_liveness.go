package resources

// This file owns the read-only, scheduled liveness check for resource
// acquisition references. It is intentionally separate from manifest
// validation: an unreachable upstream is an operator finding, not a commit
// failure.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

type UpstreamLivenessFinding struct {
	Resource      string            `json:"resource"`
	Target        int               `json:"target"`
	Kind          string            `json:"kind"`
	Reference     string            `json:"reference"`
	Predicate     map[string]string `json:"predicate,omitempty"`
	CheckedAt     time.Time         `json:"checked_at"`
	FirstFailedAt time.Time         `json:"first_failed_at,omitempty"`
	Status        int               `json:"status,omitempty"`
	Reachable     bool              `json:"reachable"`
	Stale         bool              `json:"stale"`
	Note          string            `json:"note,omitempty"`
}

type UpstreamLivenessReport struct {
	CheckedAt time.Time                 `json:"checked_at"`
	Findings  []UpstreamLivenessFinding `json:"findings"`
}

type livenessState struct {
	Failures      int       `json:"failures"`
	FirstFailedAt time.Time `json:"first_failed_at"`
}

// CheckUpstream walks all resources (or one named resource), checks each
// network-backed acquisition target, and persists consecutive-failure state.
// It returns findings for every checked target; Stale becomes true only after
// two consecutive failed observations.
func CheckUpstream(ctx context.Context, root, statePath, name string, client *http.Client, now time.Time) (UpstreamLivenessReport, error) {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	state := map[string]livenessState{}
	if data, err := os.ReadFile(statePath); err == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return UpstreamLivenessReport{}, fmt.Errorf("decode upstream liveness state: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return UpstreamLivenessReport{}, fmt.Errorf("read upstream liveness state: %w", err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "resources"))
	if err != nil {
		return UpstreamLivenessReport{}, err
	}
	report := UpstreamLivenessReport{CheckedAt: now.UTC(), Findings: []UpstreamLivenessFinding{}}
	for _, entry := range entries {
		if !entry.IsDir() || (name != "" && entry.Name() != name) {
			continue
		}
		manifestPath := filepath.Join(root, "resources", entry.Name(), "resource.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			report.Findings = append(report.Findings, UpstreamLivenessFinding{
				Resource: entry.Name(), Kind: "manifest", Reference: manifestPath,
				CheckedAt: now.UTC(), Reachable: false,
				Note: fmt.Sprintf("resource manifest unavailable: %s", err),
			})
			continue
		}
		var manifest manifestpkg.ResourceManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			report.Findings = append(report.Findings, UpstreamLivenessFinding{
				Resource: entry.Name(), Kind: "manifest", Reference: manifestPath,
				CheckedAt: now.UTC(), Reachable: false,
				Note: fmt.Sprintf("resource manifest invalid: %s", err),
			})
			continue
		}
		acquisitions := []struct{ source *binaryfetch.Acquisition }{{manifest.Acquisition}}
		if manifest.ManagedService != nil {
			acquisitions = append(acquisitions, struct{ source *binaryfetch.Acquisition }{manifest.ManagedService.Acquisition})
		}
		for _, item := range acquisitions {
			if item.source == nil {
				continue
			}
			for index, target := range item.source.Targets {
				kind := item.source.EffectiveKind(target)
				reference := target.URL
				if kind == "oci-image" {
					reference = target.Image
				}
				if kind == "npm" {
					reference = target.Package + "@" + target.Version
				}
				if kind == "none" || kind == "composed" || strings.TrimSpace(reference) == "" {
					continue
				}
				key := manifest.Name + ":" + fmt.Sprint(index) + ":" + kind + ":" + reference
				status, checkErr := checkReference(ctx, client, kind, reference)
				finding := UpstreamLivenessFinding{Resource: manifest.Name, Target: index, Kind: kind, Reference: reference, Predicate: target.When, CheckedAt: now.UTC(), Status: status, Reachable: checkErr == nil}
				if checkErr == nil {
					delete(state, key)
				} else {
					prior := state[key]
					prior.Failures++
					if prior.Failures == 1 {
						prior.FirstFailedAt = now.UTC()
					}
					state[key] = prior
					finding.FirstFailedAt = prior.FirstFailedAt
					finding.Note = checkErr.Error()
					finding.Stale = prior.Failures >= 2
				}
				report.Findings = append(report.Findings, finding)
			}
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Resource != report.Findings[j].Resource {
			return report.Findings[i].Resource < report.Findings[j].Resource
		}
		return report.Findings[i].Target < report.Findings[j].Target
	})
	if err := os.MkdirAll(filepath.Dir(statePath), 0o750); err != nil {
		return UpstreamLivenessReport{}, err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return UpstreamLivenessReport{}, err
	}
	if err := os.WriteFile(statePath, append(data, '\n'), 0o600); err != nil {
		return UpstreamLivenessReport{}, err
	}
	return report, nil
}

func checkReference(ctx context.Context, client *http.Client, kind, reference string) (int, error) {
	var endpoint string
	switch kind {
	case "url":
		endpoint = reference
	case "oci-image":
		return checkOCIReference(ctx, client, reference)
	case "npm":
		endpoint = npmVersionURL(reference)
	default:
		return 0, fmt.Errorf("unsupported liveness kind %q", kind)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, endpoint, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

func checkOCIReference(ctx context.Context, client *http.Client, reference string) (int, error) {
	endpoint := ociManifestURL(reference)
	status, header, err := probeUpstreamHTTP(ctx, client, endpoint, "")
	if err == nil || status != http.StatusUnauthorized {
		return status, err
	}
	challenge := header.Get("WWW-Authenticate")
	realm, service, scope := bearerChallenge(challenge)
	if realm == "" {
		return status, err
	}
	tokenURL, parseErr := url.Parse(realm)
	if parseErr != nil {
		return status, fmt.Errorf("parse OCI auth realm: %w", parseErr)
	}
	query := tokenURL.Query()
	if service != "" {
		query.Set("service", service)
	}
	if scope != "" {
		query.Set("scope", scope)
	}
	tokenURL.RawQuery = query.Encode()
	tokenReq, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if reqErr != nil {
		return status, reqErr
	}
	tokenResp, doErr := client.Do(tokenReq)
	if doErr != nil {
		return status, doErr
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode < 200 || tokenResp.StatusCode >= 400 {
		return tokenResp.StatusCode, fmt.Errorf("OCI auth returned HTTP %d", tokenResp.StatusCode)
	}
	var tokenPayload struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if decodeErr := json.NewDecoder(tokenResp.Body).Decode(&tokenPayload); decodeErr != nil {
		return tokenResp.StatusCode, fmt.Errorf("decode OCI auth token: %w", decodeErr)
	}
	token := tokenPayload.Token
	if token == "" {
		token = tokenPayload.AccessToken
	}
	if token == "" {
		return tokenResp.StatusCode, fmt.Errorf("OCI auth response did not contain a token")
	}
	status, _, err = probeUpstreamHTTP(ctx, client, endpoint, "Bearer "+token)
	return status, err
}

func probeUpstreamHTTP(ctx context.Context, client *http.Client, endpoint, authorization string) (int, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, endpoint, nil)
	if err != nil {
		return 0, nil, err
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	// Registries may store a multi-platform OCI index at a digest. Without
	// explicitly accepting indexes, GHCR reports MANIFEST_UNKNOWN even when
	// the pinned digest exists.
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.docker.distribution.manifest.v2+json",
	}, ", "))
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return resp.StatusCode, resp.Header, fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
	}
	return resp.StatusCode, resp.Header, nil
}

var bearerChallengePart = regexp.MustCompile(`(?i)(?:bearer\s+|,)\s*(realm|service|scope)="([^"]*)"`)

func bearerChallenge(challenge string) (realm, service, scope string) {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(challenge)), "bearer") {
		return "", "", ""
	}
	for _, match := range bearerChallengePart.FindAllStringSubmatch(challenge, -1) {
		switch strings.ToLower(match[1]) {
		case "realm":
			realm = match[2]
		case "service":
			service = match[2]
		case "scope":
			scope = match[2]
		}
	}
	return realm, service, scope
}

func ociManifestURL(reference string) string {
	if !strings.Contains(reference, "://") {
		if strings.HasPrefix(reference, "library/") || !strings.Contains(strings.SplitN(reference, "/", 2)[0], ".") {
			reference = "https://registry-1.docker.io/" + reference
		} else {
			reference = "https://" + reference
		}
	}
	u, err := url.Parse(reference)
	if err != nil || u.Host == "" {
		return reference
	}
	if u.Host == "docker.io" || u.Host == "index.docker.io" {
		u.Host = "registry-1.docker.io"
	}
	name, digest, ok := strings.Cut(u.Path, "@sha256:")
	if !ok {
		return reference
	}
	u.Path = "/v2" + strings.TrimSuffix(name, "/") + "/manifests/sha256:" + digest
	return u.String()
}

func npmVersionURL(reference string) string {
	name, version := reference, ""
	separator := strings.LastIndex(reference, "@")
	if separator > 0 {
		name, version = reference[:separator], reference[separator+1:]
	}
	ok := version != ""
	if !ok || version == "" {
		return "https://registry.npmjs.org/" + url.PathEscape(name)
	}
	return "https://registry.npmjs.org/" + url.PathEscape(name) + "/" + url.PathEscape(version)
}
