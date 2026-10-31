package heartbeat

import (
	"log"
	"os"
	"strings"
)

// readDirSafe reads directory entries, returning nil on error.
func readDirSafe(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("team_execution_store: failed to read persist dir %s: %v", dir, err)
		}
		return nil, err
	}
	return entries, nil
}

// isTeamQueueFile checks if a filename matches the team queue pattern.
func isTeamQueueFile(name string) bool {
	return strings.HasPrefix(name, "team-queue-") && strings.HasSuffix(name, ".json")
}

// extractTeamID extracts the team ID from a team queue filename.
func extractTeamID(name string) string {
	// "team-queue-{teamID}.json" -> teamID
	trimmed := strings.TrimPrefix(name, "team-queue-")
	trimmed = strings.TrimSuffix(trimmed, ".json")
	return trimmed
}
