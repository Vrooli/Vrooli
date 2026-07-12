package templateengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/templatevalidation"
	. "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
)

var unresolvedTemplatePattern = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)

// Deep validation regenerates shared packages/proto artifacts. One run must
// own that workspace at a time, otherwise a rejected concurrent run can clean
// artifacts while the active generated scenario is compiling.
var deepValidationSemaphore = make(chan struct{}, 1)

// Test Genie detail changes as individual phases discover new evidence. Keep
// the Template Manager debt identity tied to the failure class instead of that
// volatile prose, while retaining the complete message for an operator.
const (
	testGenieDeepValidationProtocolPath     = "test-genie/deep-validation/protocol"
	testGenieDeepValidationStartupPath      = "test-genie/deep-validation/startup"
	testGenieDeepValidationPhaseResultsPath = "test-genie/deep-validation/phase-results"
	testGenieDeepValidationWarningsPath     = "test-genie/deep-validation/warnings"
)

func TemplateCommandHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return func(ctx C, args []string) error {
		if len(args) == 0 {
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (TemplateListRequest, error) { return ParseTemplateListRequest(args) },
				func(ctx C, req TemplateListRequest) (cliout.Format, []TemplateInfo, error) {
					return runTemplateList(deps, ctx, req)
				},
				RenderTemplateListResponse,
			)(ctx, nil)
		}
		switch commandtree.NormalizeName(args[0]) {
		case "list":
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (TemplateListRequest, error) { return ParseTemplateListRequest(args) },
				func(ctx C, req TemplateListRequest) (cliout.Format, []TemplateInfo, error) {
					return runTemplateList(deps, ctx, req)
				},
				RenderTemplateListResponse,
			)(ctx, args[1:])
		case "show":
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (TemplateShowRequest, error) { return ParseTemplateShowRequest(args) },
				func(ctx C, req TemplateShowRequest) (cliout.Format, TemplateInfo, error) {
					return runTemplateShow(deps, ctx, req)
				},
				RenderTemplateShowResponse,
			)(ctx, args[1:])
		case "validate":
			req, err := ParseTemplateValidateRequest(args[1:])
			if err != nil {
				if helpErr, ok := err.(interface{ HelpText() string }); ok {
					commandtree.WriteHelp(deps.Stdout(ctx), helpErr.HelpText())
					return nil
				}
				return rootcli.UsageErrorf("scenario template validate", "%s", err.Error())
			}
			format, report, err := runTemplateValidate(deps, ctx, req)
			if err != nil {
				return err
			}
			if err := RenderTemplateValidateResponse(deps.Stdout(ctx), format, report); err != nil {
				return err
			}
			if len(report.Issues) > 0 {
				return fmt.Errorf("scenario template validation failed")
			}
			return nil
		case "drift":
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (TemplateDriftRequest, error) { return ParseTemplateDriftRequest(args) },
				func(ctx C, req TemplateDriftRequest) (cliout.Format, TemplateDriftReport, error) {
					return runTemplateDrift(deps, ctx, req)
				},
				RenderTemplateDriftResponse,
			)(ctx, args[1:])
		case "cleanup":
			return bindGlobal(deps.Stdout,
				func(ctx C, args []string) (TemplateCleanupRequest, error) { return ParseTemplateCleanupRequest(args) },
				func(ctx C, req TemplateCleanupRequest) (cliout.Format, TemplateCleanupResult, error) {
					return runTemplateCleanup(deps, ctx, req)
				},
				RenderTemplateCleanupResponse,
			)(ctx, args[1:])
		case "--help", "-h":
			RenderTemplateHelp(deps.Stdout(ctx))
			return nil
		default:
			return rootcli.UsageErrorf("scenario template", "unknown scenario template command: %s", args[0])
		}
	}
}

func GenerateHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return bindGlobal(deps.Stdout,
		func(ctx C, args []string) (GenerateRequest, error) {
			return ParseGenerateRequest(args, deps.Stderr(ctx), func(name string) (TemplateInfo, error) {
				return loadTemplate(deps.Root(ctx), name)
			}, ParseGenerateArgs)
		},
		func(ctx C, req GenerateRequest) (cliout.Format, GenerateResult, error) {
			return runGenerate(deps, ctx, req)
		},
		RenderGenerateResponse,
	)
}

func runTemplateList[C any](deps HandlerDeps[C], ctx C, _ TemplateListRequest) (cliout.Format, []TemplateInfo, error) {
	templates, err := loadTemplates(deps.Root(ctx))
	if err != nil {
		return "", nil, err
	}
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", nil, err
	}
	return format, templates, nil
}

func runTemplateShow[C any](deps HandlerDeps[C], ctx C, req TemplateShowRequest) (cliout.Format, TemplateInfo, error) {
	info, err := loadTemplate(deps.Root(ctx), req.Name)
	if err != nil {
		return "", TemplateInfo{}, err
	}
	return cliout.FormatHuman, info, nil
}

