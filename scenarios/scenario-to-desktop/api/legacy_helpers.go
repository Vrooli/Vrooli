package main

// legacy_helpers.go contains helper methods for legacy handlers that haven't been
// migrated to domain modules yet. These can be removed once the corresponding
// handlers are migrated to their domain modules.

import (
	"fmt"
	"path/filepath"
)

// telemetryFilePath returns the path to the telemetry file for a scenario.
// Used by: proxy_hints.go
func (s *Server) telemetryFilePath(scenario string) string {
	vrooliRoot := detectVrooliRoot()
	return filepath.Join(vrooliRoot, ".vrooli", "deployment", "telemetry", fmt.Sprintf("%s.jsonl", scenario))
}
