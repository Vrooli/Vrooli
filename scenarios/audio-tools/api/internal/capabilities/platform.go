package capabilities

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ResourcePlatformResolver reads resource.json declarations. It deliberately
// owns no compatibility table: resources remain the platform SSOT.
type ResourcePlatformResolver struct {
	FS   fs.FS
	GOOS string
}

func NewResourcePlatformResolver(fsys fs.FS, goos string) ResourcePlatformResolver {
	if goos == "" {
		goos = runtime.GOOS
	}
	return ResourcePlatformResolver{FS: fsys, GOOS: goos}
}

func (r ResourcePlatformResolver) Resolve(slug string) PlatformVerdict {
	if r.FS == nil || strings.TrimSpace(slug) == "" {
		return PlatformVerdict{Support: PlatformSupported}
	}
	raw, err := fs.ReadFile(r.FS, filepath.ToSlash(filepath.Join(slug, "resource.json")))
	if err != nil {
		return PlatformVerdict{Support: PlatformSupported}
	}
	var doc struct {
		Platforms  map[string]string `json:"platforms"`
		Deployment struct {
			Profiles map[string]map[string]struct {
				Support string `json:"support"`
				Reason  string `json:"reason"`
			} `json:"profiles"`
		} `json:"deployment"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return PlatformVerdict{Support: PlatformSupported}
	}
	key := manifestPlatformKey(r.GOOS)
	if profile, ok := doc.Deployment.Profiles["desktop"]; ok {
		if declaration, ok := profile[key]; ok && strings.EqualFold(declaration.Support, "unsupported") {
			return PlatformVerdict{Support: PlatformUnsupported, Reason: strings.TrimSpace(declaration.Reason)}
		}
		if declaration, ok := profile[key]; ok && strings.EqualFold(declaration.Support, "conditional") {
			return PlatformVerdict{Support: PlatformDegraded, Reason: strings.TrimSpace(declaration.Reason)}
		}
	}
	if strings.EqualFold(doc.Platforms[key], "unsupported") {
		return PlatformVerdict{Support: PlatformUnsupported}
	}
	return PlatformVerdict{Support: PlatformSupported}
}

func manifestPlatformKey(goos string) string {
	switch goos {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

// ResourcesFS resolves the resource tree from lifecycle environment. The
// explicit directory is useful for tests and non-standard deployments.
func ResourcesFS() fs.FS {
	if raw, configured := os.LookupEnv("VROOLI_RESOURCES_DIR"); configured && strings.TrimSpace(raw) != "" {
		root := strings.TrimSpace(raw)
		return os.DirFS(root)
	}
	if raw, configured := os.LookupEnv("VROOLI_SCENARIO_DIR"); configured && strings.TrimSpace(raw) != "" {
		scenarioDir := strings.TrimSpace(raw)
		root := filepath.Join(filepath.Dir(filepath.Dir(filepath.Clean(scenarioDir))), "resources")
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			return os.DirFS(root)
		}
	}
	return nil
}
