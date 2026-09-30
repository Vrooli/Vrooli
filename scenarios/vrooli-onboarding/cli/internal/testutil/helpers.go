// Package testutil contains CLI-only fixtures shared by onboarding tests.
// Production packages must not import this package.
package testutil

import (
	"encoding/json"
	"os"
)

// WriteJSON writes a private JSON fixture without exposing it through a
// production command path.
func WriteJSON(path string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
