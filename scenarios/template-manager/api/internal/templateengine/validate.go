package templateengine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/templatevalidation"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func runTemplateValidate[C any](deps HandlerDeps[C], ctx C, req templatecontracts.TemplateValidateRequest) (templatecontracts.TemplateValidationReport, error) {
	templates, err := loadTemplates(deps.Root(ctx))
	if err != nil {
		return templatecontracts.TemplateValidationReport{}, err
	}
	templates, err = filterTemplatesForValidation(templates, req.TemplateName)
	if err != nil {
		return templatecontracts.TemplateValidationReport{}, err
	}
	mode := req.Mode
	if mode == "" {
		mode = templatecontracts.TemplateValidationModeShallow
	}
	warningPolicy := req.WarningPolicy
	if warningPolicy == "" {
		if mode == templatecontracts.TemplateValidationModeDeep {
			warningPolicy = templatecontracts.TemplateValidationWarningPolicyReport
		} else {
			warningPolicy = templatecontracts.TemplateValidationWarningPolicyIgnore
		}
		req.WarningPolicy = warningPolicy
	}
	report := templatecontracts.TemplateValidationReport{
		Mode:          mode,
		TemplateName:  req.TemplateName,
		TestPreset:    req.TestPreset,
		WarningPolicy: warningPolicy,
		Count:         len(templates),
	}
	for _, info := range templates {
		switch mode {
		case templatecontracts.TemplateValidationModeDeep:
			run, issues := validateTemplateDeep(deps, ctx, info, req)
			report.DeepRuns = append(report.DeepRuns, run)
			report.WarningSummary = mergeTemplateValidationWarningSummaries(report.WarningSummary, run.WarningSummary)
			report.Issues = append(report.Issues, issues...)
		default:
			report.Issues = append(report.Issues, validateTemplateShallow(deps, ctx, info)...)
		}
	}
	return report, nil
}

func validateTemplateShallow[C any](deps HandlerDeps[C], ctx C, info templatecontracts.TemplateInfo) []templatecontracts.TemplateValidationIssue {
	issues := validateTemplateSource(info)
	if info.Missing {
		return append(issues, templatecontracts.TemplateValidationIssue{
			Template: info.Name,
			Message:  "template.json is missing",
		})
	}
	issues = append(issues, validateRelocationProtoSources(deps, ctx, info)...)
	tempRoot, err := os.MkdirTemp("", "vrooli-template-validate-*")
	if err != nil {
		return append(issues, templatecontracts.TemplateValidationIssue{
			Template: info.Name,
			Message:  fmt.Sprintf("create validation temp dir: %v", err),
		})
	}
	destination := filepath.Join(tempRoot, "scenario")
	return append(issues, validateTemplateShallowGeneratedCopy(deps, ctx, info, tempRoot, destination)...)
}

func validateTemplateShallowGeneratedCopy[C any](deps HandlerDeps[C], ctx C, info templatecontracts.TemplateInfo, tempRoot, destination string) []templatecontracts.TemplateValidationIssue {
	defer os.RemoveAll(tempRoot)
	values, err := buildTemplateValues(
		deps.Root(ctx),
		destination,
		info.Name,
		info.Manifest,
		templateValidationSeedValues(info),
	)
	if err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  err.Error(),
		}}
	}
	if err := copyTemplate(info.Path, destination, values, info.Manifest); err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("generate validation copy: %v", err),
		}}
	}
	design, err := resolveDesign(deps.Root(ctx), info, "", destination, values)
	if err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("resolve default design: %v", err),
		}}
	}
	if err := preflightDesignCopies(design, true); err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("preflight default design: %v", err),
		}}
	}
	if err := preflightDesignTemplateCollisions(info.Path, destination, design); err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("preflight default design: %v", err),
		}}
	}
	if err := copyDesignAssets(design, values); err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("copy default design: %v", err),
		}}
	}
	provenance := buildGenerationProvenance(info, design, time.Now().UTC())
	if err := injectScenarioProvenance(destination, provenance); err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("write provenance: %v", err),
		}}
	}
	if err := renderOrientationManifest(destination, info.Manifest, values); err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("write orientation manifest: %v", err),
		}}
	}
	if err := verifyTemplate(destination); err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  err.Error(),
		}}
	}
	// Run relocations against validation seed values so generated module
	// checks can resolve proto dependencies without leaving persistent
	// template-validation-* artifacts in shared packages.
	resolved, err := resolveRelocations(deps.Root(ctx), info, values)
	if err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("resolve relocations: %v", err),
		}}
	}
	defer cleanupRelocationTargets(resolved)
	if err := runRelocations(deps, ctx, info.Path, resolved, values, deps.Stderr(ctx)); err != nil {
		return []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("relocate validation artifacts: %v", err),
		}}
	}
	return validateGeneratedScenario(destination, deps.RunSubprocess != nil, func(spec scenarioexec.SubprocessSpec) error {
		if deps.RunSubprocess == nil {
			return nil
		}
		var err error
		spec.Env, err = templateHookEnv(deps.CommandEnv(ctx), map[string]string{"GOWORK": "off"})
		if err != nil {
			return err
		}
		spec.Stdout = io.Discard
		spec.Stderr = deps.Stderr(ctx)
		return deps.RunSubprocess(ctx, spec)
	}, info.Name, info.Manifest)
}

