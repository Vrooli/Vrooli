package modeltest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ignoredHashDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"dist":          true,
	"build":         true,
	"coverage":      true,
	"_apalache-out": true,
}

func validateFreshHash(filePath string, wantHash string, field string) error {
	if wantHash == "" {
		return fmt.Errorf("formal artifact %s is required", field)
	}
	hash, err := repoPathSHA256(filePath)
	if err != nil {
		return fmt.Errorf("read formal artifact source %s: %v", filePath, err)
	}
	if hash != wantHash {
		return fmt.Errorf("formal artifact %s=%s, want %s", field, wantHash, hash)
	}
	return nil
}

func repoPathSHA256(repoPath string) (string, error) {
	abs, err := findRepoPath(repoPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(abs)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		return hex.EncodeToString(sum[:]), nil
	}
	return repoTreeSHA256(abs)
}

func repoTreeSHA256(root string) (string, error) {
	var parts []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if ignoredHashDirs[entry.Name()] && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if strings.HasPrefix(slash, "testdata/") || strings.HasSuffix(slash, "_test.go") || strings.HasSuffix(slash, ".formal.generated.json") || strings.HasSuffix(slash, ".qnt") {
			return nil
		}
		if !(strings.HasSuffix(slash, ".go") || slash == "go.mod" || strings.HasSuffix(slash, ".schema.json")) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		parts = append(parts, slash+"\x00"+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func findRepoPath(repoPath string) (string, error) {
	if filepath.IsAbs(repoPath) {
		if _, err := os.Stat(repoPath); err != nil {
			return "", err
		}
		return repoPath, nil
	}
	var firstErr error
	candidates := repoFileCandidates(repoPath)
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		for _, candidate := range candidates {
			abs := filepath.Join(dir, candidate)
			if _, err := os.Stat(abs); err == nil {
				return abs, nil
			} else if firstErr == nil {
				firstErr = err
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", firstErr
}

func repoFileCandidates(repoPath string) []string {
	candidates := []string{repoPath}
	if trimmed, ok := strings.CutPrefix(repoPath, "api/"); ok {
		candidates = append(candidates, trimmed)
	}
	candidates = append(candidates, filepath.Base(repoPath))
	return candidates
}
