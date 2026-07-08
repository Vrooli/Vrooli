package assessment

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func validSpec() Spec {
	return Spec{
		Provider: "measures-health",
		Phase:    "measures",
		Version:  "1",
		Levels: []Level{
			{ID: "L0", Name: "No contract"},
			{ID: "L1", Name: "Domains derived"},
			{ID: "L2", Name: "Domains covered"},
			{ID: "L3", Name: "Measures tiered"},
		},
		Findings: map[string]FindingMapping{
			"measures.uncovered-domain": {
				LocalLevelImpact:    "L2",
				GlobalImpact:        ImpactCapabilityGap,
				Dimension:           "measures",
				SeverityDefault:     "ERROR",
				RecommendedSkillIDs: []string{"measures-adoption"},
			},
		},
		Fallback: FallbackPolicy{
			LocalLevelImpact: "L1",
			GlobalImpact:     ImpactUnknown,
			Dimension:        "measures",
			SeverityDefault:  "WARNING",
		},
	}
}

func validMultiCapabilitySpec() Spec {
	return Spec{
		Provider: "ui-health",
		Phase:    "ui-health",
		Version:  "1",
		Capabilities: []CapabilitySpec{
			{
				ID:          "interop",
				Label:       "Interop",
				Description: "Embedding and deployment-context correctness.",
				Levels: []Level{
					{ID: "L0", Name: "Missing", StatusLabel: "Unavailable", CapabilitySummary: "Cannot embed reliably."},
					{ID: "L1", Name: "Base", StatusLabel: "Foundation", CapabilitySummary: "Basic UI contract exists.", NextUnlock: "Proxy-safe routes and assets."},
					{ID: "L2", Name: "Proxy-safe", StatusLabel: "Ready", CapabilitySummary: "Routes and assets work under proxy.", NextUnlock: "Maximum maturity reached."},
				},
			},
			{
				ID:          "pwa_native_readiness",
				Label:       "PWA Native Readiness",
				Description: "Installability, launch, and offline-native web behavior.",
				Levels: []Level{
					{ID: "L0", Name: "Absent", StatusLabel: "Unavailable", CapabilitySummary: "Install surface is absent."},
					{ID: "L1", Name: "Install metadata", StatusLabel: "Basic", CapabilitySummary: "Manifest and launch colors are present.", NextUnlock: "Offline-safe launch and app-shell fallback."},
					{ID: "L2", Name: "Offline-ready", StatusLabel: "Ready", CapabilitySummary: "App shell is reload-safe offline."},
				},
			},
		},
		Findings: map[string]FindingMapping{
			"interop.proxy_base_missing": {
				CapabilityID:     "interop",
				LocalLevelImpact: "L2",
				GlobalImpact:     ImpactHardeningGap,
				Dimension:        "ui",
				SeverityDefault:  "ERROR",
				CleanRequirement: string(CleanRequirementRequired),
			},
			"pwa.service_worker_missing": {
				CapabilityID:     "pwa_native_readiness",
				LocalLevelImpact: "L1",
				GlobalImpact:     ImpactCapabilityGap,
				Dimension:        "ui",
				SeverityDefault:  "WARNING",
				CleanRequirement: string(CleanRequirementRequired),
			},
		},
		Fallback: FallbackPolicy{
			LocalLevelImpact: "L1",
			GlobalImpact:     ImpactUnknown,
			Dimension:        "ui",
			SeverityDefault:  "WARNING",
			CleanRequirement: string(CleanRequirementAdvisory),
		},
	}
}

func TestRepositoryProviderSpecsValidateCleanRequirementContract(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	specPaths, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "test-genie.json"))
	if err != nil {
		t.Fatalf("glob test-genie descriptors: %v", err)
	}
	if len(specPaths) == 0 {
		t.Fatal("no scenario test-genie descriptors found")
	}

	for _, specPath := range specPaths {
		specPath := specPath
		scenario := filepath.Base(filepath.Dir(filepath.Dir(specPath)))
		t.Run(scenario, func(t *testing.T) {
			specPtr, err := LoadSpecFromScenario(filepath.Dir(filepath.Dir(specPath)))
			if err != nil {
				t.Fatalf("load %s: %v", specPath, err)
			}
			spec := *specPtr
			if len(spec.Capabilities) > 0 {
				for code, mapping := range spec.Findings {
					if strings.TrimSpace(mapping.CapabilityID) == "" {
						t.Fatalf("%s multi-capability finding %q must declare capability_id", spec.Provider, code)
					}
				}
				if strings.TrimSpace(spec.Fallback.CapabilityID) == "" {
					t.Fatalf("%s multi-capability fallback must declare capability_id", spec.Provider)
				}
			}

			counts := map[CleanRequirement]int{}
			explicit := 0
			for _, mapping := range spec.Findings {
				requirement := normalizeCleanRequirement(mapping.CleanRequirement)
				counts[requirement]++
				if mapping.CleanRequirement != "" {
					explicit++
					if !IsValidCleanRequirement(requirement) {
						t.Fatalf("%s clean_requirement %q normalized to invalid value %q", spec.Provider, mapping.CleanRequirement, requirement)
					}
				}
			}
			if len(spec.Findings) > 0 && counts[CleanRequirementAdvisory]+counts[CleanRequirementRequired]+counts[CleanRequirementUncheckable] != len(spec.Findings) {
				t.Fatalf("%s clean requirement counts = %#v, want one normalized classification per finding", spec.Provider, counts)
			}
			if explicit > 0 && counts[CleanRequirementAdvisory]+counts[CleanRequirementRequired]+counts[CleanRequirementUncheckable] == 0 {
				t.Fatalf("%s declared clean requirements but none normalized: %#v", spec.Provider, counts)
			}
		})
	}
}

