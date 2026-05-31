package autosteer

import (
	"os"
	"testing"

	"github.com/ecosystem-manager/api/pkg/skillmap"
)

// TestLivePromptManagerCatalog builds the skill→dimension resolver from a real
// prompt-manager. Gated behind ECOSYSTEM_MANAGER_LIVE_PROMPTMANAGER so the
// default suite stays hermetic:
//
//	ECOSYSTEM_MANAGER_LIVE_PROMPTMANAGER=1 \
//	  go test ./pkg/autosteer/ -run TestLivePromptManagerCatalog
func TestLivePromptManagerCatalog(t *testing.T) {
	if os.Getenv("ECOSYSTEM_MANAGER_LIVE_PROMPTMANAGER") == "" {
		t.Skip("set ECOSYSTEM_MANAGER_LIVE_PROMPTMANAGER=1 to run against a live prompt-manager")
	}

	loader := NewPromptLoader(DefaultPromptLoaderConfig())
	if !loader.IsAvailable() {
		t.Skip("prompt-manager not reachable")
	}

	resolver := skillmap.NewResolver(NewPromptLoaderCatalog(loader))

	// The ux skill must declare accessibility/visual/ui (populated in P2).
	dims := resolver.DimensionsForSkill("ux")
	if len(dims) == 0 {
		t.Fatal("ux skill resolved to no dimensions; is the catalog exposing targetDimensions?")
	}
	t.Logf("ux dimensions: %v", dims)

	if skills := resolver.SkillsForDimension("tests"); len(skills) == 0 {
		t.Error("no skill targets the tests dimension; expected the test skill")
	}
}
