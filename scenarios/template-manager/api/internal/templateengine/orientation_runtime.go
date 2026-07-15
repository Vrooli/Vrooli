package templateengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/config"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func runOrientation[C any](deps HandlerDeps[C], ctx C, req templatecontracts.OrientationRequest) (templatecontracts.OrientationReport, error) {
	item, err := scenariomodel.Load(deps.Root(ctx), req.Name, scenariomodel.SandboxEnvFromEnv())
	if err != nil {
		return templatecontracts.OrientationReport{}, err
	}
	report, manifest, err := evaluateOrientation(deps, ctx, item)
	if err != nil {
		return templatecontracts.OrientationReport{}, err
	}
	if !req.Finalize {
		return report, nil
	}
	if report.Finalized {
		return report, nil
	}
	if report.Completed < report.Required {
		return templatecontracts.OrientationReport{}, fmt.Errorf("orientation is incomplete for %s: %d/%d required steps complete", report.Scenario, report.Completed, report.Required)
	}
	for _, cleanup := range manifest.Finalize.Cleanup {
		clean, err := cleanScenarioRelativePath(cleanup)
		if err != nil {
			return templatecontracts.OrientationReport{}, fmt.Errorf("invalid orientation cleanup path %q: %w", cleanup, err)
		}
		if isDurableOrientationCleanupTarget(clean) {
			return templatecontracts.OrientationReport{}, fmt.Errorf("refusing to clean durable orientation path %q", cleanup)
		}
		if err := os.RemoveAll(filepath.Join(item.Path, clean)); err != nil {
			return templatecontracts.OrientationReport{}, fmt.Errorf("clean orientation path %q: %w", cleanup, err)
		}
	}
	report.Finalized = true
	report.FinalizeRequired = false
	report.Message = manifest.Finalize.Message
	if strings.TrimSpace(report.Message) == "" {
		report.Message = "Initialization complete; use normal scenario lifecycle commands."
	}
	return report, nil
}

func evaluateOrientation[C any](deps HandlerDeps[C], ctx C, item scenariomodel.Scenario) (templatecontracts.OrientationReport, templatecontracts.TemplateOrientation, error) {
	orientationPath := filepath.Join(item.Path, ".vrooli", "orientation.json")
	report := templatecontracts.OrientationReport{
		Scenario:        item.Slug,
		ScenarioPath:    item.Path,
		OrientationPath: orientationPath,
	}
	if item.Manifest.Generation != nil {
		report.Template = templatecontracts.GenerationTemplate{
			ID:      item.Manifest.Generation.Template.ID,
			Version: item.Manifest.Generation.Template.Version,
		}
		report.Design = templatecontracts.GenerationDesign{
			ID:      item.Manifest.Generation.Design.ID,
			Version: item.Manifest.Generation.Design.Version,
			Adapter: item.Manifest.Generation.Design.Adapter,
		}
	}
	data, err := os.ReadFile(orientationPath)
	if err != nil {
		if os.IsNotExist(err) {
			report.Finalized = true
			report.Message = "No orientation metadata found. Orientation has either been finalized or this scenario was not generated from an orientation-enabled template."
			return report, templatecontracts.TemplateOrientation{}, nil
		}
		return report, templatecontracts.TemplateOrientation{}, err
	}
	var manifest templatecontracts.TemplateOrientation
	if err := json.Unmarshal(data, &manifest); err != nil {
		return report, templatecontracts.TemplateOrientation{}, fmt.Errorf("parse orientation manifest: %w", err)
	}
	report.StartDocument = strings.TrimSpace(manifest.StartDocument)
	ev := orientationEval{
		scenarioRoot:       item.Path,
		templateSourceRoot: resolveTemplateSourceRoot(deps.Root(ctx), item),
	}
	for _, step := range manifest.Steps {
		stepReport := evaluateOrientationStep(deps, ctx, ev, step)
		report.Steps = append(report.Steps, stepReport)
		if !stepReport.Required {
			continue
		}
		report.Required++
		if stepReport.Complete {
			report.Completed++
		} else if report.NextStep == nil {
			copyStep := stepReport
			report.NextStep = &copyStep
		}
	}
	if report.Required > 0 && report.Completed == report.Required {
		report.FinalizeRequired = true
		report.Message = "All required orientation steps are complete."
	}
	return report, manifest, nil
}

