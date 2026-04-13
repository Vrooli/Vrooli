package main

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

type contractValidateOutput struct {
	Success bool                         `json:"success"`
	Root    string                       `json:"root"`
	Schema  contractValidationCheck      `json:"schema"`
	Report  repocontractcheck.Report     `json:"report"`
}

type contractValidationCheck struct {
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type contractShowOutput struct {
	Success      bool              `json:"success"`
	Root         string            `json:"root"`
	ContractPath string            `json:"contract_path"`
	Schema       string            `json:"schema"`
	Version      string            `json:"version"`
	Platform     repocontract.Platform `json:"platform"`
	Markers      repocontract.RootMarkers `json:"markers"`
	Layout       repocontract.Layout `json:"layout"`
	Scenario     repocontract.ScenarioSpec `json:"scenario"`
	Resource     repocontract.ResourceSpec `json:"resource"`
	Globs        repocontract.GlobSpec `json:"globs"`
	Environment  map[string]string `json:"environment"`
	Sandbox      contractShowSandbox `json:"sandbox"`
	Profiles     map[string]repocontract.Profile `json:"profiles"`
}

type contractShowSandbox struct {
	FullRepoScopes      []string `json:"full_repo_scopes"`
	ScenarioScopePrefix string   `json:"scenario_scope_prefix"`
}

func (app *App) runContractCommand(ctx *commandContext, args []string) error {
	return runContractCommandWithApp(app, ctx, args)
}

func runContractCommandWithApp(app *App, ctx *commandContext, args []string) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		showContractHelp(ctx.Stdout)
		return nil
	}
	switch args[0] {
	case "validate":
		return runContractValidateCommand(ctx, args[1:])
	case "show":
		return runContractShowCommand(ctx, args[1:])
	case "resolve":
		return runContractResolveCommand(ctx, args[1:])
	case "match-glob":
		return runContractMatchGlobCommand(ctx, args[1:])
	default:
		return usageErrorf("contract", "unknown contract command: %s", args[0])
	}
}

func runContractValidateCommand(ctx *commandContext, args []string) error {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			showContractValidateHelp(ctx.Stdout)
			return nil
		default:
			return unknownOptionError("contract validate", arg)
		}
	}

	root, err := resolveContractRoot()
	if err != nil {
		return contractRootError(err)
	}
	schemaMessage, schemaPassed := runContractSchemaValidation(root)
	report, err := repocontractcheck.Run(root)
	if err != nil {
		return fmt.Errorf("run repo-contract checks: %w", err)
	}

	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	output := contractValidateOutput{
		Success: schemaPassed && report.Success,
		Root:    root,
		Schema: contractValidationCheck{
			Passed:  schemaPassed,
			Message: schemaMessage,
		},
		Report: report,
	}

	if format == cliout.FormatJSON {
		if err := cliout.WriteJSON(ctx.Stdout, output); err != nil {
			return err
		}
	} else {
		if err := writeContractValidateHuman(ctx.Stdout, output); err != nil {
			return err
		}
	}

	if output.Success {
		return nil
	}
	return exitCodeError{code: 1, silent: true}
}

func runContractShowCommand(ctx *commandContext, args []string) error {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			showContractShowHelp(ctx.Stdout)
			return nil
		default:
			return unknownOptionError("contract show", arg)
		}
	}

	contract, root, err := repocontract.LoadDefaultFromEnvOrCWD()
	if err != nil {
		return contractRootError(err)
	}
	output := contractShowOutput{
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
		Sandbox: contractShowSandbox{
			FullRepoScopes:      contract.SandboxFullRepoScopes(),
			ScenarioScopePrefix: contract.SandboxScenarioScopePrefix(),
		},
		Profiles: loadContractProfiles(contract),
	}

	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(ctx.Stdout, output)
	}
	return writeContractShowHuman(ctx.Stdout, output)
}

func runContractResolveCommand(ctx *commandContext, args []string) error {
	if len(args) == 0 || wantsCommandHelp(args) {
		showContractResolveHelp(ctx.Stdout)
		return nil
	}
	switch args[0] {
	case "scenario":
		return runContractResolveScenarioCommand(ctx, args[1:])
	default:
		return usageErrorf("contract resolve", "unknown resolve target: %s", args[0])
	}
}

