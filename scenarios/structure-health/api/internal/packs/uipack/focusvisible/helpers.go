package focusvisible

import auditrules "structure-health/internal/packs/auditrules"

// Check is the uniform entry point invoked by the structure-health uipack
// orchestrator. It adapts the verbatim CheckFocusVisibleStyles rule (which
// accepts content bytes and a file path) to the shared rule signature.
func Check(content string, path string, scenario string) ([]auditrules.Violation, error) {
	return CheckFocusVisibleStyles([]byte(content), path), nil
}
