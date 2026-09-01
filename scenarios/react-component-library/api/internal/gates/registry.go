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
	registeredDefinition("graph-reconciled", true, ValidateGraphReconciled, "catalog/assets/**", "library/**"),
	registeredDefinition("dependency-rank", true, ValidateDependencyRank, "catalog/assets/**", "library/**"),
	registeredDefinition("self-hosting", true, ValidateSelfHosting, "catalog/assets/**", "library/**"),
	registeredDefinition("bas-genericity", true, ValidateBASGenericity, "catalog/assets/**", "library/**"),
	registeredDefinition("token-vocabulary", true, ValidateTokenVocabulary, "catalog/config.json", "library/**"),
	registeredDefinition("fallback-parity", false, ValidateFallbackParity, "catalog/config.json", "library/**"),
	registeredDefinition("kit-compatibility", true, ValidateKitCompatibility, "catalog/config.json", "library/**"),
	registeredDefinition("affinity-compatible", true, ValidateAffinityNotBroaderThanCompatibility, "catalog/config.json", "library/**"),
	registeredDefinition("token-ramp-complete", true, ValidateTokenRampComplete, "catalog/assets/**", "library/**"),
	registeredDefinition("scenario-token-requirements", true, ValidateScenarioTokenRequirements, "catalog/config.json", "library/**", "ui/src/**"),
	registeredDefinition("released-version-immutable", true, ValidateReleasedVersionImmutable, "library/released-version-hashes.json", "library/**"),
	registeredDefinition("version-mirror-integrity", true, ValidateVersionMirrorIntegrity, "library/**"),
	registeredDefinition("specifier-shape", true, ValidateSpecifierShape, "library/**", "catalog/config.json"),
	registeredDefinition("version-shape", true, ValidateVersionShape, "catalog/version-shape.json", "library/**"),
	registeredDefinition("field-ownership", true, ValidateFieldOwnership, "catalog/assets/**", "library/**"),
	registeredDefinition("release-provenance", true, ValidateReleaseProvenance, "library/release-provenance.json", "library/**"),
	registeredDefinition("version-liveness", true, ValidateVersionLiveness, "library/**"),
	registeredDefinition("dist-resolution", true, ValidateDistResolution, "package.json", "dist/**"),
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
	registeredDefinition("console-clean", true, ValidateConsoleClean, "catalog/config.json", "library/**"),
	registeredDefinition("surface-discipline", false, ValidateSurfaceDiscipline, "catalog/config.json", "library/**"),
	registeredDefinition("composition", false, ValidateComposition, "catalog/assets/**", "library/**"),
	registeredDefinition("composition-contract", true, ValidateCompositionContract, "catalog/assets/**", "library/**"),
	registeredDefinition("documentation", false, ValidateDocumentation, "catalog/assets/**", "library/**"),
	registeredDefinition("examples", true, ValidateExamples, "catalog/assets/**", "library/**"),
	registeredDefinition("fixture-adversarial", true, ValidateFixtures, "catalog/assets/**", "library/**"),
	registeredDefinition("tokens", true, ValidateTokens, "catalog/config.json", "library/**"),
	registeredDefinition("conformance", true, ValidateConformance, "catalog/assets/**", "library/**"),
	registeredDefinition("lifecycle", true, ValidateLifecycle, "catalog/config.json", "library/**"),
	registeredDefinition("i18n", false, ValidateI18n, "catalog/config.json", "library/**"),
	registeredDefinition("selector-coverage", false, ValidateSelectorCoverage, "catalog/config.json", "library/**"),
	registeredDefinition("restyle-contract", true, ValidateRestyleContract, "catalog/assets/**", "library/**"),
	registeredDefinition("manifest-identity", true, ValidateManifestIdentity, "catalog/assets/**", "library/**"),
	registeredDefinition("manifest-metadata", true, ValidateManifestMetadata, "catalog/assets/**", "library/**"),
	registeredDefinition("declaration-coverage", true, ValidateDeclarationCoverage, "catalog/assets/**", "library/**"),
	registeredDefinition("overlay-surface-composition", true, ValidateOverlaySurfaceComposition, "catalog/assets/**", "library/**"),
	registeredDefinition("shared-style-ownership", true, ValidateSharedStyleOwnership, "catalog/assets/**", "library/**"),
	registeredDefinition("style-injection", true, ValidateStyleInjection, "catalog/assets/**", "library/**"),
	registeredDefinition("style-ownership", true, ValidateStyleOwnership, "catalog/assets/**", "library/**"),
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
	result, err := RunDefinition(definition, scope)
	return result, true, err
}

// RunDefinition is the single dispatch seam for scope semantics. Corpus gates
// receive an explicit full-corpus scope; asset gates receive the caller's
// selection. This prevents a registry declaration from becoming descriptive
// metadata that the execution path silently ignores.
func RunDefinition(definition Definition, scope Scope) (Result, error) {
	if definition.CorpusScoped {
		scope.Assets = nil
	}
	return definition.Run(scope)
}
