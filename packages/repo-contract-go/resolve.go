package repocontract

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

const (
	defaultRepoRootEnvVar   = "VROOLI_ROOT"
	defaultSourceRootEnvVar = "VROOLI_SOURCE_ROOT"
)

var (
	getwdPath      = os.Getwd
	executablePath = os.Executable
)

// FindRepoRootFromPath resolves a repo root from any descendant path.
func FindRepoRootFromPath(start string) (string, error) {
	return FindRepoRoot(start)
}

// FindRepoRootFromCWD resolves a repo root from the current working directory.
func FindRepoRootFromCWD() (string, error) {
	cwd, err := getwdPath()
	if err != nil {
		return "", &Error{Kind: ErrNotFound, Message: "resolve current working directory", Err: err}
	}
	return FindRepoRoot(cwd)
}

// ResolveRepoRoot resolves the active repo root using canonical environment and
// process-context discovery rules.
func ResolveRepoRoot() (string, error) {
	return FindRepoRootFromEnvOrCWD()
}

// FindRepoRootFromEnvOrCWD resolves the repo root using canonical environment
// variables first, then the current working directory, then the executable path.
func FindRepoRootFromEnvOrCWD() (string, error) {
	for _, key := range []string{defaultSourceRootEnvVar, defaultRepoRootEnvVar} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			root, err := FindRepoRoot(value)
			if err == nil {
				return root, nil
			}
		}
	}

	if root, err := FindRepoRootFromCWD(); err == nil {
		return root, nil
	}

	executable, err := executablePath()
	if err != nil {
		return "", &Error{Kind: ErrNotFound, Message: "resolve executable path", Err: err}
	}
	return FindRepoRoot(filepath.Dir(executable))
}

// LoadDefaultFromEnvOrCWD resolves the active repo root and loads the canonical
// repo contract from it.
func LoadDefaultFromEnvOrCWD() (*Contract, string, error) {
	root, err := FindRepoRootFromEnvOrCWD()
	if err != nil {
		return nil, "", err
	}
	contract, err := LoadDefault(root)
	if err != nil {
		return nil, "", err
	}
	return contract, root, nil
}

// ResolveScenarioPath resolves a canonical scenario root for the given repo.
func ResolveScenarioPath(repoRoot, scenario string) (string, error) {
	contract, err := LoadDefault(repoRoot)
	if err != nil {
		return "", err
	}
	return contract.ScenarioRoot(repoRoot, scenario)
}

// ResolveScenarioFile resolves a canonical well-known scenario file/path for
// the given repo.
func ResolveScenarioFile(repoRoot, scenario, key string) (string, error) {
	contract, err := LoadDefault(repoRoot)
	if err != nil {
		return "", err
	}
	return contract.ScenarioFile(repoRoot, scenario, key)
}

// ResolveResourcePath resolves a canonical resource root for the given repo.
func ResolveResourcePath(repoRoot, resource string) (string, error) {
	contract, err := LoadDefault(repoRoot)
	if err != nil {
		return "", err
	}
	return contract.ResourceRoot(repoRoot, resource)
}

// ResolveResourceFile resolves a canonical well-known resource file/path for
// the given repo.
func ResolveResourceFile(repoRoot, resource, key string) (string, error) {
	contract, err := LoadDefault(repoRoot)
	if err != nil {
		return "", err
	}
	return contract.ResourceFile(repoRoot, resource, key)
}

// ResourceExists reports whether the contract-defined resource manifest exists.
func ResourceExists(repoRoot, resource string) (bool, error) {
	manifestPath, err := ResolveResourceFile(repoRoot, resource, "manifest")
	if err != nil {
		return false, err
	}
	info, err := statPath(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &Error{Kind: ErrNotFound, Message: "stat resource manifest", Details: manifestPath, Err: err}
	}
	return !info.IsDir(), nil
}

// ScenarioExists reports whether the contract-defined scenario manifest exists.
func ScenarioExists(repoRoot, scenario string) (bool, error) {
	servicePath, err := ResolveScenarioFile(repoRoot, scenario, "service")
	if err != nil {
		return false, err
	}
	info, err := statPath(servicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &Error{Kind: ErrNotFound, Message: "stat scenario manifest", Details: servicePath, Err: err}
	}
	return !info.IsDir(), nil
}

// FileMatchCount counts repository files that match a root-relative repo glob.
func FileMatchCount(repoRoot, pattern string) (int, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return 0, &Error{Kind: ErrInvalidInput, Message: "repo root is required"}
	}
	if err := ValidateRepoGlob(pattern); err != nil {
		return 0, err
	}

	fullPattern := filepath.Join(filepath.Clean(repoRoot), filepath.FromSlash(filepathToSlashTrimmed(pattern)))
	matches, err := doublestar.FilepathGlob(fullPattern)
	if err != nil {
		return 0, &Error{Kind: ErrInvalidInput, Message: "invalid glob syntax", Details: pattern, Err: err}
	}
	count := 0
	for _, match := range matches {
		info, err := statPath(match)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, &Error{Kind: ErrNotFound, Message: "stat glob match", Details: match, Err: err}
		}
		if !info.IsDir() {
			count++
		}
	}
	return count, nil
}
