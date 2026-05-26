package scenariocli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	repocontract "github.com/vrooli/repo-contract-go"
)

type serviceManifest struct {
	CLI *cliConfig `json:"cli,omitempty"`
}

type cliConfig struct {
	Enabled   bool               `json:"enabled"`
	Command   string             `json:"command,omitempty"`
	Adapter   cliAdapterConfig   `json:"adapter,omitempty"`
	Install   []cliInstallStep   `json:"install,omitempty"`
	Freshness *cliFreshnessCheck `json:"freshness,omitempty"`
}

type cliAdapterConfig struct {
	Kind          string `json:"kind,omitempty"`
	ModuleDir     string `json:"module_dir,omitempty"`
	ScriptPath    string `json:"script_path,omitempty"`
	InstallScript string `json:"install_script,omitempty"`
}

type cliInstallStep struct {
	Kind string `json:"kind,omitempty"`
	Run  string `json:"run,omitempty"`
}

type cliFreshnessCheck struct {
	Inputs []string `json:"inputs,omitempty"`
}

type installMetadata struct {
	BinaryName  string `json:"binary_name"`
	Fingerprint string `json:"fingerprint"`
}

type scenarioCLI struct {
	name         string
	binaryName   string
	scenarioPath string
	servicePath  string
	modulePath   string
	config       *cliConfig
}

var userHomeDir = os.UserHomeDir

func ResolveExecutable(root, home, name string) (string, error) {
	return ResolveExecutableContext(context.Background(), root, home, name)
}

func ResolveExecutableContext(ctx context.Context, root, home, name string) (string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	name = strings.TrimSpace(name)
	if root == "" {
		return "", errors.New("repo root is required")
	}
	if name == "" {
		return "", errors.New("scenario name is required")
	}

	item, err := discoverScenarioCLI(root, name)
	if err != nil {
		return "", err
	}
	if err := ensureScenarioCLI(ctx, root, strings.TrimSpace(home), item); err != nil {
		return "", err
	}
	return installedBinaryPath(home, item.binaryName)
}

func ResolveExecutableFromRepoRoot(name string) (string, error) {
	return ResolveExecutableFromRepoRootContext(context.Background(), name)
}

func ResolveExecutableFromRepoRootContext(ctx context.Context, name string) (string, error) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", err
	}
	home, _ := userHomeDir()
	return ResolveExecutableContext(ctx, root, home, name)
}

func discoverScenarioCLI(root, name string) (scenarioCLI, error) {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return scenarioCLI{}, err
	}
	scenarioRoot, err := contract.ScenarioRoot(root, name)
	if err != nil {
		return scenarioCLI{}, err
	}
	servicePath, err := contract.ScenarioFile(root, name, "service")
	if err != nil {
		return scenarioCLI{}, err
	}
	if err := requireFile(servicePath); err != nil {
		return scenarioCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
	}
	manifest, err := readServiceManifest(servicePath)
	if err != nil {
		return scenarioCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
	}
	if manifest.CLI == nil || !manifest.CLI.Enabled {
		return scenarioCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, fs.ErrNotExist)
	}

	item := scenarioCLI{
		name:         name,
		binaryName:   strings.TrimSpace(manifest.CLI.Command),
		scenarioPath: scenarioRoot,
		servicePath:  servicePath,
		config:       manifest.CLI,
	}
	switch item.config.Adapter.Kind {
	case "go_module":
		item.modulePath = filepath.Join(scenarioRoot, filepath.FromSlash(item.config.Adapter.ModuleDir))
		if err := requireFile(filepath.Join(item.modulePath, "go.mod")); err != nil {
			return scenarioCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
		}
	case "shell_script":
		if err := requireFile(filepath.Join(scenarioRoot, filepath.FromSlash(item.config.Adapter.ScriptPath))); err != nil {
			return scenarioCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
		}
		if err := requireFile(filepath.Join(scenarioRoot, filepath.FromSlash(item.config.Adapter.InstallScript))); err != nil {
			return scenarioCLI{}, fmt.Errorf("discover scenario CLI %q: %w", name, err)
		}
	default:
		return scenarioCLI{}, fmt.Errorf("discover scenario CLI %q: unsupported adapter kind %q", name, item.config.Adapter.Kind)
	}
	return item, nil
}

