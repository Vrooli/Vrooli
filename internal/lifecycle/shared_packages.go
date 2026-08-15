package lifecycle

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/packagegov"
	"github.com/vrooli/vrooli/internal/scenario"
)

const sharedPackageProvisioningDisabledEnv = "VROOLI_DISABLE_SHARED_PACKAGE_PROVISIONING"

// SharedPackageProvisioningError identifies the package and declared command
// responsible for a provisioning failure. Keeping this error typed lets the
// lifecycle report the actual missing build contract instead of leaking a
// later, misleading TypeScript module-resolution error from pnpm or tsc.
type SharedPackageProvisioningError struct {
	PackageName string
	Command     string
	Reason      string
	Err         error
}

func (e *SharedPackageProvisioningError) Error() string {
	if e == nil {
		return ""
	}
	message := fmt.Sprintf("shared package %q provisioning command %q %s", e.PackageName, e.Command, e.Reason)
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	return message
}

func (e *SharedPackageProvisioningError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type sharedPackageDependency struct {
	Name       string
	Root       string
	Package    packagegov.Package
	Generation []packagegov.CommandSpec
	Build      []packagegov.CommandSpec
}

type sharedPackageProvisionOptions struct {
	Home  string
	Env   []string
	Stdin io.Reader
}

// ProvisionGeneratedPackages ensures repository-level generated packages are
// materialized before a project build imports their outputs. Unlike scenario
// setup, this path is driven by package manifests rather than a consumer UI
// package.json because the control plane itself imports packages/proto/gen.
func ProvisionGeneratedPackages(repoRoot, home string, stdout, logWriter io.Writer) error {
	packagesRoot := filepath.Join(repoRoot, "packages")
	entries, err := os.ReadDir(packagesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		packageRoot := filepath.Join(packagesRoot, entry.Name())
		if _, err := os.Stat(filepath.Join(packageRoot, ".vrooli", "package.json")); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		pkg, _, err := packagegov.LoadPackage(packageRoot)
		if err != nil {
			return err
		}
		if len(pkg.Manifest.Package.Lifecycle.Generate) == 0 {
			continue
		}
		dependency := sharedPackageDependency{
			Name:       pkg.Manifest.Package.Name,
			Root:       packageRoot,
			Package:    pkg,
			Generation: pkg.Manifest.Package.Lifecycle.Generate,
			Build:      nil,
		}
		if err := provisionSharedPackageWithOptions(dependency, stdout, logWriter, sharedPackageProvisionOptions{Home: home}); err != nil {
			return err
		}
	}
	return nil
}

// provisionSharedPackages derives shared-package work from the scenario UI's
// file dependencies. It runs before any setup step, which is required because
// pnpm copies file: dependencies during install-ui-deps.
func (r *Runner) provisionSharedPackages(item scenario.Scenario, env map[string]string, logWriter, childWriter io.Writer) error {
	uiPackageJSON := filepath.Join(item.Path, "ui", "package.json")
	dependencies, err := sharedPackageDependencies(r.Root, uiPackageJSON)
	if err != nil {
		return fmt.Errorf("resolve shared packages for scenario %q: %w", item.Slug, err)
	}
	if len(dependencies) == 0 {
		return nil
	}

	if provisioningDisabled(env) {
		dep := dependencies[0]
		for _, candidate := range dependencies {
			if candidate.Name == "@vrooli/iframe-bridge" {
				dep = candidate
				break
			}
		}
		command := firstProvisioningCommand(dep)
		return &SharedPackageProvisioningError{
			PackageName: dep.Name,
			Command:     command,
			Reason:      "provisioning is disabled",
		}
	}

	_, _ = fmt.Fprintf(logWriter, "provision-shared-packages: %d package(s) before install-ui-deps\n", len(dependencies))
	commandOptions := sharedPackageProvisionOptions{
		Home:  r.Home,
		Env:   lifecycleStepEnv("setup", env),
		Stdin: strings.NewReader(""),
	}
	for _, dependency := range dependencies {
		if err := provisionSharedPackageWithOptions(dependency, childWriter, logWriter, commandOptions); err != nil {
			return err
		}
	}
	return nil
}

func provisioningDisabled(env map[string]string) bool {
	value := strings.TrimSpace(env[sharedPackageProvisioningDisabledEnv])
	if value == "" {
		value = strings.TrimSpace(os.Getenv(sharedPackageProvisioningDisabledEnv))
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func sharedPackageDependencies(repoRoot, uiPackageJSON string) ([]sharedPackageDependency, error) {
	specs, err := fileDependenciesWithDeps(uiPackageJSON, defaultHostProbeDeps())
	if err != nil {
		return nil, err
	}
	uiRoot := filepath.Dir(uiPackageJSON)
	result := make([]sharedPackageDependency, 0, len(specs))
	seen := make(map[string]struct{})
	for _, spec := range specs {
		dependencyRoot := resolveCheckPath(uiRoot, strings.TrimPrefix(spec.Spec, "file:"))
		pkg, packageRoot, ok, err := loadGovernedPackage(repoRoot, dependencyRoot)
		if err != nil {
			return nil, fmt.Errorf("dependency %q: %w", spec.Name, err)
		}
		if !ok || !declaresModuleIdentifier(pkg, spec.Name) {
			continue
		}
		if _, exists := seen[packageRoot]; exists {
			continue
		}
		seen[packageRoot] = struct{}{}
		result = append(result, sharedPackageDependency{
			Name:       spec.Name,
			Root:       packageRoot,
			Package:    pkg,
			Generation: append([]packagegov.CommandSpec(nil), pkg.Manifest.Package.Lifecycle.Generate...),
			Build:      append([]packagegov.CommandSpec(nil), pkg.Manifest.Package.Lifecycle.Build...),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func loadGovernedPackage(repoRoot, dependencyRoot string) (packagegov.Package, string, bool, error) {
	packagesRoot := filepath.Join(repoRoot, "packages")
	dependencyRoot = filepath.Clean(dependencyRoot)
	if !pathUnderRoot(packagesRoot, dependencyRoot) || dependencyRoot == packagesRoot {
		return packagegov.Package{}, "", false, nil
	}

	for candidate := dependencyRoot; pathUnderRoot(packagesRoot, candidate) && candidate != packagesRoot; candidate = filepath.Dir(candidate) {
		manifestPath := filepath.Join(candidate, ".vrooli", "package.json")
		if _, err := os.Stat(manifestPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return packagegov.Package{}, "", false, err
		}
		pkg, _, err := packagegov.LoadPackage(candidate)
		if err != nil {
			return packagegov.Package{}, "", false, err
		}
		return pkg, candidate, true, nil
	}
	return packagegov.Package{}, "", false, nil
}

func declaresModuleIdentifier(pkg packagegov.Package, identifier string) bool {
	for _, candidate := range pkg.Manifest.Package.ModuleIdentifiers {
		if candidate == identifier {
			return true
		}
	}
	for _, output := range pkg.Manifest.Package.GeneratedOutputs {
		for _, candidate := range output.Identifiers {
			if candidate == identifier {
				return true
			}
		}
	}
	return false
}

func provisionSharedPackage(dependency sharedPackageDependency, stdout, logWriter io.Writer) error {
	return provisionSharedPackageWithOptions(dependency, stdout, logWriter, sharedPackageProvisionOptions{})
}

func provisionSharedPackageWithOptions(dependency sharedPackageDependency, stdout, logWriter io.Writer, options sharedPackageProvisionOptions) error {
	commands := append([]packagegov.CommandSpec{}, dependency.Generation...)
	commands = append(commands, dependency.Build...)
	if len(commands) == 0 {
		return &SharedPackageProvisioningError{
			PackageName: dependency.Name,
			Command:     "<declared lifecycle>",
			Reason:      "has no generate or build lifecycle",
		}
	}
	var release func()
	if strings.TrimSpace(options.Home) != "" {
		var err error
		release, err = acquireSharedPackageLock(options.Home, dependency.Name, dependency.Root, logWriter)
		if err != nil {
			return &SharedPackageProvisioningError{
				PackageName: dependency.Name,
				Command:     "shared package lock",
				Reason:      "could not acquire",
				Err:         err,
			}
		}
		defer release()
	}

	for _, command := range commands {
		commandText := strings.Join(command.Run, " ")
		if len(command.Outputs) == 0 {
			return &SharedPackageProvisioningError{
				PackageName: dependency.Name,
				Command:     commandText,
				Reason:      "declares no build outputs",
			}
		}
		fresh, err := sharedPackageOutputsFresh(dependency.Root, command.Outputs)
		if err != nil {
			return &SharedPackageProvisioningError{PackageName: dependency.Name, Command: commandText, Reason: "could not inspect declared outputs", Err: err}
		}
		if fresh {
			_, _ = fmt.Fprintf(logWriter, "shared package %s: %s outputs are fresh\n", dependency.Name, command.Name)
			continue
		}

		startedAt := time.Now()
		_, _ = fmt.Fprintf(logWriter, "shared-package-command event=start package=%q command=%q root=%q pid=%d\n", dependency.Name, commandText, dependency.Root, os.Getpid())
		err = packagegov.RunCommandsWithOptions(dependency.Root, []packagegov.CommandSpec{command}, stdout, stdout, packagegov.CommandOptions{
			Env:   options.Env,
			Stdin: options.Stdin,
		})
		_, _ = fmt.Fprintf(logWriter, "shared-package-command event=end package=%q command=%q root=%q pid=%d duration_ms=%d status=%q\n", dependency.Name, commandText, dependency.Root, os.Getpid(), time.Since(startedAt).Milliseconds(), commandStatus(err))
		if err != nil {
			return &SharedPackageProvisioningError{PackageName: dependency.Name, Command: commandText, Reason: "failed", Err: err}
		}
		files, err := declaredOutputFiles(dependency.Root, command.Outputs)
		if err != nil {
			return &SharedPackageProvisioningError{PackageName: dependency.Name, Command: commandText, Reason: "could not inspect outputs after provisioning", Err: err}
		}
		if len(files) == 0 {
			return &SharedPackageProvisioningError{PackageName: dependency.Name, Command: commandText, Reason: "completed without producing a declared output"}
		}
	}
	return nil
}

func commandStatus(err error) string {
	if err != nil {
		return "failed"
	}
	return "completed"
}

func firstProvisioningCommand(dependency sharedPackageDependency) string {
	commands := append(append([]packagegov.CommandSpec{}, dependency.Generation...), dependency.Build...)
	if len(commands) == 0 {
		return "<declared lifecycle>"
	}
	for _, command := range commands {
		if strings.EqualFold(strings.TrimSpace(command.Name), "build") {
			return strings.Join(command.Run, " ")
		}
	}
	return strings.Join(commands[0].Run, " ")
}

func sharedPackageOutputsFresh(root string, patterns []string) (bool, error) {
	outputs, err := declaredOutputFiles(root, patterns)
	if err != nil {
		return false, err
	}
	if len(outputs) == 0 {
		return false, nil
	}
	var newestSource, oldestOutput int64
	for _, output := range outputs {
		info, err := os.Stat(output)
		if err != nil {
			return false, err
		}
		mtime := info.ModTime().UnixNano()
		if oldestOutput == 0 || mtime < oldestOutput {
			oldestOutput = mtime
		}
	}
	if err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if filePath != root && shouldSkipSharedPackageDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || sharedPackageOutputMatch(root, filePath, patterns) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if mtime := info.ModTime().UnixNano(); mtime > newestSource {
			newestSource = mtime
		}
		return nil
	}); err != nil {
		return false, err
	}
	return oldestOutput >= newestSource, nil
}

func declaredOutputFiles(root string, patterns []string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, pattern := range patterns {
		pattern = filepath.Clean(filepath.FromSlash(pattern))
		if filepath.IsAbs(pattern) || pattern == ".." || strings.HasPrefix(pattern, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("output pattern %q must be relative", pattern)
		}
		if strings.Contains(filepath.ToSlash(pattern), "/**") {
			prefix := strings.SplitN(filepath.ToSlash(pattern), "/**", 2)[0]
			base := filepath.Join(root, filepath.FromSlash(prefix))
			if err := filepath.WalkDir(base, func(filePath string, entry fs.DirEntry, err error) error {
				if err != nil {
					if os.IsNotExist(err) {
						return nil
					}
					return err
				}
				if entry.Type().IsRegular() {
					seen[filepath.Clean(filePath)] = struct{}{}
				}
				return nil
			}); err != nil {
				return nil, err
			}
			continue
		}
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			info, statErr := os.Stat(match)
			if statErr != nil {
				return nil, statErr
			}
			if info.Mode().IsRegular() {
				seen[filepath.Clean(match)] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for filePath := range seen {
		result = append(result, filePath)
	}
	sort.Strings(result)
	return result, nil
}

func shouldSkipSharedPackageDir(name string) bool {
	switch name {
	case ".git", "node_modules", "coverage", ".vite", "dist":
		return true
	default:
		return false
	}
}

func sharedPackageOutputMatch(root, filePath string, patterns []string) bool {
	rel, err := filepath.Rel(root, filePath)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, raw := range patterns {
		pattern := filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw)))
		if prefix, ok := strings.CutSuffix(pattern, "/**"); ok {
			if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
			continue
		}
		if matched, _ := path.Match(pattern, rel); matched {
			return true
		}
	}
	return false
}

func sharedPackageOutputDigest(root string, patterns []string) (string, error) {
	files, err := declaredOutputFiles(root, patterns)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	for _, filePath := range files {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return "", err
		}
		_, _ = fmt.Fprintf(hash, "%s\x00", filepath.ToSlash(rel))
		_, _ = hash.Write(data)
	}
	if len(files) == 0 {
		return "missing", nil
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}
