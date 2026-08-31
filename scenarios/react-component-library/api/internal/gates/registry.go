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
	registeredDefinition("graph-reconciled", false, ValidateGraphReconciled, "catalog/assets/**", "library/**"),
	registeredDefinition("dependency-rank", true, ValidateDependencyRank, "catalog/assets/**", "library/**"),
	registeredDefinition("self-hosting", true, ValidateSelfHosting, "catalog/assets/**", "library/**"),
	registeredDefinition("bas-genericity", true, ValidateBASGenericity, "catalog/assets/**", "library/**"),
	registeredDefinition("token-vocabulary", true, ValidateTokenVocabulary, "catalog/config.json", "library/**"),
	registeredDefinition("fallback-parity", false, ValidateFallbackParity, "catalog/config.json", "library/**"),
	registeredDefinition("kit-compatibility", false, ValidateKitCompatibility, "catalog/config.json", "library/**"),
	registeredDefinition("affinity-compatible", false, ValidateAffinityNotBroaderThanCompatibility, "catalog/config.json", "library/**"),
	registeredDefinition("token-ramp-complete", true, ValidateTokenRampComplete, "catalog/assets/**", "library/**"),
	registeredDefinition("scenario-token-requirements", true, ValidateScenarioTokenRequirements, "catalog/config.json", "library/**", "ui/src/**"),
	registeredDefinition("released-version-immutable", false, ValidateReleasedVersionImmutable, "library/released-version-hashes.json", "library/**"),
	registeredDefinition("version-mirror-integrity", false, ValidateVersionMirrorIntegrity, "library/**"),
	registeredDefinition("specifier-shape", false, ValidateSpecifierShape, "library/**", "catalog/config.json"),
	registeredDefinition("release-provenance", true, ValidateReleaseProvenance, "library/release-provenance.json", "library/**"),
	registeredDefinition("version-liveness", true, ValidateVersionLiveness, "library/**"),
	registeredDefinition("types", false, func(scope Scope) (Result, error) {
		return ValidateTypes(scope)
	}, "package.json", "pnpm-lock.yaml", "library/**"),
	registeredDefinition("api", false, ValidateAPI, "catalog/assets/**", "library/**"),
	declarationOnlyDefinition("unit", false, "catalog/config.json", "ui/src/**"),
	declarationOnlyDefinition("interaction", false, "catalog/config.json", "ui/src/**"),
	declarationOnlyDefinition("accessibility", false, "catalog/config.json", "ui/src/**"),
	declarationOnlyDefinition("responsive", false, "catalog/config.json", "ui/src/**"),
	declarationOnlyDefinition("visual", false, "catalog/config.json", "ui/src/**"),
	registeredDefinition("rtl", false, ValidateRTL, "catalog/config.json", "library/**"),
	registeredDefinition("reduced-motion", false, ValidateReducedMotion, "catalog/config.json", "library/**"),
	registeredDefinition("performance", true, ValidatePerformance, "catalog/assets/**", "library/**"),
	registeredDefinition("console-clean", false, ValidateConsoleClean, "catalog/config.json", "library/**"),
	registeredDefinition("surface-discipline", false, ValidateSurfaceDiscipline, "catalog/config.json", "library/**"),
	registeredDefinition("composition", false, ValidateComposition, "catalog/assets/**", "library/**"),
	registeredDefinition("composition-contract", false, ValidateCompositionContract, "catalog/assets/**", "library/**"),
	registeredDefinition("documentation", false, ValidateDocumentation, "catalog/assets/**", "library/**"),
	registeredDefinition("examples", false, ValidateExamples, "catalog/assets/**", "library/**"),
	registeredDefinition("fixture-adversarial", false, ValidateFixtures, "catalog/assets/**", "library/**"),
	registeredDefinition("tokens", true, ValidateTokens, "catalog/config.json", "library/**"),
	registeredDefinition("conformance", true, ValidateConformance, "catalog/assets/**", "library/**"),
	registeredDefinition("lifecycle", false, ValidateLifecycle, "catalog/config.json", "library/**"),
	registeredDefinition("i18n", false, ValidateI18n, "catalog/config.json", "library/**"),
	registeredDefinition("selector-coverage", false, ValidateSelectorCoverage, "catalog/config.json", "library/**"),
	registeredDefinition("restyle-contract", true, ValidateRestyleContract, "catalog/assets/**", "library/**"),
	registeredDefinition("manifest-identity", true, ValidateManifestIdentity, "catalog/assets/**", "library/**"),
	registeredDefinition("manifest-metadata", true, ValidateManifestMetadata, "catalog/assets/**", "library/**"),
	registeredDefinition("overlay-surface-composition", true, ValidateOverlaySurfaceComposition, "catalog/assets/**", "library/**"),
	registeredDefinition("shared-style-ownership", true, ValidateSharedStyleOwnership, "catalog/assets/**", "library/**"),
	registeredDefinition("style-injection", true, ValidateStyleInjection, "catalog/assets/**", "library/**"),
	registeredDefinition("foreign-token-classes", true, ValidateForeignTokenClasses, "catalog/config.json", "library/**"),
	registeredDefinition("utility-class", true, ValidateNoUtilityClasses, "catalog/config.json", "library/**"),
	registeredDefinition("consumer-pin", true, ValidateConsumerPins, "catalog/assets/**", "library/**", "ui/src/**"),
	registeredDefinition("deprecated-import", true, ValidateDeprecatedImports, "catalog/assets/**", "library/**"),
	registeredDefinition("provenance-stamp", true, ValidateProvenanceStamp, "catalog/assets/**", "library/**"),
	registeredDefinition("story-grammar", true, ValidateStoryGrammar, "catalog/assets/**", "library/**"),
	registeredDefinition("story-distinctness", true, ValidateStoryDistinctness, "catalog/assets/**", "library/**"),
	registeredDefinition("evidence-freshness", true, ValidateEvidenceFreshness, "catalog/assets/**", "library/**"),
}

func registeredDefinition(id string, corpus bool, run Runner, inputs ...string) Definition {
	return Definition{ID: id, CorpusScoped: corpus, Run: run, DeterminismInputs: append([]string(nil), inputs...)}
}

func declarationOnlyDefinition(id string, corpus bool, inputs ...string) Definition {
	return Definition{ID: id, CorpusScoped: corpus, DeterminismInputs: append([]string(nil), inputs...)}
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
