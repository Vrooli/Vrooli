package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/storage"
)

func resolveSessionStateRoot() string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		log.Printf("session-state: storage resolver failed, using fallback: %v", err)
		return fallbackSessionStateRoot()
	}

	opts := storage.Options{ScenarioID: "web-console"}
	if _, err := storage.EnsureClassDir(resolver, opts, storage.ClassState, 0); err != nil {
		log.Printf("session-state: ensure state dir failed, using fallback: %v", err)
		return fallbackSessionStateRoot()
	}

	path, err := resolver.Path(opts, storage.ClassState, "sessions")
	if err != nil {
		log.Printf("session-state: resolve path failed, using fallback: %v", err)
		return fallbackSessionStateRoot()
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		log.Printf("session-state: create directory failed, using fallback: %v", err)
		return fallbackSessionStateRoot()
	}
	return path
}

func fallbackSessionStateRoot() string {
	exe, _ := os.Executable()
	path := filepath.Join(filepath.Dir(exe), "..", "store", "sessions")
	_ = os.MkdirAll(path, 0o755)
	return path
}
