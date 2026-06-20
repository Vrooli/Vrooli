package setupsteps

import (
	"fmt"

	auditrules "structure-health/internal/packs/auditrules"
)

// Violation aliases the shared rule-framework type so the migrated rule logic
// compiles unchanged (it referenced the unqualified Violation that config/types.go
// aliased to rules.Violation in scenario-auditor).
type Violation = auditrules.Violation

// toStringOrDefault is copied verbatim from scenario-auditor config/types.go so the
// rule logic that depends on it preserves identical behavior.
func toStringOrDefault(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
