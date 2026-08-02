// Package archtest protects the import boundaries that keep agent-manager's
// runtime recoverable and its read-side exceptions intentional.
package archtest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"agent-manager/internal/runsignal"
)

const maxScenarioSourceLines = 1500

func TestScenarioSourceFilesStayBelowSizeLimit(t *testing.T) {
	var oversized []string
	err := filepath.WalkDir(scenarioRoot(t), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "node_modules", ".git", "coverage", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".ts", ".tsx":
		default:
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		lines := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines++
		}
		closeErr := file.Close()
		if err := scanner.Err(); err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if lines > maxScenarioSourceLines {
			rel, err := filepath.Rel(scenarioRoot(t), path)
			if err != nil {
				return err
			}
			oversized = append(oversized, fmt.Sprintf("%s (%d lines)", filepath.ToSlash(rel), lines))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(oversized) > 0 {
		t.Fatalf("scenario source files must stay at or below %d lines: %s", maxScenarioSourceLines, strings.Join(oversized, ", "))
	}
}

func TestWorkflowRuntimeDoesNotImportOrchestration(t *testing.T) {
	violations := importsUnder(t, "api/internal/workflowruntime", "agent-manager/internal/orchestration")
	if len(violations) != 0 {
		t.Fatalf("workflowruntime must remain a leaf package: %v", violations)
	}
}

func TestProductionOrchestrationDoesNotImportDatabase(t *testing.T) {
	violations := importsUnder(t, "api/internal/orchestration", "agent-manager/internal/adapters/database")
	if len(violations) != 0 {
		t.Fatalf("orchestration must depend on repository interfaces, not database: %v", violations)
	}
}

// Run-signal extraction is a deterministic replayable projection. Its entire
// package must stay independent of database, model, and HTTP clients.
func TestRunSignalDoesNotImportDatabaseModelOrHTTPClients(t *testing.T) {
	for _, path := range goFiles(t, filepath.Join(scenarioRoot(t), "api/internal/runsignal")) {
		imports, err := importsFromFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range imports {
			if imported == "net/http" || strings.Contains(imported, "database") || strings.Contains(imported, "model") || strings.Contains(imported, "openai") {
				rel, _ := filepath.Rel(scenarioRoot(t), path)
				t.Fatalf("runsignal must remain deterministic and local; %s imports %q", filepath.ToSlash(rel), imported)
			}
		}
	}
}

// Every detector is both deterministic (the import boundary above) and
// represented in the checked-in labelled corpus. This makes an unlabelled
// registry addition an architecture failure rather than a best-effort test.
func TestRunSignalDetectorRegistryHasLabelledCoverage(t *testing.T) {
	type labels struct {
		Expected []string `json:"expected"`
	}
	body, err := os.ReadFile(filepath.Join(scenarioRoot(t), "api/internal/runsignal/testdata/classification/all-detectors.labels.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus labels
	if err := json.Unmarshal(body, &corpus); err != nil {
		t.Fatal(err)
	}
	covered := make(map[string]bool, len(corpus.Expected))
	for _, id := range corpus.Expected {
		covered[id] = true
	}
	registered := make(map[string]bool, len(runsignal.EpisodeDetectors())+len(runsignal.SelfReportDetectors()))
	for _, detector := range runsignal.EpisodeDetectors() {
		assertRegisteredDetector(t, registered, covered, "episode/"+detector.Identifier(), detector.ClassifierVersion(), detector.CauseScope())
	}
	for _, detector := range runsignal.SelfReportDetectors() {
		assertRegisteredDetector(t, registered, covered, "span/"+detector.Identifier(), detector.ClassifierVersion(), detector.CauseScope())
	}
	if len(registered) != len(covered) {
		t.Fatalf("labelled detector corpus has extras or duplicate names: registered=%v labels=%v", registered, corpus.Expected)
	}
}

func assertRegisteredDetector(t *testing.T, registered, covered map[string]bool, id, version, scope string) {
	t.Helper()
	if id == "episode/" || id == "span/" || version == "" || scope == "" {
		t.Fatalf("detector declares incomplete metadata: id=%q version=%q scope=%q", id, version, scope)
	}
	if registered[id] {
		t.Fatalf("duplicate registered detector %q", id)
	}
	if !covered[id] {
		t.Fatalf("registered detector %q lacks labelled corpus coverage", id)
	}
	registered[id] = true
}

func TestAvailabilityVocabularyRejectsAdHocStateLiterals(t *testing.T) {
	states := "available|unavailable|degraded|unobserved|unknown|resolved|policy_absent|oversized|not_captured|external|empty|complete"
	literal := regexp.MustCompile(`(?:State:\s*|\.State\s*(?:==|!=)\s*)"(?:` + states + `)"`)
	paths := append(
		goFiles(t, filepath.Join(scenarioRoot(t), "api/internal/runreport")),
		goFiles(t, filepath.Join(scenarioRoot(t), "api/internal/runsignal"))...,
	)
	paths = append(paths,
		filepath.Join(scenarioRoot(t), "api/internal/wiring/receipt_runtime.go"),
		filepath.Join(scenarioRoot(t), "api/internal/handlers/invocation_facts.go"),
		filepath.Join(scenarioRoot(t), "api/internal/handlers/observed_receipts.go"),
	)
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if match := literal.FindString(string(source)); match != "" {
			rel, _ := filepath.Rel(scenarioRoot(t), path)
			t.Fatalf("%s introduces ad-hoc availability state %q; use internal/availability constants", filepath.ToSlash(rel), match)
		}
	}
}

func TestDurableProjectionHasOneRuntimeWriterAndNoRetiredSchema(t *testing.T) {
	root := scenarioRoot(t)
	writer := filepath.Join(root, "api/internal/adapters/database/repository_invocation_read_model.go")
	writerSource, err := os.ReadFile(writer)
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"invocation_read_model_facts", "invocation_read_model_episodes", "invocation_read_model_self_report_spans"} {
		for _, verb := range []string{"INSERT INTO " + table, "DELETE FROM " + table} {
			if !strings.Contains(string(writerSource), verb) {
				t.Fatalf("durable projection writer is missing %q", verb)
			}
		}
	}
	for _, path := range goFiles(t, filepath.Join(root, "api/internal/adapters/database")) {
		if path == writer || strings.HasSuffix(path, "_test.go") || filepath.Base(path) == "connection.go" {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"invocation_read_model_facts", "invocation_read_model_episodes", "invocation_read_model_self_report_spans"} {
			if strings.Contains(string(source), "INSERT INTO "+table) || strings.Contains(string(source), "DELETE FROM "+table) {
				rel, _ := filepath.Rel(root, path)
				t.Fatalf("%s is a second runtime writer for %s", filepath.ToSlash(rel), table)
			}
		}
	}
	schema, err := os.ReadFile(filepath.Join(root, "api/internal/domain/schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, retired := range []string{"investigation_invocation_facts", "investigation_friction_episodes", "investigation_self_report_spans"} {
		if strings.Contains(string(schema), retired) {
			t.Fatalf("retired projection table %q remains in declarative schema", retired)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "api/internal/adapters/database/repository_investigation_projection.go")); !os.IsNotExist(err) {
		t.Fatalf("legacy investigation projection repository must be deleted, stat err=%v", err)
	}
}

// Configuration is substrate: validation errors and loading mechanics must
// remain usable without importing business entities from internal/domain.
func TestConfigDoesNotImportDomain(t *testing.T) {
	violations := importsUnder(t, "api/internal/config", "agent-manager/internal/domain")
	if len(violations) != 0 {
		t.Fatalf("config must not import domain entities: %v", violations)
	}
}

// Health is a substrate observer. It owns narrow contracts for database audit
// storage and failure observations rather than importing sibling packages.
func TestHealthDoesNotImportSiblingDomains(t *testing.T) {
	for _, forbidden := range []string{
		"agent-manager/internal/domain",
		"agent-manager/internal/fallback",
		"agent-manager/internal/sqlcompat",
	} {
		violations := importsUnder(t, "api/internal/health", forbidden)
		if len(violations) != 0 {
			t.Fatalf("health must not import sibling package %q: %v", forbidden, violations)
		}
	}
}

func TestHandlersDirectPersistenceImportsAreReadSideExceptions(t *testing.T) {
	allowed := map[string]bool{
		"events.go":            true,
		"operational_stats.go": true,
		"pricing.go":           true,
	}
	for _, path := range goFiles(t, filepath.Join(scenarioRoot(t), "api/internal/handlers")) {
		imports, err := importsFromFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range imports {
			if imported != "agent-manager/internal/eventlog" && imported != "agent-manager/internal/stats" {
				continue
			}
			if !allowed[filepath.Base(path)] {
				t.Fatalf("%s imports read-side persistence %q without a documented exception", filepath.Base(path), imported)
			}
		}
	}
}

func TestHandlersUseCapabilityBoundaries(t *testing.T) {
	path := filepath.Join(scenarioRoot(t), "api/internal/handlers/handlers.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"orchestration.Service", "*orchestration.Orchestrator"} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("handlers must depend on narrow capability interfaces, found %q", forbidden)
		}
	}
	if !strings.Contains(string(source), "orchestration.HandlerServices") {
		t.Fatal("handlers must receive the composition-root capability bundle")
	}
}

func TestOnlyWiringConstructsProductionOrchestrator(t *testing.T) {
	apiRoot := filepath.Join(scenarioRoot(t), "api")
	for _, path := range goFiles(t, apiRoot) {
		rel, err := filepath.Rel(apiRoot, path)
		if err != nil {
			t.Fatal(err)
		}
		normalized := filepath.ToSlash(rel)
		if strings.Contains(normalized, "/testutil/") || strings.HasPrefix(normalized, "internal/wiring/") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(source), "orchestration.New(") {
			t.Fatalf("%s constructs the orchestrator outside internal/wiring", normalized)
		}
	}
}

