package secrets

import (
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

// requiredSecretValueMissing defines the bundle contract for absent credentials:
// an omitted Required flag defaults to true, and whitespace-only values are empty.
func requiredSecretValueMissing(secret manifest.Secret, value string) bool {
	required := secret.Required == nil || *secret.Required
	return required && strings.TrimSpace(value) == ""
}
