package scenariohandlers

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

	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/cliout"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/shell"
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
		path, err := resolve(check.Path)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		info, err := os.Stat(path)
		report.Passed = err == nil && !info.IsDir()
		if !report.Passed {
			report.Message = "file is missing"
		}
	case "file_absent":
		path, err := resolve(check.Path)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		_, err = os.Stat(path)
		report.Passed = os.IsNotExist(err)
		if !report.Passed {
			report.Message = "file exists"
		}
	case "directory_exists":
		path, err := resolve(check.Path)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		info, err := os.Stat(path)
		report.Passed = err == nil && info.IsDir()
		if !report.Passed {
			report.Message = "directory is missing"
		}
	case "glob_present", "glob_absent":
		pattern, err := resolve(check.Pattern)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		report.Passed = len(matches) > 0
		if check.Kind == "glob_absent" {
			report.Passed = len(matches) == 0
		}
		if !report.Passed {
			report.Message = fmt.Sprintf("matched %d path(s)", len(matches))
		}
	case "text_contains", "text_absent":
		path, err := resolve(check.Path)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		data, err := os.ReadFile(path)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		contains := bytes.Contains(data, []byte(check.Text))
		report.Passed = contains
		if check.Kind == "text_absent" {
			report.Passed = !contains
		}
		if !report.Passed {
			report.Message = "text condition not satisfied"
		}
	case "text_absent_tree":
		// Recursively scan the scenario tree (text files only, skipping
		// vendored/build dirs) and fail if any file contains check.Text. This
		// is the EXAMPLE-DOMAIN residue gate: a finalized scenario must carry
		// zero example-domain markers anywhere.
		hits, err := scanTreeForText(scenarioRoot, check.Text)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		report.Passed = len(hits) == 0
		if !report.Passed {
			shown := hits
			if len(shown) > 5 {
				shown = shown[:5]
			}
			report.Message = fmt.Sprintf("found %q in %d file(s): %s — run `vrooli scenario detemplate %s`",
				check.Text, len(hits), strings.Join(shown, ", "), filepath.Base(scenarioRoot))
		}
	case "json_path_exists":
		path, err := resolve(check.Path)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		ok, err := orientationJSONPathExists(path, check.Query)
		if err != nil {
			report.Message = err.Error()
			return report
		}
		report.Passed = ok
		if !report.Passed {
			report.Message = "JSON path not found"
		}
	case "command":
		timeout, err := time.ParseDuration(check.Timeout)
		if err != nil {
			report.Message = "invalid timeout: " + err.Error()
			return report
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
			if commandCtx.Err() == context.DeadlineExceeded {
				report.Message = "command timed out after " + check.Timeout
				return report
			}
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = strings.TrimSpace(stdout.String())
			}
			if msg == "" {
				msg = err.Error()
			}
			report.Message = truncateForIssue(msg, 240)
		}
	default:
		report.Message = "unknown check kind"
	}
	return report
}

func orientationCheckLabel(check TemplateOrientationCheck) string {
	switch check.Kind {
	case "file_exists", "file_absent", "directory_exists":
		return check.Path
	case "glob_present", "glob_absent":
		return check.Pattern
	case "json_path_exists":
		return strings.TrimSpace(check.Path + ":" + check.Query)
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

func orientationJSONPathExists(path, query string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var current any
	if err := json.Unmarshal(data, &current); err != nil {
		return false, err
	}
	for _, part := range strings.Split(strings.TrimSpace(query), ".") {
		if part == "" {
			return false, nil
		}
		switch typed := current.(type) {
		case map[string]any:
			value, ok := typed[part]
			if !ok {
				return false, nil
			}
			current = value
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return false, nil
			}
			current = typed[index]
		default:
			return false, nil
		}
	}
	return current != nil, nil
}
