package recall

import "vrooli-memory/internal/module"

// Module reserves the runtime registration seam until the Connect transport
// is wired; the query service itself is independently testable above.
func Module() module.Module { return module.Empty("recall") }
