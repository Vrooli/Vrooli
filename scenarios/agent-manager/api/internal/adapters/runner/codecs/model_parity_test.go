package codecs

import (
	"slices"
	"testing"

	"agent-manager/internal/modelpolicy"
)

// TestModelCatalogCodecConformance gates the policy/runner seam without
// compiling model identifiers into codecs. The catalog declares desired
// static inventory and compatibility; codecs declare measured mechanics and
// dynamic namespaces. Boot's decorator composes the two views.
func TestModelCatalogCodecConformance(t *testing.T) {
	revision, err := modelpolicy.Load(modelpolicy.ResolvePath())
	if err != nil {
		t.Fatalf("load model policy catalog: %v", err)
	}
	catalog := revision.Catalog()

	codecSet := []Codec{
		NewClaudeForTest(),
		NewCodexForTest(),
		NewOpenCodeForTest(),
		NewGrokForTest(),
	}
	for _, codec := range codecSet {
		runnerType := codec.Type()
		t.Run(string(runnerType), func(t *testing.T) {
			inventory, ok := catalog.Runners[runnerType]
			if !ok {
				t.Fatalf("codec runner %q has no catalog inventory", runnerType)
			}
			raw := codec.Capabilities()
			if raw.SupportsRunnerDefault != inventory.SupportsRunnerDefault {
				t.Errorf("SupportsRunnerDefault = %v, catalog = %v", raw.SupportsRunnerDefault, inventory.SupportsRunnerDefault)
			}
			if !slices.Equal(raw.DynamicModelPrefixes, inventory.DynamicModelPrefixes) {
				t.Errorf("DynamicModelPrefixes = %v, catalog = %v", raw.DynamicModelPrefixes, inventory.DynamicModelPrefixes)
			}
			for _, model := range raw.SupportedModels {
				if slices.Contains(catalog.ModelIDs(runnerType), model) {
					t.Errorf("static model %q is duplicated in codec capabilities", model)
				}
			}

			composed := WithCatalogModels(codec, catalog.ModelIDs(runnerType)).Capabilities()
			for _, model := range catalog.ModelIDs(runnerType) {
				if !slices.Contains(composed.SupportedModels, model) {
					t.Errorf("catalog model %q missing from composed capabilities %v", model, composed.SupportedModels)
				}
			}
		})
	}
}

func TestCatalogModelSourceReflectsAtomicReload(t *testing.T) {
	models := []string{"first"}
	codec := WithCatalogModelSource(NewCodexForTest(), func() []string {
		return append([]string(nil), models...)
	})
	if got := codec.Capabilities().SupportedModels; !slices.Equal(got, []string{"first"}) {
		t.Fatalf("initial models = %v", got)
	}

	models = []string{"second"}
	if got := codec.Capabilities().SupportedModels; !slices.Equal(got, []string{"second"}) {
		t.Fatalf("reloaded models = %v", got)
	}
}
