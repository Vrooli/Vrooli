package healthlifecycle

import auditrules "structure-health/internal/packs/auditrules"

// Violation aliases the shared rule-framework type so the migrated rule logic
// compiles unchanged (it referenced the unqualified Violation that config/types.go
// aliased to rules.Violation in scenario-auditor).
type Violation = auditrules.Violation