func TestCleanRequirementDefaultsToAdvisory(t *testing.T) {
	spec := validSpec()
	mapping := spec.Findings["measures.uncovered-domain"]
	mapping.CleanRequirement = ""
	spec.Findings["measures.uncovered-domain"] = mapping
	assessment, err := BuildProtoAssessment(BuildInput{
		Spec:     spec,
		Scenario: "demo",
		Findings: []Finding{
			{Code: "measures.uncovered-domain", Severity: "WARNING"},
		},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	if got := assessment.GetFindingsByCleanRequirement()[string(CleanRequirementAdvisory)]; got != 1 {
		t.Fatalf("advisory clean requirement count = %d, want 1", got)
	}
	if got := assessment.GetFindings()[0].GetMaturity().GetCleanRequirement(); got != commonv1.CleanRequirement_CLEAN_REQUIREMENT_ADVISORY {
		t.Fatalf("finding clean requirement = %v, want advisory", got)
	}
}

func TestValidateSpecAllowsExplicitCleanRequirementsForAnyProvider(t *testing.T) {
	spec := validSpec()
	for name, requirement := range map[string]string{
		"required":    string(CleanRequirementRequired),
		"advisory":    string(CleanRequirementAdvisory),
		"uncheckable": string(CleanRequirementUncheckable),
	} {
		t.Run(name, func(t *testing.T) {
			spec := spec
			spec.Provider = "structure-health"
			mapping := spec.Findings["measures.uncovered-domain"]
			mapping.CleanRequirement = requirement
			spec.Findings = map[string]FindingMapping{"structure.demo": mapping}
			if err := ValidateSpec(spec); err != nil {
				t.Fatalf("ValidateSpec rejected %s for non-proto provider: %v", requirement, err)
			}
		})
	}
}

func TestValidateSpecRejectsUnknownCleanRequirement(t *testing.T) {
	spec := validSpec()
	mapping := spec.Findings["measures.uncovered-domain"]
	mapping.CleanRequirement = "sometimes"
	spec.Findings["measures.uncovered-domain"] = mapping
	if err := ValidateSpec(spec); err == nil {
		t.Fatal("ValidateSpec succeeded with unknown clean_requirement")
	}
}

func TestValidateSpecRejectsUnknownDimension(t *testing.T) {
	spec := validSpec()
	spec.Findings["bad.dimension"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     ImpactEvolvabilityGap,
		Dimension:        "not-a-dimension",
	}
	if err := ValidateSpec(spec); err == nil {
		t.Fatal("ValidateSpec succeeded with unknown dimension")
	}
}

func TestValidateSpecRejectsUnknownImpact(t *testing.T) {
	spec := validSpec()
	spec.Findings["bad.impact"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     "not_an_impact",
		Dimension:        "measures",
	}
	if err := ValidateSpec(spec); err == nil {
		t.Fatal("ValidateSpec succeeded with unknown global impact")
	}
}

func TestValidateSpecAllowsDuplicateLevelIDsAcrossCapabilities(t *testing.T) {
	spec := validMultiCapabilitySpec()
	if err := ValidateSpec(spec); err != nil {
		t.Fatalf("ValidateSpec returned error: %v", err)
	}
	spec.Findings["bad.capability"] = FindingMapping{
		CapabilityID:     "missing",
		LocalLevelImpact: "L1",
		GlobalImpact:     ImpactHardeningGap,
		Dimension:        "ui",
		CleanRequirement: string(CleanRequirementRequired),
	}
	if err := ValidateSpec(spec); err == nil {
		t.Fatal("ValidateSpec succeeded with unknown capability_id")
	}
}

func TestValidateSpecRequiresCleanRequirementForCapabilitySpecs(t *testing.T) {
	spec := validMultiCapabilitySpec()
	mapping := spec.Findings["pwa.service_worker_missing"]
	mapping.CleanRequirement = ""
	spec.Findings["pwa.service_worker_missing"] = mapping
	if err := ValidateSpec(spec); err == nil {
		t.Fatal("ValidateSpec succeeded with missing clean_requirement in capability spec")
	}
}

func TestEveryGlobalImpactHasRungSemanticMapping(t *testing.T) {
	for impact := range validImpacts {
		dims := DimensionsForImpact(impact)
		switch impact {
		case ImpactAdvisory, ImpactUnknown:
			if len(dims) != 0 {
				t.Fatalf("%s dimensions = %#v, want none", impact, dims)
			}
		default:
			if len(dims) == 0 {
				t.Fatalf("%s has no mapped dimensions", impact)
			}
		}
	}
}

func TestLocalMaturityUsesLowestBlockingLevel(t *testing.T) {
	spec := validSpec()
	got := LocalMaturity(spec, []Finding{
		{Code: "measures.uncovered-domain", Severity: "SEVERITY_ERROR"},
	})
	if got.CurrentLevel != "L1" {
		t.Fatalf("CurrentLevel = %q, want L1", got.CurrentLevel)
	}
	if got.NextLevel != "L2" {
		t.Fatalf("NextLevel = %q, want L2", got.NextLevel)
	}
	if len(got.BlockingFindingCodes) != 1 || got.BlockingFindingCodes[0] != "measures.uncovered-domain" {
		t.Fatalf("BlockingFindingCodes = %#v", got.BlockingFindingCodes)
	}
}

func TestLocalMaturityReportsFloorForL0Blocker(t *testing.T) {
	spec := validSpec()
	spec.Findings["manifest.required"] = FindingMapping{
		LocalLevelImpact: "L0",
		GlobalImpact:     ImpactFoundationBlocker,
		Dimension:        "contracts",
		SeverityDefault:  "SEVERITY_ERROR",
		CleanRequirement: string(CleanRequirementRequired),
	}
	got := LocalMaturity(spec, []Finding{{Code: "manifest.required"}})
	if got.CurrentLevel != "L0" {
		t.Fatalf("CurrentLevel = %q, want L0", got.CurrentLevel)
	}
	if got.NextLevel != "L1" {
		t.Fatalf("NextLevel = %q, want L1", got.NextLevel)
	}
}

func TestNormalizeFindingFallsBackBySourceAndDefaultSeverity(t *testing.T) {
	spec := validSpec()
	spec.Fallback.Dimension = ""
	item := NormalizeFinding(spec, Finding{
		Code:   "unmapped",
		Source: architecturev1.FindingSource_FINDING_SOURCE_SECURITY,
	})
	if item.Mapping.Dimension != "security" {
		t.Fatalf("dimension = %q, want security", item.Mapping.Dimension)
	}
	if item.Severity != architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
		t.Fatalf("severity = %v, want WARNING", item.Severity)
	}
}

func TestAdvisoryWarningDoesNotBlockLocalMaturity(t *testing.T) {
	spec := validSpec()
	spec.Findings["measures.context"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     ImpactAdvisory,
		Dimension:        "measures",
		SeverityDefault:  "WARNING",
	}
	got := LocalMaturity(spec, []Finding{{Code: "measures.context"}})
	if got.CurrentLevel != "L3" {
		t.Fatalf("CurrentLevel = %q, want L3", got.CurrentLevel)
	}
	if got.NextLevel != "" {
		t.Fatalf("NextLevel = %q, want empty", got.NextLevel)
	}
}

func TestAdvisoryErrorStillBlocksLocalMaturity(t *testing.T) {
	spec := validSpec()
	spec.Findings["measures.context"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     ImpactAdvisory,
		Dimension:        "measures",
		SeverityDefault:  "ERROR",
	}
	got := LocalMaturity(spec, []Finding{{Code: "measures.context"}})
	if got.CurrentLevel != "L0" {
		t.Fatalf("CurrentLevel = %q, want L0", got.CurrentLevel)
	}
	if got.NextLevel != "L1" {
		t.Fatalf("NextLevel = %q, want L1", got.NextLevel)
	}
}

func TestRequiredWarningBlocksLocalMaturityAndClean(t *testing.T) {
	spec := validSpec()
	spec.Findings["measures.required-context"] = FindingMapping{
		LocalLevelImpact: "L3",
		GlobalImpact:     ImpactAdvisory,
		Dimension:        "measures",
		SeverityDefault:  "WARNING",
		CleanRequirement: string(CleanRequirementRequired),
	}
	got := LocalMaturity(spec, []Finding{{Code: "measures.required-context"}})
	if got.CurrentLevel != "L2" {
		t.Fatalf("CurrentLevel = %q, want L2", got.CurrentLevel)
	}
	if got.NextLevel != "L3" {
		t.Fatalf("NextLevel = %q, want L3", got.NextLevel)
	}
	if got.Clean {
		t.Fatal("Clean = true, want false while REQUIRED findings remain")
	}
}

func TestUncheckableFindingIsUnknownAndDoesNotBlockOrCountAsDebt(t *testing.T) {
	spec := validSpec()
	spec.Findings["measures.uncheckable"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     ImpactAdvisory,
		Dimension:        "measures",
		SeverityDefault:  "ERROR",
		CleanRequirement: string(CleanRequirementUncheckable),
	}
	got := LocalMaturity(spec, []Finding{{Code: "measures.uncheckable"}})
	if got.CurrentLevel != "L3" {
		t.Fatalf("CurrentLevel = %q, want L3", got.CurrentLevel)
	}
	if got.UnknownCount != 1 {
		t.Fatalf("UnknownCount = %d, want 1", got.UnknownCount)
	}
	if !got.Clean {
		t.Fatal("Clean = false, want true because UNCHECKABLE findings are unknown, not required debt")
	}
	if DebtScore(got.Findings) != 0 {
		t.Fatalf("DebtScore = %d, want 0", DebtScore(got.Findings))
	}
}

func TestDebtByLevelExcludesPhaseErrorsAndUncheckableFindings(t *testing.T) {
	spec := validSpec()
	spec.Findings["measures.warning"] = FindingMapping{
		LocalLevelImpact: "L3",
		GlobalImpact:     ImpactHardeningGap,
		Dimension:        "measures",
		SeverityDefault:  "WARNING",
	}
	spec.Findings["measures.info"] = FindingMapping{
		LocalLevelImpact: "L2",
		GlobalImpact:     ImpactAdvisory,
		Dimension:        "measures",
		SeverityDefault:  "INFO",
	}
	spec.Findings["measures.required-warning"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     ImpactAdvisory,
		Dimension:        "measures",
		SeverityDefault:  "WARNING",
		CleanRequirement: string(CleanRequirementRequired),
	}
	spec.Findings["measures.uncheckable"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     ImpactAdvisory,
		Dimension:        "measures",
		SeverityDefault:  "WARNING",
		CleanRequirement: string(CleanRequirementUncheckable),
	}

	local := LocalMaturity(spec, []Finding{
		{Code: "measures.uncovered-domain"},
		{Code: "measures.warning"},
		{Code: "measures.info"},
		{Code: "measures.required-warning"},
		{Code: "measures.uncheckable"},
	})

	if local.CurrentLevel != "L0" {
		t.Fatalf("CurrentLevel = %q, want L0", local.CurrentLevel)
	}
	got := DebtByLevel(local.Findings)
	if DebtScore(local.Findings) != 3 {
		t.Fatalf("DebtScore = %d, want 3", DebtScore(local.Findings))
	}
	if got["L3"].Total != 1 || got["L2"].Total != 1 || got["L1"].Total != 1 {
		t.Fatalf("DebtByLevel = %#v, want one debt finding at L1, L2, and L3", got)
	}
	if _, exists := got["L2"].BySeverity[architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR]; exists {
		t.Fatalf("blocking error was counted as debt: %#v", got)
	}
}

func TestAssessmentDebtByLevelDerivesFromProtoAssessment(t *testing.T) {
	assessment := &commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "measures-health",
		Phase:    "measures",
		Version:  "1",
		Local:    &commonv1.LocalMaturityAssessment{CurrentLevel: "L3"},
		Findings: []*commonv1.AssessmentFinding{
			{
				Code:     "measures.warning",
				Severity: "WARNING",
				Maturity: &commonv1.FindingMaturity{
					LocalLevel:   "L3",
					GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_HARDENING_GAP,
					Dimension:    "measures",
				},
			},
			{
				Code:     "measures.error",
				Severity: "ERROR",
				Maturity: &commonv1.FindingMaturity{
					LocalLevel:   "L2",
					GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP,
					Dimension:    "measures",
				},
			},
		},
	}

	got := AssessmentDebtByLevel(assessment)
	if AssessmentDebtScore(assessment) != 1 {
		t.Fatalf("AssessmentDebtScore = %d, want 1", AssessmentDebtScore(assessment))
	}
	if got["L3"].Total != 1 {
		t.Fatalf("L3 debt = %#v, want one warning", got["L3"])
	}
	if got["L2"].Total != 0 {
		t.Fatalf("L2 debt = %#v, want no blocking errors counted", got["L2"])
	}
}

