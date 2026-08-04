package validation

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Service validates one scenario's storage judgment at a time. It owns the
// detect → analyze pipeline; the Connect handler and the CLI are thin
// translation layers over ValidateScenario.
type Service struct {
	repoRoot  string
	detector  Detector
	analyzers []Analyzer
	logger    *log.Logger
}

// Deps wires the Service's seams. Detector and Analyzers default to the real
// implementations when nil, so production callers pass only RepoRoot.
type Deps struct {
	RepoRoot  string
	Detector  Detector
	Analyzers []Analyzer
	Logger    *log.Logger
}

// New constructs a Service. The detector defaults to the code-facts-backed
// detector (filesystem fallback) and the analyzer set to DefaultAnalyzers().
func New(d Deps) *Service {
	detector := d.Detector
	if detector == nil {
		detector = CodeFactsDetector{}
	}
	analyzers := d.Analyzers
	if analyzers == nil {
		analyzers = DefaultAnalyzers()
	}
	logger := d.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Service{repoRoot: d.RepoRoot, detector: detector, analyzers: analyzers, logger: logger}
}

// ValidateScenario preserves the shared ScenarioValidationService entry point
// while routing through the owner-kind-agnostic validation engine.
//
// A missing target is a graceful skip (an INFO STORAGE_TARGET_UNRESOLVABLE
// finding, never a hard error) so the test-genie storage phase does not crash
// the suite when pointed at a synthetic or absent scenario.
func (s *Service) ValidateScenario(ctx context.Context, scenario string) (Report, error) {
	report, err := s.ValidateOwner(ctx, corestorage.OwnerScenario, scenario, corestorage.HostPlatform())
	if err != nil || scenario != "storage-manager" {
		return report, err
	}
	gate := s.validateNonScenarioGate(ctx, corestorage.HostPlatform())
	// Keep the single aggregate verdict for gating, but retain every owner
	// finding with typed attribution so consumers can act on the actual owner.
	report.Findings = append(report.Findings, gate.Findings...)
	if gate.ErrorCount > 0 {
		report.Findings = append(report.Findings, Finding{
			Code:        "STORAGE_OWNER_GATE_FAILED",
			Severity:    SeverityError,
			Title:       "Non-scenario storage gate failed",
			Message:     fmt.Sprintf("resource/tool/safeguard validation found %d error or blocker findings across %d owners; advisory findings: %d", gate.ErrorCount, gate.OwnerCount, gate.AdvisoryCount),
			Location:    "scenarios/storage-manager/api/handlers/validation/module.go",
			Remediation: "Fix the reported non-scenario storage declaration errors before merging.",
			Analyzer:    "storage-manager.non-scenario-gate",
		})
	} else {
		report.Findings = append(report.Findings, Finding{
			Code:        "STORAGE_OWNER_GATE_ADVISORY_TREND",
			Severity:    SeverityInfo,
			Title:       "Non-scenario storage gate passed",
			Message:     fmt.Sprintf("resource/tool/safeguard validation covered %d owners; advisory findings: %d", gate.OwnerCount, gate.AdvisoryCount),
			Location:    "scenarios/storage-manager/api/handlers/validation/module.go",
			Remediation: "Track advisory drift as a trend; only error and blocker findings gate the suite.",
			Analyzer:    "storage-manager.non-scenario-gate",
		})
	}
	sortFindings(report.Findings)
	return report, nil
}

type nonScenarioGateResult struct {
	OwnerCount    int
	ErrorCount    int
	AdvisoryCount int
	Findings      []Finding
}

func (s *Service) validateNonScenarioGate(ctx context.Context, platform corestorage.Platform) nonScenarioGateResult {
	inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: s.repoRoot, Platform: platform})
	if err != nil {
		return nonScenarioGateResult{ErrorCount: 1}
	}
	result := nonScenarioGateResult{}
	for _, owner := range inventory.Owners {
		if owner.Kind == corestorage.OwnerScenario {
			continue
		}
		result.OwnerCount++
		report, validateErr := s.ValidateOwnerFromInventoryFast(ctx, owner.Kind, owner.ID, platform, inventory)
		if validateErr != nil {
			result.ErrorCount++
			continue
		}
		for _, finding := range report.Findings {
			finding.Subject = storageTargetSubject(owner.Kind, owner.ID, s.repoRoot, owner.ManifestPath)
			result.Findings = append(result.Findings, finding)
			if finding.Severity >= SeverityError {
				result.ErrorCount++
			} else {
				result.AdvisoryCount++
			}
		}
	}
	return result
}

func storageTargetSubject(kind corestorage.OwnerKind, id, repoRoot, manifestPath string) *commonv1.ValidationTarget {
	root := ""
	if repoRoot != "" && manifestPath != "" {
		if rel, err := filepath.Rel(repoRoot, filepath.Dir(manifestPath)); err == nil {
			root = filepath.ToSlash(rel)
		}
	}
	return &commonv1.ValidationTarget{Kind: storageTargetKind(kind), Id: id, Root: root}
}

