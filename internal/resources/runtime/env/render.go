package env

import (
	"path/filepath"
	"strings"

	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

type Renderer struct {
	Root       string
	Home       string
	ResourceID string
	Paths      runtimestorage.Paths
}

func NewRenderer(root, home, resourceID string, paths runtimestorage.Paths) *Renderer {
	return &Renderer{
		Root:       filepath.Clean(root),
		Home:       filepath.Clean(home),
		ResourceID: strings.TrimSpace(resourceID),
		Paths:      paths,
	}
}

func (r *Renderer) RenderValue(input string) string {
	replacements := map[string]string{
		"HOME":                r.Home,
		"ROOT":                r.Root,
		"RESOURCE_ROOT":       filepath.Join(r.Root, "resources", r.ResourceID),
		"RESOURCE_CONFIG_DIR": r.Paths.ConfigDir,
		"RESOURCE_DATA_DIR":   r.Paths.DataDir,
		"RESOURCE_CACHE_DIR":  r.Paths.CacheDir,
		"RESOURCE_LOGS_DIR":   r.Paths.LogsDir,
		"RESOURCE_STATE_DIR":  r.Paths.StateDir,
	}

	out := input
	for key, value := range replacements {
		out = strings.ReplaceAll(out, "${"+key+"}", value)
		out = strings.ReplaceAll(out, "$"+key, value)
	}
	return out
}
