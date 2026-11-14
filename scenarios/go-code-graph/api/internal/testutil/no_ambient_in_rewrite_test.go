package testutil_test

import (
	"testing"

	"github.com/vrooli/api-core/testguard"
)

func TestNoAmbientInRewriteDomain(t *testing.T) {
	sourceDir := "../rewrite"
	// Rewrite owns its filesystem executor. Keep ambient clock/config access,
	// HTTP transport and child-process dependencies outside this domain.
	testguard.AssertNoForbiddenReferences(t, sourceDir, map[string]string{
		"net/http":    "transport belongs in the handler layer",
		"os/exec":     "process execution belongs in the sidecar layer",
		"time.Now":    "use the injected clock seam",
		"os.Getenv":   "configuration arrives through constructors",
		"log.Default": "thread the logger into the owner",
	})
}
