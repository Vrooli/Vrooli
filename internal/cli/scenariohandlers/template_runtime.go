package scenariohandlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/config"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

var unresolvedTemplatePattern = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)

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
	if info.Missing {
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  "template.json is missing",
		}}
	}
	if deps.RunSubprocess == nil {
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  "deep validation requires subprocess execution",
		}}
	}
	if deps.LocateTestGenieCLI == nil {
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  "deep validation requires test-genie CLI resolution",
		}}
	}
	cleanupStaleTemplateValidationRuns(deps, ctx)
	testGenieCLI, err := deps.LocateTestGenieCLI(ctx)
	if err != nil {
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  err.Error(),
		}}
	}
	tempRoot, err := os.MkdirTemp("", "vrooli-template-deep-*")
	if err != nil {
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("create deep validation temp dir: %v", err),
		}}
	}
	run.TempRoot = tempRoot
	run.RetainedTemp = req.RetainTemp
	destination := filepath.Join(tempRoot, "scenarios", scenarioID)
	run.ScenarioPath = destination
	marker, err := templatevalidation.NewRunMarker(templatevalidation.NewRunMarkerInput{
		RepoRoot:     deps.Root(ctx),
		Template:     info.Name,
		ScenarioID:   scenarioID,
		ScenarioPath: destination,
		TempRoot:     tempRoot,
		Retained:     req.RetainTemp,
	})
	if err != nil {
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("create deep validation marker: %v", err),
		}}
	}
	run.RunID = marker.RunID
	if err := templatevalidation.WriteMarker(marker); err != nil {
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("write deep validation marker: %v", err),
		}}
	}
	cleanupTemp := true
	if req.RetainTemp {
		cleanupTemp = false
		run.CleanupStatus = "retained"
		run.CleanupCommand = "vrooli scenario template cleanup --run " + marker.RunID
	}
	defer func() {
		marker.Completed = true
		marker.CleanupStatus = run.CleanupStatus
		if cleanupTemp {
			if err := os.RemoveAll(tempRoot); err != nil {
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
	}()

	if err := prepareDeepValidationWorkspace(deps.Root(ctx), tempRoot); err != nil {
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("prepare deep validation workspace: %v", err),
		}}
	}
	relocations, issues := generateDeepValidationScenario(deps, ctx, info, scenarioID, destination)
	run.RelocationArtifacts = relocationArtifactPaths(relocations)
	marker.RelocationArtifacts = append([]string(nil), run.RelocationArtifacts...)
	_ = templatevalidation.WriteMarker(marker)
	if len(relocations) > 0 && !req.RetainTemp {
		defer cleanupRelocationTargets(relocations)
	}
	if len(issues) > 0 {
		return run, issues
	}
	var stdout, stderr bytes.Buffer
	args := []string{
		"execute",
		scenarioID,
		"--scenario-path", destination,
		"--logical-repo-root", deps.Root(ctx),
		"--logical-scenario-relpath", filepath.Join("scenarios", scenarioID),
		"--preset", run.TestPreset,
		"--no-stream",
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
		result := parseTestGenieJSONResult(info.Name, stdout.Bytes())
		if req.WarningPolicy != TemplateValidationWarningPolicyIgnore {
			run.WarningSummary = result.WarningSummary
		}
		if result.Issue != nil && result.Success != nil && !*result.Success {
			return run, []TemplateValidationIssue{*result.Issue}
		}
		if issue := testGenieFailureIssueFromJSON(info.Name, stdout.Bytes()); issue != nil {
			return run, []TemplateValidationIssue{*issue}
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("test-genie deep validation failed: %s", truncateForIssue(message, 2000)),
		}}
	}
	result := parseTestGenieJSONResult(info.Name, stdout.Bytes())
	if req.WarningPolicy != TemplateValidationWarningPolicyIgnore {
		run.WarningSummary = result.WarningSummary
	}
	if result.Issue != nil {
		return run, []TemplateValidationIssue{*result.Issue}
	}
	if req.WarningPolicy == TemplateValidationWarningPolicyFail && result.WarningSummary.Total > 0 {
		return run, []TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("test-genie deep validation reported %d warning(s)", run.WarningSummary.Total),
		}}
	}
	return run, nil
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
		return parsedTestGenieResult{Issue: &TemplateValidationIssue{
			Template: templateName,
			Message:  "test-genie deep validation produced no JSON output",
		}}
	}
	var response testGenieResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return parsedTestGenieResult{Issue: &TemplateValidationIssue{
			Template: templateName,
			Message:  fmt.Sprintf("test-genie deep validation returned invalid JSON: %v", err),
		}}
	}
	result := parsedTestGenieResult{
		Success:        response.Success,
		WarningSummary: response.WarningSummary,
	}
	if response.Success == nil {
		result.Issue = &TemplateValidationIssue{
			Template: templateName,
			Message:  "test-genie deep validation JSON omitted success",
		}
		return result
	}
	if *response.Success {
		return result
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
	result.Issue = &TemplateValidationIssue{
		Template: templateName,
		Message:  "test-genie deep validation failed: " + truncateForIssue(summary, 2000),
	}
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
	destination := opts.Destination
	if destination == "" {
		destination = filepath.Join(deps.Root(ctx), "scenarios", opts.Values["SCENARIO_ID"])
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(deps.Root(ctx), filepath.FromSlash(destination))
	}
	destination = filepath.Clean(destination)
	values, err := buildTemplateValues(deps.Root(ctx), destination, info.Name, info.Manifest, opts.Values)
	if err != nil {
		return "", GenerateResult{}, err
	}
	resolved, err := resolveRelocations(deps.Root(ctx), info, values)
	if err != nil {
		return "", GenerateResult{}, err
	}
	design, err := resolveDesign(deps.Root(ctx), info, opts.Design, destination, values)
	if err != nil {
		return "", GenerateResult{}, err
	}
	provenance := buildGenerationProvenance(info, design, time.Now().UTC())
	if opts.DryRun {
		return cliout.FormatHuman, GenerateResult{
			TemplateName: info.Name,
			DisplayName:  coalesce(values["SCENARIO_DISPLAY_NAME"], values["SCENARIO_ID"]),
			Destination:  destination,
			Values:       values,
			Manifest:     info.Manifest,
			Design:       design,
			Relocations:  resolved,
			Provenance:   provenance,
			DryRun:       true,
		}, nil
	}
	if stat, err := os.Stat(destination); err == nil && stat != nil {
		if !opts.Force {
			return "", GenerateResult{}, fmt.Errorf("destination already exists: %s (use --force to overwrite)", destination)
		}
		if err := os.RemoveAll(destination); err != nil {
			return "", GenerateResult{}, err
		}
	}
	// Pre-flight every relocation target so the generator never partially
	// commits when a later target collides. This mirrors the destination
	// guard above; with --force we remove first, otherwise we error fast.
	for _, reloc := range resolved {
		if stat, err := os.Stat(reloc.To); err == nil && stat != nil {
			if !opts.Force {
				return "", GenerateResult{}, fmt.Errorf("relocation target already exists: %s (use --force to overwrite)", reloc.To)
			}
			if err := os.RemoveAll(reloc.To); err != nil {
				return "", GenerateResult{}, err
			}
		}
	}
	if err := preflightDesignCopies(design, opts.Force); err != nil {
		return "", GenerateResult{}, err
	}
	if err := preflightDesignTemplateCollisions(info.Path, destination, design); err != nil {
		return "", GenerateResult{}, err
	}
	if err := copyTemplate(info.Path, destination, values, info.Manifest); err != nil {
		return "", GenerateResult{}, err
	}
	if err := copyDesignAssets(design, values); err != nil {
		return "", GenerateResult{}, err
	}
	if err := injectScenarioProvenance(destination, provenance); err != nil {
		return "", GenerateResult{}, err
	}
	if err := renderOrientationManifest(destination, info.Manifest, values); err != nil {
		return "", GenerateResult{}, err
	}
	if err := verifyTemplate(destination); err != nil {
		return "", GenerateResult{}, err
	}
	if err := runRelocations(deps, ctx, info.Path, resolved, values, deps.Stdout(ctx)); err != nil {
		return "", GenerateResult{}, err
	}
	if issues := validateGeneratedScenario(destination, deps.RunSubprocess != nil, func(spec scenarioexec.SubprocessSpec) error {
		spec.Env = deps.CommandEnv(ctx)
		spec.Stdout = io.Discard
		spec.Stderr = deps.Stderr(ctx)
		return deps.RunSubprocess(ctx, spec)
	}, info.Name, info.Manifest); len(issues) > 0 {
		return "", GenerateResult{}, fmt.Errorf("%s", formatTemplateValidationIssues(issues))
	}
	result := GenerateResult{
		TemplateName: info.Name,
		DisplayName:  coalesce(values["SCENARIO_DISPLAY_NAME"], values["SCENARIO_ID"]),
		Destination:  destination,
		Values:       values,
		Manifest:     info.Manifest,
		Design:       design,
		Relocations:  resolved,
		Provenance:   provenance,
		RunHooks:     opts.RunHooks,
	}
	if opts.RunHooks {
		if err := runTemplateHooks(deps, ctx, destination, info.Manifest, deps.Stdout(ctx)); err != nil {
			return "", GenerateResult{}, err
		}
	}
	return cliout.FormatHuman, result, nil
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

func loadTemplates(root string) ([]TemplateInfo, error) {
	baseDir := config.TemplateBaseDir(root)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	templates := make([]TemplateInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		info, err := loadTemplate(root, name)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				templates = append(templates, TemplateInfo{Name: name, Path: filepath.Join(baseDir, name), Missing: true})
				continue
			}
			return nil, err
		}
		templates = append(templates, info)
	}
	sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })
	return templates, nil
}

