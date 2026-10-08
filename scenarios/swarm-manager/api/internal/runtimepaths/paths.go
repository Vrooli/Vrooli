package runtimepaths

import (
	"os"
	"strings"

	"github.com/vrooli/api-core/storage"
)

const scenarioID = "swarm-manager"

func resolver() (*storage.Resolver, error) {
	return storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
}

// Paths resolves the primary class roots used by the API's durable file
// stores. Request-scoped mutating handlers pair these roots with
// filerouting.RoutedRoots so a Test Genie lease can replace them atomically.
func Paths() (storage.Paths, error) {
	r, err := resolver()
	if err != nil {
		return storage.Paths{}, err
	}
	ns, err := storage.ScenarioNamespace(scenarioID)
	if err != nil {
		return storage.Paths{}, err
	}
	return r.Resolve(storage.Options{ScenarioID: ns})
}

func pathFor(class storage.Class, rel string) (string, error) {
	r, err := resolver()
	if err != nil {
		return "", err
	}
	// Variant-aware path scope (Baseline Modes): a shadow engagement resolves to
	// "swarm-manager_shadow" via the lifecycle-injected VROOLI_STORAGE_NAMESPACE,
	// so shadow state/data/cache never collide with live. Falls back to the bare
	// slug outside the lifecycle.
	// A developer shell can retain another scenario's lifecycle variables. Do
	// not let that stale process context redirect swarm-manager domain data into
	// a different scenario namespace. Legitimate lifecycle roots are the live
	// slug or its variant suffix (for example swarm-manager_shadow).
	ns := scenarioID
	if injected := strings.TrimSpace(os.Getenv("VROOLI_STORAGE_NAMESPACE")); injected != "" && (injected == scenarioID || strings.HasPrefix(injected, scenarioID+"_")) {
		resolved, namespaceErr := storage.ScenarioNamespace(scenarioID)
		if namespaceErr != nil {
			return "", namespaceErr
		}
		ns = resolved
	}
	return r.Path(storage.Options{ScenarioID: ns}, class, rel)
}

func StatePath(rel string) (string, error) {
	return pathFor(storage.ClassState, rel)
}

func DataPath(rel string) (string, error) {
	return pathFor(storage.ClassData, rel)
}

func CachePath(rel string) (string, error) {
	return pathFor(storage.ClassCache, rel)
}
