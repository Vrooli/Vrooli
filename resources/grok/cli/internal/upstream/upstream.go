// Package upstream wires Grok's release-channel version discovery into the
// shared upstreamcheck verb.
//
// Unlike codex (npm) or opencode (GitHub/npm), Grok Build is not published to a
// package registry: its latest version is exposed as a bare text pointer at
// https://x.ai/cli/<channel> (the same source the official x.ai/cli/install.sh
// probes), with a direct-GCS mirror as fallback. We therefore override the
// default upstreamcheck LatestFetcher with an HTTP GET of that pointer so
// `resource-grok upstream-check` reports real installed-vs-latest drift while
// staying agent-safe (any failure degrades to "unknown", never hard-fails).
package upstream

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	// SourceKind labels the upstream-check report. It is a custom transport
	// (not the built-in github/npm kinds), resolved by the fetcher below.
	SourceKind = upstreamcheck.SourceKind("xai-channel")
	// DefaultChannel is the release channel whose latest version we report.
	DefaultChannel = "stable"

	primaryBase  = "https://x.ai/cli"
	fallbackBase = "https://storage.googleapis.com/grok-build-public-artifacts/cli"
)

// DefaultBases is the channel-pointer source order: Cloudflare-fronted x.ai
// first, direct GCS mirror second.
func DefaultBases() []string { return []string{primaryBase, fallbackBase} }

// Handlers returns upstreamcheck handlers bound to Grok's channel-pointer
// version source. displayName labels the report; pinned mirrors
// resource.json upstream_cli.version_pinned.
func Handlers(displayName, pinned string) *upstreamcheck.Handlers {
	h := upstreamcheck.Default(upstreamcheck.Config{
		DisplayName:   displayName,
		InstalledCmd:  []string{"grok", "--version"},
		PinnedVersion: pinned,
		SourceKind:    SourceKind,
		SourceID:      DefaultChannel,
	})
	client := h.HTTPClient
	bases := DefaultBases()
	h.LatestFetcher = func(ctx context.Context, _ upstreamcheck.SourceKind, id string) (string, error) {
		return FetchLatest(ctx, client, bases, id)
	}
	return h
}

// FetchLatest GETs the channel pointer ("<base>/<channel>") from each base in
// order, returning the first non-empty version string. Exposed for tests.
func FetchLatest(ctx context.Context, client *http.Client, bases []string, channel string) (string, error) {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = DefaultChannel
	}
	if client == nil {
		client = &http.Client{Timeout: tuning.ControlPlaneClientTimeout()}
	}
	var lastErr error
	for _, base := range bases {
		url := strings.TrimRight(base, "/") + "/" + channel
		body, err := upstreamcheck.GetBody(ctx, client, url, "", 1<<16)
		if err != nil {
			lastErr = err
			continue
		}
		if v := strings.TrimSpace(string(body)); v != "" {
			return v, nil
		}
		lastErr = fmt.Errorf("empty version pointer at %s", url)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no channel-pointer sources configured")
	}
	return "", lastErr
}
