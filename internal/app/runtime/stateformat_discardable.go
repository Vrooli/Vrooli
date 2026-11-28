package runtimeapp

import "github.com/vrooli/vrooli/internal/stateformat"

// These owners persist bytes, but every byte is derived. Calling them
// ContractOnly would be false (they do own files) and converting them to a
// compatibility versioned store would make every unrelated edit to a large,
// actively developed package force an operator transition for data that is
// rebuilt anyway. Discardable says what is true: a reader treats a missing,
// corrupt or unknown-version record as a miss and regenerates, so a rollback
// never has to interpret it.
//
// The claim is reviewed like ContractOnly. The named readers are the evidence:
//   - lifecycle: readSharedPackageStamp returns not-found on a JSON error,
//     install-digest reads compare the digest and ignore a parse failure, and
//     the stamp carries its own version.
//   - host facts: every entry carries hostFactsSchemaVersion, and a reader
//     rejects any other version instead of interpreting it, then re-probes.

// lifecycleCacheStateFormat describes the lifecycle's persisted caches: install
// digests, shared-package stamps, closure caches and locks.
func lifecycleCacheStateFormat() (stateformat.Store, error) {
	return stateformat.DiscardableStore("lifecycle-cache")
}

// hostFactsCacheStateFormat describes the per-project host inventory cache.
func hostFactsCacheStateFormat() (stateformat.Store, error) {
	return stateformat.DiscardableStore("host-facts-cache")
}