func runContractResolveScenarioCommand(ctx *commandContext, args []string) error {
	if len(args) == 0 {
		return usageErrorf("contract resolve scenario", "contract resolve scenario requires a scenario name")
	}
	scenarioName := strings.TrimSpace(args[0])
	if scenarioName == "" {
		return usageErrorf("contract resolve scenario", "contract resolve scenario requires a scenario name")
	}

	fileKey := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			showContractResolveScenarioHelp(ctx.Stdout)
			return nil
		case "--file":
			if i+1 >= len(args) {
				return usageErrorf("contract resolve scenario", "missing value for --file")
			}
			fileKey = strings.TrimSpace(args[i+1])
			i++
		default:
			return unknownOptionError("contract resolve scenario", args[i])
		}
	}

	root, err := resolveContractRoot()
	if err != nil {
		return contractRootError(err)
	}
	var resolved string
	if fileKey == "" {
		resolved, err = repocontract.ResolveScenarioPath(root, scenarioName)
	} else {
		resolved, err = repocontract.ResolveScenarioFile(root, scenarioName, fileKey)
	}
	if err != nil {
		return err
	}

	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(ctx.Stdout, map[string]any{
			"success":  true,
			"root":     root,
			"scenario": scenarioName,
			"file":     fileKey,
			"path":     resolved,
		})
	}
	_, _ = fmt.Fprintln(ctx.Stdout, resolved)
	return nil
}

func runContractMatchGlobCommand(ctx *commandContext, args []string) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			showContractMatchGlobHelp(ctx.Stdout)
			return nil
		}
	}
	if len(args) != 2 {
		return usageErrorf("contract match-glob", "usage: vrooli contract match-glob <pattern> <path>")
	}

	matched, err := repocontract.MatchRepoGlob(args[0], args[1])
	if err != nil {
		return err
	}
	format, err := parseOutputFormat(ctx.Globals)
	if err != nil {
		return err
	}
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(ctx.Stdout, map[string]any{
			"success": true,
			"pattern": args[0],
			"path":    args[1],
			"matched": matched,
		})
	}
	if matched {
		_, _ = fmt.Fprintln(ctx.Stdout, "matched")
		return nil
	}
	_, _ = fmt.Fprintln(ctx.Stdout, "not matched")
	return nil
}

func resolveContractRoot() (string, error) {
	return repocontract.FindRepoRootFromEnvOrCWD()
}

func contractRootError(err error) error {
	return newErrorWithCategory(fmt.Errorf("resolve repo contract root: %w", err), errorCategoryEnvironment, "Run from a Vrooli repository descendant or set VROOLI_SOURCE_ROOT", nil)
}

func runContractSchemaValidation(root string) (string, bool) {
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

func loadContractProfiles(contract *repocontract.Contract) map[string]repocontract.Profile {
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

func writeContractValidateHuman(w io.Writer, output contractValidateOutput) error {
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
	if _, err := fmt.Fprintf(w, "Schema: %s\n", renderCheckLine(output.Schema.Passed, output.Schema.Message)); err != nil {
		return err
	}
	for _, check := range output.Report.Checks {
		if _, err := fmt.Fprintf(w, "%s: %s\n", check.Name, renderCheckLine(check.Passed, check.Message)); err != nil {
			return err
		}
	}
	return nil
}

func writeContractShowHuman(w io.Writer, output contractShowOutput) error {
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

func renderCheckLine(passed bool, message string) string {
	if strings.TrimSpace(message) == "" {
		message = "ok"
	}
	if passed {
		return "PASS (" + message + ")"
	}
	return "FAIL (" + message + ")"
}

func showContractHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "vrooli contract - Inspect and validate the repository contract")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli contract validate")
	_, _ = fmt.Fprintln(w, "  vrooli contract show")
	_, _ = fmt.Fprintln(w, "  vrooli contract resolve scenario <name> [--file <key>]")
	_, _ = fmt.Fprintln(w, "  vrooli contract match-glob <pattern> <path>")
}

func showContractValidateHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract validate [--json]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Runs schema validation plus in-process semantic and live drift checks.")
}

func showContractShowHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract show [--json]")
}

func showContractResolveHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract resolve scenario <name> [--file <key>] [--json]")
}

func showContractResolveScenarioHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract resolve scenario <name> [--file <key>] [--json]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Known keys: service, docs, requirements, api, ui, cli, initialization")
}

func showContractMatchGlobHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage: vrooli contract match-glob <pattern> <path> [--json]")
}