func readServiceManifest(path string) (serviceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return serviceManifest{}, err
	}
	var manifest serviceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return serviceManifest{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if manifest.CLI != nil {
		if err := manifest.CLI.validate(); err != nil {
			return serviceManifest{}, fmt.Errorf("validate cli in %s: %w", path, err)
		}
	}
	return manifest, nil
}

func (cfg *cliConfig) validate() error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	cfg.Command = strings.TrimSpace(cfg.Command)
	cfg.Adapter.Kind = strings.TrimSpace(cfg.Adapter.Kind)
	cfg.Adapter.ModuleDir = strings.TrimSpace(cfg.Adapter.ModuleDir)
	cfg.Adapter.ScriptPath = strings.TrimSpace(cfg.Adapter.ScriptPath)
	cfg.Adapter.InstallScript = strings.TrimSpace(cfg.Adapter.InstallScript)
	if cfg.Command == "" {
		return errors.New("command is required when cli.enabled=true")
	}
	switch cfg.Adapter.Kind {
	case "go_module":
		if cfg.Adapter.ModuleDir == "" {
			return errors.New("adapter.module_dir is required for cli.adapter.kind=go_module")
		}
	case "shell_script":
		if cfg.Adapter.ScriptPath == "" {
			return errors.New("adapter.script_path is required for cli.adapter.kind=shell_script")
		}
		if cfg.Adapter.InstallScript == "" {
			return errors.New("adapter.install_script is required for cli.adapter.kind=shell_script")
		}
	default:
		return fmt.Errorf("unsupported cli.adapter.kind %q", cfg.Adapter.Kind)
	}
	for i := range cfg.Install {
		cfg.Install[i].Kind = strings.TrimSpace(cfg.Install[i].Kind)
		cfg.Install[i].Run = strings.TrimSpace(cfg.Install[i].Run)
		if cfg.Install[i].Kind == "" {
			return fmt.Errorf("install[%d].kind is required", i)
		}
		if cfg.Install[i].Kind != "command" {
			return fmt.Errorf("unsupported cli.install[%d].kind %q", i, cfg.Install[i].Kind)
		}
		if cfg.Install[i].Run == "" {
			return fmt.Errorf("install[%d].run is required", i)
		}
	}
	return nil
}