func loadTemplate(root, name string) (TemplateInfo, error) {
	templateDir := filepath.Join(config.TemplateBaseDir(root), name)
	info, err := os.Stat(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return TemplateInfo{}, fmt.Errorf("template not found: %s", name)
		}
		return TemplateInfo{}, err
	}
	if !info.IsDir() {
		return TemplateInfo{}, fmt.Errorf("template path is not a directory: %s", templateDir)
	}
	data, err := os.ReadFile(filepath.Join(templateDir, "template.json"))
	if err != nil {
		return TemplateInfo{}, err
	}
	var manifest TemplateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return TemplateInfo{}, err
	}
	if manifest.Name == "" {
		manifest.Name = name
	}
	if manifest.RequiredVars == nil {
		manifest.RequiredVars = map[string]TemplateVar{}
	}
	if manifest.OptionalVars == nil {
		manifest.OptionalVars = map[string]TemplateVar{}
	}
	if manifest.Docs == nil {
		manifest.Docs = map[string]string{}
	}
	return TemplateInfo{Name: name, Path: templateDir, Manifest: manifest}, nil
}

func buildTemplateValues(root, destination, templateName string, manifest TemplateManifest, baseValues map[string]string) (map[string]string, error) {
	currentDate := time.Now().UTC().Format("2006-01-02")
	randomToken, err := randomTemplateToken()
	if err != nil {
		return nil, err
	}
	values := copyStringMap(baseValues)
	values["CURRENT_DATE"] = currentDate
	values["RANDOM_TOKEN"] = randomToken
	if err := populateTemplatePathValues(root, destination, values); err != nil {
		return nil, fmt.Errorf("resolve template path placeholders for %s: %w", templateName, err)
	}
	optionalKeys := make([]string, 0, len(manifest.OptionalVars))
	for key := range manifest.OptionalVars {
		optionalKeys = append(optionalKeys, key)
	}
	sort.Strings(optionalKeys)
	for _, key := range optionalKeys {
		if strings.TrimSpace(values[key]) == "" {
			values[key] = renderTemplateString(manifest.OptionalVars[key].Default, values)
		}
	}
	// Derive snake_case identifiers from kebab-case scenario IDs so proto
	// package directives (which forbid hyphens), Go package aliases, and
	// Python module names get a valid identifier without each template
	// having to re-implement the conversion.
	if id, ok := values["SCENARIO_ID"]; ok && strings.TrimSpace(id) != "" {
		values["SCENARIO_ID_SNAKE"] = strings.ReplaceAll(id, "-", "_")
	}
	return values, nil
}

