package journal

import "vrooli-memory/internal/module"

// Module reserves the journal composition seam before its handlers are added.
func Module() module.Module { return module.Empty("journal") }
