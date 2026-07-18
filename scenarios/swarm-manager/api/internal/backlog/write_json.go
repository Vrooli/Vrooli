package backlog

import (
	"encoding/json"
	"os"

	"swarm-manager/internal/pathredact"
)

func writeJSONRedacted(path string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if redacted, changed := pathredact.NewForArtifactPath(path).RedactBytes(path, encoded); changed {
		encoded = redacted
	}
	return os.WriteFile(path, encoded, 0o600)
}