func validateTemplateDeep[C any](deps HandlerDeps[C], ctx C, info templatecontracts.TemplateInfo, req templatecontracts.TemplateValidateRequest) (run templatecontracts.TemplateValidationDeepRun, issues []templatecontracts.TemplateValidationIssue) {
	scenarioID := "template-validation-" + info.Name + "-deep"
	run = templatecontracts.TemplateValidationDeepRun{
		Template:   info.Name,
		ScenarioID: scenarioID,
		TestPreset: coalesce(req.TestPreset, templatecontracts.DefaultTemplateValidationTestPreset),
	}
	select {
	case deepValidationSemaphore <- struct{}{}:
		defer func() { <-deepValidationSemaphore }()
	default:
		return run, []templatecontracts.TemplateValidationIssue{newTemplateValidationIssue(info.Name, "deep validation is already in progress; wait for the active run before retrying")}
	}
	if issue := validateDeepPrerequisites(deps, info); issue != nil {
		return run, []templatecontracts.TemplateValidationIssue{*issue}
	}
	cleanupStaleTemplateValidationRuns(deps, ctx)
	testGenieCLI, err := deps.LocateTestGenieCLI(ctx)
	if err != nil {
		return run, []templatecontracts.TemplateValidationIssue{newTemplateValidationIssue(info.Name, err.Error())}
	}
	run, marker, cleanupTemp, err := initDeepValidationRun(deps.Root(ctx), info.Name, scenarioID, run.TestPreset, req.RetainTemp)
	if err != nil {
		return run, []templatecontracts.TemplateValidationIssue{newTemplateValidationIssue(info.Name, err.Error())}
	}
	defer func() {
		finalizeDeepValidationRun(&run, marker, cleanupTemp)
	}()

	if err := prepareDeepValidationWorkspace(deps.Root(ctx), run.TempRoot); err != nil {
		return run, []templatecontracts.TemplateValidationIssue{newTemplateValidationIssue(info.Name, fmt.Sprintf("prepare deep validation workspace: %v", err))}
	}
	relocations, issues := generateDeepValidationScenario(deps, ctx, info, scenarioID, run.ScenarioPath)
	run.RelocationArtifacts = relocationArtifactPaths(relocations)
	marker.RelocationArtifacts = append([]string(nil), run.RelocationArtifacts...)
	_ = templatevalidation.WriteMarker(marker)
	if len(relocations) > 0 && !req.RetainTemp {
		defer func() {
			if err := cleanupDeepValidationRelocations(deps, ctx, relocations); err != nil {
				issues = append(issues, newTemplateValidationIssue(info.Name, err.Error()))
			}
		}()
	}
	if len(issues) > 0 {
		return run, issues
	}
	if issues := runDeepValidationTestGenie(deps, ctx, info.Name, testGenieCLI, req, &run); len(issues) > 0 {
		return run, issues
	}
	return run, nil
}

func validateDeepPrerequisites[C any](deps HandlerDeps[C], info templatecontracts.TemplateInfo) *templatecontracts.TemplateValidationIssue {
	switch {
	case info.Missing:
		issue := newTemplateValidationIssue(info.Name, "template.json is missing")
		return &issue
	case deps.RunSubprocess == nil:
		issue := newTemplateValidationIssue(info.Name, "deep validation requires subprocess execution")
		return &issue
	case deps.LocateTestGenieCLI == nil:
		issue := newTemplateValidationIssue(info.Name, "deep validation requires test-genie CLI resolution")
		return &issue
	default:
		return nil
	}
}

func newTemplateValidationIssue(templateName, message string) templatecontracts.TemplateValidationIssue {
	return templatecontracts.TemplateValidationIssue{Template: templateName, Message: message}
}

func mergeTemplateValidationWarningSummaries(left, right templatecontracts.TemplateValidationWarningSummary) templatecontracts.TemplateValidationWarningSummary {
	if right.Total == 0 && len(right.Phases) == 0 {
		return left
	}
	left.Total += right.Total
	left.Phases = append(left.Phases, right.Phases...)
	return left
}

func filterTemplatesForValidation(templates []templatecontracts.TemplateInfo, templateName string) ([]templatecontracts.TemplateInfo, error) {
	name := strings.TrimSpace(templateName)
	if name == "" {
		return templates, nil
	}
	for _, info := range templates {
		if info.Name == name {
			return []templatecontracts.TemplateInfo{info}, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrTemplateNotFound, name)
}
