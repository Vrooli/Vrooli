package gates

import (
	"path/filepath"
	"testing"
)

func TestBlockingStaticCalibrationFixturesDiscriminate(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	runners := map[string]GateRunner{
		"fallback-parity":             ValidateFallbackParity,
		"kit-compatibility":           ValidateKitCompatibility,
		"affinity-compatible":         ValidateAffinityNotBroaderThanCompatibility,
		"token-vocabulary":            ValidateTokenVocabulary,
		"token-ramp-complete":         ValidateTokenRampComplete,
		"scenario-token-requirements": ValidateScenarioTokenRequirements,
		"scenario-canonical-layer":    ValidateScenarioCanonicalLayer,
		"released-version-immutable":  ValidateReleasedVersionImmutable,
		"types":                       ValidateTypes,
		"api":                         ValidateAPI,
		"rtl":                         ValidateRTL,
		"reduced-motion":              ValidateReducedMotion,
		"documentation":               ValidateDocumentation,
		"examples":                    ValidateExamples,
		"fixture-adversarial":         ValidateFixtures,
		"tokens":                      ValidateTokens,
		"lifecycle":                   ValidateLifecycle,
		"console-clean":               ValidateConsoleClean,
		"performance":                 ValidatePerformance,
		"restyle-contract":            ValidateRestyleContract,
		"manifest-identity":           ValidateManifestIdentity,
		"manifest-metadata":           ValidateManifestMetadata,
		"overlay-surface-composition": ValidateOverlaySurfaceComposition,
		"shared-style-ownership":      ValidateSharedStyleOwnership,
		"style-injection":             ValidateStyleInjection,
		"foreign-token-classes":       ValidateForeignTokenClasses,
		"utility-class":               ValidateNoUtilityClasses,
		"consumer-pin":                ValidateConsumerPins,
		"deprecated-import":           ValidateDeprecatedImports,
		"provenance-stamp":            ValidateProvenanceStamp,
		"story-distinctness":          ValidateStoryDistinctness,
	}
	for gate, runner := range runners {
		t.Run(gate, func(t *testing.T) {
			report, err := Calibrate(root, gate, runner)
			if err != nil {
				t.Fatal(err)
			}
			if report.NonDiscriminating {
				t.Fatalf("calibration did not discriminate: %+v", report.Results)
			}
		})
	}
}

func TestCalibrationQuarantinesAlwaysPassingRunner(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	report, err := Calibrate(root, "token-vocabulary", func(Scope) (Result, error) {
		return Result{Inspected: 1}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.NonDiscriminating || len(report.Results) != 1 || report.Results[0].Status != "non-discriminating" {
		t.Fatalf("always-passing runner was not quarantined: %+v", report)
	}
}

func TestCalibrationDelegatesExternalRunner(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	report, err := Calibrate(root, "unit", GateRunnerFor("unit"))
	if err != nil {
		t.Fatal(err)
	}
	if report.NonDiscriminating || !report.Delegated || len(report.Results) != 1 || report.Results[0].Status != "delegated" {
		t.Fatalf("external calibration was not delegated: %+v", report)
	}
}
