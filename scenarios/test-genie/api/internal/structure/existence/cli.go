package existence

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CLIApproach represents the type of CLI implementation used by a scenario.
type CLIApproach int

const (
	// CLIApproachUnknown indicates the CLI approach could not be determined.
	CLIApproachUnknown CLIApproach = iota
	// CLIApproachLegacy indicates a bash script CLI (cli/<name> is a shell script).
	CLIApproachLegacy
	// CLIApproachCrossPlatform indicates a Go-based cross-platform CLI.
	CLIApproachCrossPlatform
)

// String returns a human-readable name for the CLI approach.
func (a CLIApproach) String() string {
	switch a {
	case CLIApproachLegacy:
		return "legacy"
	case CLIApproachCrossPlatform:
		return "cross-platform"
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

// ValidateCLI validates the CLI directory structure based on the detected approach.
func ValidateCLI(scenarioDir, scenarioName string, logWriter io.Writer) CLIResult {
	cliDir := filepath.Join(scenarioDir, "cli")

	// First verify cli/ directory exists
	if err := ensureDir(cliDir); err != nil {
		return CLIResult{
			Approach: CLIApproachUnknown,
			Result: FailMisconfiguration(
				fmt.Errorf("cli directory missing"),
				"Create the 'cli' directory to match the scenario template.",
			),
		}
	}

	approach := DetectCLIApproach(scenarioDir, scenarioName)
	logStep(logWriter, "Detected CLI approach: %s", approach)

	switch approach {
	case CLIApproachCrossPlatform:
		return validateCrossPlatformCLI(scenarioDir, scenarioName, logWriter)
	case CLIApproachLegacy:
		return validateLegacyCLI(scenarioDir, scenarioName, logWriter)
	default:
		return validateUnknownCLI(scenarioDir, scenarioName, logWriter)
	}
}

// DetectCLIApproach determines whether a scenario uses the legacy bash CLI
// or the newer cross-platform Go CLI based on file presence.
//
// Cross-platform detection: cli/main.go + cli/go.mod exist
// Legacy detection: cli/<scenario-name> exists and is not a compiled binary
func DetectCLIApproach(scenarioDir, scenarioName string) CLIApproach {
	cliDir := filepath.Join(scenarioDir, "cli")

	// Check for cross-platform indicators: Go module with main.go
	mainGo := filepath.Join(cliDir, "main.go")
	goMod := filepath.Join(cliDir, "go.mod")

	if FileExists(mainGo) && FileExists(goMod) {
		return CLIApproachCrossPlatform
	}

	// Check for legacy indicator: cli/<scenario-name> bash script
	cliScript := filepath.Join(cliDir, scenarioName)
	if FileExists(cliScript) {
		// Verify it's a text file (bash script), not a compiled binary
		if isTextFile(cliScript) {
			return CLIApproachLegacy
		}
		// If it's a binary, this is likely a cross-platform CLI that was built locally
		// Check if Go sources also exist
		if FileExists(mainGo) {
			return CLIApproachCrossPlatform
		}
	}

	return CLIApproachUnknown
}

// validateCrossPlatformCLI validates a Go-based cross-platform CLI structure.
func validateCrossPlatformCLI(scenarioDir, scenarioName string, logWriter io.Writer) CLIResult {
	cliDir := filepath.Join(scenarioDir, "cli")
	var observations []Observation

	requiredFiles := []string{
		"main.go",
		"go.mod",
		"install.sh",
	}

	for _, file := range requiredFiles {
		path := filepath.Join(cliDir, file)
		if err := ensureFile(path); err != nil {
			logError(logWriter, "Missing cross-platform CLI file: cli/%s", file)
			return CLIResult{
				Approach: CLIApproachCrossPlatform,
				Result: FailMisconfiguration(
					fmt.Errorf("missing cli/%s", file),
					fmt.Sprintf("Create cli/%s for the cross-platform Go CLI. See packages/cli-core/README.md for guidance.", file),
				),
			}
		}
		logStep(logWriter, "  ✓ cli/%s", file)
	}

	// Optional but recommended: install.ps1 for Windows support
	installPs1 := filepath.Join(cliDir, "install.ps1")
	if FileExists(installPs1) {
		logStep(logWriter, "  ✓ cli/install.ps1 (Windows support)")
		observations = append(observations, NewSuccessObservation("Cross-platform CLI with Windows support"))
	} else {
		logStep(logWriter, "  ⚠ cli/install.ps1 missing (no Windows support)")
		observations = append(observations, NewInfoObservation("Cross-platform CLI detected (add install.ps1 for Windows support)"))
	}

	observations = append(observations, NewSuccessObservation("Go cross-platform CLI structure valid"))

	return CLIResult{
		Approach: CLIApproachCrossPlatform,
		Result:   OK().WithObservations(observations...),
	}
}

// validateLegacyCLI validates a legacy bash script CLI structure.
func validateLegacyCLI(scenarioDir, scenarioName string, logWriter io.Writer) CLIResult {
	cliDir := filepath.Join(scenarioDir, "cli")
	var observations []Observation

	// Legacy CLI requires the bash script and install.sh
	requiredFiles := []string{
		scenarioName,
		"install.sh",
	}

	for _, file := range requiredFiles {
		path := filepath.Join(cliDir, file)
		if err := ensureFile(path); err != nil {
			logError(logWriter, "Missing legacy CLI file: cli/%s", file)
			return CLIResult{
				Approach: CLIApproachLegacy,
				Result: FailMisconfiguration(
					fmt.Errorf("missing cli/%s", file),
					fmt.Sprintf("Create cli/%s for the legacy bash CLI.", file),
				),
			}
		}
		logStep(logWriter, "  ✓ cli/%s", file)
	}

	// Verify the CLI script is executable
	cliScript := filepath.Join(cliDir, scenarioName)
	info, err := os.Stat(cliScript)
	if err == nil && info.Mode()&0111 == 0 {
		logWarn(logWriter, "cli/%s is not executable", scenarioName)
		observations = append(observations, NewWarningObservation(fmt.Sprintf("cli/%s should be executable (chmod +x)", scenarioName)))
	}

	observations = append(observations, NewSuccessObservation("Legacy bash CLI structure valid"))
	observations = append(observations, NewInfoObservation("Legacy bash CLI detected. Cross-platform Go CLI available - see docs/phases/structure/cli-approaches.md"))

	return CLIResult{
		Approach: CLIApproachLegacy,
		Result:   OK().WithObservations(observations...),
	}
}

// validateUnknownCLI handles cases where the CLI approach cannot be determined.
func validateUnknownCLI(scenarioDir, scenarioName string, logWriter io.Writer) CLIResult {
	cliDir := filepath.Join(scenarioDir, "cli")

	// Check what files exist to provide helpful guidance
	hasMainGo := FileExists(filepath.Join(cliDir, "main.go"))
	hasGoMod := FileExists(filepath.Join(cliDir, "go.mod"))
	hasScript := FileExists(filepath.Join(cliDir, scenarioName))
	hasInstallSh := FileExists(filepath.Join(cliDir, "install.sh"))

	var remediation string
	switch {
	case hasMainGo && !hasGoMod:
		remediation = "cli/main.go exists but cli/go.mod is missing. Run 'go mod init' in cli/ or see packages/cli-core/README.md."
	case hasGoMod && !hasMainGo:
		remediation = "cli/go.mod exists but cli/main.go is missing. Create main.go or see packages/cli-core/README.md."
	case !hasInstallSh:
		remediation = "cli/install.sh is required. Create it to enable CLI installation."
	case !hasScript && !hasMainGo:
		remediation = fmt.Sprintf("No CLI implementation found. Create either cli/%s (bash script) or cli/main.go + cli/go.mod (Go CLI).", scenarioName)
	default:
		remediation = "CLI structure is incomplete. See docs/phases/structure/cli-approaches.md for valid patterns."
	}

	logError(logWriter, "Could not determine CLI approach")
	return CLIResult{
		Approach: CLIApproachUnknown,
		Result: FailMisconfiguration(
			fmt.Errorf("CLI structure incomplete or unrecognized"),
			remediation,
		),
	}
}

// isTextFile performs a simple heuristic check to determine if a file is text (not binary).
// It reads the first 512 bytes and checks for null bytes.
func isTextFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return false
	}

	// Check for null bytes which indicate binary content
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return false
		}
	}
	return true
}
