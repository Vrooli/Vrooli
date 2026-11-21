package exec

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

func validatePolicyFiles(s *types.Sandbox, cfg BwrapConfig) error {
	if cfg.SharePID {
		return fmt.Errorf("policy files require a private PID namespace")
	}
	var aliases []string
	for _, file := range cfg.PolicyFiles {
		for _, path := range []string{file.Source, file.Target} {
			if !filepath.IsAbs(path) || filepath.Clean(path) != path || path == "/" {
				return fmt.Errorf("policy file paths must be clean absolute file paths")
			}
			for _, root := range append(workspaceWritePolicyRoots(s), s.LowerDir, "/workspace-readonly", "/proc", "/dev") {
				if pathsOverlap(path, root) {
					return fmt.Errorf("policy file overlaps workspace or runtime filesystem: %q", path)
				}
			}
			for source, target := range cfg.ReadWriteBinds {
				if pathsOverlap(path, source) || pathsOverlap(path, target) {
					return fmt.Errorf("policy file overlaps a writable profile bind: %q", path)
				}
			}
			for _, mask := range cfg.MaskPaths {
				if pathsOverlap(path, mask) {
					return fmt.Errorf("policy file overlaps masked path: %q", path)
				}
			}
			for _, other := range aliases {
				if pathsOverlap(path, other) {
					return fmt.Errorf("policy file aliases overlap: %q and %q", path, other)
				}
			}
		}
		aliases = append(aliases, file.Source, file.Target)
		resolved, err := filepath.EvalSymlinks(file.Source)
		if err != nil || resolved != file.Source {
			return fmt.Errorf("policy source must exist without symlink components: %q", file.Source)
		}
		info, err := os.Lstat(file.Source)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("policy source must be a regular file: %q", file.Source)
		}
		expected, err := hex.DecodeString(file.SHA256)
		if err != nil || len(expected) != sha256.Size {
			return fmt.Errorf("policy file requires a SHA-256 content identity")
		}
		input, err := os.Open(file.Source)
		if err != nil {
			return err
		}
		digest := sha256.New()
		_, err = io.Copy(digest, input)
		closeErr := input.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if !strings.EqualFold(hex.EncodeToString(digest.Sum(nil)), file.SHA256) {
			return fmt.Errorf("policy file content identity changed: %q", file.Source)
		}
	}
	return nil
}

// workspaceWritePolicyRoots includes aliases even when profile mirroring is
// disabled: the home overlay can otherwise expose an independently writable
// copy at ProjectRoot. All aliases must see the same constrained workspace.
func workspaceWritePolicyRoots(s *types.Sandbox) []string {
	roots := []string{driver.NamespaceWorkspacePath}
	for _, path := range []string{s.MergedDir, s.ProjectRoot} {
		if path != "" && !slices.Contains(roots, filepath.Clean(path)) {
			roots = append(roots, filepath.Clean(path))
		}
	}
	sort.Strings(roots)
	return roots
}

// Validate physical grants immediately before every execution, including
// continuations. Missing paths, symlink components and writable profile aliases
// refuse launch rather than silently widening a persisted owner policy.
func validateWorkspaceWritePolicy(s *types.Sandbox, cfg BwrapConfig) error {
	if cfg.SharePID {
		return fmt.Errorf("workspace write policy requires a private PID namespace")
	}
	policy := s.Behavior.WritePolicy
	if err := policy.Validate(); err != nil {
		return err
	}
	roots := workspaceWritePolicyRoots(s)
	for i, root := range roots {
		if !filepath.IsAbs(root) || root == "/" {
			return fmt.Errorf("invalid workspace alias %q", root)
		}
		for _, other := range roots[:i] {
			if pathsOverlap(root, other) {
				return fmt.Errorf("overlapping workspace aliases %q and %q", root, other)
			}
		}
	}
	for src, dst := range cfg.ReadWriteBinds {
		for _, root := range roots {
			if pathsOverlap(src, root) || pathsOverlap(dst, root) {
				return fmt.Errorf("writable profile bind %q -> %q overlaps constrained workspace", src, dst)
			}
		}
	}
	for _, path := range policy.Paths {
		full := filepath.Join(s.MergedDir, path)
		resolved, err := filepath.EvalSymlinks(full)
		if err != nil {
			return fmt.Errorf("resolve write grant %q: %w", path, err)
		}
		if resolved != full {
			return fmt.Errorf("write grant %q traverses a symlink", path)
		}
		info, err := os.Stat(full)
		if err != nil {
			return fmt.Errorf("stat write grant %q: %w", path, err)
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("write grant %q is not a regular file or directory", path)
		}
	}
	return nil
}
