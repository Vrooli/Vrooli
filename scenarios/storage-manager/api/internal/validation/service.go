package validation

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
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

// ValidateScenario detects the scenario's storage surface, runs every
// applicable analyzer, and aggregates normalized findings into a Report.
//
// A missing target is a graceful skip (an INFO STORAGE_TARGET_UNRESOLVABLE
// finding, never a hard error) so the test-genie storage phase does not crash
// the suite when pointed at a synthetic or absent scenario.
func (s *Service) ValidateScenario(ctx context.Context, scenario string) (Report, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Report{}, fmt.Errorf("scenario name is required")
	}

	collector := metricsFrom(ctx)

	detect := collector.Stage("detect-storage-surface")
	scenarioDir, ok := resolveScenarioDir(s.repoRoot, scenario)
	if !ok {
		detect.End()
		return Report{
			Scenario:     scenario,
			StorageStage: "greenfield",
			Findings: []Finding{{
				Code:        "STORAGE_TARGET_UNRESOLVABLE",
				Severity:    SeverityInfo,
				Title:       fmt.Sprintf("Scenario %q not found under scenarios/", scenario),
				Message:     fmt.Sprintf("No directory scenarios/%s exists, so there is no storage surface to validate.", scenario),
				Remediation: "Check the scenario id, or generate it via `vrooli scenario generate`.",
				Analyzer:    "storage-manager",
			}},
		}, nil
	}

	detection := s.detector.Detect(ctx, scenario, scenarioDir)
	engines := detectEngines(scenarioDir)
	stage, hasMigrations := deriveStorageStage(scenarioDir)
	detect.End()

	apiDir := filepath.Join(scenarioDir, "api")
	if info, err := os.Stat(apiDir); err != nil || !info.IsDir() {
		apiDir = ""
	}

	ac := AnalyzerContext{
		RepoRoot:      s.repoRoot,
		Scenario:      scenario,
		ScenarioDir:   scenarioDir,
		APIDir:        apiDir,
		Language:      detection.Language,
		Engines:       engines,
		Domains:       detection.Domains,
		StorageStage:  stage,
		HasMigrations: hasMigrations,
	}
	if inventory, inventoryErr := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: s.repoRoot, Platform: corestorage.Platform(runtime.GOOS)}); inventoryErr == nil {
		for i := range inventory.Owners {
			owner := &inventory.Owners[i]
			if owner.Kind == corestorage.OwnerScenario && owner.ID == scenario {
				ac.Owner = owner
				break
			}
		}
	}

	var findings []Finding
	for _, a := range s.analyzers {
		if !a.Applies(ac) {
			continue
		}
		st := collector.Stage("analyze:" + a.Name())
		got, err := a.Analyze(ctx, ac)
		st.End()
		if err != nil {
			s.logger.Printf("storage-manager: analyzer %q failed (downgraded to no-op): %v", a.Name(), err)
			continue
		}
		findings = append(findings, got...)
	}

	sortFindings(findings)

	return Report{
		Scenario:      scenario,
		ScenarioDir:   scenarioDir,
		Language:      detection.Language,
		Engines:       engines,
		StorageStage:  stage,
		HasMigrations: hasMigrations,
		Findings:      findings,
	}, nil
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