func evaluateOrientationStep[C any](deps HandlerDeps[C], ctx C, ev orientationEval, step templatecontracts.TemplateOrientationStep) templatecontracts.OrientationStepReport {
	report := templatecontracts.OrientationStepReport{
		ID:          step.ID,
		Title:       step.Title,
		Description: step.Description,
		Docs:        append([]string(nil), step.Docs...),
		Required:    orientationStepRequired(step),
		Complete:    true,
	}
	if len(step.Checks) == 0 {
		report.Complete = false
		return report
	}
	// Declared checks first, then engine-derived content-quality companions so
	// green orientation certifies real adaptation work, not just shipped files.
	checks := append(append([]templatecontracts.TemplateOrientationCheck(nil), step.Checks...), deriveContentAdaptedChecks(step.Checks, ev)...)
	for _, check := range checks {
		checkReport := evaluateOrientationCheck(deps, ctx, ev, check)
		report.Checks = append(report.Checks, checkReport)
		if !checkReport.Passed && !checkReport.Optional {
			report.Complete = false
		}
	}
	return report
}

func evaluateOrientationCheck[C any](deps HandlerDeps[C], ctx C, ev orientationEval, check templatecontracts.TemplateOrientationCheck) templatecontracts.OrientationCheckReport {
	report := templatecontracts.OrientationCheckReport{
		Kind:     check.Kind,
		Label:    orientationCheckLabel(check),
		Optional: check.Optional,
	}
	resolve := func(path string) (string, error) {
		clean, err := cleanScenarioRelativePath(path)
		if err != nil {
			return "", err
		}
		return filepath.Join(ev.scenarioRoot, clean), nil
	}
	switch check.Kind {
	case "file_exists":
		evaluateOrientationFileExists(&report, check, resolve)
	case "file_absent":
		evaluateOrientationFileAbsent(&report, check, resolve)
	case "directory_exists":
		evaluateOrientationDirectoryExists(&report, check, resolve)
	case "glob_present", "glob_absent":
		evaluateOrientationGlobPresence(&report, check, resolve)
	case "glob_min_count":
		evaluateOrientationGlobMinCount(&report, check, resolve)
	case "text_contains", "text_absent":
		evaluateOrientationTextCondition(&report, check, resolve)
	case "text_absent_tree":
		evaluateOrientationTextAbsentTree(&report, check, ev.scenarioRoot)
	case "json_path_exists":
		evaluateOrientationJSONPathExists(&report, check, resolve)
	case "json_min_entries":
		evaluateOrientationJSONMinEntries(&report, check, resolve)
	case "content_adapted":
		evaluateOrientationContentAdapted(&report, ev, check)
	case "command":
		evaluateOrientationCommand(&report, deps, ctx, ev.scenarioRoot, check)
	default:
		report.Message = "unknown check kind"
	}
	return report
}

type orientationPathResolver func(string) (string, error)

func evaluateOrientationFileExists(report *templatecontracts.OrientationCheckReport, check templatecontracts.TemplateOrientationCheck, resolve orientationPathResolver) {
	path, err := resolve(check.Path)
	if err != nil {
		report.Message = err.Error()
		return
	}
	info, err := os.Stat(path)
	report.Passed = err == nil && !info.IsDir()
	if !report.Passed {
		report.Message = "file is missing"
	}
}

func evaluateOrientationFileAbsent(report *templatecontracts.OrientationCheckReport, check templatecontracts.TemplateOrientationCheck, resolve orientationPathResolver) {
	path, err := resolve(check.Path)
	if err != nil {
		report.Message = err.Error()
		return
	}
	_, err = os.Stat(path)
	report.Passed = os.IsNotExist(err)
	if !report.Passed {
		report.Message = "file exists"
	}
}

func evaluateOrientationDirectoryExists(report *templatecontracts.OrientationCheckReport, check templatecontracts.TemplateOrientationCheck, resolve orientationPathResolver) {
	path, err := resolve(check.Path)
	if err != nil {
		report.Message = err.Error()
		return
	}
	info, err := os.Stat(path)
	report.Passed = err == nil && info.IsDir()
	if !report.Passed {
		report.Message = "directory is missing"
	}
}

