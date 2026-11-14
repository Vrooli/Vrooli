package components

import "strings"

// VersionLock is the generated `versions/<version>/dependencies.json` record:
// the exact releases one version was built against. The package lock
// generator (packages/react-component-library/tooling/generate-locks.mjs), the
// publish path, the indexer, and retention all read or write this one shape.
//
// Schema v2 records each dependency's imported major and the release it
// resolved to under `observed`. Schema v1 recorded the release under
// `version`; it is still read so older locks keep their edges, and is never
// written.
type VersionLock struct {
	SchemaVersion int                     `json:"schemaVersion"`
	LibraryID     string                  `json:"libraryId"`
	Version       string                  `json:"version"`
	ResolvedAt    string                  `json:"resolvedAt"`
	Dependencies  []VersionLockDependency `json:"dependencies"`
}

// VersionLockDependency is one pinned dependency of a VersionLock.
type VersionLockDependency struct {
	LibraryID string `json:"libraryId"`
	Version   string `json:"version,omitempty"`
	Major     int    `json:"major"`
	Observed  string `json:"observed"`
	Rank      int    `json:"rank"`
}

// Pin is the exact release this entry pins, whichever schema wrote it. A
// reader that consults only one of the two fields sees no edge at all for
// locks written under the other schema.
func (d VersionLockDependency) Pin() string {
	if observed := strings.TrimSpace(d.Observed); observed != "" {
		return observed
	}
	return strings.TrimSpace(d.Version)
}
