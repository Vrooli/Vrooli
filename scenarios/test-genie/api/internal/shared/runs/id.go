// Package runs owns the append-only run index and the canonical run-ID format
// that keys all of test-genie's artifact storage under coverage/runs/<runID>/.
package runs

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NewRunID mints a sortable, collision-resistant run identifier of the form
// YYYYmmdd-HHMMSS-<uuid8>. It is the single source of truth for run IDs; the
// orchestrator and any standalone runners (e.g. UI smoke) must use it so every
// run is addressable in the run index.
func NewRunID() string {
	return fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102-150405"), uuid.NewString()[:8])
}