func populateTemplatePathValues(root, destination string, values map[string]string) error {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return err
	}
	repoRoot := filepath.Clean(root)
	packagesDir, err := contract.TopLevelDir(root, "packages")
	if err != nil {
		return err
	}
	for key, dir := range map[string]string{
		"API":     filepath.Join(destination, "api"),
		"CLI":     filepath.Join(destination, "cli"),
		"RUNTIME": filepath.Join(destination, "runtime"),
	} {
		repoRel, err := filepath.Rel(dir, repoRoot)
		if err != nil {
			return err
		}
		packagesRel, err := filepath.Rel(dir, packagesDir)
		if err != nil {
			return err
		}
		values["REPO_ROOT_REL_FROM_"+key] = filepath.ToSlash(repoRel)
		values["PACKAGES_REL_FROM_"+key] = filepath.ToSlash(packagesRel)
	}
	return nil
}

func copyTemplate(templateDir, destination string, values map[string]string, manifest TemplateManifest) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	// Build the set of relocation source paths (template-relative, cleaned)
	// so the walk can prune them — they're handled by runRelocations and
	// must not also land in the scenario destination.
	relocSources := make(map[string]struct{}, len(manifest.Relocations))
	for _, reloc := range manifest.Relocations {
		from := strings.TrimSpace(reloc.From)
		if from == "" {
			continue
		}
		relocSources[filepath.Clean(filepath.FromSlash(from))] = struct{}{}
	}
	copyExcludes := make(map[string]struct{}, len(manifest.CopyExcludes))
	for _, exclude := range manifest.CopyExcludes {
		exclude = strings.TrimSpace(exclude)
		if exclude == "" {
			continue
		}
		copyExcludes[filepath.Clean(filepath.FromSlash(exclude))] = struct{}{}
	}
	return filepath.WalkDir(templateDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == templateDir {
			return nil
		}
		if entry.IsDir() && shouldSkipTemplateCopyDir(entry.Name()) {
			return filepath.SkipDir
		}
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if _, skip := relocSources[filepath.Clean(relPath)]; skip {
				return filepath.SkipDir
			}
			if _, skip := copyExcludes[filepath.Clean(relPath)]; skip {
				return filepath.SkipDir
			}
		}
		if filepath.Base(path) == ".DS_Store" || relPath == "template.json" {
			return nil
		}
		if _, skip := copyExcludes[filepath.Clean(relPath)]; skip {
			return nil
		}
		targetPath := filepath.Join(destination, renderTemplateString(relPath, values))
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

