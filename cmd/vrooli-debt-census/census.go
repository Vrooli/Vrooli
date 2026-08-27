package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

const (
	budgetSchemaVersion             = 1
	metricOK            metricState = "ok"
	metricFailed        metricState = "failed"
)

type metricState string

// Metric describes one independently measured debt signal.
type Metric struct {
	Name        string
	Description string
	BudgetKey   string
	Phase       string
	Measure     func(root string) (MetricResult, error)
}

// MetricResult is the typed result of one measurement. Value is meaningful
// only when State is ok; zero is therefore distinguishable from failure.
type MetricResult struct {
	Value int
	State metricState
	Error error
}

type metricResult struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Value       *int        `json:"value,omitempty"`
	Budget      int         `json:"budget"`
	Delta       *int        `json:"delta"`
	Phase       string      `json:"phase"`
	Status      metricState `json:"state"`
	Error       string      `json:"error,omitempty"`
}

type censusDocument struct {
	SchemaVersion int            `json:"schemaVersion"`
	Metrics       []metricResult `json:"metrics"`
}

type budgetDocument struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Ratchet       bool                   `json:"ratchet"`
	Metrics       map[string]budgetEntry `json:"metrics"`
}

type budgetEntry struct {
	Budget          int  `json:"budget"`
	Baseline        int  `json:"baseline"`
	BudgetPresent   bool `json:"-"`
	BaselinePresent bool `json:"-"`
}

func (entry *budgetEntry) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("metric budget must be an object: %w", err)
	}
	for field := range fields {
		if field != "budget" && field != "baseline" {
			return fmt.Errorf("unknown metric budget property %q", field)
		}
	}
	budgetRaw, budgetOK := fields["budget"]
	baselineRaw, baselineOK := fields["baseline"]
	if !budgetOK || !baselineOK {
		return errors.New("metric budget requires both budget and baseline")
	}
	if err := json.Unmarshal(budgetRaw, &entry.Budget); err != nil {
		return fmt.Errorf("budget must be an integer: %w", err)
	}
	if err := json.Unmarshal(baselineRaw, &entry.Baseline); err != nil {
		return fmt.Errorf("baseline must be an integer: %w", err)
	}
	entry.BudgetPresent = true
	entry.BaselinePresent = true
	return nil
}

