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
	return r.Path(storage.Options{ScenarioID: scenarioID}, class, rel)
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
