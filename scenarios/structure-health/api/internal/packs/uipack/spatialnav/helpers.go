package spatialnav

import auditrules "structure-health/internal/packs/auditrules"

// Check is the uniform entry point invoked by the structure-health uipack
// orchestrator. It adapts the verbatim CheckSpatialNavProvider rule (which
// accepts content bytes and a file path) to the shared rule signature.
func Check(content string, path string, scenario string) ([]auditrules.Violation, error) {
	return CheckSpatialNavProvider([]byte(content), path), nil
}