func runTemplateValidate[C any](deps HandlerDeps[C], ctx C, req TemplateValidateRequest) (cliout.Format, TemplateValidationReport, error) {
	templates, err := loadTemplates(deps.Root(ctx))
	if err != nil {
		return "", TemplateValidationReport{}, err
	}
	templates, err = filterTemplatesForValidation(templates, req.TemplateName)
	if err != nil {
		return "", TemplateValidationReport{}, err
	}
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", TemplateValidationReport{}, err
	}
	mode := req.Mode
	if mode == "" {
		mode = TemplateValidationModeShallow
	}
	warningPolicy := req.WarningPolicy
	if warningPolicy == "" {
		if mode == TemplateValidationModeDeep {
			warningPolicy = TemplateValidationWarningPolicyReport
		} else {
			warningPolicy = TemplateValidationWarningPolicyIgnore
		}
		req.WarningPolicy = warningPolicy
	}
	report := TemplateValidationReport{
		Mode:          mode,
		TemplateName:  req.TemplateName,
		TestPreset:    req.TestPreset,
		WarningPolicy: warningPolicy,
		Count:         len(templates),
	}
	for _, info := range templates {
		switch mode {
		case TemplateValidationModeDeep:
			run, issues := validateTemplateDeep(deps, ctx, info, req)
			report.DeepRuns = append(report.DeepRuns, run)
			report.WarningSummary = mergeTemplateValidationWarningSummaries(report.WarningSummary, run.WarningSummary)
			report.Issues = append(report.Issues, issues...)
		default:
			report.Issues = append(report.Issues, validateTemplateShallow(deps, ctx, info)...)
		}
	}
	return format, report, nil
}

func validateTemplateShallow[C any](deps HandlerDeps[C], ctx C, info TemplateInfo) []TemplateValidationIssue {
	issues := validateTemplateSource(info)
	if info.Missing {
		return append(issues, TemplateValidationIssue{
			Template: info.Name,
			Message:  "template.json is missing",
		})
	}
	issues = append(issues, validateRelocationProtoSources(deps, ctx, info)...)
	tempRoot, err := os.MkdirTemp("", "vrooli-template-validate-*")
	if err != nil {
		return append(issues, TemplateValidationIssue{
			Template: info.Name,
			Message:  fmt.Sprintf("create validation temp dir: %v", err),
		})
	}
	destination := filepath.Join(tempRoot, "scenario")
	return append(issues, validateTemplateShallowGeneratedCopy(deps, ctx, info, tempRoot, destination)...)
}

func validateTemplateShallowGeneratedCopy[C any](deps HandlerDeps[C], ctx C, info TemplateInfo, tempRoot, destination string) []TemplateValidationIssue {
	defer os.RemoveAll(tempRoot)
	values, err := buildTemplateValues(
		deps.Root(ctx),
		destination,
		info.Name,
		info.Manifest,
		templateValidationSeedValues(info),
	)
	if err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  err.Error(),
		}}
	}
	if err := copyTemplate(info.Path, destination, values, info.Manifest); err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("generate validation copy: %v", err),
		}}
	}
	design, err := resolveDesign(deps.Root(ctx), info, "", destination, values)
	if err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("resolve default design: %v", err),
		}}
	}
	if err := preflightDesignCopies(design, true); err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("preflight default design: %v", err),
		}}
	}
	if err := preflightDesignTemplateCollisions(info.Path, destination, design); err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("preflight default design: %v", err),
		}}
	}
	if err := copyDesignAssets(design, values); err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("copy default design: %v", err),
		}}
	}
	provenance := buildGenerationProvenance(info, design, time.Now().UTC())
	if err := injectScenarioProvenance(destination, provenance); err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("write provenance: %v", err),
		}}
	}
	if err := renderOrientationManifest(destination, info.Manifest, values); err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("write orientation manifest: %v", err),
		}}
	}
	if err := verifyTemplate(destination); err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  err.Error(),
		}}
	}
	// Run relocations against validation seed values so generated module
	// checks can resolve proto dependencies without leaving persistent
	// template-validation-* artifacts in shared packages.
	resolved, err := resolveRelocations(deps.Root(ctx), info, values)
	if err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("resolve relocations: %v", err),
		}}
	}
	defer cleanupRelocationTargets(resolved)
	if err := runRelocations(deps, ctx, info.Path, resolved, values, deps.Stderr(ctx)); err != nil {
		return []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("relocate validation artifacts: %v", err),
		}}
	}
	return validateGeneratedScenario(destination, deps.RunSubprocess != nil, func(spec scenarioexec.SubprocessSpec) error {
		if deps.RunSubprocess == nil {
			return nil
		}
		spec.Env = deps.CommandEnv(ctx)
		spec.Stdout = io.Discard
		spec.Stderr = deps.Stderr(ctx)
		return deps.RunSubprocess(ctx, spec)
	}, info.Name, info.Manifest)
}

