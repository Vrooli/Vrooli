package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reporoot"
)

// acceleratedResourceManifest is the smallest shape that answers "does this
// resource declare a non-CPU accelerator backend". It reads the new
// acceleration block and the two deprecated surfaces it replaced, so the
// mode-drift family covers the fleet during the migration rather than after it.
type acceleratedResourceManifest struct {
	Acceleration *struct {
		Backends []string `json:"backends"`
	} `json:"acceleration"`
	GPU          json.RawMessage `json:"gpu"`
	Requirements struct {
		GPU json.RawMessage `json:"gpu"`
	} `json:"requirements"`
}

// declaresAccelerator reports whether the manifest asks for any backend other
// than the CPU.
func (m acceleratedResourceManifest) declaresAccelerator() bool {
	if m.Acceleration != nil {
		for _, backend := range m.Acceleration.Backends {
			if strings.TrimSpace(strings.ToLower(backend)) != "cpu" {
				return true
			}
		}
		// An acceleration block naming only cpu is a deliberate statement that
		// this resource does no accelerated work.
		return false
	}
	return hasJSONObject(m.GPU) || hasJSONObject(m.Requirements.GPU)
}

func hasJSONObject(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

// resourceDeclaresAccelerator reads one resource manifest from the repository.
// A resource whose manifest cannot be read is reported as not accelerated:
// registering a placement check for something that may not exist would produce
// a check that can only ever fail.
func resourceDeclaresAccelerator(root, name string) bool {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(name) == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(root, "resources", name, "resource.json"))
	if err != nil {
		return false
	}
	var manifest acceleratedResourceManifest
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	return manifest.declaresAccelerator()
}

// resourceExists reports whether a monitored name is actually a resource in the
// repository. A monitored name that is not is a stale configuration entry: it
// produces a check that fails forever and heals nothing.
func resourceExists(root, name string) bool {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(name) == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(root, "resources", name, "resource.json"))
	return err == nil && !info.IsDir()
}

// acceleratedResources filters a monitored list down to the resources that
// declare an accelerator, in the order they were given.
func acceleratedResources(names []string) []string {
	root := reporoot.ResolveFromOS()
	if root == "" {
		return nil
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		if resourceDeclaresAccelerator(root, name) {
			out = append(out, name)
		}
	}
	return out
}

// PruneMissingResources drops monitored names that are not resources in this
// repository. It returns the surviving names and the dropped ones, so the
// caller can say what it ignored rather than silently narrowing the fleet.
func PruneMissingResources(names []string) (kept, dropped []string) {
	root := reporoot.ResolveFromOS()
	if root == "" {
		// With no repository to check against, every name is kept: dropping a
		// resource because the root could not be resolved would be worse than
		// monitoring one that turns out to be absent.
		return names, nil
	}
	for _, name := range names {
		if resourceExists(root, name) {
			kept = append(kept, name)
			continue
		}
		dropped = append(dropped, name)
	}
	return kept, dropped
}
