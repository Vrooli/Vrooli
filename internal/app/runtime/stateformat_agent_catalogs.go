package runtimeapp

import (
	"reflect"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/usagemode"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// agentCatalogSchemaDigest pins the stored shapes of the shared cli-core
// caches: the coding-role and live-model catalogs, the context-window records
// and the usage-mode file. A deliberate shape change must bump the store's
// write version and re-pin.
const agentCatalogSchemaDigest = "9073998c6087fdc91a59078f41309521522ec88d8ead11142be9c7e001cbc12d"

// agentCatalogStatePaths are the cli-core caches that live inside the runtime
// home. A harness's own model cache (".codex/models_cache.json") sits in the
// user's home, outside this store.
const (
	agentCatalogContextWindowsPath = "state/vrooli/claude-code/model-context-windows.json"
	agentCatalogUsageModePath      = "state/vrooli/usage-mode/mode.json"
)

// agentCatalogsStateFormat describes the shared cli-core caches as a versioned
// store. Its types live in a separate module, so this descriptor is owned here
// with the rest of the root control plane's persistence declarations.
func agentCatalogsStateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[agentcatalog.CodingRoleCatalog](),
		reflect.TypeFor[agentcatalog.LiveModelCatalog](),
		reflect.TypeFor[agentcatalog.ModelResolution](),
		reflect.TypeFor[harness.ModelWindowRecord](),
		reflect.TypeFor[usagemode.File](),
		reflect.TypeFor[usagemode.LaunchMarker](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "agent-catalogs",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			agentcatalog.CodingRolePolicySchemaVersion,
			agentcatalog.LiveCatalogCurated,
			agentcatalog.ModelDiscoverySchemaVersion,
			agentcatalog.ModelSourceClaudeSubscription,
			agentcatalog.TokenWeightPolicyVersion,
			harness.WindowSourceClaudeStatusline,
			harness.WindowSourceCodexModelsCache,
			usagemode.ArmChallenger,
			usagemode.ArmIncumbent,
			usagemode.ExceptionExplicitModel,
			usagemode.ExceptionFullPower,
			usagemode.KindLearn,
			usagemode.KindSave,
			usagemode.UntilOff,
			usagemode.UntilTime,
			usagemode.UntilWindowReset,
		),
		StatePaths: []string{agentCatalogContextWindowsPath, agentCatalogUsageModePath},
	}, nil
}
