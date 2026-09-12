package runtime

import (
	"test-genie/internal/dbexec"
	"test-genie/internal/persistence"
)

// ApplySchema remains the runtime-facing entry point; persistence owns the
// schema authority so offline cutover uses exactly the startup contract.
func ApplySchema(db dbexec.Executor, includeSeed bool) error {
	return persistence.ApplySchema(db, includeSeed)
}

func ensureDatabaseSchema(db dbexec.Executor) error {
	return persistence.ApplySchema(db, true)
}