func validateTemplateDeep[C any](deps HandlerDeps[C], ctx C, info TemplateInfo, req TemplateValidateRequest) (run TemplateValidationDeepRun, issues []TemplateValidationIssue) {
	scenarioID := "template-validation-" + info.Name + "-deep"
	run = TemplateValidationDeepRun{
		Template:   info.Name,
		ScenarioID: scenarioID,
		TestPreset: coalesce(req.TestPreset, DefaultTemplateValidationTestPreset),
	}
	select {
	case deepValidationSemaphore <- struct{}{}:
		defer func() { <-deepValidationSemaphore }()
	default:
		return run, []TemplateValidationIssue{newTemplateValidationIssue(info.Name, "deep validation is already in progress; wait for the active run before retrying")}
	}
	if issue := validateDeepPrerequisites(deps, info); issue != nil {
		return run, []TemplateValidationIssue{*issue}
	}
	cleanupStaleTemplateValidationRuns(deps, ctx)
	testGenieCLI, err := deps.LocateTestGenieCLI(ctx)
	if err != nil {
		return run, []TemplateValidationIssue{newTemplateValidationIssue(info.Name, err.Error())}
	}
	run, marker, cleanupTemp, err := initDeepValidationRun(deps.Root(ctx), info.Name, scenarioID, run.TestPreset, req.RetainTemp)
	if err != nil {
		return run, []TemplateValidationIssue{newTemplateValidationIssue(info.Name, err.Error())}
	}
	defer func() {
		finalizeDeepValidationRun(&run, marker, cleanupTemp)
	}()

	if err := prepareDeepValidationWorkspace(deps.Root(ctx), run.TempRoot); err != nil {
		return run, []TemplateValidationIssue{newTemplateValidationIssue(info.Name, fmt.Sprintf("prepare deep validation workspace: %v", err))}
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

func validateDeepPrerequisites[C any](deps HandlerDeps[C], info TemplateInfo) *TemplateValidationIssue {
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

func newTemplateValidationIssue(templateName, message string) TemplateValidationIssue {
	return TemplateValidationIssue{Template: templateName, Message: message}
}

func newTestGenieDeepValidationIssue(templateName, path, message string) TemplateValidationIssue {
	return TemplateValidationIssue{Template: templateName, Path: path, Message: message}
}

func initDeepValidationRun(repoRoot, templateName, scenarioID, testPreset string, retainTemp bool) (TemplateValidationDeepRun, templatevalidation.RunMarker, bool, error) {
	run := TemplateValidationDeepRun{
		Template:     templateName,
		ScenarioID:   scenarioID,
		TestPreset:   testPreset,
		RetainedTemp: retainTemp,
	}
	tempRoot, err := os.MkdirTemp("", "vrooli-template-deep-*")
	if err != nil {
		return run, templatevalidation.RunMarker{}, false, fmt.Errorf("create deep validation temp dir: %v", err)
	}
	run.TempRoot = tempRoot
	run.ScenarioPath = filepath.Join(tempRoot, "scenarios", scenarioID)
	marker, err := templatevalidation.NewRunMarker(templatevalidation.NewRunMarkerInput{
		RepoRoot:     repoRoot,
		Template:     templateName,
		ScenarioID:   scenarioID,
		ScenarioPath: run.ScenarioPath,
		TempRoot:     tempRoot,
		Retained:     retainTemp,
	})
	if err != nil {
		return run, templatevalidation.RunMarker{}, false, fmt.Errorf("create deep validation marker: %v", err)
	}
	run.RunID = marker.RunID
	if err := templatevalidation.WriteMarker(marker); err != nil {
		return run, templatevalidation.RunMarker{}, false, fmt.Errorf("write deep validation marker: %v", err)
	}
	if retainTemp {
		run.CleanupStatus = "retained"
		run.CleanupCommand = "template-manager template cleanup --run " + marker.RunID
		return run, marker, false, nil
	}
	return run, marker, true, nil
}

func finalizeDeepValidationRun(run *TemplateValidationDeepRun, marker templatevalidation.RunMarker, cleanupTemp bool) {
	marker.Completed = true
	marker.CleanupStatus = run.CleanupStatus
	if cleanupTemp {
		if err := os.RemoveAll(run.TempRoot); err != nil {
			run.CleanupStatus = "cleanup failed: " + err.Error()
			marker.CleanupStatus = run.CleanupStatus
			_ = templatevalidation.WriteMarker(marker)
			return
		}
		run.CleanupStatus = "removed"
		marker.CleanupStatus = run.CleanupStatus
		return
	}
	_ = templatevalidation.WriteMarker(marker)
}

func runDeepValidationTestGenie[C any](deps HandlerDeps[C], ctx C, templateName, testGenieCLI string, req TemplateValidateRequest, run *TemplateValidationDeepRun) []TemplateValidationIssue {
	var stdout, stderr bytes.Buffer
	args := []string{
		"execute",
		run.ScenarioID,
		"--scenario-path", run.ScenarioPath,
		"--logical-repo-root", deps.Root(ctx),
		"--logical-scenario-relpath", filepath.Join("scenarios", run.ScenarioID),
		"--preset", run.TestPreset,
		"--wait",
		"--json",
	}
	if err := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
		Name:   testGenieCLI,
		Args:   args,
		Dir:    deps.Root(ctx),
		Env:    deps.CommandEnv(ctx),
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		result := parseTestGenieJSONResult(templateName, stdout.Bytes())
		if req.WarningPolicy != TemplateValidationWarningPolicyIgnore {
			run.WarningSummary = result.WarningSummary
		}
		if result.Issue != nil && result.Success != nil && !*result.Success {
			return []TemplateValidationIssue{*result.Issue}
		}
		if issue := testGenieFailureIssueFromJSON(templateName, stdout.Bytes()); issue != nil {
			return []TemplateValidationIssue{*issue}
		}
		return []TemplateValidationIssue{newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationProtocolPath, "test-genie deep validation failed: "+deepValidationFailureMessage(stdout.String(), stderr.String(), err))}
	}
	result := parseTestGenieJSONResult(templateName, stdout.Bytes())
	if req.WarningPolicy != TemplateValidationWarningPolicyIgnore {
		run.WarningSummary = result.WarningSummary
	}
	if result.Issue != nil {
		return []TemplateValidationIssue{*result.Issue}
	}
	if req.WarningPolicy == TemplateValidationWarningPolicyFail && result.WarningSummary.Total > 0 {
		return []TemplateValidationIssue{newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationWarningsPath, fmt.Sprintf("test-genie deep validation reported %d warning(s)", run.WarningSummary.Total))}
	}
	return nil
}

