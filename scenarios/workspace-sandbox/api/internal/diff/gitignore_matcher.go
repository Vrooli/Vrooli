package diff

import (
	"context"
	"strings"
)

// GitIgnoreMatcher applies repository ignore policy without making the
// changedetect leaf package depend on git. Failures intentionally return an
// empty set: capture is safer than silently losing agent work.
type GitIgnoreMatcher struct {
	projectRoot string
	runner      CommandRunner
}

func NewGitIgnoreMatcher(projectRoot string, runner CommandRunner) *GitIgnoreMatcher {
	return &GitIgnoreMatcher{projectRoot: projectRoot, runner: runner}
}

func (m *GitIgnoreMatcher) Ignored(ctx context.Context, paths []string) (map[string]struct{}, error) {
	ignored := make(map[string]struct{})
	if m == nil || m.projectRoot == "" || m.runner == nil || len(paths) == 0 {
		return ignored, nil
	}
	if check := m.runner.Run(ctx, "", "", "git", "-C", m.projectRoot, "rev-parse", "--is-inside-work-tree"); check.Err != nil {
		return ignored, nil
	}
	const chunkSize = 200
	for start := 0; start < len(paths); start += chunkSize {
		end := start + chunkSize
		if end > len(paths) {
			end = len(paths)
		}
		chunk := paths[start:end]
		args := append([]string{"git", "-C", m.projectRoot, "check-ignore", "-z", "--"}, chunk...)
		result := m.runner.Run(ctx, "", "", args...)
		if result.Err != nil && result.ExitCode != 1 {
			return map[string]struct{}{}, nil
		}
		for _, p := range nulPaths(result.Stdout) {
			ignored[p] = struct{}{}
		}
		trackedArgs := append([]string{"git", "-C", m.projectRoot, "ls-files", "-z", "--"}, chunk...)
		tracked := m.runner.Run(ctx, "", "", trackedArgs...)
		if tracked.Err != nil {
			return map[string]struct{}{}, nil
		}
		for _, p := range nulPaths(tracked.Stdout) {
			delete(ignored, p)
		}
	}
	return ignored, nil
}

func nulPaths(output string) []string {
	paths := strings.Split(output, "\x00")
	if len(paths) > 0 && paths[len(paths)-1] == "" {
		paths = paths[:len(paths)-1]
	}
	return paths
}