func TestBuildProtoAssessmentWithZeroFindings(t *testing.T) {
	got, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     validSpec(),
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	if got.GetScenario() != "demo" || got.GetProvider() != "measures-health" || got.GetPhase() != "measures" {
		t.Fatalf("assessment identity wrong: %+v", got)
	}
	if got.GetLocal().GetCurrentLevel() != "L3" || got.GetLocal().GetNextLevel() != "" {
		t.Fatalf("local maturity = current %q next %q, want L3/empty", got.GetLocal().GetCurrentLevel(), got.GetLocal().GetNextLevel())
	}
	if !got.GetLocal().GetClean() || got.GetLocal().GetUnknownCount() != 0 {
		t.Fatalf("local clean/unknown = %v/%d, want true/0", got.GetLocal().GetClean(), got.GetLocal().GetUnknownCount())
	}
	if len(got.GetFindings()) != 0 {
		t.Fatalf("findings = %d, want zero", len(got.GetFindings()))
	}
}

func TestBuildProtoAssessmentWithZeroFindingsAndCapabilityOnlySpec(t *testing.T) {
	got, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     validMultiCapabilitySpec(),
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	if got.GetLocal().GetCurrentLevel() == "" {
		t.Fatal("local current level must be set for a clean capability-only spec")
	}
	if len(got.GetCapabilities()) != 2 {
		t.Fatalf("capabilities = %d, want 2", len(got.GetCapabilities()))
	}
	for _, capability := range got.GetCapabilities() {
		if capability.GetCurrentLevel() == "" {
			t.Fatalf("capability %q current level is empty", capability.GetId())
		}
	}
}

