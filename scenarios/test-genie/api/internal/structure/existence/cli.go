package existence

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CLIApproach represents the manifest-declared CLI adapter used by a scenario.
type CLIApproach int

const (
	// CLIApproachUnknown indicates the CLI contract could not be determined.
	CLIApproachUnknown CLIApproach = iota
	// CLIApproachGoModule indicates a Go module adapter.
	CLIApproachGoModule
	// CLIApproachShellScript indicates a shell script adapter.
	CLIApproachShellScript
)

// String returns a human-readable name for the CLI approach.
func (a CLIApproach) String() string {
	switch a {
	case CLIApproachGoModule:
		return "go_module"
	case CLIApproachShellScript:
		return "shell_script"
	default:
		return "unknown"
	}
}

// CLIResult contains the outcome of CLI structure validation.
type CLIResult struct {
	// Approach indicates which CLI implementation type was detected.
	Approach CLIApproach

	// Result contains the validation outcome.
	Result Result
}

// CLIValidator validates CLI directory structure.
type CLIValidator interface {
	// Validate checks the CLI structure and returns the result.
	Validate() CLIResult
}

// cliValidator is the default implementation of CLIValidator.
type cliValidator struct {
	scenarioDir  string
	scenarioName string
	logWriter    io.Writer
}

type serviceManifest struct {
	Service struct {
		Name string `json:"name"`
	} `json:"service"`
	CLI *serviceCLIConfig `json:"cli"`
}

type serviceCLIConfig struct {
	Enabled   bool                 `json:"enabled"`
	Command   string               `json:"command"`
	Adapter   serviceCLIAdapter    `json:"adapter"`
	Install   []serviceCLIInstall  `json:"install"`
	Invoke    serviceCLIInvoke     `json:"invoke"`
	Freshness *serviceCLIFreshness `json:"freshness,omitempty"`
}

type serviceCLIAdapter struct {
	Kind          string `json:"kind"`
	ModuleDir     string `json:"module_dir"`
	ScriptPath    string `json:"script_path"`
	InstallScript string `json:"install_script"`
}

type serviceCLIInstall struct {
	OS   []string `json:"os"`
	Kind string   `json:"kind"`
	Run  string   `json:"run"`
}

type serviceCLIInvoke struct {
	Kind    string `json:"kind"`
	Command string `json:"command"`
}

type serviceCLIFreshness struct {
	Inputs []string `json:"inputs"`
}

// NewCLIValidator creates a new CLI validator.
func NewCLIValidator(scenarioDir, scenarioName string, logWriter io.Writer) CLIValidator {
	return &cliValidator{
		scenarioDir:  scenarioDir,
		scenarioName: scenarioName,
		logWriter:    logWriter,
	}
}

// Validate implements CLIValidator.
func (v *cliValidator) Validate() CLIResult {
	return ValidateCLI(v.scenarioDir, v.scenarioName, v.logWriter)
}

// ValidateCLI validates the CLI directory structure based on the declared manifest adapter.
func ValidateCLI(scenarioDir, scenarioName string, logWriter io.Writer) CLIResult {
	manifest, err := LoadServiceManifest(scenarioDir)
	if err != nil {
		return CLIResult{
			Approach: CLIApproachUnknown,
			Result: FailMisconfiguration(
				err,
				"Add a valid .vrooli/service.json with a top-level cli contract.",
			),
		}
	}

	if strings.TrimSpace(manifest.Service.Name) == "" {
		manifest.Service.Name = scenarioName
	}

	approach := DetectCLIApproach(manifest)
	logStep(logWriter, "Detected CLI approach from manifest: %s", approach)

	switch approach {
	case CLIApproachGoModule:
		return validateGoModuleCLI(scenarioDir, manifest, logWriter)
	case CLIApproachShellScript:
		return validateShellScriptCLI(scenarioDir, manifest, logWriter)
	default:
		return validateUnknownCLI(manifest)
	}
}

// LoadServiceManifest reads the scenario service manifest for CLI inspection.
func LoadServiceManifest(scenarioDir string) (serviceManifest, error) {
	return loadServiceManifest(scenarioDir)
}