func buildGenerationProvenance(info TemplateInfo, design ResolvedDesign, now time.Time) GenerationProvenance {
	return GenerationProvenance{
		Template: GenerationTemplate{
			ID:      info.Name,
			Version: strings.TrimSpace(info.Manifest.Version),
		},
		GeneratedAt: now.UTC().Format(time.RFC3339),
		Design: GenerationDesign{
			ID:      strings.TrimSpace(design.KitID),
			Version: strings.TrimSpace(design.Version),
			Adapter: strings.TrimSpace(design.AdapterID),
		},
	}
}

func injectScenarioProvenance(destination string, provenance GenerationProvenance) error {
	servicePath := filepath.Join(destination, ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read service manifest: %w", err)
	}
	var manifest scenariomodel.ServiceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse service manifest: %w", err)
	}
	manifest.Generation = &scenariomodel.GenerationMetadata{
		Template: scenariomodel.GenerationTemplate{
			ID:      provenance.Template.ID,
			Version: provenance.Template.Version,
		},
		GeneratedAt: provenance.GeneratedAt,
		Design: scenariomodel.GenerationDesign{
			ID:      provenance.Design.ID,
			Version: provenance.Design.Version,
			Adapter: provenance.Design.Adapter,
		},
	}
	rendered, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("render service manifest: %w", err)
	}
	rendered = append(rendered, '\n')
	return os.WriteFile(servicePath, rendered, 0o644)
}

func renderOrientationManifest(destination string, manifest TemplateManifest, values map[string]string) error {
	if manifest.Orientation == nil {
		return nil
	}
	copyTo := strings.TrimSpace(manifest.Orientation.CopyTo)
	if copyTo == "" {
		return fmt.Errorf("orientation.copyTo is required")
	}
	cleanPath, err := cleanScenarioRelativePath(copyTo)
	if err != nil {
		return fmt.Errorf("orientation.copyTo: %w", err)
	}
	data, err := json.MarshalIndent(manifest.Orientation, "", "  ")
	if err != nil {
		return fmt.Errorf("render orientation manifest: %w", err)
	}
	data = []byte(renderTemplateString(string(data), values))
	target := filepath.Join(destination, cleanPath)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(target, data, 0o644)
}

