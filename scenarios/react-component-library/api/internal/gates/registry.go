package gates

import "database/sql"

// Scope identifies the repository and, optionally, the assets a gate must
// inspect. An empty Assets slice means the full corpus.
type Scope struct {
	Root   string
	Assets []string
	DB     *sql.DB
}

func (s Scope) IsFullCorpus() bool { return len(s.Assets) == 0 }

// Runner executes one catalog gate.
type Runner func(Scope) (Result, error)

// RuleSource identifies the declaration layer that imposed a rule.
type RuleSource string

const (
	RuleSourceUniversal RuleSource = "universal"
	RuleSourceKind      RuleSource = "kind"
	RuleSourceAsset     RuleSource = "asset"
	RuleSourceCorpus    RuleSource = "corpus"
)

// Definition is the single executable registration record for a gate.
type Definition struct {
	ID                string
	Run               Runner
	CorpusScoped      bool
	DeterminismInputs []string
}

var registry = []Definition{
	registeredDefinition("graph-reconciled", false, ValidateGraphReconciled),
	registeredDefinition("dependency-rank", true, ValidateDependencyRank),
	registeredDefinition("self-hosting", true, ValidateSelfHosting),
	registeredDefinition("bas-genericity", true, ValidateBASGenericity),
	registeredDefinition("token-vocabulary", true, ValidateTokenVocabulary),
	registeredDefinition("fallback-parity", false, ValidateFallbackParity),
	registeredDefinition("kit-compatibility", false, ValidateKitCompatibility),
	registeredDefinition("affinity-compatible", false, ValidateAffinityNotBroaderThanCompatibility),
	registeredDefinition("token-ramp-complete", true, ValidateTokenRampComplete),
	registeredDefinition("scenario-token-requirements", true, ValidateScenarioTokenRequirements),
	registeredDefinition("released-version-immutable", false, ValidateReleasedVersionImmutable),
	registeredDefinition("release-provenance", true, ValidateReleaseProvenance),
	registeredDefinition("version-liveness", true, ValidateVersionLiveness),
	scopedDefinition("types", false, func(scope Scope) (Result, error) {
		return ValidateTypes(scope)
	}),
	registeredDefinition("api", false, ValidateAPI),
	declarationOnlyDefinition("unit", false),
	declarationOnlyDefinition("interaction", false),
	declarationOnlyDefinition("accessibility", false),
	declarationOnlyDefinition("responsive", false),
	declarationOnlyDefinition("visual", false),
	registeredDefinition("rtl", false, ValidateRTL),
	registeredDefinition("reduced-motion", false, ValidateReducedMotion),
	registeredDefinition("performance", true, ValidatePerformance),
	registeredDefinition("console-clean", false, ValidateConsoleClean),
	registeredDefinition("surface-discipline", false, ValidateSurfaceDiscipline),
	registeredDefinition("composition", false, ValidateComposition),
	registeredDefinition("composition-contract", false, ValidateCompositionContract),
	registeredDefinition("documentation", false, ValidateDocumentation),
	registeredDefinition("examples", false, ValidateExamples),
	registeredDefinition("fixture-adversarial", false, ValidateFixtures),
	registeredDefinition("tokens", true, ValidateTokens),
	registeredDefinition("conformance", true, ValidateConformance),
	registeredDefinition("lifecycle", false, ValidateLifecycle),
	registeredDefinition("i18n", false, ValidateI18n),
	scopedDefinition("selector-coverage", false, ValidateSelectorCoverage),
	registeredDefinition("restyle-contract", true, ValidateRestyleContract),
	registeredDefinition("manifest-identity", true, ValidateManifestIdentity),
	registeredDefinition("manifest-metadata", true, ValidateManifestMetadata),
	registeredDefinition("overlay-surface-composition", true, ValidateOverlaySurfaceComposition),
	registeredDefinition("shared-style-ownership", true, ValidateSharedStyleOwnership),
	registeredDefinition("style-injection", true, ValidateStyleInjection),
	registeredDefinition("foreign-token-classes", true, ValidateForeignTokenClasses),
	registeredDefinition("utility-class", true, ValidateNoUtilityClasses),
	registeredDefinition("consumer-pin", true, ValidateConsumerPins),
	registeredDefinition("deprecated-import", true, ValidateDeprecatedImports),
	registeredDefinition("provenance-stamp", true, ValidateProvenanceStamp),
	registeredDefinition("story-grammar", true, ValidateStoryGrammar),
	registeredDefinition("story-distinctness", true, ValidateStoryDistinctness),
	registeredDefinition("evidence-freshness", true, ValidateEvidenceFreshness),
}

func registeredDefinition(id string, corpus bool, run Runner) Definition {
	return Definition{ID: id, CorpusScoped: corpus, Run: run, DeterminismInputs: []string{"catalog/config.json", "library/**"}}
}

func scopedDefinition(id string, corpus bool, run Runner) Definition {
	return Definition{ID: id, CorpusScoped: corpus, Run: run, DeterminismInputs: []string{"catalog/config.json", "library/**"}}
}

func declarationOnlyDefinition(id string, corpus bool) Definition {
	return Definition{ID: id, CorpusScoped: corpus, DeterminismInputs: []string{"catalog/config.json", "library/**"}}
}

// Definitions returns a copy so callers cannot mutate the executable order.
func Definitions() []Definition {
	return append([]Definition(nil), registry...)
}

// Register adds a gate for tests and controlled extensions.
func Register(definition Definition) {
	registry = append(registry, definition)
}

func Lookup(id string) (Definition, bool) {
	for _, definition := range registry {
		if definition.ID == id {
			return definition, true
		}
	}
	return Definition{}, false
}

// Run executes a registered gate with the caller's scope. Keeping lookup and
// execution together prevents future dispatch code from silently discarding
// the asset set while still allowing declaration-only gates to remain
// explicitly unmeasured.
func Run(id string, scope Scope) (Result, bool, error) {
	definition, ok := Lookup(id)
	if !ok || definition.Run == nil {
		return Result{}, false, nil
	}
	result, err := definition.Run(scope)
	return result, true, err
}