func TestBuildProtoAssessmentWithZeroFindingsForRepositoryCapabilitySpecs(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	specPaths, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "test-genie.json"))
	if err != nil {
		t.Fatalf("glob test-genie descriptors: %v", err)
	}
	for _, specPath := range specPaths {
		spec, err := LoadSpecFromScenario(filepath.Dir(filepath.Dir(specPath)))
		if err != nil {
			t.Fatalf("load %s: %v", specPath, err)
		}
		if len(spec.Capabilities) == 0 {
			continue
		}
		t.Run(filepath.Base(filepath.Dir(filepath.Dir(specPath))), func(t *testing.T) {
			got, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: *spec})
			if err != nil {
				t.Fatalf("BuildProtoAssessment returned error: %v", err)
			}
			if got.GetLocal().GetCurrentLevel() == "" {
				t.Fatal("local current level must be set")
			}
		})
	}
}

func TestBuildProtoAssessmentIncludesBlockingFindingMetadata(t *testing.T) {
	got, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     validSpec(),
		Findings: []Finding{{
			Code:        "measures.uncovered-domain",
			Severity:    "ERROR",
			Title:       "Domain missing measures",
			Message:     "payments has no measures",
			Location:    "measures.json",
			Remediation: "Add a measure",
			Source:      architecturev1.FindingSource_FINDING_SOURCE_MEASURES,
			Phase:       "measures",
		}},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	if got.GetLocal().GetCurrentLevel() != "L1" || got.GetLocal().GetNextLevel() != "L2" {
		t.Fatalf("local maturity = current %q next %q, want L1/L2", got.GetLocal().GetCurrentLevel(), got.GetLocal().GetNextLevel())
	}
	if got.GetLocal().GetBlockingFindingCodes()[0] != "measures.uncovered-domain" {
		t.Fatalf("blocking codes = %#v", got.GetLocal().GetBlockingFindingCodes())
	}
	finding := got.GetFindings()[0]
	if finding.GetMaturity().GetDimension() != "measures" {
		t.Fatalf("dimension = %q, want measures", finding.GetMaturity().GetDimension())
	}
	if got.GetFindingsByGlobalImpact()["capability_gap"] != 1 {
		t.Fatalf("impact counts = %#v", got.GetFindingsByGlobalImpact())
	}
	if finding.GetMaturity().GetCleanRequirement() != commonv1.CleanRequirement_CLEAN_REQUIREMENT_ADVISORY {
		t.Fatalf("clean requirement = %v, want ADVISORY default", finding.GetMaturity().GetCleanRequirement())
	}
	if got.GetFindingsByCleanRequirement()["advisory"] != 1 {
		t.Fatalf("clean requirement counts = %#v", got.GetFindingsByCleanRequirement())
	}
}

