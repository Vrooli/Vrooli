package lifecycle

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"

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
		fresh, err := sharedPackageOutputsFresh(options.Home, dependency.Root, command.Name, command.Outputs)
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
		// Record the sources behind these outputs so an unchanged package is
		// not regenerated on the next start. Failing to record is not fatal —
		// the cost is only regenerating again — but it must never be silent,
		// because a stamp that is never written looks exactly like a cache
		// that never helps.
		if err := recordSharedPackageStamp(options.Home, dependency, command); err != nil {
			_, _ = fmt.Fprintf(logWriter, "shared package %s: could not record %s freshness stamp: %v\n", dependency.Name, command.Name, err)
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

// sharedPackageOutputsFresh reports whether a package's declared outputs
// already reflect its current sources, so the generating command can be
// skipped.
//
// Freshness is decided by hashing sources, not by comparing mtimes. A
// generator's publish step is typically content-addressed — it does not
// rewrite a file whose bytes are unchanged — so an output's mtime can stay
// frozen for weeks while the package is regenerated on every start. An mtime
// comparison then reports "stale" forever, and does so most stubbornly for
// the packages whose publish works best. Hashing also survives a fresh clone,
// where checkout order rather than authorship decides mtimes, and filesystems
// whose mtime granularity is coarser than a build.
//
// Scope: the digest covers files inside the package root only. A generator
// whose toolchain is pinned outside the package (a globally installed plugin
// binary, say) is not covered; upgrading such a tool needs an explicit
// regeneration. Output integrity is likewise out of scope — that is what a
// generator's own verify command and its lock manifests are for.
func sharedPackageOutputsFresh(home, root, commandName string, patterns []string) (bool, error) {
	outputs, err := declaredOutputFiles(root, patterns)
	if err != nil {
		return false, err
	}
	// No outputs on disk means nothing has been generated yet, whatever a
	// stamp claims.
	if len(outputs) == 0 {
		return false, nil
	}
	stampPath, err := sharedPackageStampPath(home, root, commandName)
	if err != nil {
		return false, err
	}
	// Without a runtime home there is nowhere to record what produced the
	// outputs, so the only safe answer is to regenerate.
	if stampPath == "" {
		return false, nil
	}
	stamp, found, err := readSharedPackageStamp(stampPath)
	if err != nil || !found {
		return false, err
	}
	if stamp.Version != sharedPackageStampVersion {
		return false, nil
	}
	if sharedPackageOutputsListDigest(root, outputs) != stamp.OutputsDigest {
		return false, nil
	}
	digest, err := sharedPackageSourceDigest(root, patterns)
	if err != nil {
		return false, err
	}
	return digest == stamp.SourceDigest, nil
}

// sharedPackageOutputsListDigest hashes the sorted relative paths of the
// declared outputs. declaredOutputFiles already returns them sorted, so this
// is stable, and it reads no file contents.
func sharedPackageOutputsListDigest(root string, outputs []string) string {
	hash := sha256.New()
	for _, filePath := range outputs {
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			rel = filePath
		}
		_, _ = fmt.Fprintf(hash, "%s\n", filepath.ToSlash(rel))
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

// sharedPackageSourceDigest hashes every regular file under root that is not a
// declared output. Path and content both feed the hash so a rename is a
// change, and WalkDir's lexical order makes the result stable across runs and
// machines.
func sharedPackageSourceDigest(root string, outputPatterns []string) (string, error) {
	hash := sha256.New()
	err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if filePath != root && shouldSkipSharedPackageDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || sharedPackageOutputMatch(root, filePath, outputPatterns) {
			return nil
		}
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00", filepath.ToSlash(rel), len(data))
		_, _ = hash.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

// sharedPackageStampVersion invalidates every recorded stamp when the meaning
// of the digest changes. Bump it alongside any change to what is hashed.
const sharedPackageStampVersion = 2

// sharedPackageStamp records which sources produced a package's outputs.
type sharedPackageStamp struct {
	Version int    `json:"version"`
	Package string `json:"package"`
	Command string `json:"command"`
	// SourceDigest identifies the inputs that produced the outputs.
	SourceDigest string `json:"source_digest"`
	// OutputsDigest identifies the *set* of files produced — paths only, not
	// content. Generators publish by mirroring a staging tree, which deletes
	// target files the staging tree lacks, so a partial or interrupted publish
	// can silently drop outputs. Without this, unchanged sources would report
	// fresh forever and the gap would never heal. Hashing paths rather than
	// bytes keeps the check cheap on a large generated tree.
	OutputsDigest string `json:"outputs_digest"`
	RecordedAt    string `json:"recorded_at"`
}

// sharedPackageStampPath locates the stamp for one package command. The file
// name is a digest rather than the package name because package names carry
// characters ("@", "/") that are not portable in a path. Returns "" when no
// runtime home is available.
func sharedPackageStampPath(home, root, commandName string) (string, error) {
	if strings.TrimSpace(home) == "" {
		return "", nil
	}
	// home is the user's home directory; the runtime home lives beneath it and
	// its layout is owned by the repo contract, not by this file.
	cacheDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyCache)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	key := sha256.Sum256([]byte(filepath.ToSlash(canonical) + "\x00" + commandName))
	return filepath.Join(cacheDir, "shared-packages", fmt.Sprintf("%x.json", key[:16])), nil
}

// readSharedPackageStamp reads a stamp. A missing or unreadable stamp is not
// an error: it simply means the package must be regenerated.
func readSharedPackageStamp(stampPath string) (sharedPackageStamp, bool, error) {
	data, err := os.ReadFile(stampPath)
	if errors.Is(err, os.ErrNotExist) {
		return sharedPackageStamp{}, false, nil
	}
	if err != nil {
		return sharedPackageStamp{}, false, err
	}
	var stamp sharedPackageStamp
	if err := json.Unmarshal(data, &stamp); err != nil {
		// A corrupt stamp must never be load-bearing; regenerate instead.
		return sharedPackageStamp{}, false, nil
	}
	return stamp, true, nil
}

// recordSharedPackageStamp captures the current source digest for a command
// that has just produced its declared outputs.
func recordSharedPackageStamp(home string, dependency sharedPackageDependency, command packagegov.CommandSpec) error {
	stampPath, err := sharedPackageStampPath(home, dependency.Root, command.Name)
	if err != nil {
		return err
	}
	if stampPath == "" {
		return fmt.Errorf("no runtime home is configured, so freshness cannot be cached")
	}
	digest, err := sharedPackageSourceDigest(dependency.Root, command.Outputs)
	if err != nil {
		return fmt.Errorf("hash sources: %w", err)
	}
	outputs, err := declaredOutputFiles(dependency.Root, command.Outputs)
	if err != nil {
		return fmt.Errorf("list outputs: %w", err)
	}
	return writeSharedPackageStamp(stampPath, dependency.Name, command.Name, digest, sharedPackageOutputsListDigest(dependency.Root, outputs))
}

// writeSharedPackageStamp records the sources that produced the current
// outputs. It is written only after a command succeeds and its outputs are
// confirmed present.
func writeSharedPackageStamp(stampPath, packageName, commandName, digest, outputsDigest string) error {
	if stampPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(stampPath), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(sharedPackageStamp{
		Version:       sharedPackageStampVersion,
		Package:       packageName,
		Command:       commandName,
		SourceDigest:  digest,
		OutputsDigest: outputsDigest,
		RecordedAt:    time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	return os.WriteFile(stampPath, data, 0o644)
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
