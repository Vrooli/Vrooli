package templateengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
	. "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
)

func OrientationHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return bindGlobal(deps.Stdout,
		func(ctx C, args []string) (OrientationRequest, error) {
			return ParseOrientationRequest(deps.Globals(ctx).JSON, args)
		},
		func(ctx C, req OrientationRequest) (cliout.Format, OrientationReport, error) {
			return runOrientation(deps, ctx, req)
		},
		RenderOrientationResponse,
	)
}

func runOrientation[C any](deps HandlerDeps[C], ctx C, req OrientationRequest) (cliout.Format, OrientationReport, error) {
	format := cliout.FormatHuman
	if req.JSON {
		format = cliout.FormatJSON
	}
	item, err := scenariomodel.Load(deps.Root(ctx), req.Name, scenariomodel.SandboxEnvFromEnv())
	if err != nil {
		return "", OrientationReport{}, err
	}
	report, manifest, err := evaluateOrientation(deps, ctx, item)
	if err != nil {
		return "", OrientationReport{}, err
	}
	if !req.Finalize {
		return format, report, nil
	}
	if report.Finalized {
		return format, report, nil
	}
	if report.Completed < report.Required {
		return "", OrientationReport{}, fmt.Errorf("orientation is incomplete for %s: %d/%d required steps complete", report.Scenario, report.Completed, report.Required)
	}
	for _, cleanup := range manifest.Finalize.Cleanup {
		clean, err := cleanScenarioRelativePath(cleanup)
		if err != nil {
			return "", OrientationReport{}, fmt.Errorf("invalid orientation cleanup path %q: %w", cleanup, err)
		}
		if isDurableOrientationCleanupTarget(clean) {
			return "", OrientationReport{}, fmt.Errorf("refusing to clean durable orientation path %q", cleanup)
		}
		if err := os.RemoveAll(filepath.Join(item.Path, clean)); err != nil {
			return "", OrientationReport{}, fmt.Errorf("clean orientation path %q: %w", cleanup, err)
		}
	}
	report.Finalized = true
	report.FinalizeRequired = false
	report.Message = manifest.Finalize.Message
	if strings.TrimSpace(report.Message) == "" {
		report.Message = "Initialization complete; use normal scenario lifecycle commands."
	}
	return format, report, nil
}

func evaluateOrientation[C any](deps HandlerDeps[C], ctx C, item scenariomodel.Scenario) (OrientationReport, TemplateOrientation, error) {
	orientationPath := filepath.Join(item.Path, ".vrooli", "orientation.json")
	report := OrientationReport{
		Scenario:        item.Slug,
		ScenarioPath:    item.Path,
		OrientationPath: orientationPath,
	}
	if item.Manifest.Generation != nil {
		report.Template = GenerationTemplate{
			ID:      item.Manifest.Generation.Template.ID,
			Version: item.Manifest.Generation.Template.Version,
		}
		report.Design = GenerationDesign{
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
			return report, TemplateOrientation{}, nil
		}
		return report, TemplateOrientation{}, err
	}
	var manifest TemplateOrientation
	if err := json.Unmarshal(data, &manifest); err != nil {
		return report, TemplateOrientation{}, fmt.Errorf("parse orientation manifest: %w", err)
	}
	report.StartDocument = strings.TrimSpace(manifest.StartDocument)
	for _, step := range manifest.Steps {
		stepReport := evaluateOrientationStep(deps, ctx, item.Path, step)
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

func evaluateOrientationStep[C any](deps HandlerDeps[C], ctx C, scenarioRoot string, step TemplateOrientationStep) OrientationStepReport {
	report := OrientationStepReport{
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
	for _, check := range step.Checks {
		checkReport := evaluateOrientationCheck(deps, ctx, scenarioRoot, check)
		report.Checks = append(report.Checks, checkReport)
		if !checkReport.Passed && !checkReport.Optional {
			report.Complete = false
		}
	}
	return report
}

func evaluateOrientationCheck[C any](deps HandlerDeps[C], ctx C, scenarioRoot string, check TemplateOrientationCheck) OrientationCheckReport {
	report := OrientationCheckReport{
		Kind:     check.Kind,
		Label:    orientationCheckLabel(check),
		Optional: check.Optional,
	}
	resolve := func(path string) (string, error) {
		clean, err := cleanScenarioRelativePath(path)
		if err != nil {
			return "", err
		}
		return filepath.Join(scenarioRoot, clean), nil
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
		evaluateOrientationTextAbsentTree(&report, check, scenarioRoot)
	case "json_path_exists":
		evaluateOrientationJSONPathExists(&report, check, resolve)
	case "json_min_entries":
		evaluateOrientationJSONMinEntries(&report, check, resolve)
	case "command":
		evaluateOrientationCommand(&report, deps, ctx, scenarioRoot, check)
	default:
		report.Message = "unknown check kind"
	}
	return report
}

type orientationPathResolver func(string) (string, error)

func evaluateOrientationFileExists(report *OrientationCheckReport, check TemplateOrientationCheck, resolve orientationPathResolver) {
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

func evaluateOrientationFileAbsent(report *OrientationCheckReport, check TemplateOrientationCheck, resolve orientationPathResolver) {
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

func evaluateOrientationDirectoryExists(report *OrientationCheckReport, check TemplateOrientationCheck, resolve orientationPathResolver) {
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

func evaluateOrientationGlobPresence(report *OrientationCheckReport, check TemplateOrientationCheck, resolve orientationPathResolver) {
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

func evaluateOrientationGlobMinCount(report *OrientationCheckReport, check TemplateOrientationCheck, resolve orientationPathResolver) {
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

func evaluateOrientationTextCondition(report *OrientationCheckReport, check TemplateOrientationCheck, resolve orientationPathResolver) {
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

func evaluateOrientationTextAbsentTree(report *OrientationCheckReport, check TemplateOrientationCheck, scenarioRoot string) {
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

func evaluateOrientationJSONPathExists(report *OrientationCheckReport, check TemplateOrientationCheck, resolve orientationPathResolver) {
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

func evaluateOrientationJSONMinEntries(report *OrientationCheckReport, check TemplateOrientationCheck, resolve orientationPathResolver) {
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

func orientationResolvedJSONPathValue(check TemplateOrientationCheck, resolve orientationPathResolver) (any, bool, error) {
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

func evaluateOrientationCommand[C any](report *OrientationCheckReport, deps HandlerDeps[C], ctx C, scenarioRoot string, check TemplateOrientationCheck) {
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

func orientationCheckLabel(check TemplateOrientationCheck) string {
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
