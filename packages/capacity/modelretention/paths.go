package modelretention

import (
	"fmt"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	hostconfig "github.com/vrooli/vrooli/internal/config"
)

// Paths are the per-resource files shared by the activity edge, the owner CLI
// and storage-manager. They are derived from the runtime home so every party
// agrees without configuration.
type Paths struct {
	// UsageFile is durable state: last request per model.
	UsageFile string
	// ReceiptsFile is durable state: one JSON line per eviction.
	ReceiptsFile string
	// FenceDir holds the per-model lock files. Locks are process ownership,
	// so they live beside the resource's other process records.
	FenceDir string
}

// StorePaths resolves the shared files for one model-store resource.
func StorePaths(resource string) (Paths, error) {
	home, err := hostconfig.HomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve runtime home for %s model retention: %w", resource, err)
	}
	return StorePathsAt(home, resource)
}

// StorePathsAt resolves the shared files under an explicit home directory.
func StorePathsAt(home, resource string) (Paths, error) {
	resource = strings.TrimSpace(resource)
	if resource == "" || strings.ContainsAny(resource, `/\`) || resource == "." || resource == ".." {
		return Paths{}, fmt.Errorf("model retention resource name %q is invalid", resource)
	}
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve runtime state directory: %w", err)
	}
	processes, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyProcesses)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve runtime processes directory: %w", err)
	}
	stateDir := filepath.Join(state, "resources", resource)
	return Paths{
		UsageFile:    filepath.Join(stateDir, "model-usage.json"),
		ReceiptsFile: filepath.Join(stateDir, "model-evictions.jsonl"),
		FenceDir:     filepath.Join(processes, "resources", resource, "model-fence"),
	}, nil
}

// defaultRegistry and defaultNamespace are implied when a name omits them.
const (
	defaultRegistry  = "registry.ollama.ai"
	defaultNamespace = "library"
)

// CanonicalName is the identity used for usage, fences, receipts and every
// per-model policy lookup. Model stores accept an untagged name as
// name:latest, match names case-insensitively and accept the default registry
// and namespace written out (registry.ollama.ai/library/x is x), so every
// spelling that reaches the same model must land on one key.
func CanonicalName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}
	if at := strings.Index(name, "@"); at >= 0 {
		name = name[:at]
	}
	name = strings.TrimPrefix(name, defaultRegistry+"/")
	name = strings.TrimPrefix(name, defaultNamespace+"/")
	last := name
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		last = name[slash+1:]
	}
	if last != "" && !strings.Contains(last, ":") {
		name += ":latest"
	}
	return name
}