func TestProductionRunStateCallsNeverUseAnEmptyRootLiteral(t *testing.T) {
	for _, path := range goFiles(t, filepath.Join(scenarioRoot(t), "api")) {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"runstate.RunDir(\"\"", "runstate.Load(\"\""} {
			if strings.Contains(string(source), forbidden) {
				rel, _ := filepath.Rel(scenarioRoot(t), path)
				t.Fatalf("%s resolves run state from an empty root: %s", filepath.ToSlash(rel), forbidden)
			}
		}
	}
}

func TestOnlyWiringMayResolveDatabaseDataDir(t *testing.T) {
	for _, path := range goFiles(t, filepath.Join(scenarioRoot(t), "api")) {
		rel, err := filepath.Rel(filepath.Join(scenarioRoot(t), "api"), path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(filepath.ToSlash(rel), "internal/wiring/") || filepath.ToSlash(rel) == "internal/adapters/database/connection.go" {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(source), "database.DataDir(") {
			t.Fatalf("%s resolves database.DataDir outside wiring", filepath.ToSlash(rel))
		}
	}
}

func TestScenarioForbidsPartNumberedSourceFiles(t *testing.T) {
	var violations []string
	err := filepath.WalkDir(scenarioRoot(t), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if matched, _ := filepath.Match("*_part[0-9]*.go", entry.Name()); matched {
			violations = append(violations, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("part-numbered source files hide their domain: %v", violations)
	}
}

func TestOrchestrationSourceFilesDeclareTheirResponsibility(t *testing.T) {
	var missing []string
	for _, path := range goFiles(t, filepath.Join(scenarioRoot(t), "api/internal/orchestration")) {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		firstLine := strings.TrimSpace(strings.SplitN(string(source), "\n", 2)[0])
		if !strings.HasPrefix(firstLine, "//") {
			rel, _ := filepath.Rel(scenarioRoot(t), path)
			missing = append(missing, filepath.ToSlash(rel))
		}
	}
	if len(missing) != 0 {
		t.Fatalf("orchestration source files need a leading responsibility comment: %v", missing)
	}
}

func TestInteractiveProductionCodeDoesNotTranslateByRunnerType(t *testing.T) {
	// Runner-native control flags belong to codecs. A runner-type switch in the
	// launch command builder is a structural drift signal: it recreates a
	// second control translation instead of asking the resolved runner for
	// ControlArgs. Transcript discovery legitimately branches by runner.
	path := filepath.Join(scenarioRoot(t), "api/internal/orchestration/interactive/launch.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "case domain.RunnerType") {
		rel, _ := filepath.Rel(scenarioRoot(t), path)
		t.Fatalf("%s switches on domain.RunnerType; use the codec control-argument seam", filepath.ToSlash(rel))
	}
}

func TestCapabilityBoundaryDetectorRejectsLegacyType(t *testing.T) {
	fixture := `package handlers; type Handler struct { svc orchestration.Service }`
	if !strings.Contains(fixture, "orchestration.Service") {
		t.Fatal("legacy handler fixture did not trigger the capability detector")
	}
}

func TestBoundaryDetectorRejectsViolatingFixtures(t *testing.T) {
	for _, tc := range []struct {
		name, source, forbidden string
	}{
		{"workflow runtime orchestration import", `package workflowruntime; import "agent-manager/internal/orchestration"`, "agent-manager/internal/orchestration"},
		{"orchestration database import", `package orchestration; import "agent-manager/internal/adapters/database"`, "agent-manager/internal/adapters/database"},
		{"config domain import", `package config; import "agent-manager/internal/domain"`, "agent-manager/internal/domain"},
		{"health fallback import", `package health; import "agent-manager/internal/fallback"`, "agent-manager/internal/fallback"},
		{"health sqlcompat import", `package health; import "agent-manager/internal/sqlcompat"`, "agent-manager/internal/sqlcompat"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			imports, err := importsFromSource(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			if !contains(imports, tc.forbidden) {
				t.Fatalf("fixture did not trigger %q: %v", tc.forbidden, imports)
			}
		})
	}
}

func TestInteractiveRunnerTypeSwitchDetectorRejectsFixture(t *testing.T) {
	fixture := `switch p.RunnerType { case domain.RunnerTypeCodex: }`
	if !strings.Contains(fixture, "case domain.RunnerType") {
		t.Fatal("runner-type switch fixture did not trigger detector")
	}
}

func importsUnder(t *testing.T, relativeDir, forbidden string) []string {
	t.Helper()
	var violations []string
	for _, path := range goFiles(t, filepath.Join(scenarioRoot(t), relativeDir)) {
		// Helpers under testutil are compiled so cross-package integration tests
		// can import them; they are test infrastructure, not runtime code.
		if strings.Contains(filepath.ToSlash(path), "/testutil/") {
			continue
		}
		imports, err := importsFromFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if contains(imports, forbidden) {
			violations = append(violations, filepath.ToSlash(path))
		}
	}
	return violations
}

func goFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func importsFromFile(path string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return importPaths(file), nil
}

func importsFromSource(source string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", source, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	return importPaths(file), nil
}

func importPaths(file *ast.File) []string {
	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		imports = append(imports, strings.Trim(spec.Path.Value, `"`))
	}
	return imports
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func scenarioRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve archtest source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