func verifyTemplate(destination string) error {
	var unresolved []string
	err := filepath.WalkDir(destination, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if unresolvedTemplatePattern.MatchString(path) {
			unresolved = append(unresolved, path)
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if looksLikeTextFile(data) && unresolvedTemplatePattern.Match(data) {
			unresolved = append(unresolved, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(unresolved) == 0 {
		return nil
	}
	sort.Strings(unresolved)
	return fmt.Errorf("unresolved placeholders remain in: %s", strings.Join(unresolved, ", "))
}

func shouldSkipTemplateCopyDir(name string) bool {
	switch strings.TrimSpace(name) {
	case "node_modules", "dist", "build", "coverage", ".turbo", ".vite":
		return true
	default:
		return false
	}
}

func looksLikeTextFile(data []byte) bool {
	return len(data) == 0 || (bytes.IndexByte(data, 0) < 0 && utf8.Valid(data))
}

func LooksLikeTextFile(data []byte) bool {
	return looksLikeTextFile(data)
}

func renderTemplateString(value string, values map[string]string) string {
	rendered := value
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", values[key])
	}
	return rendered
}

func templateValidationSeedValues(info TemplateInfo) map[string]string {
	values := map[string]string{}
	requiredKeys := make([]string, 0, len(info.Manifest.RequiredVars))
	for key := range info.Manifest.RequiredVars {
		requiredKeys = append(requiredKeys, key)
	}
	sort.Strings(requiredKeys)
	for _, key := range requiredKeys {
		switch key {
		case "SCENARIO_ID":
			values[key] = "template-validation-" + info.Name
		case "SCENARIO_DISPLAY_NAME":
			values[key] = coalesce(info.Manifest.DisplayName, info.Name+" Validation")
		case "SCENARIO_DESCRIPTION":
			values[key] = coalesce(info.Manifest.Description, "Validation scenario generated from "+info.Name)
		default:
			if fallback := strings.TrimSpace(info.Manifest.RequiredVars[key].Default); fallback != "" {
				values[key] = fallback
			} else {
				values[key] = strings.ToLower(strings.ReplaceAll(key, "_", "-"))
			}
		}
	}
	return values
}

func templateValidationSeedValuesForScenarioID(info TemplateInfo, scenarioID string) map[string]string {
	values := templateValidationSeedValues(info)
	if strings.TrimSpace(scenarioID) != "" {
		values["SCENARIO_ID"] = scenarioID
	}
	return values
}

func truncateForIssue(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "... (truncated)"
}

func validateTemplateSource(info TemplateInfo) []TemplateValidationIssue {
	if info.Missing {
		return nil
	}
	var issues []TemplateValidationIssue
	issues = append(issues, validateOrientationSource(info)...)
	_ = filepath.WalkDir(info.Path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Base(path) != "go.mod" {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(filepath.Base(path)),
				Message:  fmt.Sprintf("read go.mod: %v", readErr),
			})
			return nil
		}
		for _, target := range parseLocalReplaceTargets(string(data)) {
			if strings.Contains(target, "{{") {
				continue
			}
			rel, relErr := filepath.Rel(info.Path, path)
			if relErr != nil {
				rel = path
			}
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(rel),
				Message:  fmt.Sprintf("go.mod local replace target %q must use generator-computed placeholders", target),
			})
		}
		return nil
	})
	return issues
}

func validateOrientationSource(info TemplateInfo) []TemplateValidationIssue {
	if info.Manifest.Orientation == nil {
		return nil
	}
	orientation := info.Manifest.Orientation
	var issues []TemplateValidationIssue
	add := func(path, message string) {
		issues = append(issues, TemplateValidationIssue{Template: info.Name, Path: path, Message: message})
	}
	if strings.TrimSpace(info.Manifest.Version) == "" {
		add("template.json", "version is required when orientation is declared")
	}
	if _, err := cleanScenarioRelativePath(orientation.CopyTo); err != nil {
		add("orientation.copyTo", err.Error())
	}
	startDocument := strings.TrimSpace(orientation.StartDocument)
	if startDocument == "" {
		startDocument = strings.TrimSpace(info.Manifest.StartDocument)
	}
	if startDocument != "" {
		if _, err := cleanScenarioRelativePath(startDocument); err != nil {
			add("orientation.startDocument", err.Error())
		}
	}
	seen := map[string]struct{}{}
	for index, step := range orientation.Steps {
		stepPath := fmt.Sprintf("orientation.steps[%d]", index)
		id := strings.TrimSpace(step.ID)
		if id == "" {
			add(stepPath, "step id is required")
		} else if _, ok := seen[id]; ok {
			add(stepPath, fmt.Sprintf("duplicate step id %q", id))
		}
		seen[id] = struct{}{}
		required := orientationStepRequired(step)
		if required && len(step.Checks) == 0 {
			add(stepPath, "required step must declare at least one check")
		}
		for checkIndex, check := range step.Checks {
			checkPath := fmt.Sprintf("%s.checks[%d]", stepPath, checkIndex)
			if !validOrientationCheckKind(check.Kind) {
				add(checkPath, fmt.Sprintf("unknown check kind %q", check.Kind))
			}
			switch check.Kind {
			case "file_exists", "file_absent", "directory_exists":
				if _, err := cleanScenarioRelativePath(check.Path); err != nil {
					add(checkPath+".path", err.Error())
				}
			case "glob_present", "glob_absent":
				if strings.TrimSpace(check.Pattern) == "" {
					add(checkPath+".pattern", "pattern is required")
				} else if _, err := cleanScenarioRelativePath(check.Pattern); err != nil {
					add(checkPath+".pattern", err.Error())
				}
			case "json_path_exists":
				if _, err := cleanScenarioRelativePath(check.Path); err != nil {
					add(checkPath+".path", err.Error())
				}
				if strings.TrimSpace(check.Query) == "" {
					add(checkPath+".query", "query is required")
				}
			case "text_contains", "text_absent":
				if _, err := cleanScenarioRelativePath(check.Path); err != nil {
					add(checkPath+".path", err.Error())
				}
				if strings.TrimSpace(check.Text) == "" {
					add(checkPath+".text", "text is required")
				}
			case "command":
				if strings.TrimSpace(check.Run) == "" {
					add(checkPath+".run", "run is required")
				}
				if strings.TrimSpace(check.Timeout) == "" {
					add(checkPath+".timeout", "timeout is required")
				} else if _, err := time.ParseDuration(check.Timeout); err != nil {
					add(checkPath+".timeout", fmt.Sprintf("invalid timeout: %v", err))
				}
			}
		}
	}
	for _, cleanup := range orientation.Finalize.Cleanup {
		clean, err := cleanScenarioRelativePath(cleanup)
		if err != nil {
			add("orientation.finalize.cleanup", err.Error())
			continue
		}
		if isDurableOrientationCleanupTarget(clean) {
			add("orientation.finalize.cleanup", fmt.Sprintf("cleanup path %q targets durable scenario content", cleanup))
		}
	}
	return issues
}