func evaluateOrientationGlobPresence(report *templatecontracts.OrientationCheckReport, check templatecontracts.TemplateOrientationCheck, resolve orientationPathResolver) {
	matches, err := orientationGlobMatches(check.Pattern, resolve)
	if err != nil {
		report.Message = err.Error()
		return
	}
	report.Passed = len(matches) > 0
	if check.Kind == "glob_absent" {
		report.Passed = len(matches) == 0
	}
	if !report.Passed {
		report.Message = fmt.Sprintf("matched %d path(s)", len(matches))
	}
}

func evaluateOrientationGlobMinCount(report *templatecontracts.OrientationCheckReport, check templatecontracts.TemplateOrientationCheck, resolve orientationPathResolver) {
	matches, err := orientationGlobMatches(check.Pattern, resolve)
	if err != nil {
		report.Message = err.Error()
		return
	}
	report.Passed = len(matches) >= check.MinCount
	if !report.Passed {
		report.Message = fmt.Sprintf("matched %d path(s), need at least %d", len(matches), check.MinCount)
	}
}

func orientationGlobMatches(pattern string, resolve orientationPathResolver) ([]string, error) {
	resolved, err := resolve(pattern)
	if err != nil {
		return nil, err
	}
	return filepath.Glob(resolved)
}

func evaluateOrientationTextCondition(report *templatecontracts.OrientationCheckReport, check templatecontracts.TemplateOrientationCheck, resolve orientationPathResolver) {
	path, err := resolve(check.Path)
	if err != nil {
		report.Message = err.Error()
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		report.Message = err.Error()
		return
	}
	contains := bytes.Contains(data, []byte(check.Text))
	report.Passed = contains
	if check.Kind == "text_absent" {
		report.Passed = !contains
	}
	if !report.Passed {
		report.Message = "text condition not satisfied"
	}
}

func evaluateOrientationTextAbsentTree(report *templatecontracts.OrientationCheckReport, check templatecontracts.TemplateOrientationCheck, scenarioRoot string) {
	hits, err := scanTreeForText(scenarioRoot, check.Text)
	if err != nil {
		report.Message = err.Error()
		return
	}
	report.Passed = len(hits) == 0
	if report.Passed {
		return
	}
	shown := hits
	if len(shown) > 5 {
		shown = shown[:5]
	}
	report.Message = fmt.Sprintf("found %q in %d file(s): %s; run `template-manager lifecycle detemplate %s`",
		check.Text, len(hits), strings.Join(shown, ", "), filepath.Base(scenarioRoot))
}

func evaluateOrientationJSONPathExists(report *templatecontracts.OrientationCheckReport, check templatecontracts.TemplateOrientationCheck, resolve orientationPathResolver) {
	value, ok, err := orientationResolvedJSONPathValue(check, resolve)
	if err != nil {
		report.Message = err.Error()
		return
	}
	report.Passed = ok && value != nil
	if !report.Passed {
		report.Message = "JSON path not found"
	}
}

func evaluateOrientationJSONMinEntries(report *templatecontracts.OrientationCheckReport, check templatecontracts.TemplateOrientationCheck, resolve orientationPathResolver) {
	value, ok, err := orientationResolvedJSONPathValue(check, resolve)
	if err != nil {
		report.Message = err.Error()
		return
	}
	if !ok {
		report.Message = "JSON path not found"
		return
	}
	count, ok := orientationJSONEntryCount(value)
	report.Passed = ok && count >= check.MinCount
	if !report.Passed {
		report.Message = orientationJSONMinEntriesMessage(ok, count, check.MinCount)
	}
}

func orientationResolvedJSONPathValue(check templatecontracts.TemplateOrientationCheck, resolve orientationPathResolver) (any, bool, error) {
	path, err := resolve(check.Path)
	if err != nil {
		return nil, false, err
	}
	return orientationJSONPathValue(path, check.Query)
}

func orientationJSONMinEntriesMessage(ok bool, count, minCount int) string {
	if !ok {
		return "JSON path is not an array or object"
	}
	return fmt.Sprintf("JSON path has %d entrie(s), need at least %d", count, minCount)
}

func evaluateOrientationCommand[C any](report *templatecontracts.OrientationCheckReport, deps HandlerDeps[C], ctx C, scenarioRoot string, check templatecontracts.TemplateOrientationCheck) {
	timeout, err := time.ParseDuration(check.Timeout)
	if err != nil {
		report.Message = "invalid timeout: " + err.Error()
		return
	}
	commandCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var stdout, stderr bytes.Buffer
	err = shell.BashCommand(check.Run, shell.Spec{
		Context: commandCtx,
		Dir:     scenarioRoot,
		Env:     deps.CommandEnv(ctx),
		Stdout:  &stdout,
		Stderr:  &stderr,
	}).Run()
	report.Passed = err == nil
	if err != nil {
		report.Message = orientationCommandFailureMessage(commandCtx, check.Timeout, stdout.String(), stderr.String(), err)
	}
}