// DetectCLIApproach determines the CLI adapter from service.json.
func DetectCLIApproach(manifest serviceManifest) CLIApproach {
	if manifest.CLI == nil || !manifest.CLI.Enabled {
		return CLIApproachUnknown
	}
	switch strings.TrimSpace(manifest.CLI.Adapter.Kind) {
	case "go_module":
		return CLIApproachGoModule
	case "shell_script":
		return CLIApproachShellScript
	default:
		return CLIApproachUnknown
	}
}

func validateGoModuleCLI(scenarioDir string, manifest serviceManifest, logWriter io.Writer) CLIResult {
	if result := validateCommonCLIContract(manifest); !result.Success {
		return CLIResult{Approach: CLIApproachGoModule, Result: result}
	}

	moduleDir := filepath.Join(scenarioDir, filepath.FromSlash(manifest.CLI.Adapter.ModuleDir))
	if err := ensureDir(moduleDir); err != nil {
		return CLIResult{
			Approach: CLIApproachGoModule,
			Result: FailMisconfiguration(
				fmt.Errorf("missing CLI module directory %q", manifest.CLI.Adapter.ModuleDir),
				fmt.Sprintf("Create %s and keep it in sync with cli.adapter.module_dir.", manifest.CLI.Adapter.ModuleDir),
			),
		}
	}

	requiredFiles := []string{"go.mod"}
	for _, file := range requiredFiles {
		path := filepath.Join(moduleDir, file)
		if err := ensureFile(path); err != nil {
			return CLIResult{
				Approach: CLIApproachGoModule,
				Result: FailMisconfiguration(
					fmt.Errorf("missing %s/%s", manifest.CLI.Adapter.ModuleDir, file),
					fmt.Sprintf("Create %s/%s for the go_module CLI adapter.", manifest.CLI.Adapter.ModuleDir, file),
				),
			}
		}
		logStep(logWriter, "  ✓ %s/%s", manifest.CLI.Adapter.ModuleDir, file)
	}

	if !dirContainsGoSource(moduleDir) {
		return CLIResult{
			Approach: CLIApproachGoModule,
			Result: FailMisconfiguration(
				fmt.Errorf("no Go source files found in %s", manifest.CLI.Adapter.ModuleDir),
				fmt.Sprintf("Add Go source files under %s so the CLI installer has an entrypoint to build.", manifest.CLI.Adapter.ModuleDir),
			),
		}
	}

	observations := []Observation{
		NewSuccessObservation("CLI manifest contract valid"),
		NewSuccessObservation("Go module CLI structure valid"),
	}
	if !hasLocalExecutableCandidate(moduleDir, manifest.CLI.Command) {
		observations = append(observations, NewWarningObservation(
			fmt.Sprintf("Recommended local CLI artifact missing: %s/%s (installed_command validation works without it, but a scenario-local binary improves deterministic debugging)", manifest.CLI.Adapter.ModuleDir, manifest.CLI.Command),
		))
	}
	return CLIResult{
		Approach: CLIApproachGoModule,
		Result:   OK().WithObservations(observations...),
	}
}

func validateShellScriptCLI(scenarioDir string, manifest serviceManifest, logWriter io.Writer) CLIResult {
	if result := validateCommonCLIContract(manifest); !result.Success {
		return CLIResult{Approach: CLIApproachShellScript, Result: result}
	}

	requiredPaths := []string{
		manifest.CLI.Adapter.ScriptPath,
		manifest.CLI.Adapter.InstallScript,
	}
	for _, rel := range requiredPaths {
		path := filepath.Join(scenarioDir, filepath.FromSlash(rel))
		if err := ensureFile(path); err != nil {
			return CLIResult{
				Approach: CLIApproachShellScript,
				Result: FailMisconfiguration(
					fmt.Errorf("missing %s", rel),
					fmt.Sprintf("Create %s for the shell_script CLI adapter.", rel),
				),
			}
		}
		logStep(logWriter, "  ✓ %s", rel)
	}

	var observations []Observation
	scriptPath := filepath.Join(scenarioDir, filepath.FromSlash(manifest.CLI.Adapter.ScriptPath))
	info, err := os.Stat(scriptPath)
	if err == nil && info.Mode()&0o111 == 0 {
		observations = append(observations, NewWarningObservation(fmt.Sprintf("%s should be executable (chmod +x)", manifest.CLI.Adapter.ScriptPath)))
	}

	observations = append(observations,
		NewSuccessObservation("CLI manifest contract valid"),
		NewSuccessObservation("Shell script CLI structure valid"),
	)
	return CLIResult{
		Approach: CLIApproachShellScript,
		Result:   OK().WithObservations(observations...),
	}
}