func deepValidationFailureMessage(stdout, stderr string, err error) string {
	message := strings.TrimSpace(stderr)
	if message == "" {
		message = strings.TrimSpace(stdout)
	}
	if message == "" {
		message = err.Error()
	}
	return truncateForIssue(message, 2000)
}

func runTemplateCleanup[C any](deps HandlerDeps[C], ctx C, req TemplateCleanupRequest) (cliout.Format, TemplateCleanupResult, error) {
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", TemplateCleanupResult{}, err
	}
	opts, err := templateCleanupOptions(deps.Root(ctx), req)
	if err != nil {
		return "", TemplateCleanupResult{}, err
	}
	plan := templatevalidation.PlanCleanup(opts)
	result := templatevalidation.ExecuteCleanup(plan)
	if err := runProtoGenerateForCleanupResult(deps, ctx, &result); err != nil {
		return "", result, err
	}
	return format, result, templatevalidation.ResultError(result)
}

func cleanupStaleTemplateValidationRuns[C any](deps HandlerDeps[C], ctx C) {
	opts := templatevalidation.CleanupOptions{
		RepoRoot:  deps.Root(ctx),
		OlderThan: templatevalidation.DefaultCleanupOlderThan,
	}
	result := templatevalidation.ExecuteCleanup(templatevalidation.PlanCleanup(opts))
	_ = runProtoGenerateForCleanupResult(deps, ctx, &result)
}

func templateCleanupOptions(repoRoot string, req TemplateCleanupRequest) (templatevalidation.CleanupOptions, error) {
	olderThan := templatevalidation.DefaultCleanupOlderThan
	if strings.TrimSpace(req.OlderThan) != "" {
		parsed, err := time.ParseDuration(req.OlderThan)
		if err != nil {
			return templatevalidation.CleanupOptions{}, fmt.Errorf("invalid --older-than duration %q: %w", req.OlderThan, err)
		}
		olderThan = parsed
	}
	return templatevalidation.CleanupOptions{
		RepoRoot:        repoRoot,
		OlderThan:       olderThan,
		IncludeRetained: req.IncludeRetained,
		RunID:           req.RunID,
		DryRun:          req.DryRun,
	}, nil
}

func runProtoGenerateForCleanupResult[C any](deps HandlerDeps[C], ctx C, result *templatevalidation.CleanupResult) error {
	if result == nil || result.DryRun || !result.NeedsProtoGenerate || len(result.Failures) > 0 {
		return nil
	}
	if deps.RunSubprocess == nil {
		return nil
	}
	protoDir := filepath.Join(deps.Root(ctx), "packages", "proto")
	if _, err := os.Stat(filepath.Join(protoDir, "Makefile")); err != nil {
		return nil
	}
	if err := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
		Name:   "make",
		Args:   []string{"generate"},
		Dir:    protoDir,
		Env:    deps.CommandEnv(ctx),
		Stdout: deps.Stderr(ctx),
		Stderr: deps.Stderr(ctx),
	}); err != nil {
		return fmt.Errorf("regenerate proto artifacts after template validation cleanup: %w", err)
	}
	result.ProtoGenerateRan = true
	return nil
}

func cleanupDeepValidationRelocations[C any](deps HandlerDeps[C], ctx C, relocations []ResolvedRelocation) error {
	cleanupRelocationTargets(relocations)
	if !hasProtoRelocation(deps.Root(ctx), relocations) {
		return nil
	}
	return runProtoGenerateForCleanupResult(deps, ctx, &templatevalidation.CleanupResult{NeedsProtoGenerate: true})
}