func (budget *budgetDocument) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("budget document must be an object: %w", err)
	}
	for field := range fields {
		if field != "schemaVersion" && field != "ratchet" && field != "metrics" {
			return fmt.Errorf("unknown budget document property %q", field)
		}
	}
	var decoded struct {
		SchemaVersion int                    `json:"schemaVersion"`
		Ratchet       *bool                  `json:"ratchet"`
		Metrics       map[string]budgetEntry `json:"metrics"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	budget.SchemaVersion = decoded.SchemaVersion
	if decoded.Ratchet == nil {
		return errors.New("budget document requires ratchet")
	}
	budget.Ratchet = *decoded.Ratchet
	budget.Metrics = decoded.Metrics
	return nil
}

func metrics() []Metric {
	return []Metric{
		{Name: "os_rename_outside_config", Description: "non-test files under internal outside config calling os.Rename", BudgetKey: "os_rename_outside_config", Phase: "Phase 3", Measure: measureOSRename},
		{Name: "fprintf_tab_rows", Description: "non-test Fprintf calls with two or more literal tab separators", BudgetKey: "fprintf_tab_rows", Phase: "Phase 4", Measure: measureFprintfTabRows},
		{Name: "private_json_writers", Description: "non-test functions outside cliout constructing dynamic protobuf JSON values", BudgetKey: "private_json_writers", Phase: "Phase 5", Measure: measurePrivateJSONWriters},
		{Name: "bind_stanzas", Description: "non-test calls to rootcli.BindGlobalCommand", BudgetKey: "bind_stanzas", Phase: "Phase 6", Measure: measureBindStanzas},
		{Name: "hand_written_dispatchers", Description: "non-test switch args[0] dispatchers under internal/app", BudgetKey: "hand_written_dispatchers", Phase: "Phase 7", Measure: measureHandWrittenDispatchers},
		{Name: "duplicate_runner_interfaces", Description: "non-test interfaces outside shell declaring LookPath", BudgetKey: "duplicate_runner_interfaces", Phase: "Phase 8", Measure: measureDuplicateRunnerInterfaces},
		{Name: "test_lookpath_fakes", Description: "test methods outside shell named LookPath", BudgetKey: "test_lookpath_fakes", Phase: "Phase 8", Measure: measureTestLookPathFakes},
		{Name: "inline_shell_heredocs", Description: "test lines outside shelltest containing shell shebangs", BudgetKey: "inline_shell_heredocs", Phase: "Phase 8", Measure: measureInlineShellHeredocs},
		{Name: "stamped_handler_tests", Description: "packages defining the stamped TestNameAndKind", BudgetKey: "stamped_handler_tests", Phase: "Phase 10", Measure: measureStampedHandlerTests},
		{Name: "bare_vrooli_path_literals", Description: "non-test .vrooli and scenarios path literals", BudgetKey: "bare_vrooli_path_literals", Phase: "Phase 12", Measure: measureBarePathLiterals},
		{Name: "unnamed_duration_literals", Description: "non-test duration multiplications outside tuning", BudgetKey: "unnamed_duration_literals", Phase: "Phase 13", Measure: measureUnnamedDurationLiterals},
		{Name: "bare_octal_file_modes", Description: "non-test octal numeric literals outside tuning", BudgetKey: "bare_octal_file_modes", Phase: "Phase 13", Measure: measureBareOctalLiterals},
		{Name: "private_binder_dialects", Description: "private binder declarations under internal/cli", BudgetKey: "private_binder_dialects", Phase: "Phase 5", Measure: measurePrivateBinderDialects},
		{Name: "private_binder_call_sites", Description: "calls to private binder declarations", BudgetKey: "private_binder_call_sites", Phase: "Phase 5", Measure: measurePrivateBinderCallSites},
		{Name: "private_json_writers_by_name", Description: "private functions named write*JSON", BudgetKey: "private_json_writers_by_name", Phase: "Phase 6", Measure: measurePrivateJSONWritersByName},
		{Name: "render_format_prologues", Description: "comparisons against cliout.FormatJSON", BudgetKey: "render_format_prologues", Phase: "Phase 6", Measure: measureRenderFormatPrologues},
		{Name: "handler_dirs_without_tests", Description: "tool and safeguard handler directories without Go tests", BudgetKey: "handler_dirs_without_tests", Phase: "Phase 11", Measure: measureHandlerDirsWithoutTests},
		{Name: "handler_dirs_without_conformance_suite", Description: "tested handler directories without hostreq conformance", BudgetKey: "handler_dirs_without_conformance_suite", Phase: "Phase 11", Measure: measureHandlerDirsWithoutConformanceSuite},
		{Name: "appeasement_number_constants", Description: "mnd-number constants retained by name", BudgetKey: "appeasement_number_constants", Phase: "Phase 10", Measure: func(root string) (MetricResult, error) {
			return measureNamedConstants(root, true)
		}},
		{Name: "appeasement_string_constants", Description: "literal-named string constants retained by name", BudgetKey: "appeasement_string_constants", Phase: "Phase 10", Measure: func(root string) (MetricResult, error) {
			return measureNamedConstants(root, false)
		}},
		{Name: "value_named_duration_constants", Description: "duration constants named only by their value", BudgetKey: "value_named_duration_constants", Phase: "Phase 8", Measure: measureValueNamedDurationConstants},
		{Name: "targets_without_budget", Description: "declared control-plane and project targets without testing budgets", BudgetKey: "targets_without_budget", Phase: "Phase 13", Measure: measureTargetsWithoutBudget},
		{Name: "phase_providers_without_control_plane", Description: "scenario phase providers lacking control-plane target kind", BudgetKey: "phase_providers_without_control_plane", Phase: "Phase 13", Measure: measurePhaseProvidersWithoutControlPlane},
		{Name: "flag_flagset_call_sites", Description: "direct flag.NewFlagSet call sites under internal", BudgetKey: "flag_flagset_call_sites", Phase: "Phase 9", Measure: measureFlagFlagSetCallSites},
	}
}

func loadBudget(path string) (budgetDocument, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return budgetDocument{}, err
	}
	var budget budgetDocument
	if err := json.Unmarshal(raw, &budget); err != nil {
		return budgetDocument{}, err
	}
	if budget.SchemaVersion != budgetSchemaVersion {
		return budgetDocument{}, fmt.Errorf("unsupported debt budget schema version %d", budget.SchemaVersion)
	}
	if len(budget.Metrics) == 0 {
		return budgetDocument{}, errors.New("budget document requires metrics")
	}
	for name, entry := range budget.Metrics {
		if entry.Budget < 0 || entry.Baseline < 0 {
			return budgetDocument{}, fmt.Errorf("metric %s budget and baseline must be non-negative", name)
		}
	}
	known := make(map[string]struct{}, len(metrics()))
	for _, metric := range metrics() {
		known[metric.BudgetKey] = struct{}{}
		if _, ok := budget.Metrics[metric.BudgetKey]; !ok {
			return budgetDocument{}, fmt.Errorf("budget document is missing metric %s", metric.BudgetKey)
		}
	}
	for name := range budget.Metrics {
		if _, ok := known[name]; !ok {
			return budgetDocument{}, fmt.Errorf("budget document contains unknown metric %s", name)
		}
	}
	return budget, nil
}

func collect(root string, budget budgetDocument) (censusDocument, []string) {
	doc := censusDocument{SchemaVersion: budgetSchemaVersion, Metrics: make([]metricResult, 0, len(metrics()))}
	var failures []string
	for _, metric := range metrics() {
		entry, ok := budget.Metrics[metric.BudgetKey]
		if !ok {
			failures = append(failures, fmt.Sprintf("%s has no budget entry", metric.Name))
			continue
		}
		measurement, err := metric.Measure(root)
		result := metricResult{Name: metric.Name, Description: metric.Description, Budget: entry.Budget, Phase: metric.Phase, Status: measurement.State}
		if measurement.State != metricOK {
			if measurement.Error != nil {
				result.Error = measurement.Error.Error()
			} else if err != nil {
				result.Error = err.Error()
			}
			if result.Error == "" {
				result.Error = "measurement did not complete"
			}
			doc.Metrics = append(doc.Metrics, result)
			continue
		}
		if err != nil {
			result.Status = metricFailed
			result.Error = err.Error()
			failures = append(failures, fmt.Sprintf("%s failed: %v", metric.Name, err))
			doc.Metrics = append(doc.Metrics, result)
			continue
		}
		value := measurement.Value
		result.Value = &value
		delta := value - entry.Budget
		result.Delta = &delta
		failures = append(failures, ratchetFailures(metric, entry, value, budget.Ratchet)...)
		doc.Metrics = append(doc.Metrics, result)
	}
	return doc, failures
}

func ratchetFailures(metric Metric, entry budgetEntry, value int, ratchet bool) []string {
	var failures []string
	if value > entry.Budget {
		failures = append(failures, fmt.Sprintf("%s=%d exceeds budget=%d baseline=%d delta=%d (lowered by %s)", metric.Name, value, entry.Budget, entry.Baseline, value-entry.Budget, metric.Phase))
	}
	if ratchet && entry.Budget > entry.Baseline {
		failures = append(failures, fmt.Sprintf("%s ratchet_loosened_budget: budget=%d baseline=%d delta=%d", metric.Name, entry.Budget, entry.Baseline, entry.Budget-entry.Baseline))
	}
	if ratchet && value > entry.Baseline {
		failures = append(failures, fmt.Sprintf("%s ratchet_worsened_debt: value=%d baseline=%d delta=%d (lowered by %s)", metric.Name, value, entry.Baseline, value-entry.Baseline, metric.Phase))
	}
	return failures
}

func failedMeasurementFailures(doc censusDocument, ciMode bool) []string {
	if !ciMode {
		return nil
	}
	var failures []string
	for _, metric := range doc.Metrics {
		if metric.Status != metricOK {
			failures = append(failures, fmt.Sprintf("%s %s in CI: %s", metric.Name, metric.Status, metric.Error))
		}
	}
	return failures
}

func measurementOK(value int) MetricResult {
	return MetricResult{Value: value, State: metricOK}
}

func measurementError(err error) MetricResult { return MetricResult{State: metricFailed, Error: err} }

type parsedFile struct {
	path string
	file *ast.File
	fset *token.FileSet
}

func goFiles(root string, includeTests bool, excludeDir string) ([]parsedFile, error) {
	return goFilesAt(root, "internal", includeTests, excludeDir)
}

func goFilesAt(root, relativeBase string, includeTests bool, excludeDir string) ([]parsedFile, error) {
	base := filepath.Join(root, filepath.FromSlash(relativeBase))
	var files []parsedFile
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if excludeDir != "" && path == filepath.Join(base, filepath.FromSlash(excludeDir)) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || (!includeTests && strings.HasSuffix(path, "_test.go")) {
			return nil
		}
		if excludeDir != "" && strings.HasPrefix(path, filepath.Join(base, filepath.FromSlash(excludeDir))+string(filepath.Separator)) {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		files = append(files, parsedFile{path: path, file: file, fset: fset})
		return nil
	})
	return files, err
}

func selectorIs(expr ast.Expr, pkg, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && ident.Name == pkg && selector.Sel.Name == name
}

func callSelector(call *ast.CallExpr, pkg, name string) bool { return selectorIs(call.Fun, pkg, name) }

func measureOSRename(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "config")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok && callSelector(call, "os", "Rename") && !hasForbidigoDirective(parsed, call.Pos()) {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func hasForbidigoDirective(parsed parsedFile, position token.Pos) bool {
	line := parsed.fset.Position(position).Line
	raw, err := os.ReadFile(parsed.path)
	if err != nil {
		return false
	}
	lines := strings.Split(string(raw), "\n")
	if line > 0 && line <= len(lines) && strings.Contains(lines[line-1], "nolint:forbidigo") {
		return true
	}
	return false
}

func measureFprintfTabRows(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || !selectorIs(call.Fun, "fmt", "Fprintf") || len(call.Args) < 2 {
				return true
			}
			literal, ok := call.Args[1].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			format, err := strconv.Unquote(literal.Value)
			if err == nil && strings.Count(format, "\t") >= 2 {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func measurePrivateJSONWriters(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "cliout")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			decl, ok := node.(*ast.FuncDecl)
			if !ok || decl.Body == nil {
				return true
			}
			found := false
			ast.Inspect(decl.Body, func(inner ast.Node) bool {
				call, ok := inner.(*ast.CallExpr)
				if ok && (selectorIs(call.Fun, "structpb", "NewValue") || selectorIs(call.Fun, "structpb", "NewStruct")) {
					found = true
				}
				return !found
			})
			if found {
				count++
			}
			return false
		})
	}
	return measurementOK(count), nil
}

func measureBindStanzas(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok && selectorIs(call.Fun, "rootcli", "BindGlobalCommand") {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func isZeroIndex(expr ast.Expr) bool {
	index, ok := expr.(*ast.IndexExpr)
	if !ok {
		return false
	}
	ident, ok := index.X.(*ast.Ident)
	if !ok || ident.Name != "args" {
		return false
	}
	literal, ok := index.Index.(*ast.BasicLit)
	return ok && literal.Kind == token.INT && literal.Value == "0"
}

func measureHandWrittenDispatchers(root string) (MetricResult, error) {
	base := filepath.Join(root, "internal", "app")
	count := 0
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switchStmt, ok := node.(*ast.SwitchStmt)
			if ok && isZeroIndex(switchStmt.Tag) {
				count++
			}
			return true
		})
		return nil
	})
	if err != nil {
		return measurementError(err), err
	}
	return measurementOK(count), nil
}

func measureDuplicateRunnerInterfaces(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "shell")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			iface, ok := spec.Type.(*ast.InterfaceType)
			if !ok {
				return true
			}
			for _, field := range iface.Methods.List {
				if len(field.Names) == 1 && field.Names[0].Name == "LookPath" {
					count++
					break
				}
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func measureTestLookPathFakes(root string) (MetricResult, error) {
	files, err := goFiles(root, true, "shell")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		if !strings.HasSuffix(parsed.path, "_test.go") {
			continue
		}
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			decl, ok := node.(*ast.FuncDecl)
			if ok && decl.Recv != nil && decl.Name.Name == "LookPath" {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func measureInlineShellHeredocs(root string) (MetricResult, error) {
	files, err := goFiles(root, true, "shell/shelltest")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		if !strings.HasSuffix(parsed.path, "_test.go") {
			continue
		}
		file, err := os.Open(parsed.path)
		if err != nil {
			return measurementError(err), err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "#!/bin/sh") || strings.Contains(line, "#!/bin/bash") || strings.Contains(line, "#!/usr/bin/env bash") {
				count++
			}
		}
		closeErr := file.Close()
		if err := scanner.Err(); err != nil {
			return measurementError(err), err
		}
		if closeErr != nil {
			return measurementError(closeErr), closeErr
		}
	}
	return measurementOK(count), nil
}

func measureStampedHandlerTests(root string) (MetricResult, error) {
	files, err := goFiles(root, true, "")
	if err != nil {
		return measurementError(err), err
	}
	packages := map[string]struct{}{}
	for _, parsed := range files {
		if !strings.HasSuffix(parsed.path, "_test.go") {
			continue
		}
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			decl, ok := node.(*ast.FuncDecl)
			if ok && decl.Recv == nil && decl.Name.Name == "TestNameAndKind" {
				packages[filepath.Dir(parsed.path)] = struct{}{}
			}
			return true
		})
	}
	return measurementOK(len(packages)), nil
}

func measureBarePathLiterals(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		// These two files are the contract-owned declarations/serialization
		// boundary, not consumer-owned path literals. Keep their canonical
		// values centralized while measuring path construction by consumers.
		if strings.HasSuffix(filepath.ToSlash(parsed.path), "/internal/repocontractmeta/paths.go") ||
			strings.HasSuffix(filepath.ToSlash(parsed.path), "/internal/app/hygiene/types.go") {
			continue
		}
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err == nil && (value == ".vrooli" || value == "scenarios") {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func measureUnnamedDurationLiterals(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "tuning")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Walk(durationLiteralVisitor{count: &count}, parsed.file)
	}
	return measurementOK(count), nil
}

type durationLiteralVisitor struct {
	count   *int
	inConst bool
}

func (v durationLiteralVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	if declaration, ok := node.(*ast.GenDecl); ok && declaration.Tok == token.CONST {
		return durationLiteralVisitor{count: v.count, inConst: true}
	}
	binary, ok := node.(*ast.BinaryExpr)
	if !ok || v.inConst || binary.Op != token.MUL {
		return v
	}
	_, leftNumber := binary.X.(*ast.BasicLit)
	if leftNumber && (selectorIs(binary.Y, "time", "Second") || selectorIs(binary.Y, "time", "Minute") || selectorIs(binary.Y, "time", "Hour") || selectorIs(binary.Y, "time", "Millisecond") || selectorIs(binary.Y, "time", "Microsecond") || selectorIs(binary.Y, "time", "Nanosecond")) {
		(*v.count)++
	}
	return v
}

func measureBareOctalLiterals(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "tuning")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.INT {
				return true
			}
			value := literal.Value
			if strings.HasPrefix(value, "0o") || (len(value) > 1 && value[0] == '0' && !strings.HasPrefix(value, "0x") && !strings.HasPrefix(value, "0b")) {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func measurePrivateBinderDialects(root string) (MetricResult, error) {
	files, err := goFilesAt(root, "internal/cli", false, "rootcli")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			decl, ok := node.(*ast.FuncDecl)
			if ok && decl.Recv == nil && isPrivateBinderName(decl.Name.Name) {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func measurePrivateBinderCallSites(root string) (MetricResult, error) {
	files, err := goFilesAt(root, "internal/cli", false, "rootcli")
	if err != nil {
		return measurementError(err), err
	}
	declarations := make(map[string]struct{})
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			decl, ok := node.(*ast.FuncDecl)
			if ok && decl.Recv == nil && isPrivateBinderName(decl.Name.Name) {
				declarations[decl.Name.Name] = struct{}{}
			}
			return true
		})
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch function := call.Fun.(type) {
			case *ast.Ident:
				name = function.Name
			case *ast.SelectorExpr:
				name = function.Sel.Name
			}
			if _, ok := declarations[name]; ok {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func isPrivateBinderName(name string) bool {
	return len(name) > len("bind") && strings.HasPrefix(name, "bind") && unicode.IsUpper(rune(name[len("bind")]))
}

func measurePrivateJSONWritersByName(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "cliout")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			decl, ok := node.(*ast.FuncDecl)
			if ok && decl.Recv == nil && isJSONWriterName(decl.Name.Name) {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func isJSONWriterName(name string) bool {
	return len(name) > len("writeJSON") && strings.HasPrefix(name, "write") && strings.HasSuffix(name, "JSON") && unicode.IsUpper(rune(name[len("write")]))
}

func measureRenderFormatPrologues(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "cliout")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			binary, ok := node.(*ast.BinaryExpr)
			if !ok || binary.Op != token.EQL {
				return true
			}
			if selectorIs(binary.X, "cliout", "FormatJSON") || selectorIs(binary.Y, "cliout", "FormatJSON") {
				count++
			}
			return true
		})
	}
	return measurementOK(count), nil
}

func handlerDirs(root string) ([]string, error) {
	var dirs []string
	for _, relative := range []string{"internal/tools", "internal/safeguards"} {
		entries, err := os.ReadDir(filepath.Join(root, relative))
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, filepath.Join(root, relative, entry.Name()))
			}
		}
	}
	return dirs, nil
}

func directoryHasGoTest(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
			return true, nil
		}
	}
	return false, nil
}

func measureHandlerDirsWithoutTests(root string) (MetricResult, error) {
	dirs, err := handlerDirs(root)
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, dir := range dirs {
		hasTest, err := directoryHasGoTest(dir)
		if err != nil {
			return measurementError(err), err
		}
		if !hasTest {
			count++
		}
	}
	return measurementOK(count), nil
}

func measureHandlerDirsWithoutConformanceSuite(root string) (MetricResult, error) {
	dirs, err := handlerDirs(root)
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, dir := range dirs {
		hasTest, err := directoryHasGoTest(dir)
		if err != nil {
			return measurementError(err), err
		}
		if !hasTest {
			continue
		}
		found := false
		err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() && strings.HasSuffix(path, ".go") {
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				found = found || strings.Contains(string(data), "hostreqkittest")
			}
			return nil
		})
		if err != nil {
			return measurementError(err), err
		}
		if !found {
			count++
		}
	}
	return measurementOK(count), nil
}

func measureNamedConstants(root string, number bool) (MetricResult, error) {
	files, err := goFiles(root, false, "")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		for _, declaration := range parsed.file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				valueSpec := spec.(*ast.ValueSpec)
				for index, name := range valueSpec.Names {
					if number && isAppeasementNumberName(name.Name) {
						count++
						continue
					}
					if !number && isAppeasementStringName(name.Name, valueSpec, index) {
						count++
					}
				}
			}
		}
	}
	return measurementOK(count), nil
}

func isAppeasementNumberName(name string) bool {
	if !strings.HasPrefix(name, "mnd") || !strings.Contains(name, "NumberValue") {
		return false
	}
	return unicode.IsUpper(rune(name[3])) && strings.TrimLeft(name[strings.Index(name, "NumberValue")+len("NumberValue"):], "0123456789") == ""
}

func isAppeasementStringName(name string, valueSpec *ast.ValueSpec, index int) bool {
	if len(name) <= len("literal") || !strings.HasPrefix(name, "literal") || !unicode.IsUpper(rune(name[len("literal")])) || index >= len(valueSpec.Values) {
		return false
	}
	literal, ok := valueSpec.Values[index].(*ast.BasicLit)
	return ok && literal.Kind == token.STRING
}

func measureValueNamedDurationConstants(root string) (MetricResult, error) {
	files, err := goFilesAt(root, "internal/tuning", false, "")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		for _, declaration := range parsed.file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				for _, name := range spec.(*ast.ValueSpec).Names {
					if isValueNamedDuration(name.Name) {
						count++
					}
				}
			}
		}
	}
	return measurementOK(count), nil
}

func isValueNamedDuration(name string) bool {
	if !strings.HasPrefix(name, "Duration") {
		return false
	}
	index := len("Duration")
	for index < len(name) && name[index] >= '0' && name[index] <= '9' {
		index++
	}
	if index == len("Duration") || index == len(name) {
		return false
	}
	return name[index:] == "ms" || name[index:] == "s" || name[index:] == "m" || name[index:] == "h" || name[index:] == "d"
}

func measureTargetsWithoutBudget(root string) (MetricResult, error) {
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "repo-contract.json"))
	if err != nil {
		return measurementError(err), err
	}
	var contract struct {
		Targets struct {
			Kinds map[string]struct {
				Roots []string `json:"roots"`
			} `json:"kinds"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, kind := range []string{"control-plane", "project"} {
		for _, pattern := range contract.Targets.Kinds[kind].Roots {
			matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
			if err != nil {
				return measurementError(err), err
			}
			for _, match := range matches {
				if _, err := os.Stat(filepath.Join(match, ".vrooli", "testing.json")); errors.Is(err, os.ErrNotExist) {
					count++
				} else if err != nil {
					return measurementError(err), err
				}
			}
		}
	}
	return measurementOK(count), nil
}

func measurePhaseProvidersWithoutControlPlane(root string) (MetricResult, error) {
	entries, err := os.ReadDir(filepath.Join(root, "scenarios"))
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, "scenarios", entry.Name(), ".vrooli", "test-genie.json")
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return measurementError(err), err
		}
		var descriptor struct {
			Kinds   []string `json:"kinds"`
			Targets struct {
				Kinds []string `json:"kinds"`
			} `json:"targets"`
		}
		if err := json.Unmarshal(data, &descriptor); err != nil {
			return measurementError(err), err
		}
		kinds := descriptor.Kinds
		if len(kinds) == 0 {
			kinds = descriptor.Targets.Kinds
		}
		if !contains(kinds, "control-plane") {
			count++
		}
	}
	return measurementOK(count), nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func measureFlagFlagSetCallSites(root string) (MetricResult, error) {
	files, err := goFiles(root, false, "")
	if err != nil {
		return measurementError(err), err
	}
	count := 0
	for _, parsed := range files {
		found := false
		ast.Inspect(parsed.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if ok && selectorIs(call.Fun, "flag", "NewFlagSet") {
				found = true
			}
			return true
		})
		if found {
			count++
		}
	}
	return measurementOK(count), nil
}