func validateUnknownCLI(manifest serviceManifest) CLIResult {
	remediation := "Define service.json cli.enabled=true, set cli.command, declare a supported cli.adapter.kind, and configure cli.invoke."
	if manifest.CLI == nil {
		remediation = "Add a top-level cli block to .vrooli/service.json so the scenario declares how its CLI is installed and invoked."
	} else if !manifest.CLI.Enabled {
		remediation = "Set cli.enabled=true for scenarios that ship a CLI, or update structure expectations if this scenario intentionally has no CLI."
	} else if strings.TrimSpace(manifest.CLI.Adapter.Kind) == "" {
		remediation = "Set cli.adapter.kind to one of: go_module, shell_script."
	}

	return CLIResult{
		Approach: CLIApproachUnknown,
		Result: FailMisconfiguration(
			fmt.Errorf("CLI manifest contract incomplete or unsupported"),
			remediation,
		),
	}
}

func validateCommonCLIContract(manifest serviceManifest) Result {
	if manifest.CLI == nil {
		return FailMisconfiguration(
			fmt.Errorf("missing cli manifest"),
			"Add a top-level cli block to .vrooli/service.json.",
		)
	}
	if !manifest.CLI.Enabled {
		return FailMisconfiguration(
			fmt.Errorf("cli.enabled is false"),
			"Set cli.enabled=true for scenarios that declare a CLI.",
		)
	}
	if strings.TrimSpace(manifest.CLI.Command) == "" {
		return FailMisconfiguration(
			fmt.Errorf("cli.command is required"),
			"Set cli.command to the installed executable name.",
		)
	}
	if strings.TrimSpace(manifest.CLI.Invoke.Kind) != "installed_command" {
		return FailMisconfiguration(
			fmt.Errorf("cli.invoke.kind must be installed_command"),
			"Set cli.invoke.kind to 'installed_command' so runtime resolution matches the manifest.",
		)
	}
	if strings.TrimSpace(manifest.CLI.Invoke.Command) != strings.TrimSpace(manifest.CLI.Command) {
		return FailMisconfiguration(
			fmt.Errorf("cli.invoke.command must match cli.command"),
			"Set cli.invoke.command to the same value as cli.command.",
		)
	}
	if len(manifest.CLI.Install) == 0 {
		return FailMisconfiguration(
			fmt.Errorf("cli.install is required"),
			"Declare at least one cli.install command strategy in .vrooli/service.json.",
		)
	}
	for _, step := range manifest.CLI.Install {
		if strings.TrimSpace(step.Kind) != "command" || strings.TrimSpace(step.Run) == "" {
			return FailMisconfiguration(
				fmt.Errorf("cli.install entries must declare kind=command and a run string"),
				"Define each cli.install entry with kind 'command' and a concrete run command.",
			)
		}
	}
	return OK()
}

func loadServiceManifest(scenarioDir string) (serviceManifest, error) {
	servicePath := filepath.Join(scenarioDir, ".vrooli", "service.json")
	data, err := os.ReadFile(servicePath)
	if err != nil {
		return serviceManifest{}, fmt.Errorf("read %s: %w", servicePath, err)
	}
	var manifest serviceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return serviceManifest{}, fmt.Errorf("parse %s: %w", servicePath, err)
	}
	return manifest, nil
}

func dirContainsGoSource(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(d.Name(), ".go") {
			found = true
			return io.EOF
		}
		return nil
	})
	return found
}

func hasLocalExecutableCandidate(moduleDir, command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	path := filepath.Join(moduleDir, command)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
