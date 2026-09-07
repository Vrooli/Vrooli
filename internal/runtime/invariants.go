package runtime

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/safeguards"
)

// embeddedSafeguardInvariants reads declaration data at the runtime boundary;
// hostreqkit owns evaluation and does not depend on safeguard manifests.
func embeddedSafeguardInvariants(name string) ([]hostreqkit.Invariant, error) {
	data, err := safeguards.Manifests.ReadFile(name + "/safeguard.json")
	if err != nil {
		return nil, fmt.Errorf("read safeguard %q: %w", name, err)
	}
	return hostreqkit.DecodeInvariantDeclarations(data)
}