func orientationStepRequired(step TemplateOrientationStep) bool {
	return step.Required == nil || *step.Required
}

func validOrientationCheckKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "file_exists", "file_absent", "directory_exists", "glob_present", "glob_absent", "json_path_exists", "text_contains", "text_absent", "command":
		return true
	default:
		return false
	}
}

func cleanScenarioRelativePath(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	clean := filepath.Clean(filepath.FromSlash(trimmed))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%q must be a scenario-relative path", value)
	}
	return clean, nil
}

func isDurableOrientationCleanupTarget(path string) bool {
	slash := filepath.ToSlash(path)
	if slash == "docs" || strings.HasPrefix(slash, "docs/") || slash == "DESIGN.md" || slash == "requirements" || strings.HasPrefix(slash, "requirements/") {
		return true
	}
	for _, prefix := range []string{"api/", "cli/", "ui/", "proto/", "runtime/"} {
		if strings.HasPrefix(slash, prefix) {
			return true
		}
	}
	return false
}

// validateRelocationProtoSources runs `buf lint` against template-side
// proto source folders so schema-level mistakes (missing package
// directive, syntax errors, naming convention violations) surface in
// template validation rather than after a real scenario generation.
//
// The "is this proto?" decision is heuristic: any relocation source that
// contains a .proto file is treated as one. Future non-proto relocations
// (e.g., scripts) won't be confused for protos because they won't have
// .proto files inside.
//
// Implementation note: `buf lint --path` only accepts paths inside the
// buf module (packages/proto/schemas/). The template's protos live
// outside that module pre-substitution, so we copy them into a temp
// subdirectory under schemas/ with template-validation seed values
// applied, lint there, and clean up. The temp directory name is
// prefixed with `.tmp-validate-` so it can never collide with a real
// scenario schema directory.
//
// Skipped entirely when deps.RunSubprocess is nil (mirrors the pattern
// used by validateGeneratedScenario for `go mod tidy`).
func validateRelocationProtoSources[C any](deps HandlerDeps[C], ctx C, info TemplateInfo) []TemplateValidationIssue {
	if deps.RunSubprocess == nil {
		return nil
	}
	if len(info.Manifest.Relocations) == 0 {
		return nil
	}
	repoRoot := deps.Root(ctx)
	protoPackageDir := filepath.Join(repoRoot, "packages", "proto")
	schemasDir := filepath.Join(protoPackageDir, "schemas")
	if _, err := os.Stat(schemasDir); err != nil {
		// No proto module in this repo (e.g., test fixtures with a
		// minimal repo-contract). The template's claim is that protos
		// belong here, so absence isn't a per-template issue — the
		// generator would fail at make-generate time, which is a
		// separate failure mode.
		return nil
	}
	var issues []TemplateValidationIssue
	values := templateValidationSeedValues(info)
	if id, ok := values["SCENARIO_ID"]; ok && strings.TrimSpace(id) != "" {
		values["SCENARIO_ID_SNAKE"] = strings.ReplaceAll(id, "-", "_")
	}
	for _, reloc := range info.Manifest.Relocations {
		from := strings.TrimSpace(reloc.From)
		if from == "" {
			continue
		}
		srcDir := filepath.Join(info.Path, filepath.FromSlash(from))
		if !directoryContainsProto(srcDir) {
			continue
		}
		tmpDir, err := os.MkdirTemp(schemasDir, ".tmp-validate-"+info.Name+"-")
		if err != nil {
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(from),
				Message:  fmt.Sprintf("create lint temp dir: %v", err),
			})
			continue
		}
		// Best-effort cleanup; lint failures are surfaced through
		// `issues` regardless of whether the cleanup succeeds.
		shouldClean := true
		defer func(path string, doClean *bool) {
			if *doClean {
				_ = os.RemoveAll(path)
			}
		}(tmpDir, &shouldClean)
		if err := copyRelocationTree(srcDir, tmpDir, values); err != nil {
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(from),
				Message:  fmt.Sprintf("substitute proto sources for lint: %v", err),
			})
			continue
		}
		// `buf lint --path` is now scoped to the temp dir which lives
		// inside the buf module, so the lint succeeds.
		//
		// `buf lint` writes lint diagnostics to stdout (one per line) and
		// exits non-zero. We capture both streams and prefer stdout for the
		// surfaced message because that's where the actionable detail lives.
		// The temp-dir path prefix in each diagnostic line is also stripped
		// so the surfaced message matches what an author would see if they
		// ran `buf lint` directly against the template's proto/.
		var stdout, stderr bytes.Buffer
		relTmp, err := filepath.Rel(protoPackageDir, tmpDir)
		if err != nil {
			relTmp = tmpDir
		}
		err = deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   "bash",
			Args:   []string{"-lc", fmt.Sprintf("buf lint --path %s", shellQuote(relTmp))},
			Dir:    protoPackageDir,
			Env:    deps.CommandEnv(ctx),
			Stdout: &stdout,
			Stderr: &stderr,
		})
		if err != nil {
			msg := strings.TrimSpace(stdout.String())
			if msg == "" {
				msg = strings.TrimSpace(stderr.String())
			}
			if msg == "" {
				msg = err.Error()
			}
			// Strip the temp-dir prefix so diagnostics read as if buf lint
			// had been run against the template's source proto/ directly.
			fromPrefix := strings.TrimRight(filepath.ToSlash(from), "/") + "/"
			msg = strings.ReplaceAll(msg, filepath.ToSlash(relTmp)+"/", fromPrefix)
			issues = append(issues, TemplateValidationIssue{
				Template: info.Name,
				Path:     filepath.ToSlash(from),
				Message:  fmt.Sprintf("buf lint failed: %s", msg),
			})
		}
	}
	return issues
}

