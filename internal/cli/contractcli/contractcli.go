package contractcli

import (
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/repocontractcheck"
)

type ValidationOutput struct {
	Success bool                     `json:"success"`
	Root    string                   `json:"root"`
	Schema  ValidationCheck          `json:"schema"`
	Report  repocontractcheck.Report `json:"report"`
}

type ValidationCheck struct {
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type ShowOutput struct {
	Success      bool                            `json:"success"`
	Root         string                          `json:"root"`
	ContractPath string                          `json:"contract_path"`
	Schema       string                          `json:"schema"`
	Version      string                          `json:"version"`
	Platform     repocontract.Platform           `json:"platform"`
	Markers      repocontract.RootMarkers        `json:"markers"`
	Layout       repocontract.Layout             `json:"layout"`
	Scenario     repocontract.ScenarioSpec       `json:"scenario"`
	Resource     repocontract.ResourceSpec       `json:"resource"`
	Globs        repocontract.GlobSpec           `json:"globs"`
	Environment  map[string]string               `json:"environment"`
	Sandbox      ShowSandbox                     `json:"sandbox"`
	Profiles     map[string]repocontract.Profile `json:"profiles"`
}

type ShowSandbox struct {
	FullRepoScopes      []string `json:"full_repo_scopes"`
	ScenarioScopePrefix string   `json:"scenario_scope_prefix"`
}

type ResolveScenarioOutput struct {
	Success  bool   `json:"success"`
	Root     string `json:"root"`
	Scenario string `json:"scenario"`
	File     string `json:"file"`
	Path     string `json:"path"`
}

type MatchGlobOutput struct {
	Success bool   `json:"success"`
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Matched bool   `json:"matched"`
}

func ResolveRoot() (string, error) {
	return repocontract.FindRepoRootFromEnvOrCWD()
}

func RunSchemaValidation(root string) (string, bool) {
	cmd := exec.Command("python3", filepath.Join(root, ".vrooli", "schemas", "validate-repo-contract.py"))
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	message := strings.TrimSpace(string(output))
	if err != nil {
		if message == "" {
			message = err.Error()
		}
		return message, false
	}
	if message == "" {
		message = "ok"
	}
	return message, true
}

func Validate(root string) (ValidationOutput, error) {
	schemaMessage, schemaPassed := RunSchemaValidation(root)
	report, err := repocontractcheck.Run(root)
	if err != nil {
		return ValidationOutput{}, fmt.Errorf("run repo-contract checks: %w", err)
	}

	return ValidationOutput{
		Success: schemaPassed && report.Success,
		Root:    root,
		Schema: ValidationCheck{
			Passed:  schemaPassed,
			Message: schemaMessage,
		},
		Report: report,
	}, nil
}

func LoadShowOutput() (ShowOutput, error) {
	contract, root, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return ShowOutput{}, err
	}
	return ShowOutput{
		Success:      true,
		Root:         root,
		ContractPath: filepath.Join(root, ".vrooli", "repo-contract.json"),
		Schema:       contract.Schema(),
		Version:      contract.Version(),
		Platform:     contract.Platform(),
		Markers:      contract.RootMarkers(),
		Layout:       contract.Layout(),
		Scenario:     contract.Scenario(),
		Resource:     contract.Resource(),
		Globs:        contract.Globs(),
		Environment:  contract.EnvironmentVariables(),
		Sandbox: ShowSandbox{
			FullRepoScopes:      contract.SandboxFullRepoScopes(),
			ScenarioScopePrefix: contract.SandboxScenarioScopePrefix(),
		},
		Profiles: LoadProfiles(contract),
	}, nil
}

func ResolveScenario(root, scenarioName, fileKey string) (ResolveScenarioOutput, error) {
	var (
		resolved string
		err      error
	)
	if fileKey == "" {
		resolved, err = repocontract.ResolveScenarioPath(root, scenarioName)
	} else {
		resolved, err = repocontract.ResolveScenarioFile(root, scenarioName, fileKey)
	}
	if err != nil {
		return ResolveScenarioOutput{}, err
	}
	return ResolveScenarioOutput{
		Success:  true,
		Root:     root,
		Scenario: scenarioName,
		File:     fileKey,
		Path:     resolved,
	}, nil
}

