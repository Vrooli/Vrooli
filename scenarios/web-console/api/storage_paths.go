package main

import (
	"log"
	"os"

	"github.com/vrooli/api-core/storage"
)

const webConsoleScenarioID = "web-console"

func mustResolveScenarioStoragePath(class storage.Class, rel string) string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		log.Fatalf("storage: build resolver for class %q: %v", class, err)
	}

	opts := storage.Options{ScenarioID: webConsoleScenarioID}
	if _, err := storage.EnsureClassDir(resolver, opts, class, 0); err != nil {
		log.Fatalf("storage: ensure class dir for %q: %v", class, err)
	}

	path, err := resolver.Path(opts, class, rel)
	if err != nil {
		log.Fatalf("storage: resolve path for class %q rel %q: %v", class, rel, err)
	}

	return path
}

func mustResolveScenarioStorageDir(class storage.Class, rel string) string {
	path := mustResolveScenarioStoragePath(class, rel)
	if err := os.MkdirAll(path, 0o755); err != nil {
		log.Fatalf("storage: create dir for class %q rel %q: %v", class, rel, err)
	}
	return path
}

// scenarioPrimaryPaths resolves web-console's per-class storage roots. These
// are the roots a routed file lease shadows: filerouting.RoutedRoots hands a
// request the leased throwaway tree when the request context carries the
// test-mode marker, and these primary roots otherwise.
func scenarioPrimaryPaths() storage.Paths {
	return storage.Paths{
		ConfigDir: mustResolveScenarioStorageDir(storage.ClassConfig, ""),
		DataDir:   mustResolveScenarioStorageDir(storage.ClassData, ""),
		CacheDir:  mustResolveScenarioStorageDir(storage.ClassCache, ""),
		LogsDir:   mustResolveScenarioStorageDir(storage.ClassLogs, ""),
		StateDir:  mustResolveScenarioStorageDir(storage.ClassState, ""),
	}
}