// directoryContainsProto reports whether the directory tree rooted at
// path contains any .proto files. Walks until the first match.
func directoryContainsProto(path string) bool {
	stat, err := os.Stat(path)
	if err != nil || !stat.IsDir() {
		return false
	}
	found := false
	_ = filepath.WalkDir(path, func(p string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(p), ".proto") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// shellQuote returns a single-quoted shell argument that survives buf's
// `bash -lc` invocation. Used for absolute paths that may contain
// shell-special characters; deliberately conservative.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func validateGeneratedScenario(destination string, runCommands bool, run func(scenarioexec.SubprocessSpec) error, templateName string, manifest TemplateManifest) []TemplateValidationIssue {
	var issues []TemplateValidationIssue
	issues = append(issues, validateGeneratedStartDocument(destination, templateName, manifest)...)
	_ = filepath.WalkDir(destination, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Base(path) != "go.mod" {
			return err
		}
		moduleDir := filepath.Dir(path)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			issues = append(issues, TemplateValidationIssue{
				Template: templateName,
				Path:     filepath.ToSlash(strings.TrimPrefix(path, destination+string(filepath.Separator))),
				Message:  fmt.Sprintf("read generated go.mod: %v", readErr),
			})
			return nil
		}
		for _, target := range parseLocalReplaceTargets(string(data)) {
			resolved := target
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(moduleDir, filepath.FromSlash(target))
			}
			if _, statErr := os.Stat(filepath.Clean(resolved)); statErr != nil {
				issues = append(issues, TemplateValidationIssue{
					Template: templateName,
					Path:     filepath.ToSlash(strings.TrimPrefix(path, destination+string(filepath.Separator))),
					Message:  fmt.Sprintf("go.mod replace target %q does not resolve from generated module: %v", target, statErr),
				})
			}
		}
		if runCommands && moduleHasGoFiles(moduleDir) {
			if execErr := run(scenarioexec.SubprocessSpec{
				Name: "bash",
				Args: []string{"-lc", "GOWORK=off go mod tidy"},
				Dir:  moduleDir,
			}); execErr != nil {
				issues = append(issues, TemplateValidationIssue{
					Template: templateName,
					Path:     filepath.ToSlash(strings.TrimPrefix(path, destination+string(filepath.Separator))),
					Message:  fmt.Sprintf("generated module validation failed: %v", execErr),
				})
			}
		}
		return nil
	})
	return issues
}