func hasProtoRelocation(repoRoot string, relocations []ResolvedRelocation) bool {
	protoSchemasRoot := filepath.Clean(filepath.Join(repoRoot, "packages", "proto", "schemas"))
	for _, relocation := range relocations {
		target := filepath.Clean(relocation.To)
		if target == protoSchemasRoot || strings.HasPrefix(target, protoSchemasRoot+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func testGenieFailureIssueFromJSON(templateName string, output []byte) *TemplateValidationIssue {
	type testGenieResponse struct {
		Success *bool `json:"success"`
	}
	data := bytes.TrimSpace(output)
	if len(data) == 0 {
		return nil
	}
	var response testGenieResponse
	if err := json.Unmarshal(data, &response); err != nil || response.Success == nil || *response.Success {
		return nil
	}
	return validateTestGenieJSONSuccess(templateName, data)
}

func validateTestGenieJSONSuccess(templateName string, output []byte) *TemplateValidationIssue {
	return parseTestGenieJSONResult(templateName, output).Issue
}

type parsedTestGenieResult struct {
	Success        *bool
	WarningSummary TemplateValidationWarningSummary
	Issue          *TemplateValidationIssue
}

func parseTestGenieJSONResult(templateName string, output []byte) parsedTestGenieResult {
	type testGenieResponse struct {
		Success       *bool    `json:"success"`
		Error         string   `json:"error"`
		ErrorMessages []string `json:"errors"`
		PhaseSummary  struct {
			Total  int `json:"total"`
			Passed int `json:"passed"`
			Failed int `json:"failed"`
		} `json:"phaseSummary"`
		Phases []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		} `json:"phases"`
		WarningSummary TemplateValidationWarningSummary `json:"warningSummary"`
	}

	data := bytes.TrimSpace(output)
	if len(data) == 0 {
		issue := newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationProtocolPath, "test-genie deep validation produced no JSON output")
		return parsedTestGenieResult{Issue: &issue}
	}
	var response testGenieResponse
	if err := json.Unmarshal(data, &response); err != nil {
		issue := newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationProtocolPath, fmt.Sprintf("test-genie deep validation returned invalid JSON: %v", err))
		return parsedTestGenieResult{Issue: &issue}
	}
	result := parsedTestGenieResult{
		Success:        response.Success,
		WarningSummary: response.WarningSummary,
	}
	if response.Success == nil {
		issue := newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationProtocolPath, "test-genie deep validation JSON omitted success")
		result.Issue = &issue
		return result
	}
	if *response.Success {
		return result
	}
	if response.PhaseSummary.Total == 0 && response.PhaseSummary.Failed == 0 {
		startupFailure := append([]string(nil), response.ErrorMessages...)
		if detail := strings.TrimSpace(response.Error); detail != "" {
			startupFailure = append(startupFailure, detail)
		}
		if len(startupFailure) > 0 {
			issue := newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationStartupPath, "test-genie deep validation startup failed before phases: "+truncateForIssue(strings.Join(startupFailure, "; "), 2000))
			result.Issue = &issue
			return result
		}
	}
	var failed []string
	for _, phase := range response.Phases {
		if phase.Status != "failed" {
			continue
		}
		if strings.TrimSpace(phase.Error) == "" {
			failed = append(failed, phase.Name)
			continue
		}
		failed = append(failed, fmt.Sprintf("%s: %s", phase.Name, phase.Error))
	}
	for _, msg := range response.ErrorMessages {
		if strings.TrimSpace(msg) != "" {
			failed = append(failed, strings.TrimSpace(msg))
		}
	}
	summary := fmt.Sprintf("%d passed, %d failed, %d total", response.PhaseSummary.Passed, response.PhaseSummary.Failed, response.PhaseSummary.Total)
	if len(failed) > 0 {
		summary += "; failed phases: " + strings.Join(failed, "; ")
	} else if strings.TrimSpace(response.Error) != "" {
		summary += "; " + strings.TrimSpace(response.Error)
	}
	issue := newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationPhaseResultsPath, "test-genie deep validation failed: "+truncateForIssue(summary, 2000))
	result.Issue = &issue
	return result
}

func mergeTemplateValidationWarningSummaries(left, right TemplateValidationWarningSummary) TemplateValidationWarningSummary {
	if right.Total == 0 && len(right.Phases) == 0 {
		return left
	}
	left.Total += right.Total
	left.Phases = append(left.Phases, right.Phases...)
	return left
}