func MatchGlob(pattern, path string) (MatchGlobOutput, error) {
	matched, err := repocontract.MatchRepoGlob(pattern, path)
	if err != nil {
		return MatchGlobOutput{}, err
	}
	return MatchGlobOutput{
		Success: true,
		Pattern: pattern,
		Path:    path,
		Matched: matched,
	}, nil
}

func LoadProfiles(contract *repocontract.Contract) map[string]repocontract.Profile {
	names := []string{"mini_vrooli_bundle"}
	profiles := make(map[string]repocontract.Profile, len(names))
	for _, name := range names {
		profile, err := contract.Profile(name)
		if err == nil {
			profiles[name] = profile
		}
	}
	return profiles
}

func RenderValidate(w io.Writer, format cliout.Format, output ValidationOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, output)
	}
	status := "passed"
	if !output.Success {
		status = "failed"
	}
	if _, err := fmt.Fprintf(w, "Repo contract validation %s\n", status); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Root: %s\n", output.Root); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Schema: %s\n", RenderCheckLine(output.Schema.Passed, output.Schema.Message)); err != nil {
		return err
	}
	for _, check := range output.Report.Checks {
		if _, err := fmt.Fprintf(w, "%s: %s\n", check.Name, RenderCheckLine(check.Passed, check.Message)); err != nil {
			return err
		}
	}
	return nil
}

func RenderShow(w io.Writer, format cliout.Format, output ShowOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, output)
	}
	if _, err := fmt.Fprintln(w, "Repo contract"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Root: %s\n", output.Root); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Contract: %s\n", output.ContractPath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Version: %s\n", output.Version); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Platform mode: %s\n", output.Platform.Mode); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Markers: dirs=%s files=%s\n", strings.Join(output.Markers.RequiredDirs, ","), strings.Join(output.Markers.RequiredFiles, ",")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Layout: scenarios=%s resources=%s packages=%s cmd=%s internal=%s docs=%s\n",
		output.Layout.ScenarioDir, output.Layout.ResourceDir, output.Layout.PackageDir, output.Layout.CommandDir, output.Layout.InternalDir, output.Layout.DocsDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Scenario service path: %s\n", output.Scenario.WellKnownPaths["service"]); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Resource manifest path: %s\n", output.Resource.Manifest); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Glob policy: syntax=%s root_relative=%t case_sensitive=%t allow_absolute=%t path_format=%s\n",
		output.Globs.Syntax, output.Globs.RootRelative, output.Globs.CaseSensitive, output.Globs.AllowAbsolute, output.Globs.PathFormat); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Sandbox scopes: %s prefix=%s\n", strings.Join(output.Sandbox.FullRepoScopes, ","), output.Sandbox.ScenarioScopePrefix); err != nil {
		return err
	}
	profileNames := make([]string, 0, len(output.Profiles))
	for name := range output.Profiles {
		profileNames = append(profileNames, name)
	}
	slices.Sort(profileNames)
	if _, err := fmt.Fprintf(w, "Profiles: %s\n", strings.Join(profileNames, ",")); err != nil {
		return err
	}
	return nil
}

func RenderResolveScenario(w io.Writer, format cliout.Format, output ResolveScenarioOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, output)
	}
	_, err := fmt.Fprintln(w, output.Path)
	return err
}

func RenderMatchGlob(w io.Writer, format cliout.Format, output MatchGlobOutput) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, output)
	}
	if output.Matched {
		_, err := fmt.Fprintln(w, "matched")
		return err
	}
	_, err := fmt.Fprintln(w, "not matched")
	return err
}

func RenderCheckLine(passed bool, message string) string {
	if strings.TrimSpace(message) == "" {
		message = "ok"
	}
	if passed {
		return "PASS (" + message + ")"
	}
	return "FAIL (" + message + ")"
}