func TestBuildProtoAssessmentEmitsCapabilityAssessmentsAndRollup(t *testing.T) {
	got, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     validMultiCapabilitySpec(),
		Findings: []Finding{
			{Code: "interop.proxy_base_missing"},
			{Code: "pwa.service_worker_missing"},
		},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	if got.GetLocal().GetCurrentLevel() != "L0" || got.GetLocal().GetNextLevel() != "L1" {
		t.Fatalf("rollup local = current %q next %q, want L0/L1", got.GetLocal().GetCurrentLevel(), got.GetLocal().GetNextLevel())
	}
	if got.GetHighestPriorityCapability().GetCapabilityId() != "pwa_native_readiness" {
		t.Fatalf("highest priority = %#v, want pwa_native_readiness", got.GetHighestPriorityCapability())
	}
	if len(got.GetCapabilities()) != 2 {
		t.Fatalf("capabilities = %d, want 2", len(got.GetCapabilities()))
	}
	pwa := got.GetCapabilities()[1]
	if pwa.GetCurrentLevel() != "L0" || pwa.GetNextLevel() != "L1" {
		t.Fatalf("pwa maturity = current %q next %q, want L0/L1", pwa.GetCurrentLevel(), pwa.GetNextLevel())
	}
	if pwa.GetCurrentSummary() != "Install surface is absent." {
		t.Fatalf("pwa current summary = %q", pwa.GetCurrentSummary())
	}
	if pwa.GetNextUnlock() != "Offline-safe launch and app-shell fallback." {
		t.Fatalf("pwa next unlock = %q", pwa.GetNextUnlock())
	}
	if got.GetFindings()[0].GetMaturity().GetCapabilityId() != "interop" {
		t.Fatalf("finding capability = %q, want interop", got.GetFindings()[0].GetMaturity().GetCapabilityId())
	}
	if got.GetFindings()[1].GetMaturity().GetCapabilityId() != "pwa_native_readiness" {
		t.Fatalf("finding capability = %q, want pwa_native_readiness", got.GetFindings()[1].GetMaturity().GetCapabilityId())
	}
}

