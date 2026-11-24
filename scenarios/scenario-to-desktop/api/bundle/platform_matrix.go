package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

// PlatformMatrix is the bundle owner's finite target matrix. It reports the
// declared service artifact path or source-build fallback for each supported
// desktop OS/architecture pair. It does not qualify resource payloads or a
// successful build for targets that were not selected.
type PlatformMatrix struct {
	SchemaVersion string                 `json:"schema_version"`
	Targets       []PlatformMatrixTarget `json:"targets"`
}

type PlatformMatrixTarget struct {
	Key             string                  `json:"key"`
	OS              string                  `json:"os"`
	Architecture    string                  `json:"architecture"`
	Selected        bool                    `json:"selected"`
	Runtime         string                  `json:"runtime"`
	RuntimeArtifact string                  `json:"runtime_artifact,omitempty"`
	Services        []PlatformMatrixService `json:"services"`
}

type PlatformMatrixService struct {
	ID           string `json:"id"`
	Disposition  string `json:"disposition"`
	ArtifactPath string `json:"artifact_path,omitempty"`
	Bundled      bool   `json:"bundled"`
	Reason       string `json:"reason,omitempty"`
}

var desktopPlatformKeys = []string{
	"linux-amd64", "linux-arm64",
	"darwin-amd64", "darwin-arm64",
	"windows-amd64", "windows-arm64",
}

func (p *DefaultPackager) planPlatformMatrix(manifest *bundlemanifest.Manifest, requested []string, manifestRoot string) (PlatformMatrix, []string, error) {
	if manifest.SchemaVersion == "" {
		return PlatformMatrix{}, nil, fmt.Errorf("schema_version is required")
	}
	if manifest.Target != "desktop" {
		return PlatformMatrix{}, nil, fmt.Errorf("unsupported target %q (expected desktop)", manifest.Target)
	}
	if len(manifest.Services) == 0 {
		return PlatformMatrix{}, nil, fmt.Errorf("manifest requires at least one service")
	}
	if err := validateEmbeddedPeerServices(manifest); err != nil {
		return PlatformMatrix{}, nil, err
	}

	requested = append([]string(nil), requested...)
	if len(requested) == 0 {
		requested = collectPlatforms(*manifest)
	}
	if len(requested) == 0 {
		return PlatformMatrix{}, nil, fmt.Errorf("manifest has no platform binaries to package")
	}

	selected := make(map[string]struct{}, len(requested))
	selectedKeys := make([]string, 0, len(requested))
	for _, requestedKey := range requested {
		goos, goarch, err := p.platform.ParseKey(requestedKey)
		if err != nil {
			return PlatformMatrix{}, nil, fmt.Errorf("unsupported desktop target %q: %w", requestedKey, err)
		}
		key := canonicalPlatformKey(goos, goarch)
		if key == "" {
			return PlatformMatrix{}, nil, fmt.Errorf("unsupported desktop target %q (%s/%s)", requestedKey, goos, goarch)
		}
		if _, exists := selected[key]; !exists {
			selected[key] = struct{}{}
			selectedKeys = append(selectedKeys, key)
		}
	}
	sort.Strings(selectedKeys)

	matrix := PlatformMatrix{SchemaVersion: "v1", Targets: make([]PlatformMatrixTarget, 0, len(desktopPlatformKeys))}
	blocked := make([]string, 0)
	for _, key := range desktopPlatformKeys {
		goos, goarch, err := p.platform.ParseKey(key)
		if err != nil {
			// The declared finite matrix is shared by the default resolver. A
			// custom resolver can expose fewer cells; those cells remain explicit.
			matrix.Targets = append(matrix.Targets, PlatformMatrixTarget{
				Key: key, Selected: containsString(selectedKeys, key), Runtime: "unsupported",
			})
			if containsString(selectedKeys, key) {
				blocked = append(blocked, fmt.Sprintf("%s: runtime target is unsupported by this packager", key))
			}
			continue
		}
		row := PlatformMatrixTarget{
			Key: key, OS: goos, Architecture: goarch, Runtime: "build-supported",
			Selected: containsString(selectedKeys, key), Services: make([]PlatformMatrixService, 0, len(manifest.Services)),
		}
		for _, service := range manifest.Services {
			item := p.planServicePlatform(service, key, manifestRoot)
			row.Services = append(row.Services, item)
			if row.Selected && item.Disposition == "unsupported" {
				blocked = append(blocked, fmt.Sprintf("%s: service %s %s", key, item.ID, item.Reason))
			}
		}
		matrix.Targets = append(matrix.Targets, row)
	}
	if len(blocked) > 0 {
		return matrix, nil, fmt.Errorf("selected bundle platform matrix is incomplete: %s", strings.Join(blocked, "; "))
	}
	return matrix, selectedKeys, nil
}

