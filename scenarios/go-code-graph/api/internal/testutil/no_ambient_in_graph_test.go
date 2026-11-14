package testutil_test

import (
	"testing"

	"github.com/vrooli/api-core/testguard"
)

func TestNoAmbientInGraphDomain(t *testing.T) {
	sourceDir := "../graph"
	// Graph owns repository reads and its filesystem cache, and exposes
	// time.Duration metrics. Guard the actual ambient operations and foreign
	// transport/process boundaries rather than banning those packages wholesale.
	testguard.AssertNoForbiddenReferences(t, sourceDir, map[string]string{
		"net/http":    "transport belongs in the handler layer",
		"os/exec":     "process execution belongs in the sidecar layer",
		"time.Now":    "use the injected clock seam",
		"os.Getenv":   "configuration arrives through constructors",
		"log.Default": "thread the logger into the owner",
	})
}