func TestCapabilityPriorityScoring(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*Spec)
		findings   []Finding
		wantFocus  string
		wantRanks  map[string]int32
		wantReason string
	}{
		{
			name: "required lower level warning outranks higher level error",
			findings: []Finding{
				{Code: "pwa.service_worker_missing"},
				{Code: "interop.proxy_base_missing"},
			},
			wantFocus: "pwa_native_readiness",
			wantRanks: map[string]int32{
				"pwa_native_readiness": 1,
				"interop":              2,
			},
			wantReason: "lowest current level with required/blocking findings WARNING capability_gap debt=1",
		},
		{
			name: "same level foundation impact outranks advisory",
			mutate: func(spec *Spec) {
				spec.Findings["interop.proxy_base_missing"] = FindingMapping{
					CapabilityID:     "interop",
					LocalLevelImpact: "L1",
					GlobalImpact:     ImpactFoundationBlocker,
					Dimension:        "ui",
					SeverityDefault:  "ERROR",
					CleanRequirement: string(CleanRequirementRequired),
				}
				spec.Findings["pwa.service_worker_missing"] = FindingMapping{
					CapabilityID:     "pwa_native_readiness",
					LocalLevelImpact: "L1",
					GlobalImpact:     ImpactAdvisory,
					Dimension:        "ui",
					SeverityDefault:  "ERROR",
					CleanRequirement: string(CleanRequirementAdvisory),
				}
			},
			findings: []Finding{
				{Code: "pwa.service_worker_missing"},
				{Code: "interop.proxy_base_missing"},
			},
			wantFocus:  "interop",
			wantReason: "lowest current level with required/blocking findings ERROR foundation_blocker",
		},
		{
			name: "advisory debt outranks fully clean capability",
			mutate: func(spec *Spec) {
				delete(spec.Findings, "pwa.service_worker_missing")
				spec.Findings["interop.proxy_base_missing"] = FindingMapping{
					CapabilityID:     "interop",
					LocalLevelImpact: "L2",
					GlobalImpact:     ImpactHardeningGap,
					Dimension:        "ui",
					SeverityDefault:  "WARNING",
					CleanRequirement: string(CleanRequirementAdvisory),
				}
			},
			findings:   []Finding{{Code: "interop.proxy_base_missing"}},
			wantFocus:  "interop",
			wantReason: "lowest current level with advisory findings WARNING hardening_gap debt=1",
		},
		{
			name: "debt bearing capability outranks earlier clean capability at same rung",
			mutate: func(spec *Spec) {
				spec.Capabilities = []CapabilitySpec{
					spec.Capabilities[1],
					spec.Capabilities[0],
				}
				spec.Findings["interop.proxy_base_missing"] = FindingMapping{
					CapabilityID:     "interop",
					LocalLevelImpact: "L2",
					GlobalImpact:     ImpactHardeningGap,
					Dimension:        "ui",
					SeverityDefault:  "WARNING",
					CleanRequirement: string(CleanRequirementAdvisory),
				}
				delete(spec.Findings, "pwa.service_worker_missing")
			},
			findings:   []Finding{{Code: "interop.proxy_base_missing"}},
			wantFocus:  "interop",
			wantReason: "lowest current level with advisory findings WARNING hardening_gap debt=1",
		},
		{
			name: "fixable tie break applies after severity and impact",
			mutate: func(spec *Spec) {
				spec.Findings["interop.proxy_base_missing"] = FindingMapping{
					CapabilityID:     "interop",
					LocalLevelImpact: "L1",
					GlobalImpact:     ImpactHardeningGap,
					Dimension:        "ui",
					SeverityDefault:  "ERROR",
					CleanRequirement: string(CleanRequirementRequired),
				}
				spec.Findings["pwa.service_worker_missing"] = FindingMapping{
					CapabilityID:     "pwa_native_readiness",
					LocalLevelImpact: "L1",
					GlobalImpact:     ImpactHardeningGap,
					Dimension:        "ui",
					SeverityDefault:  "ERROR",
					CleanRequirement: string(CleanRequirementRequired),
					FixClass:         FixClassAuto,
					FixerStatus:      FixerStatusPending,
				}
			},
			findings: []Finding{
				{Code: "pwa.service_worker_missing"},
				{Code: "interop.proxy_base_missing"},
			},
			wantFocus:  "pwa_native_readiness",
			wantReason: "lowest current level with required/blocking findings ERROR hardening_gap fixable=1",
		},
		{
			name: "declaration order is final stable tie break",
			mutate: func(spec *Spec) {
				spec.Findings["interop.proxy_base_missing"] = FindingMapping{
					CapabilityID:     "interop",
					LocalLevelImpact: "L1",
					GlobalImpact:     ImpactHardeningGap,
					Dimension:        "ui",
					SeverityDefault:  "ERROR",
					CleanRequirement: string(CleanRequirementRequired),
				}
				spec.Findings["pwa.service_worker_missing"] = FindingMapping{
					CapabilityID:     "pwa_native_readiness",
					LocalLevelImpact: "L1",
					GlobalImpact:     ImpactHardeningGap,
					Dimension:        "ui",
					SeverityDefault:  "ERROR",
					CleanRequirement: string(CleanRequirementRequired),
				}
			},
			findings: []Finding{
				{Code: "pwa.service_worker_missing"},
				{Code: "interop.proxy_base_missing"},
			},
			wantFocus:  "interop",
			wantReason: "lowest current level with required/blocking findings ERROR hardening_gap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := validMultiCapabilitySpec()
			if tt.mutate != nil {
				tt.mutate(&spec)
			}
			got, err := BuildProtoAssessment(BuildInput{
				Scenario: "demo",
				Spec:     spec,
				Findings: tt.findings,
			})
			if err != nil {
				t.Fatalf("BuildProtoAssessment returned error: %v", err)
			}
			if got.GetHighestPriorityCapability().GetCapabilityId() != tt.wantFocus {
				t.Fatalf("focus = %#v, want %s", got.GetHighestPriorityCapability(), tt.wantFocus)
			}
			if got.GetHighestPriorityCapability().GetReason() != tt.wantReason {
				t.Fatalf("focus reason = %q, want %q", got.GetHighestPriorityCapability().GetReason(), tt.wantReason)
			}
			for _, capability := range got.GetCapabilities() {
				if want, ok := tt.wantRanks[capability.GetId()]; ok && capability.GetPriorityRank() != want {
					t.Fatalf("%s priority rank = %d, want %d", capability.GetId(), capability.GetPriorityRank(), want)
				}
			}
		})
	}
}

