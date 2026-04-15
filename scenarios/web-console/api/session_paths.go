package main

import "github.com/vrooli/api-core/storage"

func resolveSessionStateRoot() string {
	return mustResolveScenarioStorageDir(storage.ClassState, "sessions")
}