func prepareDeepValidationWorkspace(repoRoot, tempRoot string) error {
	if err := os.MkdirAll(filepath.Join(tempRoot, "scenarios"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(tempRoot, ".vrooli"), 0o755); err != nil {
		return err
	}
	contractSrc := filepath.Join(repoRoot, ".vrooli", "repo-contract.json")
	if data, err := os.ReadFile(contractSrc); err == nil {
		if err := os.WriteFile(filepath.Join(tempRoot, ".vrooli", "repo-contract.json"), data, 0o644); err != nil {
			return err
		}
	}
	// Generated scenario manifests resolve their schemas relative to the
	// repository root. The temporary validation workspace is deliberately a
	// minimal repo, so retain the canonical schema directory as a shared,
	// read-only dependency rather than making generated scenarios special-case
	// schema validation.
	schemasSrc := filepath.Join(repoRoot, ".vrooli", "schemas")
	schemasDst := filepath.Join(tempRoot, ".vrooli", "schemas")
	if _, err := os.Stat(schemasSrc); err == nil {
		if err := os.Symlink(schemasSrc, schemasDst); err != nil && !os.IsExist(err) {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, dir := range []string{"packages", "resources", "templates", "cmd", "internal"} {
		src := filepath.Join(repoRoot, dir)
		dst := filepath.Join(tempRoot, dir)
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				if mkErr := os.MkdirAll(dst, 0o755); mkErr != nil {
					return mkErr
				}
				continue
			}
			return err
		}
		if err := os.Symlink(src, dst); err != nil {
			if os.IsExist(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func generateDeepValidationScenario[C any](deps HandlerDeps[C], ctx C, info TemplateInfo, scenarioID, destination string) ([]ResolvedRelocation, []TemplateValidationIssue) {
	values, err := buildTemplateValues(
		deps.Root(ctx),
		destination,
		info.Name,
		info.Manifest,
		templateValidationSeedValuesForScenarioID(info, scenarioID),
	)
	if err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  err.Error(),
		}}
	}
	if err := copyTemplate(info.Path, destination, values, info.Manifest); err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("generate deep validation scenario: %v", err),
		}}
	}
	design, err := resolveDesign(deps.Root(ctx), info, "", destination, values)
	if err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("resolve default design: %v", err),
		}}
	}
	if err := preflightDesignCopies(design, true); err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("preflight default design: %v", err),
		}}
	}
	if err := preflightDesignTemplateCollisions(info.Path, destination, design); err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("preflight default design: %v", err),
		}}
	}
	if err := copyDesignAssets(design, values); err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("copy default design: %v", err),
		}}
	}
	provenance := buildGenerationProvenance(info, design, time.Now().UTC())
	if err := injectScenarioProvenance(destination, provenance); err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("write provenance: %v", err),
		}}
	}
	if err := renderOrientationManifest(destination, info.Manifest, values); err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("write orientation manifest: %v", err),
		}}
	}
	if err := verifyTemplate(destination); err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  err.Error(),
		}}
	}
	resolved, err := resolveRelocations(deps.Root(ctx), info, values)
	if err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("resolve relocations: %v", err),
		}}
	}
	if err := runRelocations(deps, ctx, info.Path, resolved, values, deps.Stderr(ctx)); err != nil {
		return nil, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("relocate deep validation artifacts: %v", err),
		}}
	}
	if issues := validateGeneratedScenario(destination, deps.RunSubprocess != nil, func(spec scenarioexec.SubprocessSpec) error {
		spec.Env = deps.CommandEnv(ctx)
		spec.Stdout = io.Discard
		spec.Stderr = deps.Stderr(ctx)
		return deps.RunSubprocess(ctx, spec)
	}, info.Name, info.Manifest); len(issues) > 0 {
		return resolved, issues
	}
	if err := runTemplateHooks(deps, ctx, destination, info.Manifest, deps.Stderr(ctx)); err != nil {
		return resolved, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("run template post hooks: %v", err),
		}}
	}
	return resolved, nil
}

func filterTemplatesForValidation(templates []TemplateInfo, templateName string) ([]TemplateInfo, error) {
	name := strings.TrimSpace(templateName)
	if name == "" {
		return templates, nil
	}
	for _, info := range templates {
		if info.Name == name {
			return []TemplateInfo{info}, nil
		}
	}
	return nil, fmt.Errorf("template not found: %s", name)
}

func runGenerate[C any](deps HandlerDeps[C], ctx C, req GenerateRequest) (cliout.Format, GenerateResult, error) {
	info := req.TemplateInfo
	opts := req.Options
	prepared, err := prepareGenerate(deps.Root(ctx), info, opts)
	if err != nil {
		return "", GenerateResult{}, err
	}
	if opts.DryRun {
		return cliout.FormatHuman, prepared.result(opts.RunHooks, true), nil
	}
	if err := preflightGenerate(prepared, opts.Force); err != nil {
		return "", GenerateResult{}, err
	}
	if err := writeGeneratedScenario(deps, ctx, prepared); err != nil {
		return "", GenerateResult{}, err
	}
	if err := validateGeneratedScenarioResult(deps, ctx, prepared); err != nil {
		return "", GenerateResult{}, err
	}
	result := prepared.result(opts.RunHooks, false)
	if opts.RunHooks {
		if err := runTemplateHooks(deps, ctx, prepared.destination, info.Manifest, deps.Stdout(ctx)); err != nil {
			return "", GenerateResult{}, err
		}
	}
	return cliout.FormatHuman, result, nil
}

type preparedGenerate struct {
	info        TemplateInfo
	destination string
	values      map[string]string
	relocations []ResolvedRelocation
	design      ResolvedDesign
	provenance  GenerationProvenance
}

