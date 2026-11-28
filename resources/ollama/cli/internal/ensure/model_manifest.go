package ensure

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InstalledManifestDigest reads the exact locally installed Ollama manifest
// without starting the daemon. It is an authorization preflight for wake-on-
// demand only; inference still rechecks the live daemon inventory afterward.
func InstalledManifestDigest(modelRoot, modelRef string) (string, error) {
	root, err := filepath.Abs(strings.TrimSpace(modelRoot))
	if err != nil || strings.TrimSpace(modelRoot) == "" {
		return "", errors.New("Ollama model store root must be absolute")
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("Ollama model store root is unavailable or linked")
	}
	repository, tag, err := parseOriginalRef(modelRef)
	if err != nil {
		return "", fmt.Errorf("resolve installed Ollama model identity: %w", err)
	}
	parts := append([]string{root, "manifests", RegistryHost}, strings.Split(repository, "/")...)
	parts = append(parts, tag)
	manifestPath := filepath.Join(parts...)
	rel, err := filepath.Rel(root, manifestPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("Ollama model manifest escapes the model store")
	}
	current := root
	segments := strings.Split(rel, string(filepath.Separator))
	for index, segment := range segments {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return "", fmt.Errorf("inspect installed Ollama model manifest: %w", err)
		}
		if index == len(segments)-1 {
			if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return "", errors.New("installed Ollama model manifest is not a regular file")
			}
		} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("installed Ollama model manifest path contains a linked or invalid directory")
		}
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("read installed Ollama model manifest: %w", err)
	}
	if len(data) == 0 {
		return "", errors.New("installed Ollama model manifest is empty")
	}
	return "sha256:" + sha256Hex(data), nil
}
