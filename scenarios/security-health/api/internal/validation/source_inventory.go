package validation

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// commitEligibleFiles returns tracked plus non-ignored untracked files. That is
// the security boundary gitleaks gates: content which can enter a commit. It
// excludes installed dependencies, build outputs, runtime databases, and local
// secret files by the repository's own ignore policy.
func commitEligibleFiles(ctx context.Context, cmd Commander, scenarioDir string) ([]string, error) {
	stdout, _, exitCode, err := cmd.Run(ctx, scenarioDir, "git", "ls-files", "-c", "-o", "--exclude-standard", "-z", "--", ".")
	if err == nil && exitCode == 0 {
		var files []string
		for _, raw := range strings.Split(string(stdout), "\x00") {
			rel := filepath.Clean(strings.TrimSpace(raw))
			if rel == "." || rel == "" || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				continue
			}
			path := filepath.Join(scenarioDir, rel)
			info, statErr := os.Lstat(path)
			if statErr != nil || info.IsDir() {
				continue
			}
			files = append(files, path)
		}
		sort.Strings(files)
		return files, nil
	}
	// A standalone scenario fixture may not be in a git worktree. Preserve the
	// same first-party boundary using the repository's generated-dir policy.
	return walkEvidenceFiles(scenarioDir, true)
}

func prepareCommitEligibleSnapshot(ctx context.Context, cmd Commander, scenarioDir string) (string, func(), error) {
	files, err := commitEligibleFiles(ctx, cmd, scenarioDir)
	if err != nil {
		return "", nil, err
	}
	snapshot, err := os.MkdirTemp("", "security-health-gitleaks-")
	if err != nil {
		return "", nil, fmt.Errorf("create gitleaks source snapshot: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(snapshot) }
	for _, source := range files {
		rel, relErr := filepath.Rel(scenarioDir, source)
		if relErr != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			cleanup()
			return "", nil, fmt.Errorf("invalid gitleaks source path %q", source)
		}
		destination := filepath.Join(snapshot, rel)
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("create gitleaks snapshot directory: %w", err)
		}
		info, err := os.Lstat(source)
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("stat gitleaks source %q: %w", source, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, readErr := os.Readlink(source)
			if readErr != nil {
				cleanup()
				return "", nil, fmt.Errorf("read gitleaks symlink %q: %w", source, readErr)
			}
			if err := os.WriteFile(destination, []byte(target), 0o600); err != nil {
				cleanup()
				return "", nil, fmt.Errorf("stage gitleaks symlink %q: %w", source, err)
			}
			continue
		}
		if err := os.Link(source, destination); err == nil {
			continue
		}
		if err := copySnapshotFile(source, destination, info.Mode().Perm()); err != nil {
			cleanup()
			return "", nil, err
		}
	}
	return snapshot, cleanup, nil
}

func copySnapshotFile(source, destination string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open gitleaks source %q: %w", source, err)
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create gitleaks snapshot file %q: %w", destination, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy gitleaks source %q: %w", source, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close gitleaks snapshot file %q: %w", destination, err)
	}
	return nil
}
