package scenariocli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
	return installedBinaryPath(home, item.binaryName), nil
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
	servicePath := filepath.Join(scenarioRoot, ".vrooli", "service.json")
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
	binaryPath := installedBinaryPath(home, item.binaryName)
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
	installDir := installDir(home)
	switch item.config.Adapter.Kind {
	case "go_module":
		installerDir, err := cliInstallerDir(root, item.modulePath)
		if err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, "go", "run", "./cmd/cli-installer",
			"--module", item.modulePath,
			"--name", item.binaryName,
			"--install-dir", installDir,
		)
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

func installDir(home string) string {
	home = strings.TrimSpace(home)
	if home == "" {
		var err error
		home, err = userHomeDir()
		if err != nil {
			return filepath.Join(".", ".vrooli", "bin")
		}
	}
	return filepath.Join(home, ".vrooli", "bin")
}

func installedBinaryPath(home, binaryName string) string {
	return filepath.Join(installDir(home), strings.TrimSpace(binaryName))
}

func installMetadataPath(home string, item scenarioCLI) string {
	return installedBinaryPath(home, item.binaryName) + ".build.meta"
}

func readInstallMetadata(home string, item scenarioCLI) (installMetadata, bool, error) {
	data, err := os.ReadFile(installMetadataPath(home, item))
	if err != nil {
		if os.IsNotExist(err) {
			return installMetadata{}, false, nil
		}
		return installMetadata{}, false, err
	}
	var meta installMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return installMetadata{}, false, fmt.Errorf("parse install metadata %s: %w", installMetadataPath(home, item), err)
	}
	return meta, true, nil
}

func writeInstallMetadata(home string, item scenarioCLI, meta installMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := installMetadataPath(home, item)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func computeScenarioCLIFingerprint(item scenarioCLI) (string, error) {
	switch item.config.Adapter.Kind {
	case "go_module":
		return computeFingerprint(item.modulePath, item.binaryName)
	case "shell_script":
		if item.config.Freshness != nil && len(item.config.Freshness.Inputs) > 0 {
			return computeFingerprintFromScenarioInputs(item.scenarioPath, item.config.Freshness.Inputs, item.binaryName)
		}
		serviceRel, err := filepath.Rel(item.scenarioPath, item.servicePath)
		if err != nil {
			return "", err
		}
		return computeFingerprintFromScenarioInputs(item.scenarioPath, []string{
			item.config.Adapter.ScriptPath,
			item.config.Adapter.InstallScript,
			filepath.ToSlash(serviceRel),
		}, item.binaryName)
	default:
		return "", fmt.Errorf("unsupported scenario CLI adapter kind %q", item.config.Adapter.Kind)
	}
}

type fileEntry struct {
	rel  string
	size int64
	hash [32]byte
}

var skipDirs = []string{
	".git",
	".vscode",
	".idea",
	"coverage",
	"dist",
	"build",
	"tmp",
	"data",
	"node_modules",
}

var skipFiles = []string{
	"build.meta",
}

func computeFingerprint(root string, extraSkipFiles ...string) (string, error) {
	var entries []fileEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if shouldSkipDir(rel) && d.IsDir() {
			return filepath.SkipDir
		}
		if d.IsDir() || shouldSkipFile(rel, extraSkipFiles) {
			return nil
		}
		content, readErr := fs.ReadFile(os.DirFS(root), rel)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", rel, readErr)
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}
		entries = append(entries, fileEntry{rel: rel, size: info.Size(), hash: sha256.Sum256(content)})
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].rel < entries[j].rel })
	hasher := sha256.New()
	for _, entry := range entries {
		fmt.Fprintf(hasher, "%s|%d|%x\n", entry.rel, entry.size, entry.hash)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func computeFingerprintFromScenarioInputs(scenarioRoot string, inputs []string, extraSkipFiles ...string) (string, error) {
	entries := make(map[string]fileEntry)
	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		matches, err := expandScenarioInputPaths(scenarioRoot, input)
		if err != nil {
			return "", err
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return "", err
			}
			if info.IsDir() {
				if err := filepath.WalkDir(match, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					rel, relErr := filepath.Rel(scenarioRoot, path)
					if relErr != nil {
						return relErr
					}
					rel = filepath.ToSlash(rel)
					if rel == "." {
						return nil
					}
					if shouldSkipDir(rel) && d.IsDir() {
						return filepath.SkipDir
					}
					if d.IsDir() || shouldSkipFile(rel, extraSkipFiles) {
						return nil
					}
					content, readErr := os.ReadFile(path)
					if readErr != nil {
						return readErr
					}
					fileInfo, infoErr := d.Info()
					if infoErr != nil {
						return infoErr
					}
					entries[rel] = fileEntry{rel: rel, size: fileInfo.Size(), hash: sha256.Sum256(content)}
					return nil
				}); err != nil {
					return "", err
				}
				continue
			}
			rel, err := filepath.Rel(scenarioRoot, match)
			if err != nil {
				return "", err
			}
			rel = filepath.ToSlash(rel)
			if shouldSkipFile(rel, extraSkipFiles) {
				continue
			}
			content, err := os.ReadFile(match)
			if err != nil {
				return "", err
			}
			entries[rel] = fileEntry{rel: rel, size: info.Size(), hash: sha256.Sum256(content)}
		}
	}
	list := make([]fileEntry, 0, len(entries))
	for _, entry := range entries {
		list = append(list, entry)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].rel < list[j].rel })
	hasher := sha256.New()
	for _, entry := range list {
		fmt.Fprintf(hasher, "%s|%d|%x\n", entry.rel, entry.size, entry.hash)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func expandScenarioInputPaths(scenarioRoot, input string) ([]string, error) {
	if strings.ContainsAny(input, "*?[") {
		return filepath.Glob(filepath.Join(scenarioRoot, filepath.FromSlash(input)))
	}
	return []string{filepath.Join(scenarioRoot, filepath.FromSlash(input))}, nil
}

func shouldSkipDir(path string) bool {
	path = filepath.ToSlash(path)
	for _, skip := range skipDirs {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
}

func shouldSkipFile(path string, extra []string) bool {
	path = filepath.ToSlash(path)
	for _, skip := range skipFiles {
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	for _, skip := range extra {
		skip = filepath.ToSlash(skip)
		if path == skip || strings.HasPrefix(path, skip+"/") {
			return true
		}
	}
	return false
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