// TestPriorityAmongMaxedPrefersDeepestLadder proves that when every capability is
// clean at its ceiling (a fully-mature phase), the focus that represents the
// phase (its rung + North Star) is the DEEPEST ladder, not the shortest one that
// happens to sit at the lowest numeric rung. This is what lets a maxed phase
// showcase its most aspirational capability.
func TestPriorityAmongMaxedPrefersDeepestLadder(t *testing.T) {
	spec := Spec{
		Provider: "cli-health",
		Phase:    "contracts",
		Version:  "1",
		Capabilities: []CapabilitySpec{
			{
				ID:    "discovery_readiness",
				Label: "Discovery Readiness",
				Levels: []Level{
					{ID: "L0", Name: "Blocked", EntryCriteria: []string{"a"}, ExitCriteria: []string{"b"}, NextUnlock: "up"},
					{ID: "L1", Name: "Ready", EntryCriteria: []string{"a"}, ExitCriteria: []string{"b"}, NextUnlock: "up"},
					{ID: "L2", Name: "Clean", EntryCriteria: []string{"a"}, ExitCriteria: []string{"b"}, CapabilitySummary: "Discovery readiness is clean."},
				},
			},
			{
				ID:    "command_architecture",
				Label: "Command Architecture",
				Levels: []Level{
					{ID: "L0", Name: "Unclassifiable", EntryCriteria: []string{"a"}, ExitCriteria: []string{"b"}, NextUnlock: "up"},
					{ID: "L1", Name: "Shell", EntryCriteria: []string{"a"}, ExitCriteria: []string{"b"}, NextUnlock: "up"},
					{ID: "L2", Name: "Declarative", EntryCriteria: []string{"a"}, ExitCriteria: []string{"b"}, NextUnlock: "up"},
					{ID: "L3", Name: "Declared", EntryCriteria: []string{"a"}, ExitCriteria: []string{"b"}, NextUnlock: "up"},
					{ID: "L4", Name: "Verified", EntryCriteria: []string{"a"}, ExitCriteria: []string{"b"}, CapabilitySummary: "Verified renderer-separated primitives."},
				},
			},
		},
		Findings: map[string]FindingMapping{},
		Fallback: FallbackPolicy{LocalLevelImpact: "L1", GlobalImpact: ImpactUnknown, Dimension: "contracts", SeverityDefault: "WARNING", CleanRequirement: string(CleanRequirementAdvisory)},
	}
	got, err := BuildProtoAssessment(BuildInput{Scenario: "cli-health", Spec: spec})
	if err != nil {
		t.Fatalf("BuildProtoAssessment: %v", err)
	}
	if focus := got.GetHighestPriorityCapability().GetCapabilityId(); focus != "command_architecture" {
		t.Fatalf("focus among maxed capabilities = %q, want the deepest ladder command_architecture", focus)
	}
	// The phase-level standing therefore reflects the deepest ladder's rung and
	// North Star, not the shortest capability's.
	if got.GetLocal().GetCurrentLevel() != "L4" {
		t.Fatalf("local rung = %q, want L4 (the deepest maxed ladder)", got.GetLocal().GetCurrentLevel())
	}
}

func TestBuildProtoAssessmentRejectsInvalidSpec(t *testing.T) {
	spec := validSpec()
	spec.Provider = ""
	if _, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: spec}); err == nil {
		t.Fatal("BuildProtoAssessment succeeded with invalid spec")
	}
}

func TestValidateAssessmentRejectsMalformedAssessment(t *testing.T) {
	got, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: validSpec()})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	got.Local.CurrentLevel = ""
	if err := ValidateAssessment(got); err == nil {
		t.Fatal("ValidateAssessment succeeded with empty current level")
	}
}

