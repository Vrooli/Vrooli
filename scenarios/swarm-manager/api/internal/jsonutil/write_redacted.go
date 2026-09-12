package jsonutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"swarm-manager/internal/pathredact"
)

// WriteFileRedacted writes a durable JSON artifact after removing
// operator-specific paths. Every on-disk JSON writer should use this boundary.
func WriteFileRedacted(path string, value any) error {
	if redacted, changed, err := pathredact.NewForArtifactPath(path).RedactJSONValue(value); err != nil {
		return fmt.Errorf("redact %s: %w", filepath.Base(path), err)
	} else if changed {
		value = redacted
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}
