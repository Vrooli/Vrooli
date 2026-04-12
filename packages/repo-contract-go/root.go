package repocontract

import (
	"os"
	"path/filepath"
	"strings"
)

// FindRepoRoot locates the nearest ancestor that contains a valid repo contract
// and matches its required root markers.
func FindRepoRoot(start string) (string, error) {
	start = strings.TrimSpace(start)
	if start == "" {
		return "", &Error{Kind: ErrInvalidInput, Message: "start path is required"}
	}

	current := filepath.Clean(start)
	info, err := os.Stat(current)
	if err != nil {
		return "", &Error{Kind: ErrNotFound, Message: "stat start path", Details: start, Err: err}
	}
	if !info.IsDir() {
		current = filepath.Dir(current)
	}

	for {
		contractPath := filepath.Join(current, filepath.FromSlash(defaultContractRelPath))
		if _, err := os.Stat(contractPath); err == nil {
			contract, loadErr := Load(contractPath)
			if loadErr != nil {
				return "", loadErr
			}
			ok, matchErr := candidateMatchesRootMarkers(current, contract.RootMarkers())
			if matchErr != nil {
				return "", matchErr
			}
			if ok {
				return current, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", &Error{Kind: ErrNotFound, Message: "repo root not found", Details: start}
}

func candidateMatchesRootMarkers(candidate string, markers RootMarkers) (bool, error) {
	for _, dir := range markers.RequiredDirs {
		path := filepath.Join(candidate, filepath.FromSlash(dir))
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, &Error{Kind: ErrNotFound, Message: "stat required dir", Details: path, Err: err}
		}
		if !info.IsDir() {
			return false, nil
		}
	}
	for _, file := range markers.RequiredFiles {
		path := filepath.Join(candidate, filepath.FromSlash(file))
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, &Error{Kind: ErrNotFound, Message: "stat required file", Details: path, Err: err}
		}
		if info.IsDir() {
			return false, nil
		}
	}
	return true, nil
}
