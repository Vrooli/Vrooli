// Package artifactdelivery implements the node-local half of Bridge artifact
// placement. Device Sync Hub owns the bytes; this package only authenticates
// the target device, streams one directed item, and atomically installs it at
// the signed destination path.
package artifactdelivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxBytes matches the device-sync-hub single-file relay ceiling. The
	// limit is enforced again on the node so a malicious or misconfigured hub
	// cannot make the agent consume unbounded disk space.
	MaxBytes          int64 = 2 << 30
	deviceTokenHeader       = "X-Device-Token" // #nosec G101 -- this is an HTTP header name, not credential material.
)

type Request struct {
	ItemID          string
	Name            string
	DestinationPath string
}

type Config struct {
	BaseURL     string
	DeviceToken string
	WorkDir     string
}

type Result struct {
	Path      string
	SizeBytes int64
	SHA256    string
}

func Deliver(ctx context.Context, client *http.Client, cfg Config, in Request) (Result, error) {
	itemID := strings.TrimSpace(in.ItemID)
	if itemID == "" {
		return Result{}, errors.New("artifact delivery requires an item id")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return Result{}, errors.New("artifact delivery requires a device-sync-hub URL")
	}
	token := strings.TrimSpace(cfg.DeviceToken)
	if token == "" {
		return Result{}, errors.New("artifact delivery requires the target device token")
	}
	destination, err := resolveDestination(cfg.WorkDir, in.DestinationPath)
	if err != nil {
		return Result{}, err
	}

	endpoint := baseURL + "/api/v1/transfer/items/" + url.PathEscape(itemID) + "/content"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Result{}, fmt.Errorf("create artifact download request: %w", err)
	}
	req.Header.Set(deviceTokenHeader, token)
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("download artifact item %q: %w", itemID, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return Result{}, fmt.Errorf("download artifact item %q returned %s: %s", itemID, resp.Status, strings.TrimSpace(string(body)))
	}
	if resp.ContentLength > MaxBytes {
		return Result{}, fmt.Errorf("artifact item %q exceeds the %d-byte node limit", itemID, MaxBytes)
	}

	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return Result{}, fmt.Errorf("create artifact destination directory %q: %w", parent, err)
	}
	tmp, err := os.CreateTemp(parent, ".vrooli-artifact-*")
	if err != nil {
		return Result{}, fmt.Errorf("create temporary artifact in %q: %w", parent, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // best-effort cleanup after rename or failure
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return Result{}, fmt.Errorf("secure temporary artifact: %w", err)
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(resp.Body, MaxBytes+1))
	if copyErr != nil {
		_ = tmp.Close()
		return Result{}, fmt.Errorf("write artifact %q: %w", destination, copyErr)
	}
	if written > MaxBytes {
		_ = tmp.Close()
		return Result{}, fmt.Errorf("artifact item %q exceeds the %d-byte node limit", itemID, MaxBytes)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return Result{}, fmt.Errorf("sync artifact %q: %w", destination, err)
	}
	if err := tmp.Close(); err != nil {
		return Result{}, fmt.Errorf("close artifact %q: %w", destination, err)
	}
	if err := os.Rename(tmpPath, destination); err != nil {
		return Result{}, fmt.Errorf("install artifact at %q: %w", destination, err)
	}
	return Result{Path: destination, SizeBytes: written, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func resolveDestination(workDir, raw string) (string, error) {
	requested := strings.TrimSpace(raw)
	if requested == "" || strings.ContainsRune(requested, 0) {
		return "", errors.New("artifact destination path is required")
	}
	clean := filepath.Clean(requested)
	if clean == "." || clean == string(filepath.Separator) {
		return "", errors.New("artifact destination path must name a file")
	}
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	base := strings.TrimSpace(workDir)
	if base == "" {
		return "", errors.New("relative artifact destination requires the agent work directory")
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve agent work directory: %w", err)
	}
	joined := filepath.Join(base, clean)
	rel, err := filepath.Rel(base, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("relative artifact destination escapes the agent work directory")
	}
	return joined, nil
}