func prepareGenerate(root string, info TemplateInfo, opts GenerateOptions) (preparedGenerate, error) {
	destination := cleanGenerateDestination(root, opts)
	values, err := buildTemplateValues(root, destination, info.Name, info.Manifest, opts.Values)
	if err != nil {
		return preparedGenerate{}, err
	}
	resolved, err := resolveRelocations(root, info, values)
	if err != nil {
		return preparedGenerate{}, err
	}
	design, err := resolveDesign(root, info, opts.Design, destination, values)
	if err != nil {
		return preparedGenerate{}, err
	}
	return preparedGenerate{
		info:        info,
		destination: destination,
		values:      values,
		relocations: resolved,
		design:      design,
		provenance:  buildGenerationProvenance(info, design, time.Now().UTC()),
	}, nil
}

func cleanGenerateDestination(root string, opts GenerateOptions) string {
	destination := opts.Destination
	if destination == "" {
		destination = filepath.Join(root, "scenarios", opts.Values["SCENARIO_ID"])
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(root, filepath.FromSlash(destination))
	}
	return filepath.Clean(destination)
}

func (prepared preparedGenerate) result(runHooks, dryRun bool) GenerateResult {
	return GenerateResult{
		TemplateName: prepared.info.Name,
		DisplayName:  coalesce(prepared.values["SCENARIO_DISPLAY_NAME"], prepared.values["SCENARIO_ID"]),
		Destination:  prepared.destination,
		Values:       prepared.values,
		Manifest:     prepared.info.Manifest,
		Design:       prepared.design,
		Relocations:  prepared.relocations,
		Provenance:   prepared.provenance,
		RunHooks:     runHooks,
		DryRun:       dryRun,
	}
}

func preflightGenerate(prepared preparedGenerate, force bool) error {
	if err := removeExistingGenerateTarget(prepared.destination, "destination", force); err != nil {
		return err
	}
	for _, reloc := range prepared.relocations {
		if err := removeExistingGenerateTarget(reloc.To, "relocation target", force); err != nil {
			return err
		}
	}
	if err := preflightDesignCopies(prepared.design, force); err != nil {
		return err
	}
	return preflightDesignTemplateCollisions(prepared.info.Path, prepared.destination, prepared.design)
}

func removeExistingGenerateTarget(path, label string, force bool) error {
	if stat, err := os.Stat(path); err == nil && stat != nil {
		if !force {
			return fmt.Errorf("%s already exists: %s (use --force to overwrite)", label, path)
		}
		return os.RemoveAll(path)
	}
	return nil
}

func writeGeneratedScenario[C any](deps HandlerDeps[C], ctx C, prepared preparedGenerate) error {
	if err := copyTemplate(prepared.info.Path, prepared.destination, prepared.values, prepared.info.Manifest); err != nil {
		return err
	}
	if err := copyDesignAssets(prepared.design, prepared.values); err != nil {
		return err
	}
	if err := injectScenarioProvenance(prepared.destination, prepared.provenance); err != nil {
		return err
	}
	if err := renderOrientationManifest(prepared.destination, prepared.info.Manifest, prepared.values); err != nil {
		return err
	}
	if err := verifyTemplate(prepared.destination); err != nil {
		return err
	}
	return runRelocations(deps, ctx, prepared.info.Path, prepared.relocations, prepared.values, deps.Stdout(ctx))
}

func validateGeneratedScenarioResult[C any](deps HandlerDeps[C], ctx C, prepared preparedGenerate) error {
	issues := validateGeneratedScenario(prepared.destination, deps.RunSubprocess != nil, func(spec scenarioexec.SubprocessSpec) error {
		spec.Env = deps.CommandEnv(ctx)
		spec.Stdout = io.Discard
		spec.Stderr = deps.Stderr(ctx)
		return deps.RunSubprocess(ctx, spec)
	}, prepared.info.Name, prepared.info.Manifest)
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("%s", formatTemplateValidationIssues(issues))
}

// resolveRelocations renders each relocation's To path against the
// substitution values and returns absolute repo-rooted destinations.
// The From paths are left template-relative — the caller resolves them
// against info.Path when copying. Errors signal misconfigured manifests
// (empty From/To, From referencing outside the template tree, etc.).
func resolveRelocations(root string, info TemplateInfo, values map[string]string) ([]ResolvedRelocation, error) {
	if len(info.Manifest.Relocations) == 0 {
		return nil, nil
	}
	repoRoot := filepath.Clean(root)
	resolved := make([]ResolvedRelocation, 0, len(info.Manifest.Relocations))
	for index, reloc := range info.Manifest.Relocations {
		from := strings.TrimSpace(reloc.From)
		if from == "" {
			return nil, fmt.Errorf("relocation %d: from is required", index)
		}
		// Reject path traversal so a manifest can't escape the template tree.
		cleanFrom := filepath.Clean(filepath.FromSlash(from))
		if cleanFrom == ".." || strings.HasPrefix(cleanFrom, ".."+string(filepath.Separator)) || filepath.IsAbs(cleanFrom) {
			return nil, fmt.Errorf("relocation %d: from %q must be a template-relative path", index, reloc.From)
		}
		toRendered := strings.TrimSpace(renderTemplateString(reloc.To, values))
		if toRendered == "" {
			return nil, fmt.Errorf("relocation %d: to is required (rendered from %q)", index, reloc.To)
		}
		toAbs := toRendered
		if !filepath.IsAbs(toAbs) {
			toAbs = filepath.Join(repoRoot, filepath.FromSlash(toAbs))
		}
		toAbs = filepath.Clean(toAbs)
		// The resolved To must stay within the repo root — relocations
		// declare in-repo placement, not arbitrary writes.
		rel, err := filepath.Rel(repoRoot, toAbs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("relocation %d: to %q resolves outside repo root", index, reloc.To)
		}
		resolved = append(resolved, ResolvedRelocation{
			Description: reloc.Description,
			From:        cleanFrom,
			To:          toAbs,
			Post:        reloc.Post,
		})
	}
	return resolved, nil
}