func storageTargetKind(kind corestorage.OwnerKind) commonv1.ValidationTargetKind {
	switch kind {
	case corestorage.OwnerScenario:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO
	case corestorage.OwnerResource:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE
	case corestorage.OwnerTool:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_TOOL
	case corestorage.OwnerSafeguard:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD
	default:
		return commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED
	}
}

// ValidateOwner validates any native storage owner. Scenario-shaped analysis
// is populated only for scenarios; declaration analyzers still run for every
// owner kind through the same registry.
func (s *Service) ValidateOwner(ctx context.Context, kind corestorage.OwnerKind, id string, requested corestorage.Platform) (Report, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Report{}, fmt.Errorf("%s name is required", kind)
	}
	platform := corestorage.NormalizePlatform(string(requested))
	if platform == "" {
		platform = corestorage.HostPlatform()
	}
	if platform == "" {
		return Report{}, fmt.Errorf("unsupported storage platform %q", requested)
	}

	inventory, inventoryErr := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: s.repoRoot, Platform: platform})
	if inventoryErr != nil {
		return Report{}, inventoryErr
	}
	return s.validateOwnerFromInventory(ctx, kind, id, platform, inventory)
}

// ValidateOwnerFromInventory validates against a caller-owned inventory. Fleet
// validation uses this seam so one repository walk feeds every owner report.
func (s *Service) ValidateOwnerFromInventory(ctx context.Context, kind corestorage.OwnerKind, id string, requested corestorage.Platform, inventory corestorage.OwnerInventory) (Report, error) {
	return s.validateOwnerFromInventoryWithDetector(ctx, kind, id, requested, inventory, s.detector, false)
}

// ValidateOwnerFromInventoryFast validates from a caller-owned inventory using
// only local filesystem facts and declaration analyzers. Fleet validation uses
// this path so one unavailable code-facts provider cannot add its timeout once
// per scenario, while scenario-shaped source analyzers remain the responsibility
// of the single-owner and Test Genie scenario surfaces.
func (s *Service) ValidateOwnerFromInventoryFast(ctx context.Context, kind corestorage.OwnerKind, id string, requested corestorage.Platform, inventory corestorage.OwnerInventory) (Report, error) {
	return s.validateOwnerFromInventoryWithDetector(ctx, kind, id, requested, inventory, FilesystemDetector{}, true)
}

func (s *Service) validateOwnerFromInventory(ctx context.Context, kind corestorage.OwnerKind, id string, platform corestorage.Platform, inventory corestorage.OwnerInventory) (Report, error) {
	return s.validateOwnerFromInventoryWithDetector(ctx, kind, id, platform, inventory, s.detector, false)
}