func orientationCommandFailureMessage(ctx context.Context, timeout, stdout, stderr string, err error) string {
	if ctx.Err() == context.DeadlineExceeded {
		return "command timed out after " + timeout
	}
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = strings.TrimSpace(stdout)
	}
	if msg == "" {
		msg = err.Error()
	}
	return truncateForIssue(msg, 240)
}

func orientationCheckLabel(check templatecontracts.TemplateOrientationCheck) string {
	switch check.Kind {
	case "file_exists", "file_absent", "directory_exists":
		return check.Path
	case "glob_present", "glob_absent":
		return check.Pattern
	case "glob_min_count":
		return fmt.Sprintf("%s >= %d", check.Pattern, check.MinCount)
	case "json_path_exists":
		return strings.TrimSpace(check.Path + ":" + check.Query)
	case "json_min_entries":
		return fmt.Sprintf("%s:%s >= %d", check.Path, check.Query, check.MinCount)
	case "text_contains", "text_absent":
		return check.Path
	case "text_absent_tree":
		return check.Text
	case "content_adapted":
		return check.Path + " (adapted from template)"
	case "command":
		return check.Run
	default:
		return check.Kind
	}
}

func orientationJSONPathValue(path, query string) (any, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	var current any
	if err := json.Unmarshal(data, &current); err != nil {
		return nil, false, err
	}
	for _, part := range strings.Split(strings.TrimSpace(query), ".") {
		if part == "" {
			return nil, false, nil
		}
		switch typed := current.(type) {
		case map[string]any:
			value, ok := typed[part]
			if !ok {
				return nil, false, nil
			}
			current = value
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false, nil
			}
			current = typed[index]
		default:
			return nil, false, nil
		}
	}
	return current, true, nil
}

func orientationJSONEntryCount(value any) (int, bool) {
	switch typed := value.(type) {
	case []any:
		return len(typed), true
	case map[string]any:
		return len(typed), true
	default:
		return 0, false
	}
}

// orientationEval carries the per-scenario evaluation context. templateSourceRoot
// is the on-disk source directory of the scenario's originating template (empty
// when the scenario was not generated from a discoverable template), which lets
// content-quality checks measure divergence from the template scaffold.
type orientationEval struct {
	scenarioRoot       string
	templateSourceRoot string
}

// resolveTemplateSourceRoot maps a generated scenario back to its template source
// directory using the recorded generation manifest. Returns "" when the scenario
// has no generation metadata or the template is no longer present.
func resolveTemplateSourceRoot(root string, item scenariomodel.Scenario) string {
	if item.Manifest.Generation == nil {
		return ""
	}
	id := strings.TrimSpace(item.Manifest.Generation.Template.ID)
	if id == "" {
		return ""
	}
	candidate := filepath.Join(config.TemplateBaseDir(root), id)
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}

