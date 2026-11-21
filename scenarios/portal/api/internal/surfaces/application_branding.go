package surfaces

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// ApplicationBranding projects applied scenario metadata. It does not create a
// second application registry or consult mutable brand candidates.
type ApplicationBranding struct {
	ScenarioRoot func(string) (string, error)
	ServicePath  func(string) (string, error)
}

type ApplicationMetadata struct {
	Label    string
	IconPath string
}

func DefaultApplicationBranding() *ApplicationBranding {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return nil
	}
	return &ApplicationBranding{
		ScenarioRoot: func(name string) (string, error) { return repocontract.ResolveScenarioPath(root, name) },
		ServicePath:  func(name string) (string, error) { return repocontract.ScenarioServiceManifestPath(root, name) },
	}
}

func (b *ApplicationBranding) Metadata(ctx context.Context, name string) (ApplicationMetadata, error) {
	if err := ctx.Err(); err != nil {
		return ApplicationMetadata{}, err
	}
	if !scenarioName.MatchString(name) {
		return ApplicationMetadata{}, fmt.Errorf("invalid application identity")
	}
	path, err := b.ServicePath(name)
	if err != nil {
		return ApplicationMetadata{}, err
	}
	data, err := boundedRegular(path, 1<<20)
	if err != nil {
		return ApplicationMetadata{}, err
	}
	var declaration struct {
		Service struct {
			DisplayName string `json:"displayName"`
		} `json:"service"`
	}
	if err := json.Unmarshal(data, &declaration); err != nil {
		return ApplicationMetadata{}, err
	}
	label := strings.TrimSpace(declaration.Service.DisplayName)
	if len(label) > 160 {
		label = ""
	}
	result := ApplicationMetadata{Label: label}
	if _, err := b.iconFile(name); err == nil {
		result.IconPath = "/api/v1/applications/" + name + "/icon"
	}
	return result, nil
}

func (b *ApplicationBranding) iconFile(name string) (string, error) {
	if !scenarioName.MatchString(name) {
		return "", fmt.Errorf("invalid application identity")
	}
	root, err := b.ScenarioRoot(name)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	// A template asset is not approved product branding, even when it is a
	// technically valid PNG. Prefer the actual product wordmark in that case.
	if logo, readErr := boundedRegular(filepath.Join(root, "ui", "public", "public", "logo.svg"), 128<<10); readErr == nil && bytes.Contains(bytes.ToLower(logo), []byte("placeholder logo")) {
		return "", fmt.Errorf("application artwork is a placeholder")
	}
	// This is the applied web-public-v1 asset, never a path from catalogue input.
	path := filepath.Join(root, "ui", "public", "public", "icon-192.png")
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(resolved, root+string(filepath.Separator)) {
		return "", fmt.Errorf("artwork escapes application root")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 512<<10 {
		return "", fmt.Errorf("artwork unavailable")
	}
	return resolved, nil
}

// Icon returns only a bounded, decoded PNG with a content identity for caching.
func (b *ApplicationBranding) Icon(ctx context.Context, name string) ([]byte, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	path, err := b.iconFile(name)
	if err != nil {
		return nil, "", err
	}
	data, err := boundedRegular(path, 512<<10)
	if err != nil {
		return nil, "", err
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 1024 || config.Height > 1024 {
		return nil, "", fmt.Errorf("invalid application artwork")
	}
	digest := sha256.Sum256(data)
	return data, hex.EncodeToString(digest[:]), nil
}

func boundedRegular(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("bounded file unavailable")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("bounded file unavailable")
	}
	return data, nil
}