func (s *Service) validateOwnerFromInventoryWithDetector(ctx context.Context, kind corestorage.OwnerKind, id string, platform corestorage.Platform, inventory corestorage.OwnerInventory, detector Detector, declarationOnly bool) (Report, error) {
	if id == "" {
		return Report{}, fmt.Errorf("%s name is required", kind)
	}
	collector := metricsFrom(ctx)
	detect := collector.Stage("detect-storage-surface")
	var owner *corestorage.OwnerManifest
	for i := range inventory.Owners {
		if inventory.Owners[i].Kind == kind && inventory.Owners[i].ID == id {
			owner = &inventory.Owners[i]
			break
		}
	}
	if owner == nil && kind == corestorage.OwnerScenario {
		if scenarioDir, ok := resolveScenarioDir(s.repoRoot, id); ok {
			owner = &corestorage.OwnerManifest{Kind: kind, ID: id, ManifestPath: filepath.Join(scenarioDir, ".vrooli", "service.json")}
		}
	}
	if owner == nil {
		detect.End()
		return unresolvedOwnerReport(kind, id, platform), nil
	}

	var scenarioDir, apiDir string
	var detection Detection
	var engines []Engine
	var stage string
	var hasMigrations bool
	if kind == corestorage.OwnerScenario {
		var ok bool
		scenarioDir, ok = resolveScenarioDir(s.repoRoot, id)
		if !ok {
			detect.End()
			return unresolvedOwnerReport(kind, id, platform), nil
		}

		detection = detector.Detect(ctx, id, scenarioDir)
		engines = detectEngines(scenarioDir)
		stage, hasMigrations = deriveStorageStage(scenarioDir)
		apiDir = filepath.Join(scenarioDir, "api")
		if info, err := os.Stat(apiDir); err != nil || !info.IsDir() {
			apiDir = ""
		}
	}
	detect.End()

	ac := AnalyzerContext{
		RepoRoot:      s.repoRoot,
		Scenario:      id,
		OwnerKind:     kind,
		Platform:      platform,
		ScenarioDir:   scenarioDir,
		APIDir:        apiDir,
		Language:      detection.Language,
		Engines:       engines,
		Domains:       detection.Domains,
		StorageStage:  stage,
		HasMigrations: hasMigrations,
		Owner:         owner,
	}

	var findings []Finding
	analyzerResults := make([]AnalyzerResult, 0, len(s.analyzers))
	for _, a := range s.analyzers {
		if declarationOnly && len(a.Kinds()) > 0 {
			analyzerResults = append(analyzerResults, AnalyzerResult{Name: a.Name(), Reason: "fleet validation is declaration-only for scenario-shaped analyzers"})
			continue
		}
		if !analyzerSupportsKind(a, kind) {
			analyzerResults = append(analyzerResults, AnalyzerResult{Name: a.Name(), Reason: fmt.Sprintf("not applicable to owner kind %s", kind)})
			continue
		}
		if !a.Applies(ac) {
			analyzerResults = append(analyzerResults, AnalyzerResult{Name: a.Name(), Reason: "context not applicable"})
			continue
		}
		st := collector.Stage("analyze:" + a.Name())
		got, err := a.Analyze(ctx, ac)
		st.End()
		if err != nil {
			s.logger.Printf("storage-manager: analyzer %q failed (downgraded to no-op): %v", a.Name(), err)
			analyzerResults = append(analyzerResults, AnalyzerResult{Name: a.Name(), Applicable: true, Reason: "analyzer error"})
			continue
		}
		findings = append(findings, got...)
		codes := make([]string, 0, len(got))
		for _, finding := range got {
			codes = append(codes, finding.Code)
		}
		analyzerResults = append(analyzerResults, AnalyzerResult{Name: a.Name(), Applicable: true, FindingCode: codes})
	}

	// The accountability rung is a function of every other analyzer's output, so
	// it runs last. It always runs: an owner that declares nothing produces no
	// findings, and without a marker the maturity engine would score that
	// silence as the top rung.
	rungStage := collector.Stage("analyze:" + accountabilityAnalyzer)
	rung := accountabilityFindings(ac, findings)
	rungStage.End()
	rungCodes := make([]string, 0, len(rung))
	for _, finding := range rung {
		rungCodes = append(rungCodes, finding.Code)
	}
	findings = append(findings, rung...)
	analyzerResults = append(analyzerResults, AnalyzerResult{Name: accountabilityAnalyzer, Applicable: true, FindingCode: rungCodes})

	sortFindings(findings)

	return Report{
		Scenario:      id,
		OwnerKind:     kind,
		OwnerID:       id,
		Platform:      platform,
		Status:        reportStatus(findings),
		Analyzers:     analyzerResults,
		ScenarioDir:   scenarioDir,
		Language:      detection.Language,
		Engines:       engines,
		StorageStage:  stage,
		HasMigrations: hasMigrations,
		Findings:      findings,
	}, nil
}

func analyzerSupportsKind(analyzer Analyzer, kind corestorage.OwnerKind) bool {
	for _, supported := range analyzer.Kinds() {
		if supported == kind {
			return true
		}
	}
	return len(analyzer.Kinds()) == 0
}

func unresolvedOwnerReport(kind corestorage.OwnerKind, id string, platform corestorage.Platform) Report {
	return Report{
		Scenario:     id,
		OwnerKind:    kind,
		OwnerID:      id,
		Platform:     platform,
		Status:       "unresolvable",
		StorageStage: "greenfield",
		Findings: []Finding{{
			Code:        "STORAGE_TARGET_UNRESOLVABLE",
			Severity:    SeverityInfo,
			Title:       fmt.Sprintf("%s %q not found", kind, id),
			Message:     fmt.Sprintf("No %s owner named %q exists in the repository inventory.", kind, id),
			Remediation: "Check the owner id and its native manifest path.",
			Analyzer:    "storage-manager",
		}},
	}
}

func reportStatus(findings []Finding) string {
	for _, finding := range findings {
		if finding.Severity >= SeverityError {
			return "failed"
		}
	}
	return "passed"
}

// resolveScenarioDir joins a repo root and scenario id into the absolute
// scenario directory, verifying it exists. Returns ok=false when absent.
func resolveScenarioDir(repoRoot, scenario string) (string, bool) {
	dir := filepath.Join(repoRoot, "scenarios", scenario)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return dir, true
}

// sortFindings orders findings deterministically: severity desc (errors
// first), then analyzer, code, and location — so the report is stable across
// runs and the diff-based regression checks are meaningful.
func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Severity != b.Severity {
			return a.Severity > b.Severity
		}
		if a.Analyzer != b.Analyzer {
			return a.Analyzer < b.Analyzer
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Location < b.Location
	})
}
