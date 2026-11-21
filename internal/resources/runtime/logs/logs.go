package logs

import (
	"path/filepath"
	"strings"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

// CandidatePaths returns plausible managed log locations for a resource.
//
// The first candidate is always the canonical resource logs directory. Manifest
// volume sources that appear to back log targets are appended as additional
// candidates for migration and driver compatibility use.
func CandidatePaths(manifest manifestpkg.ResourceManifest, paths runtimestorage.Paths) []string {
	candidates := []string{}
	if strings.TrimSpace(paths.LogsDir) != "" {
		candidates = append(candidates, filepath.Clean(paths.LogsDir))
	}
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		seen[candidate] = struct{}{}
	}
	for _, volume := range manifest.Runtime.Volumes {
		target := strings.ToLower(strings.TrimSpace(volume.Target))
		if target == "" || !looksLikeLogsTarget(target) {
			continue
		}
		source := filepath.Clean(filepath.FromSlash(strings.TrimSpace(volume.Source)))
		if source == "." || source == "" {
			continue
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		candidates = append(candidates, source)
	}
	return candidates
}

func looksLikeLogsTarget(target string) bool {
	return strings.Contains(target, "/log") || strings.HasSuffix(target, "log") || strings.HasSuffix(target, "logs")
}
