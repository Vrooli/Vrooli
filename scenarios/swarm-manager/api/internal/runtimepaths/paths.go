package runtimepaths

import (
	"github.com/vrooli/api-core/storage"
)

const scenarioID = "swarm-manager"

func resolver() (*storage.Resolver, error) {
	return storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
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
	ns, err := storage.ScenarioNamespace(scenarioID)
	if err != nil {
		return "", err
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
