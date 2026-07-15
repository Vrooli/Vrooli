package templateengine

import (
	"bytes"
	"encoding/json"
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

func newTestGenieDeepValidationIssue(templateName, path, message string) templatecontracts.TemplateValidationIssue {
	return templatecontracts.TemplateValidationIssue{Template: templateName, Path: path, Message: message}
}

func initDeepValidationRun(repoRoot, templateName, scenarioID, testPreset string, retainTemp bool) (templatecontracts.TemplateValidationDeepRun, templatevalidation.RunMarker, bool, error) {
	run := templatecontracts.TemplateValidationDeepRun{
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

func finalizeDeepValidationRun(run *templatecontracts.TemplateValidationDeepRun, marker templatevalidation.RunMarker, cleanupTemp bool) {
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

func runDeepValidationTestGenie[C any](deps HandlerDeps[C], ctx C, templateName, testGenieCLI string, req templatecontracts.TemplateValidateRequest, run *templatecontracts.TemplateValidationDeepRun) []templatecontracts.TemplateValidationIssue {
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
		if req.WarningPolicy != templatecontracts.TemplateValidationWarningPolicyIgnore {
			run.WarningSummary = result.WarningSummary
		}
		if result.Issue != nil && result.Success != nil && !*result.Success {
			return []templatecontracts.TemplateValidationIssue{*result.Issue}
		}
		if issue := testGenieFailureIssueFromJSON(templateName, stdout.Bytes()); issue != nil {
			return []templatecontracts.TemplateValidationIssue{*issue}
		}
		return []templatecontracts.TemplateValidationIssue{newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationProtocolPath, "test-genie deep validation failed: "+deepValidationFailureMessage(stdout.String(), stderr.String(), err))}
	}
	result := parseTestGenieJSONResult(templateName, stdout.Bytes())
	if req.WarningPolicy != templatecontracts.TemplateValidationWarningPolicyIgnore {
		run.WarningSummary = result.WarningSummary
	}
	if result.Issue != nil {
		return []templatecontracts.TemplateValidationIssue{*result.Issue}
	}
	if req.WarningPolicy == templatecontracts.TemplateValidationWarningPolicyFail && result.WarningSummary.Total > 0 {
		return []templatecontracts.TemplateValidationIssue{newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationWarningsPath, fmt.Sprintf("test-genie deep validation reported %d warning(s)", run.WarningSummary.Total))}
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

func testGenieFailureIssueFromJSON(templateName string, output []byte) *templatecontracts.TemplateValidationIssue {
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

func validateTestGenieJSONSuccess(templateName string, output []byte) *templatecontracts.TemplateValidationIssue {
	return parseTestGenieJSONResult(templateName, output).Issue
}

type parsedTestGenieResult struct {
	Success        *bool
	WarningSummary templatecontracts.TemplateValidationWarningSummary
	Issue          *templatecontracts.TemplateValidationIssue
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
		WarningSummary templatecontracts.TemplateValidationWarningSummary `json:"warningSummary"`
	}

	data, extractErr := terminalTestGenieJSON(output)
	if len(data) == 0 {
		issue := newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationProtocolPath, "test-genie deep validation produced no JSON output")
		return parsedTestGenieResult{Issue: &issue}
	}
	if extractErr != nil {
		issue := newTestGenieDeepValidationIssue(templateName, testGenieDeepValidationProtocolPath, fmt.Sprintf("test-genie deep validation returned invalid JSON: %v", extractErr))
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

// terminalTestGenieJSON tolerates the durable-run handle emitted before a
// terminal result. The Test Genie --json contract keeps that handle on stderr,
// but accepting it here protects persisted template evidence when a wrapper
// merges streams. Only an object carrying success is a terminal result; an
// event-only handle never becomes a fabricated validation outcome.
func terminalTestGenieJSON(output []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(output))
	var last, terminal json.RawMessage
	for {
		var value json.RawMessage
		err := decoder.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		last = append(last[:0], value...)
		var envelope struct {
			Success *bool `json:"success"`
		}
		if err := json.Unmarshal(value, &envelope); err != nil {
			return nil, err
		}
		if envelope.Success != nil {
			terminal = append(terminal[:0], value...)
		}
	}
	if len(terminal) > 0 {
		return terminal, nil
	}
	return last, nil
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

func generateDeepValidationScenario[C any](deps HandlerDeps[C], ctx C, info templatecontracts.TemplateInfo, scenarioID, destination string) ([]templatecontracts.ResolvedRelocation, []templatecontracts.TemplateValidationIssue) {
	values, err := buildTemplateValues(
		deps.Root(ctx),
		destination,
		info.Name,
		info.Manifest,
		templateValidationSeedValuesForScenarioID(info, scenarioID),
	)
	if err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  err.Error(),
		}}
	}
	if err := copyTemplate(info.Path, destination, values, info.Manifest); err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("generate deep validation scenario: %v", err),
		}}
	}
	design, err := resolveDesign(deps.Root(ctx), info, "", destination, values)
	if err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("resolve default design: %v", err),
		}}
	}
	if err := preflightDesignCopies(design, true); err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("preflight default design: %v", err),
		}}
	}
	if err := preflightDesignTemplateCollisions(info.Path, destination, design); err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("preflight default design: %v", err),
		}}
	}
	if err := copyDesignAssets(design, values); err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("copy default design: %v", err),
		}}
	}
	provenance := buildGenerationProvenance(info, design, time.Now().UTC())
	if err := injectScenarioProvenance(destination, provenance); err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("write provenance: %v", err),
		}}
	}
	if err := renderOrientationManifest(destination, info.Manifest, values); err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("write orientation manifest: %v", err),
		}}
	}
	if err := verifyTemplate(destination); err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  err.Error(),
		}}
	}
	resolved, err := resolveRelocations(deps.Root(ctx), info, values)
	if err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("resolve relocations: %v", err),
		}}
	}
	if err := runRelocations(deps, ctx, info.Path, resolved, values, deps.Stderr(ctx)); err != nil {
		return nil, []templatecontracts.TemplateValidationIssue{{
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
		return resolved, []templatecontracts.TemplateValidationIssue{{
			Template: info.Name,
			Message:  fmt.Sprintf("run template post hooks: %v", err),
		}}
	}
	return resolved, nil
}
