// Package upstream wires Antigravity's release-manifest version discovery into
// the shared upstreamcheck verb.
//
// Unlike codex (npm) or opencode (GitHub/npm), Antigravity is not published to a
// package registry: its latest version is served as a per-platform JSON manifest
// from the auto-updater service (the same source the official
// antigravity.google/cli/install.sh probes):
//
//	GET <base>/manifests/<platform>.json
//	  -> { "version": "...", "url": "<artifact>", "sha512": "<hex>" }
//
// We therefore override the default upstreamcheck LatestFetcher with an HTTP GET
// of that manifest so `resource-antigravity upstream-check` reports real
// installed-vs-latest drift while staying agent-safe (any failure degrades to
// "unknown", never hard-fails).
package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/vrooli/vrooli/internal/tuning"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/vrooli/cli-core/upstreamcheck"
)

const (
	// SourceKind labels the upstream-check report. It is a custom transport
	// (not the built-in github/npm kinds), resolved by the fetcher below.
	SourceKind = upstreamcheck.SourceKind("antigravity-manifest")

	// ManifestBase is the auto-updater service that serves the per-platform
	// release manifests.
	ManifestBase = "https://antigravity-cli-auto-updater-974169037036.us-central1.run.app"
)

// manifest is the flat per-platform release descriptor.
type manifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA512  string `json:"sha512"`
}

// HostPlatform maps the running GOOS/GOARCH to the manifest platform key
// (e.g. "linux_amd64", "darwin_arm64", "windows_amd64"). Antigravity publishes
// {linux,darwin}_{amd64,arm64} and windows_{amd64,arm64}; arches other than
// amd64/arm64 have no build and resolve to an empty string (fetch then fails
// cleanly, reporting "unknown").
func HostPlatform() string {
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		return ""
	}
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		return runtime.GOOS + "_" + arch
	default:
		return ""
	}
}

// Handlers returns upstreamcheck handlers bound to Antigravity's release-manifest
// version source. displayName labels the report; pinned mirrors
// resource.json upstream_cli.version_pinned.
func Handlers(displayName, pinned string) *upstreamcheck.Handlers {
	platform := HostPlatform()
	h := upstreamcheck.Default(upstreamcheck.Config{
		DisplayName:   displayName,
		InstalledCmd:  []string{"agy", "--version"},
		PinnedVersion: pinned,
		SourceKind:    SourceKind,
		SourceID:      platform,
	})
	client := h.HTTPClient
	h.LatestFetcher = func(ctx context.Context, _ upstreamcheck.SourceKind, id string) (string, error) {
		return FetchLatest(ctx, client, ManifestBase, id)
	}
	return h
}

// FetchLatest GETs the per-platform manifest ("<base>/manifests/<platform>.json")
// and returns its `version` field. Exposed for tests.
func FetchLatest(ctx context.Context, client *http.Client, base, platform string) (string, error) {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		platform = HostPlatform()
	}
	if platform == "" {
		return "", fmt.Errorf("no Antigravity build published for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if client == nil {
		client = &http.Client{Timeout: tuning.ControlPlaneClientTimeout()}
	}
	url := strings.TrimRight(base, "/") + "/manifests/" + platform + ".json"
	m, err := getManifest(ctx, client, url)
	if err != nil {
		return "", err
	}
	if v := strings.TrimSpace(m.Version); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("manifest at %s had no version", url)
}

func getManifest(ctx context.Context, client *http.Client, url string) (manifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return manifest{}, err
	}
	req.Header.Set("User-Agent", "vrooli-upstream-check")
	resp, err := client.Do(req)
	if err != nil {
		return manifest{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return manifest{}, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return manifest{}, err
	}
	var m manifest
	if err := json.Unmarshal(body, &m); err != nil {
		return manifest{}, fmt.Errorf("parse manifest %s: %w", url, err)
	}
	return m, nil
}
