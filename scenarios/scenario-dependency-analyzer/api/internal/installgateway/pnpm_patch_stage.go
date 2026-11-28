package installgateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxPnpmPatchFiles      = 32
	maxPnpmPatchFileBytes  = 2 << 20
	maxPnpmPatchTotalBytes = 4 << 20
)

type pnpmPatchManifestJSON struct {
	Pnpm struct {
		PatchedDependencies map[string]string `json:"patchedDependencies"`
	} `json:"pnpm"`
}

type stagedPnpmPatch struct {
	relativePath string
	content      []byte
}

func readPnpmPatchInputs(surfaceRoot string, manifest []byte) ([]stagedPnpmPatch, int64, error) {
	var parsed pnpmPatchManifestJSON
	if err := json.Unmarshal(manifest, &parsed); err != nil {
		return nil, 0, fmt.Errorf("read pnpm patchedDependencies from package.json: %w", err)
	}
	declared := parsed.Pnpm.PatchedDependencies
	if len(declared) == 0 {
		return nil, 0, nil
	}
	if len(declared) > maxPnpmPatchFiles {
		return nil, 0, fmt.Errorf("pnpm patchedDependencies declares %d files; limit is %d", len(declared), maxPnpmPatchFiles)
	}

	paths := make([]string, 0, len(declared))
	seen := make(map[string]struct{}, len(declared))
	for _, raw := range declared {
		relativePath, err := cleanPnpmPatchPath(raw)
		if err != nil {
			return nil, 0, err
		}
		if _, exists := seen[relativePath]; exists {
			continue
		}
		seen[relativePath] = struct{}{}
		paths = append(paths, relativePath)
	}
	sort.Strings(paths)

	patches := make([]stagedPnpmPatch, 0, len(paths))
	var total int64
	for _, relativePath := range paths {
		content, err := readLocalPnpmPatch(surfaceRoot, relativePath)
		if err != nil {
			return nil, 0, err
		}
		if int64(len(content)) > maxPnpmPatchTotalBytes-total {
			return nil, 0, fmt.Errorf("pnpm patch files exceed the %d-byte aggregate limit", maxPnpmPatchTotalBytes)
		}
		total += int64(len(content))
		patches = append(patches, stagedPnpmPatch{relativePath: relativePath, content: content})
	}
	return patches, total, nil
}

func cleanPnpmPatchPath(raw string) (string, error) {
	if raw == "" || strings.ContainsRune(raw, '\x00') || strings.Contains(raw, `\`) || pathpkg.IsAbs(raw) || (len(raw) >= 2 && raw[1] == ':') {
		return "", fmt.Errorf("pnpm patch path %q must be a confined relative file path", raw)
	}
	clean := pathpkg.Clean(raw)
	if clean == "." || clean == ".." || clean != raw || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("pnpm patch path %q must not contain traversal or non-canonical components", raw)
	}
	for _, component := range strings.Split(raw, "/") {
		if component == "" || component == "." || component == ".." {
			return "", fmt.Errorf("pnpm patch path %q must not contain traversal or empty components", raw)
		}
	}
	if clean == "package.json" || clean == "pnpm-lock.yaml" || clean == "pnpm-workspace.yaml" || clean == ".npmrc" || strings.HasPrefix(clean, ".sda-") {
		return "", fmt.Errorf("pnpm patch path %q collides with reserved staging input", raw)
	}
	return clean, nil
}

func readLocalPnpmPatch(surfaceRoot, relativePath string) ([]byte, error) {
	clean, err := cleanPnpmPatchPath(relativePath)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(surfaceRoot)
	if err != nil {
		return nil, fmt.Errorf("open pnpm patch root: %w", err)
	}
	defer root.Close()
	current := ""
	components := strings.Split(clean, "/")
	for index, component := range components {
		current = filepath.Join(current, component)
		info, err := root.Lstat(current)
		if err != nil {
			return nil, fmt.Errorf("inspect pnpm patch %q: %w", relativePath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("pnpm patch path %q contains a symbolic link", relativePath)
		}
		last := index == len(components)-1
		if last && !info.Mode().IsRegular() {
			return nil, fmt.Errorf("pnpm patch path %q must name a regular file", relativePath)
		}
		if !last && !info.IsDir() {
			return nil, fmt.Errorf("pnpm patch path %q has a non-directory parent", relativePath)
		}
	}
	// Re-check the opened file identity so a path swapped between Lstat and
	// Open cannot redirect the bounded read through a symlink or replacement.
	opened, err := root.Open(clean)
	if err != nil {
		return nil, fmt.Errorf("open pnpm patch %q: %w", relativePath, err)
	}
	defer opened.Close()
	openedInfo, err := opened.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("pnpm patch %q changed before it could be read", relativePath)
	}
	pathInfo, err := root.Lstat(clean)
	if err != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(openedInfo, pathInfo) {
		return nil, fmt.Errorf("pnpm patch %q changed before it could be read", relativePath)
	}
	if openedInfo.Size() > maxPnpmPatchFileBytes {
		return nil, fmt.Errorf("pnpm patch %q exceeds the %d-byte limit", relativePath, maxPnpmPatchFileBytes)
	}
	content, err := io.ReadAll(io.LimitReader(opened, maxPnpmPatchFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read pnpm patch %q: %w", relativePath, err)
	}
	if int64(len(content)) > maxPnpmPatchFileBytes {
		return nil, fmt.Errorf("pnpm patch %q exceeds the %d-byte limit", relativePath, maxPnpmPatchFileBytes)
	}
	return content, nil
}

func stagePnpmPatchFiles(stageDir string, patches []stagedPnpmPatch) error {
	for _, patch := range patches {
		destination := filepath.Join(stageDir, filepath.FromSlash(patch.relativePath))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return fmt.Errorf("create staged pnpm patch directory for %q: %w", patch.relativePath, err)
		}
		if err := os.WriteFile(destination, patch.content, 0o600); err != nil {
			return fmt.Errorf("stage pnpm patch %q: %w", patch.relativePath, err)
		}
	}
	return nil
}

func (s *JSStage) verifyPnpmPatchInputsUnchanged() error {
	for _, patch := range s.pnpmPatches {
		current, err := readLocalPnpmPatch(s.surfaceRoot, patch.relativePath)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, patch.content) {
			return fmt.Errorf("pnpm patch %q changed after dependency resolution; nothing was published", patch.relativePath)
		}
	}
	return nil
}