// deriveContentAdaptedChecks upgrades the meaning of declared placeholder-removal
// gates: for every declared text_absent check whose target file the template also
// ships, the engine adds a content_adapted companion. A text_absent marker means
// "the template ships replaceable placeholder text here", so the engine also
// requires the file to carry materially adapted content — closing the loophole
// where deleting the marker line alone flips the gate green. This makes react-vite
// (and any template) content-aware without editing template.json. Derivation is
// skipped entirely when the template source is unavailable, and never duplicates a
// content_adapted check the template already declares for the same path.
func deriveContentAdaptedChecks(checks []templatecontracts.TemplateOrientationCheck, ev orientationEval) []templatecontracts.TemplateOrientationCheck {
	if ev.templateSourceRoot == "" {
		return nil
	}
	seen := map[string]struct{}{}
	for _, check := range checks {
		if check.Kind != "content_adapted" {
			continue
		}
		if clean, err := cleanScenarioRelativePath(check.Path); err == nil {
			seen[clean] = struct{}{}
		}
	}
	var derived []templatecontracts.TemplateOrientationCheck
	for _, check := range checks {
		if check.Kind != "text_absent" {
			continue
		}
		clean, err := cleanScenarioRelativePath(check.Path)
		if err != nil {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		if !orientationRegularFileExists(filepath.Join(ev.templateSourceRoot, clean)) {
			continue
		}
		seen[clean] = struct{}{}
		derived = append(derived, templatecontracts.TemplateOrientationCheck{Kind: "content_adapted", Path: check.Path})
	}
	return derived
}

func evaluateOrientationContentAdapted(report *templatecontracts.OrientationCheckReport, ev orientationEval, check templatecontracts.TemplateOrientationCheck) {
	clean, err := cleanScenarioRelativePath(check.Path)
	if err != nil {
		report.Message = err.Error()
		return
	}
	if ev.templateSourceRoot == "" {
		report.Passed = true
		report.Skipped = true
		report.Message = "no generation template recorded; content adaptation not evaluated"
		return
	}
	templateFile := filepath.Join(ev.templateSourceRoot, clean)
	if !orientationRegularFileExists(templateFile) {
		report.Passed = true
		report.Skipped = true
		report.Message = "template does not ship this file; content adaptation not evaluated"
		return
	}
	netNew, err := templateNetNewContentLines(templateFile, filepath.Join(ev.scenarioRoot, clean))
	if err != nil {
		report.Message = err.Error()
		return
	}
	threshold := contentAdaptedThreshold(clean, check.MinCount)
	report.Passed = netNew >= threshold
	if !report.Passed {
		report.Message = fmt.Sprintf("only %d line(s) of scenario-specific content beyond the template scaffold; need at least %d — replace the generated boilerplate with real content", netNew, threshold)
	}
}

// contentAdaptedThreshold returns the minimum number of scenario-specific content
// lines required. JSON registries legitimately keep most structural scaffolding
// and are oriented by re-pointing a small number of values, so they require only
// one adapted line; prose/markdown files are meant to be rewritten and require
// more. A template-declared minCount override always wins.
func contentAdaptedThreshold(relPath string, override int) int {
	if override > 0 {
		return override
	}
	if strings.EqualFold(filepath.Ext(relPath), ".json") {
		return 1
	}
	return 3
}

func orientationRegularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

var orientationPlaceholderPattern = regexp.MustCompile(`\{\{[^}]*\}\}`)

type templateLineMatcher struct {
	exact    string
	segments []string
}

// templateNetNewContentLines counts scenario content lines that are NOT derived
// from the template source. Matching is placeholder-aware: a scenario line is
// template-derived when it exactly equals a plain template line, or contains a
// placeholder template line's literal segments in order, so generator variable
// substitution never registers as adaptation. A missing scenario file yields 0.
func templateNetNewContentLines(templateFile, scenarioFile string) (int, error) {
	templateData, err := os.ReadFile(templateFile)
	if err != nil {
		return 0, err
	}
	scenarioData, err := os.ReadFile(scenarioFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	matchers := buildTemplateLineMatchers(string(templateData))
	netNew := 0
	for _, raw := range strings.Split(string(scenarioData), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if !templateLineMatches(line, matchers) {
			netNew++
		}
	}
	return netNew, nil
}

func buildTemplateLineMatchers(content string) []templateLineMatcher {
	var matchers []templateLineMatcher
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.Contains(line, "{{") {
			segments := placeholderLiteralSegments(line)
			// Pure-placeholder lines (no literal anchor) are dropped so they
			// never wildcard-match unrelated scenario content.
			if len(segments) == 0 {
				continue
			}
			matchers = append(matchers, templateLineMatcher{segments: segments})
			continue
		}
		matchers = append(matchers, templateLineMatcher{exact: line})
	}
	return matchers
}

func placeholderLiteralSegments(line string) []string {
	var segments []string
	for _, part := range orientationPlaceholderPattern.Split(line, -1) {
		if part != "" {
			segments = append(segments, part)
		}
	}
	return segments
}

func templateLineMatches(line string, matchers []templateLineMatcher) bool {
	for _, matcher := range matchers {
		if len(matcher.segments) == 0 {
			if line == matcher.exact {
				return true
			}
			continue
		}
		idx := 0
		matched := true
		for _, segment := range matcher.segments {
			offset := strings.Index(line[idx:], segment)
			if offset < 0 {
				matched = false
				break
			}
			idx += offset + len(segment)
		}
		if matched {
			return true
		}
	}
	return false
}