func ensureScenarioCLI(ctx context.Context, root, home string, item scenarioCLI) error {
	binaryPath, err := installedBinaryPath(home, item.binaryName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(binaryPath); err == nil {
		current, currentErr := installedBinaryCurrent(home, item)
		if currentErr != nil {
			return currentErr
		}
		if current {
			return nil
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return installScenarioCLI(ctx, root, home, item)
}

func installedBinaryCurrent(home string, item scenarioCLI) (bool, error) {
	meta, ok, err := readInstallMetadata(home, item)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	fingerprint, err := computeScenarioCLIFingerprint(item)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(meta.Fingerprint) == fingerprint, nil
}

func installScenarioCLI(ctx context.Context, root, home string, item scenarioCLI) error {
	installDir, err := installDir(home)
	if err != nil {
		return err
	}
	switch item.config.Adapter.Kind {
	case "go_module":
		installerDir, err := cliInstallerDir(root, item.modulePath)
		if err != nil {
			return err
		}
		spec, err := scenarioFreshnessSpec(item)
		if err != nil {
			return err
		}
		args := []string{
			"run", "./cmd/cli-installer",
			"--module", item.modulePath,
			"--name", item.binaryName,
			"--install-dir", installDir,
		}
		if strings.TrimSpace(spec.ContextRoot) != "" && filepath.Clean(spec.ContextRoot) != filepath.Clean(item.modulePath) {
			args = append(args, "--context-root", spec.ContextRoot)
		}
		for _, input := range spec.Inputs {
			args = append(args, "--freshness-input", input)
		}
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir = installerDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return err
		}
	case "shell_script":
		cmd := exec.CommandContext(ctx, "bash", filepath.Join(item.scenarioPath, filepath.FromSlash(item.config.Adapter.InstallScript)))
		cmd.Dir = item.scenarioPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported scenario CLI adapter kind %q", item.config.Adapter.Kind)
	}

	fingerprint, err := computeScenarioCLIFingerprint(item)
	if err != nil {
		return err
	}
	return writeInstallMetadata(home, item, installMetadata{
		BinaryName:  item.binaryName,
		Fingerprint: fingerprint,
	})
}

// scenarioFreshnessSpec returns the canonical freshness contract used by both
// cli-installer (at install time) and cli-core's runtime StaleChecker. Both
// must evaluate the same spec to avoid perpetual reinstall loops.
func scenarioFreshnessSpec(item scenarioCLI) (cliutil.FreshnessSpec, error) {
	var customInputs []string
	if item.config != nil && item.config.Freshness != nil {
		customInputs = item.config.Freshness.Inputs
	}
	switch item.config.Adapter.Kind {
	case "go_module":
		return cliutil.CanonicalScenarioGoModuleFreshnessSpec(item.scenarioPath, item.modulePath, item.binaryName, customInputs), nil
	case "shell_script":
		manifestRel, err := filepath.Rel(item.scenarioPath, item.servicePath)
		if err != nil {
			return cliutil.FreshnessSpec{}, err
		}
		return cliutil.CanonicalShellScriptFreshnessSpec(item.scenarioPath, item.config.Adapter.ScriptPath, item.config.Adapter.InstallScript, manifestRel, item.binaryName, customInputs), nil
	default:
		return cliutil.FreshnessSpec{}, fmt.Errorf("unsupported scenario CLI adapter kind %q", item.config.Adapter.Kind)
	}
}

func cliInstallerDir(root, modulePath string) (string, error) {
	for _, base := range []string{root, filepath.Clean(modulePath)} {
		dir := filepath.Clean(base)
		for {
			candidate := filepath.Join(dir, "packages", "cli-core")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "", fmt.Errorf("locate cli installer for module %s", modulePath)
}

func installDir(home string) (string, error) {
	home = strings.TrimSpace(home)
	if home == "" {
		resolved, err := userHomeDir()
		if err != nil {
			return "", err
		}
		home = resolved
	}
	return repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
}

func installedBinaryPath(home, binaryName string) (string, error) {
	dir, err := installDir(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, strings.TrimSpace(binaryName)), nil
}

func installMetadataPath(home string, item scenarioCLI) (string, error) {
	binPath, err := installedBinaryPath(home, item.binaryName)
	if err != nil {
		return "", err
	}
	return binPath + ".build.meta", nil
}

func readInstallMetadata(home string, item scenarioCLI) (installMetadata, bool, error) {
	metaPath, err := installMetadataPath(home, item)
	if err != nil {
		return installMetadata{}, false, err
	}
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return installMetadata{}, false, nil
		}
		return installMetadata{}, false, err
	}
	var meta installMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return installMetadata{}, false, fmt.Errorf("parse install metadata %s: %w", metaPath, err)
	}
	return meta, true, nil
}

func writeInstallMetadata(home string, item scenarioCLI, meta installMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path, err := installMetadataPath(home, item)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func computeScenarioCLIFingerprint(item scenarioCLI) (string, error) {
	spec, err := scenarioFreshnessSpec(item)
	if err != nil {
		return "", err
	}
	return cliutil.ComputeFreshnessFingerprint(spec)
}

func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}
