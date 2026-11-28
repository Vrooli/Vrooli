package graph_test

import (
	"testing"

	"github.com/vrooli/api-core/testguard"
)

func TestGraphPackageNoSubstrateImports(t *testing.T) {
	testguard.AssertNoForbiddenReferences(t, ".", map[string]string{
		"time":     "time ownership belongs to orchestration",
		"os":       "filesystem ownership belongs to the sidecar",
		"net/http": "transport belongs in the handler layer",
		"os/exec":  "process execution belongs in the sidecar layer",
	})
}