func (p *DefaultPackager) planServicePlatform(service bundlemanifest.Service, platform, manifestRoot string) PlatformMatrixService {
	if isUIService(service.Type) {
		return PlatformMatrixService{ID: service.ID, Disposition: "shared-ui"}
	}
	if binary, ok := p.platform.ResolveBinaryForPlatform(service, platform); ok {
		item := PlatformMatrixService{ID: service.ID, ArtifactPath: binary.Path}
		if manifestRoot == "" {
			item.Disposition = "prebuilt-declared"
			return item
		}
		resolved, err := resolveManifestPath(p.fileOps, manifestRoot, binary.Path)
		if err != nil {
			item.Disposition = "unsupported"
			item.Reason = fmt.Sprintf("binary path %q is invalid: %v", binary.Path, err)
			return item
		}
		if _, err := os.Stat(resolved); err == nil {
			item.Disposition = "prebuilt-available"
			return item
		}
		if service.Build == nil || strings.TrimSpace(service.Build.Type) == "" {
			item.Disposition = "unsupported"
			item.Reason = fmt.Sprintf("prebuilt artifact %q is missing and no build config is declared", binary.Path)
			return item
		}
		item.Disposition = "build-configured"
		item.Reason = "declared artifact is unavailable; source build is configured"
		return item
	}
	if service.Build != nil && strings.TrimSpace(service.Build.Type) != "" {
		return PlatformMatrixService{ID: service.ID, Disposition: "build-configured"}
	}
	return PlatformMatrixService{ID: service.ID, Disposition: "unsupported", Reason: "no matching binary artifact or build config"}
}

func canonicalPlatformKey(goos, goarch string) string {
	if goarch != "amd64" && goarch != "arm64" {
		return ""
	}
	switch goos {
	case "linux", "darwin", "windows":
		return goos + "-" + goarch
	default:
		return ""
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (matrix *PlatformMatrix) markSelectedBundlesComplete(bundleDir string, runtimeBinaries, serviceArtifacts map[string]string) {
	for index := range matrix.Targets {
		target := &matrix.Targets[index]
		if !target.Selected {
			continue
		}
		target.Runtime = "built"
		if path := runtimeBinaries[target.Key]; path != "" {
			target.RuntimeArtifact = relativeBundleArtifact(bundleDir, path)
		}
		for serviceIndex := range target.Services {
			service := &target.Services[serviceIndex]
			if path := serviceArtifacts[serviceArtifactID(service.ID, target.Key)]; path != "" {
				service.Bundled = true
				service.ArtifactPath = relativeBundleArtifact(bundleDir, path)
				if service.Disposition == "build-configured" {
					service.Disposition = "compiled"
				}
			} else if service.Disposition == "shared-ui" {
				// UI services are staged once for the whole bundle, not per target.
				service.Bundled = true
			}
		}
	}
}

func serviceArtifactID(serviceID, platform string) string { return serviceID + "\x00" + platform }

func relativeBundleArtifact(bundleDir, path string) string {
	relative, err := filepath.Rel(bundleDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}

func writePlatformMatrix(bundleDir string, matrix PlatformMatrix) (string, error) {
	payload, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode platform matrix: %w", err)
	}
	path := filepath.Join(bundleDir, "platform-matrix.json")
	if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write platform matrix: %w", err)
	}
	return path, nil
}
