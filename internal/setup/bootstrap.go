package setup

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	vrooliruntime "github.com/vrooli/vrooli/internal/runtime"
	onboardingapplyprivileges "github.com/vrooli/vrooli/internal/safeguards/onboarding-apply-privileges"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	bootstrapParameterA            = 2
	setupDockerTool                = "docker"
	setupCredentialStorePassphrase = "credential-store-passphrase"
	setupOwnerVrooli               = "vrooli"
	setupStageComplete             = "complete"
)

func bootstrapAwareRequirements(resolution hostreq.Resolution) hostreq.Resolution {
	byName := make(map[string]hostreq.ResolvedRequirement, len(resolution.Tools)+bootstrapParameterA)
	for _, requirement := range resolution.Tools {
		name := strings.ToLower(strings.TrimSpace(requirement.Name))
		if name == "" || name == setupDockerTool {
			continue
		}
		byName[name] = requirement
	}
	for _, name := range []string{"git", "go"} {
		if _, ok := byName[name]; ok {
			requirement := byName[name]
			requirement.Required = true
			requirement.Environments = []string{"development", "production", "minimal"}
			byName[name] = requirement
			continue
		}
		byName[name] = hostreq.ResolvedRequirement{
			Name:         name,
			Kind:         hostreq.KindTool,
			Required:     true,
			Reasons:      []string{"Bootstrap source operations and subsequent source rebuilds"},
			When:         []string{"setup"},
			Environments: []string{"development", "production", "minimal"},
			Provenance: []hostreq.Provenance{{
				Kind: "root", Name: "vrooli-bootstrap", Path: "internal/setup/setup.go", Source: "internal/setup/setup.go",
			}},
		}
	}
	ordered := make([]hostreq.ResolvedRequirement, 0, len(byName))
	for _, name := range []string{"git", "go"} {
		ordered = append(ordered, byName[name])
		delete(byName, name)
	}
	if requirement, ok := byName["rasdaemon"]; ok {
		ordered = append(ordered, requirement)
		delete(byName, "rasdaemon")
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		ordered = append(ordered, byName[name])
	}
	resolution.Tools = ordered
	return resolution
}

func addOnboardingApplyPrivilegeRequirement(resolution hostreq.Resolution, executable string) hostreq.Resolution {
	tools := make([]string, 0)
	for _, requirement := range resolution.Tools {
		if requirement.Privilege == hostreqspec.PrivilegeElevated {
			tools = append(tools, requirement.Name)
		}
	}
	safeguards := make([]string, 0)
	for _, requirement := range resolution.Safeguards {
		if requirement.Privilege == hostreqspec.PrivilegeElevated {
			safeguards = append(safeguards, requirement.Name)
		}
	}
	if len(tools) == 0 && len(safeguards) == 0 {
		return resolution
	}
	grant := hostreq.ResolvedRequirement{
		Name:       "onboarding_apply_privileges",
		Kind:       hostreq.KindSafeguard,
		Required:   true,
		Privilege:  hostreqspec.PrivilegeElevated,
		Platforms:  []string{"linux", "macos"},
		Config:     onboardingapplyprivileges.ConfigForRequirements(executable, tools, safeguards),
		Reasons:    []string{"Allow onboarding apply to execute selected elevated host requirements without a second prompt"},
		Provenance: []hostreq.Provenance{{Kind: "root", Name: "vrooli-setup", Path: "internal/setup/setup.go", Source: "internal/setup/setup.go"}},
	}
	resolution.Safeguards = append([]hostreq.ResolvedRequirement{grant}, resolution.Safeguards...)
	return resolution
}

func ensureBootstrapHostTools(home string, opts vrooliruntime.EnsureOptions) error {
	host := vrooliruntime.Current()
	if err := ensureBootstrapPackageManager(host, home, opts, exec.LookPath, shell.Run); err != nil {
		return err
	}
	for _, name := range []string{"git", "go"} {
		if opts.OnOperation != nil {
			opts.OnOperation("Checking bootstrap tool " + name)
		}
		status, err := vrooliruntime.EnsureTool(name, opts)
		if err != nil {
			return fmt.Errorf("install bootstrap host tool %s: %w", name, err)
		}
		if !bootstrapToolSatisfied(status) {
			detail := strings.TrimSpace(strings.Join(status.Notes, "; "))
			if detail == "" {
				detail = string(status.ExecutionState)
			}
			return fmt.Errorf("bootstrap host tool %s is unavailable: %s", name, detail)
		}
		recoverHostToolPATH(home)
	}
	return nil
}

const homebrewInstallURL = "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh"

// ensureBootstrapPackageManager is the Homebrew-specific bootstrap for macOS's
// only remaining chicken-and-egg:
// a fresh host has curl but no package manager capable of installing Go. Linux
// distributions already ship their package manager, and Windows needs no
// bootstrap because winget ships with supported Windows installations.
func ensureBootstrapPackageManager(
	host vrooliruntime.Host,
	home string,
	opts vrooliruntime.EnsureOptions,
	lookPath func(string) (string, error),
	run func(shell.Spec) error,
) error {
	if host.OS != "darwin" || strings.TrimSpace(host.PackageManager) != "" {
		return nil
	}
	if _, err := lookPath("curl"); err != nil {
		return fmt.Errorf("bootstrap Homebrew: curl is required")
	}
	script, err := os.CreateTemp("", "vrooli-homebrew-install-*.sh")
	if err != nil {
		return fmt.Errorf("bootstrap Homebrew: create installer staging file: %w", err)
	}
	scriptPath := script.Name()
	if err := script.Close(); err != nil {
		_ = os.Remove(scriptPath)
		return fmt.Errorf("bootstrap Homebrew: close installer staging file: %w", err)
	}
	defer os.Remove(scriptPath)
	if err := run(shell.Spec{
		Name: "curl", Args: []string{"-fsSL", "--proto", "=https", "--tlsv1.2", "-o", scriptPath, homebrewInstallURL},
		Stdout: opts.Stdout, Stderr: opts.Stderr,
	}); err != nil {
		return fmt.Errorf("bootstrap Homebrew: download official installer: %w", err)
	}
	env := envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"NONINTERACTIVE=1", "HOME=" + home})
	if err := run(shell.Spec{
		Name: "/bin/bash", Args: []string{scriptPath}, Env: env,
		Stdout: opts.Stdout, Stderr: opts.Stderr, Stdin: os.Stdin,
	}); err != nil {
		return fmt.Errorf("bootstrap Homebrew: run official installer: %w", err)
	}
	recoverHostToolPATH(home)
	if _, err := lookPath("brew"); err != nil {
		return fmt.Errorf("bootstrap Homebrew: installer completed but brew is not available on PATH")
	}
	return nil
}

func bootstrapToolSatisfied(status vrooliruntime.ItemStatus) bool {
	switch status.ExecutionState {
	case vrooliruntime.ExecutionAlreadyPresent, vrooliruntime.ExecutionInstalled:
		return true
	default:
		return false
	}
}

func recoverHostToolPATH(home string) {
	_ = os.Setenv("PATH", hostreqkit.AugmentUserToolPath(home, os.Getenv("PATH"), os.Getenv("LOCALAPPDATA")))
}