// runRelocations writes each resolved relocation to disk, substituting
// {{...}} placeholders in both file content and path components, and
// then invokes Post commands at the repo root. It must run AFTER
// copyTemplate so the in-tree skip-list has already filtered the
// relocated source dirs out of the scenario destination.
func runRelocations[C any](deps HandlerDeps[C], ctx C, templateDir string, relocations []ResolvedRelocation, values map[string]string, output io.Writer) error {
	if len(relocations) == 0 {
		return nil
	}
	if output == nil {
		output = io.Discard
	}
	for _, reloc := range relocations {
		srcDir := filepath.Join(templateDir, reloc.From)
		stat, err := os.Stat(srcDir)
		if err != nil {
			return fmt.Errorf("relocation source %q: %w", reloc.From, err)
		}
		if !stat.IsDir() {
			return fmt.Errorf("relocation source %q is not a directory", reloc.From)
		}
		if err := copyRelocationTree(srcDir, reloc.To, values); err != nil {
			return fmt.Errorf("relocate %s -> %s: %w", reloc.From, reloc.To, err)
		}
		if err := verifyTemplate(reloc.To); err != nil {
			return fmt.Errorf("relocate %s -> %s: %w", reloc.From, reloc.To, err)
		}
	}
	// Post commands run from the repo root, NOT the scenario destination —
	// this is the structural difference from runTemplateHooks. They're
	// declared per-relocation but executed in declaration order after every
	// relocation has been written, so a single `make generate` covers all
	// of them when multiple relocations are siblings.
	repoRoot := deps.Root(ctx)
	for _, reloc := range relocations {
		for _, hook := range reloc.Post {
			cmd := strings.TrimSpace(hook.Cmd)
			if cmd == "" {
				continue
			}
			cwd := repoRoot
			if hookCwd := strings.TrimSpace(hook.Cwd); hookCwd != "" && hookCwd != "." {
				cwd = filepath.Join(repoRoot, filepath.FromSlash(hookCwd))
			}
			description := strings.TrimSpace(hook.Description)
			if description == "" {
				description = cmd
			}
			_, _ = fmt.Fprintf(output, "[Relocation post] %s\n", description)
			if err := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
				Name:   "bash",
				Args:   []string{"-lc", cmd},
				Dir:    cwd,
				Env:    deps.CommandEnv(ctx),
				Stdout: output,
				Stderr: deps.Stderr(ctx),
			}); err != nil {
				return fmt.Errorf("relocation post-command %q: %w", cmd, err)
			}
		}
	}
	return nil
}

// cleanupRelocationTargets removes the resolved To paths and any artifacts
// each post-command would have produced under packages/proto/gen/ for the
// validation scenario. Best-effort: errors are swallowed because the
// validation flow has already completed by the time cleanup runs.
//
// The proto/gen/ artifact paths mirror the repository's generated output
// layout. Go uses the scenario id directly, TypeScript nests generated JS
// under gen/typescript/js/<scenario-id>, and Python rewrites hyphens to
// underscores for package names.
func cleanupRelocationTargets(relocations []ResolvedRelocation) {
	for _, path := range relocationArtifactPaths(relocations) {
		_ = os.RemoveAll(path)
	}
}

func relocationArtifactPaths(relocations []ResolvedRelocation) []string {
	targets := make([]string, 0, len(relocations))
	for _, reloc := range relocations {
		targets = append(targets, reloc.To)
	}
	return templatevalidation.RelocationArtifactPaths(targets)
}

// copyRelocationTree mirrors copyTemplate's substitution logic but writes
// into an arbitrary repo-relative target instead of the scenario destination.
// File mode is preserved from the source; path components and text content
// are both rendered through renderTemplateString.
func copyRelocationTree(srcDir, destDir string, values map[string]string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(srcDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}
		if entry.IsDir() && shouldSkipTemplateCopyDir(entry.Name()) {
			return filepath.SkipDir
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if filepath.Base(path) == ".DS_Store" {
			return nil
		}
		targetPath := filepath.Join(destDir, renderTemplateString(relPath, values))
		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if looksLikeTextFile(data) {
			data = []byte(renderTemplateString(string(data), values))
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}