func validateGeneratedStartDocument(destination, templateName string, manifest TemplateManifest) []TemplateValidationIssue {
	startDocument := strings.TrimSpace(manifest.StartDocument)
	if startDocument == "" {
		return nil
	}
	cleanPath := filepath.Clean(filepath.FromSlash(startDocument))
	if cleanPath == "." || filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) || cleanPath == ".." {
		return []TemplateValidationIssue{{
			Template: templateName,
			Path:     startDocument,
			Message:  "startDocument must be a scenario-relative path",
		}}
	}
	stat, err := os.Stat(filepath.Join(destination, cleanPath))
	if err != nil {
		return []TemplateValidationIssue{{
			Template: templateName,
			Path:     filepath.ToSlash(cleanPath),
			Message:  fmt.Sprintf("startDocument is declared but missing from generated scenario: %v", err),
		}}
	}
	if stat.IsDir() {
		return []TemplateValidationIssue{{
			Template: templateName,
			Path:     filepath.ToSlash(cleanPath),
			Message:  "startDocument must point to a file, not a directory",
		}}
	}
	return nil
}

var goModReplaceLinePattern = regexp.MustCompile(`^\s*([A-Za-z0-9._/\-{}]+)(?:\s+[^\s]+)?\s*=>\s*([^\s]+)(?:\s+[^\s]+)?\s*(?://.*)?$`)

func parseLocalReplaceTargets(content string) []string {
	var targets []string
	var inReplaceBlock bool
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		switch line {
		case "replace (":
			inReplaceBlock = true
			continue
		case ")":
			inReplaceBlock = false
			continue
		}
		switch {
		case strings.HasPrefix(line, "replace "):
			if target, ok := parseGoReplaceTarget(strings.TrimSpace(strings.TrimPrefix(line, "replace "))); ok {
				targets = append(targets, target)
			}
		case inReplaceBlock:
			if target, ok := parseGoReplaceTarget(line); ok {
				targets = append(targets, target)
			}
		}
	}
	return targets
}

func parseGoReplaceTarget(line string) (string, bool) {
	matches := goModReplaceLinePattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return "", false
	}
	target := strings.TrimSpace(matches[2])
	if target == "" {
		return "", false
	}
	if !(strings.HasPrefix(target, ".") || strings.HasPrefix(target, "/") || strings.Contains(target, "{{")) {
		return "", false
	}
	return target, true
}

func moduleHasGoFiles(moduleDir string) bool {
	entries, err := os.ReadDir(moduleDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			return true
		}
	}
	return false
}

func formatTemplateValidationIssues(issues []TemplateValidationIssue) string {
	if len(issues) == 0 {
		return ""
	}
	lines := make([]string, 0, len(issues))
	for _, issue := range issues {
		line := issue.Template
		if strings.TrimSpace(issue.Path) != "" {
			line += " [" + issue.Path + "]"
		}
		line += ": " + issue.Message
		lines = append(lines, line)
	}
	return strings.Join(lines, "; ")
}

func runTemplateHooks[C any](deps HandlerDeps[C], ctx C, destination string, manifest TemplateManifest, output io.Writer) error {
	if output == nil {
		output = io.Discard
	}
	if len(manifest.PostHooks) == 0 {
		_, _ = fmt.Fprintln(output, "No post hooks defined for this template")
		return nil
	}
	for index, hook := range manifest.PostHooks {
		description := strings.TrimSpace(hook.Description)
		if description == "" {
			description = hook.Cmd
		}
		_, _ = fmt.Fprintf(output, "[Hook %d] %s\n", index+1, description)
		cwd := destination
		if strings.TrimSpace(hook.Cwd) != "" && hook.Cwd != "." {
			cwd = filepath.Join(destination, filepath.FromSlash(hook.Cwd))
		}
		if err := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   "bash",
			Args:   []string{"-lc", hook.Cmd},
			Dir:    cwd,
			Env:    deps.CommandEnv(ctx),
			Stdout: output,
			Stderr: deps.Stderr(ctx),
		}); err != nil {
			return err
		}
	}
	return nil
}

func randomTemplateToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func copyStringMap(values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func FormatTemplateRequiredFlags(manifest TemplateManifest) string {
	keys := make([]string, 0, len(manifest.RequiredVars))
	for key := range manifest.RequiredVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return " --id <slug>"
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		flag := manifest.RequiredVars[key].Flag
		if flag == "" {
			flag = strings.ToLower(key)
		}
		parts = append(parts, fmt.Sprintf(" --%s <%s>", flag, strings.ToLower(key)))
	}
	return strings.Join(parts, "")
}