func TestBuildProtoAssessmentDeduplicatesRecommendedSkills(t *testing.T) {
	spec := validSpec()
	spec.Findings["measures.extra"] = FindingMapping{
		LocalLevelImpact:    "L2",
		GlobalImpact:        ImpactCapabilityGap,
		Dimension:           "measures",
		SeverityDefault:     "ERROR",
		RecommendedSkillIDs: []string{"quality-health", "measures-adoption"},
	}
	got, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     spec,
		Findings: []Finding{
			{Code: "measures.extra"},
			{Code: "measures.uncovered-domain"},
		},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	want := []string{"measures-adoption", "quality-health"}
	if !slices.Equal(got.GetRecommendedSkillIds(), want) {
		t.Fatalf("recommended skills = %#v, want %#v", got.GetRecommendedSkillIds(), want)
	}
}

func TestBuildValidationResponseDerivesStatusAndPacksNativeDetail(t *testing.T) {
	assessment, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     validSpec(),
		Findings: []Finding{{Code: "measures.uncovered-domain"}},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	native := &commonv1.LocalMaturityLevel{Id: "native"}
	got, err := BuildValidationResponse("demo", assessment, native, nil)
	if err != nil {
		t.Fatalf("BuildValidationResponse returned error: %v", err)
	}
	if got.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("status = %v, want FAILED", got.GetStatus())
	}
	if got.GetNativeDetail() == nil {
		t.Fatal("native_detail is nil")
	}
	unpacked := &commonv1.LocalMaturityLevel{}
	if err := got.GetNativeDetail().UnmarshalTo(unpacked); err != nil {
		t.Fatalf("native_detail unmarshal failed: %v", err)
	}
	if unpacked.GetId() != "native" {
		t.Fatalf("native detail id = %q, want native", unpacked.GetId())
	}
}

func TestBuildValidationResponseCanOverrideDerivedStatus(t *testing.T) {
	assessment, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: validSpec()})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	got, err := BuildValidationResponse(
		"demo",
		assessment,
		nil,
		nil,
		WithValidationStatus(scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED),
	)
	if err != nil {
		t.Fatalf("BuildValidationResponse returned error: %v", err)
	}
	if got.GetStatus() != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_DEGRADED {
		t.Fatalf("status = %v, want DEGRADED", got.GetStatus())
	}
}

func TestDeriveValidationStatusPassesWithNoErrorSeverity(t *testing.T) {
	assessment, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: validSpec()})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	assessment.FindingsBySeverity["FINDING_SEVERITY_WARNING"] = 2
	if got := DeriveValidationStatus(assessment); got != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED {
		t.Fatalf("status = %v, want PASSED", got)
	}
}

func TestAssessmentToArchitectureFindingsRoutesByDimension(t *testing.T) {
	assessment := &commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "unit-health",
		Phase:    "unit",
		Version:  "1",
		Local:    &commonv1.LocalMaturityAssessment{CurrentLevel: "L1"},
		Findings: []*commonv1.AssessmentFinding{{
			Code:        "coverage.low",
			Severity:    "ERROR",
			Title:       "Coverage below threshold",
			Message:     "api package is under target",
			Location:    "api/service.go",
			Remediation: "Add focused tests",
			Maturity: &commonv1.FindingMaturity{
				GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_HARDENING_GAP,
				Dimension:    "coverage",
			},
		}},
	}
	got := AssessmentToArchitectureFindings("demo", assessment, architecturev1.FindingSource_FINDING_SOURCE_CLI)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1", len(got))
	}
	finding := got[0]
	if finding.GetSource() != architecturev1.FindingSource_FINDING_SOURCE_COVERAGE {
		t.Fatalf("source = %v, want COVERAGE", finding.GetSource())
	}
	if finding.GetStableId() == "" {
		t.Fatal("stable_id is empty")
	}
	if finding.GetMessage() != "Coverage below threshold: api package is under target" {
		t.Fatalf("message = %q", finding.GetMessage())
	}
}

func TestAssessmentToArchitectureFindingsFallsBackWhenDimensionUnknown(t *testing.T) {
	assessment := &commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "custom-health",
		Phase:    "custom",
		Version:  "1",
		Local:    &commonv1.LocalMaturityAssessment{CurrentLevel: "L1"},
		Findings: []*commonv1.AssessmentFinding{{
			Code:     "custom.notice",
			Severity: "INFO",
			Maturity: &commonv1.FindingMaturity{
				GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY,
				Dimension:    "custom",
			},
		}},
	}
	got := AssessmentToArchitectureFindings("", assessment, architecturev1.FindingSource_FINDING_SOURCE_MEASURES)
	if len(got) != 1 {
		t.Fatalf("findings = %d, want 1", len(got))
	}
	if got[0].GetScenario() != "demo" {
		t.Fatalf("scenario = %q, want demo", got[0].GetScenario())
	}
	if got[0].GetSource() != architecturev1.FindingSource_FINDING_SOURCE_MEASURES {
		t.Fatalf("source = %v, want MEASURES", got[0].GetSource())
	}
}
